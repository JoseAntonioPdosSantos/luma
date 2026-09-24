# Assumptions and decisions worth knowing about

This is the engineering log: places the implementation made an explicit
choice, diverged from the original product spec
([`flashcard_learning_system_agent.md`](../flashcard_learning_system_agent.md)),
or added something the spec didn't ask for. If you're new to the
codebase and something looks surprising, check here before assuming it's
a bug.

- **Module path**: the Go module is named `flashcard-backend` (no GitHub
  org/repo existed yet when it was created). Rename it in `go.mod` and
  update imports if this project is published under a different module
  path.
- **No stateful "study session" resource**: the spec sketches
  `POST/GET .../study-sessions`, `.../reviews`, `.../finish`, but also
  explicitly allows a simpler design. There is no session entity: a new
  card's `dueAt` is set to its creation time, so it's already due the
  moment it's created — the single due-cards query already covers "if no
  due cards, offer new cards" without special-casing. `ReviewEvent` itself
  has no session ID in the domain model, confirming this. The frontend
  computes the session summary (cards studied, ratings) purely
  client-side from the sequence of review responses. One consequence:
  there's no server-side guard against a duplicate review submission for
  the same card — resubmitting just reschedules it again.
- **DELETE = archive, not hard delete**: `DELETE /api/v1/decks/{id}` and
  `DELETE /api/v1/flashcards/{id}` archive the resource (`archivedAt` set)
  rather than removing it, preserving review history and any flashcards
  under an archived deck. The spec never calls for permanent deletion.
  Archived items can be listed with `?archived=true` and restored with
  `POST /api/v1/decks/{id}/restore` and `POST /api/v1/flashcards/{id}/restore`.
  Archived items can then be deleted for good with
  `DELETE /api/v1/archived-decks/{id}` and
  `DELETE /api/v1/archived-flashcards/{id}` (active items cannot).
- **Deck groups have no soft delete**: unlike decks/flashcards, deleting a
  deck group is immediate and permanent. There's no data to lose —
  a group is purely an organizational label; deleting it just files its
  decks under no group again, so there was no archive/restore case worth
  building.
- **Mature card threshold**: a REVIEW card is promoted to MATURE once its
  interval reaches 21 days. The spec doesn't set this number; 21 days is a
  common spaced-repetition convention (used here as a placeholder, not a
  claim about optimal learning science).
- **Session tokens**: implemented as a signed (HMAC-SHA256), stateless,
  30-day cookie — no server-side session store. This satisfies the spec's
  "signed, short-lived token" option and keeps auth behind the
  `session.Manager` interface so it can be swapped later.
- **Session rate limiting**: a basic in-memory, per-IP, 10-requests/minute
  limiter on `POST /api/v1/session`. It resets on process restart and does
  not coordinate across instances — adequate for a single-instance
  deployment, not for a multi-instance one.
- **MongoDB driver**: using `go.mongodb.org/mongo-driver` v1 rather than
  the newer v2, because v2 requires Go 1.25+ while this project targets Go
  1.22 — v1 remains fully supported and keeps the toolchain simple.
- **MongoDB port**: docker-compose maps MongoDB to host port 27018 (not
  the default 27017), so it doesn't clash with a MongoDB you may already
  have running locally. Containers still reach it at `mongo:27017`.
- The health endpoint pings MongoDB and reports `"database": "ok"` or
  `"unavailable"`.
- **Audio upload has no multipart form**: the request body is the raw
  audio bytes; the content type comes only from the `Content-Type`
  header, never from a client-supplied filename (the spec warns against
  trusting filenames — this sidesteps that entirely by never accepting
  one). GridFS's own generated file ID is the only reference ever stored.
- **No audio duration**: computing it server-side would need an
  audio-format parsing dependency (MP3/OGG duration isn't in a simple
  header the way WAV's is) for little benefit — the browser's `<audio>`
  element already reports it once played. `DurationMs` is always 0.
- **Audio upload only in edit mode**: the flashcard editor only shows the
  audio upload control once a flashcard exists (i.e., after the first
  save), since audio attaches to an existing flashcard ID via its own
  endpoint. Creating a new card with audio in one step would need a
  two-phase form; instead, save the card, then edit it again to add
  audio.
- **No oversized-file orphan window**: the GridFS adapter aborts the
  upload stream (rather than finishing it and deleting after) the moment
  it detects more than `MAX_AUDIO_SIZE_BYTES` bytes, so a rejected upload
  never briefly exists in storage.
- **Deck-name uniqueness index removed**: the spec's example index list
  includes `decks: userId + name`, to support rejecting duplicate active
  deck names. That rule is explicitly optional, and was never
  implemented — no code queries decks by name — so the unused index was
  removed, per the spec's own instruction not to create an index without
  a backing query. Add it back if duplicate-name rejection is implemented
  later.
- **Flashcard due-date indexes consolidated**: the spec's example index
  list has two separate two-field indexes (`userId + scheduling.dueAt`,
  `deckId + scheduling.dueAt`), but every real query filters by `userId`
  **and** `deckId` **and** `archivedAt` together. A single four-field
  compound index (`userId, deckId, archivedAt, scheduling.dueAt`) serves
  `ListByDeck`, `ListDue`, and `CountByDeck` — MongoDB can use a prefix of
  a compound index — so it replaced the two narrower, unused ones.
- **Structured logging includes error code and resource IDs**: the
  single access-log line per request includes the `errorCode` when the
  request failed and any `deckId`/`flashcardId` from the URL path, not
  just method/path/status/duration.
- **Internationalization (Portuguese, English, Spanish, French)**: the
  frontend detects the browser's language on first visit
  (`i18next-browser-languagedetector`) and falls back to Portuguese.
  Signing in applies the language saved on the account instead, if any;
  the picker in Settings saves an explicit choice back to the account
  (`PUT /api/v1/me/language`) so it follows the user to any device,
  mirroring the `ActiveStudyProfileID` pattern. Every backend error
  response also carries a stable `key` (e.g. `deck.name.tooLong`, see
  `backend/docs/API.md`'s Error format section) alongside the English
  `message`; the frontend looks the key up in `src/i18n/locales/*.json`'s
  flat `errors` map (keyed by the backend's own dotted key, not a nested
  i18next path — some keys, like `request.invalidJson` and
  `request.invalidJson.profileId`, would otherwise collide as both a leaf
  and a parent under normal i18next dot-nesting) via
  `useApiErrorMessage`, falling back to a page-specific translated
  message if the key is unrecognized. `message` itself is never shown to
  the user — it's an English fallback/log string. "Card"/"cards" is kept
  as an English loanword in the Portuguese UI (an existing, deliberate
  convention) rather than translated to "cartão"/"cartões".
- **Collection groups are single-level and single-membership**: a group
  holds decks directly and never holds other groups, and a deck belongs
  to at most one group at a time — closer to a folder than a tag. This
  was a deliberate simplification over a more flexible (multi-group,
  nested) design, chosen because it matches how people actually described
  wanting to organize collections (e.g. "Inglês" holding "Phrasal verbs",
  "Verbos", "Gramática") without the added UI and data-model complexity
  of tags or nesting.
- **The visual design system** (`frontend/src/styles/global.css`'s color
  tokens) follows [`luma-design-agent.md`](../luma-design-agent.md): a
  calm, mostly-neutral palette with the brand blue reserved for actions
  and semantic colors (success/warning/error) reserved for state — e.g.
  collection badges use one soft tone per badge *type* (collection vs.
  group) rather than a different saturated color per item, and a
  collection's "N para revisar" / "Tudo em dia" status is a small tinted
  pill rather than coloring the whole card.

## Known limitations and future improvements

These are deliberately out of scope for now, not oversights:

- Passwordless magic-link authentication, OAuth — see "Session tokens"
  above for what's implemented instead.
- FSRS or any other more advanced scheduler — the current one is
  deterministic and interface-isolated (`domain/study.Scheduler`)
  specifically so it can be swapped without touching the study flow.
- Multiple exercise types (cloze deletion, translation mode,
  pronunciation practice) — only question → reveal-answer exists.
- Text-to-speech generation — audio must be uploaded, never generated.
- External object storage — audio lives in MongoDB GridFS only.
- Import/export, learning analytics beyond the current statistics
  screen, notifications, shared/collaborative collections, offline
  support, a mobile-native app, and more advanced permission models —
  none exist.
- The screenshot-to-flashcard importer described in
  [`screenshot_to_flashcard_importer_agent.md`](../screenshot_to_flashcard_importer_agent.md)
  is a **separate, unbuilt** system; nothing in this backend or frontend
  depends on it existing.
- A real secrets manager for `SESSION_SECRET` and other production
  hardening — see [`.env.example`](../.env.example) for what's
  configurable today.

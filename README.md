# Luma

A simple, extensible flashcard app for studying any subject through
question-and-answer cards, with optional audio and examples. Built per
[`flashcard_learning_system_agent.md`](./flashcard_learning_system_agent.md)
(the spec file's own name predates the "Luma" product name).

Status: **All 6 core phases done** (project foundation, domain/
persistence, email-only access, deck/flashcard management, GridFS audio,
study session flow). See that file's section 23 for the full phase plan.
What exists so far:

- Go backend: domain model (User, Deck, Flashcard, ReviewEvent, spaced-
  repetition Scheduler), MongoDB repositories with ownership-scoped
  queries and indexes, email-only session auth (signed cookie, rate
  limited), GridFS-backed audio storage, and REST endpoints for
  session/decks/flashcards/audio/study.
- React + TypeScript + Vite frontend: login screen, dashboard, deck
  details, flashcard editor (with audio upload/playback/removal), and a
  full study session screen (question → reveal answer → optional audio →
  rate → session summary), talking to the API over cookies.
- Docker Compose for backend, frontend, and MongoDB.
- CI (GitHub Actions) running format/vet/build/test for both sides.

The MVP is functionally complete end-to-end: create a deck, add
flashcards (with optional audio), and study them with spaced repetition.
Phase 7 (quality/documentation polish, no new functionality) is also
done — see `backend/docs/` and "Known limitations" below.

## Documentation

- [`backend/docs/API.md`](./backend/docs/API.md) — full HTTP reference:
  every endpoint, request/response shapes, and the error-code table.
- [`backend/docs/ARCHITECTURE.md`](./backend/docs/ARCHITECTURE.md) — the
  hexagonal-architecture layering, request lifecycle, and where to add a
  new feature.
- [`docs/BACKUP.md`](./docs/BACKUP.md) — daily backups: how they work, how to
  check and restore them.

## Running locally

### With Docker Compose (recommended)

```bash
cp .env.example .env   # adjust SESSION_SECRET etc. if desired
docker compose up --build
```

- Backend: http://localhost:8080/health
- Frontend: http://localhost:5173
- MongoDB: localhost:27018 (host-only; other containers reach it at
  `mongo:27017` — 27018 avoids clashing with a MongoDB you may already
  have running locally on the default port)

### Backend only

Requires Go 1.22+.

```bash
cd backend
go run ./cmd/api
```

### Frontend only

Requires Node.js 20+ (the system Node on this machine is v12 and cannot run
Vite; use Docker or a newer local Node install, e.g. via nvm).

```bash
cd frontend
npm install
npm run dev
```

## Testing

```bash
# backend — unit tests only
cd backend && go vet ./... && go test ./...

# backend — static analysis (same check CI runs)
go install honnef.co/go/tools/cmd/staticcheck@v0.5.1
staticcheck ./...

# backend — including MongoDB integration tests
docker compose up -d mongo
cd backend && MONGODB_TEST_URI="mongodb://localhost:27018" go test ./...

# frontend
cd frontend && npm run lint && npm run build && npm test -- --run
```

MongoDB-backed tests (`internal/adapters/mongodb`, and the deck/flashcard
HTTP integration tests in `internal/adapters/http`) skip themselves with a
clear message if no MongoDB instance is reachable, so `go test ./...`
still passes without Docker running — you just get less coverage.

## API overview

See [`backend/docs/API.md`](./backend/docs/API.md) for the full
reference (request/response shapes, error codes). Quick summary:

```
POST   /api/v1/session                        create/find user, issue session cookie
DELETE /api/v1/session                        clear session cookie
GET    /api/v1/me                             current user (requires session)

GET    /api/v1/decks                          list active decks + card counts
POST   /api/v1/decks
GET    /api/v1/decks/{deckID}
PATCH  /api/v1/decks/{deckID}
DELETE /api/v1/decks/{deckID}                  archives (does not hard-delete)

GET    /api/v1/decks/{deckID}/flashcards
POST   /api/v1/decks/{deckID}/flashcards
GET    /api/v1/flashcards/{flashcardID}
PATCH  /api/v1/flashcards/{flashcardID}
DELETE /api/v1/flashcards/{flashcardID}        archives (does not hard-delete)

POST   /api/v1/flashcards/{flashcardID}/audio   upload/replace audio (body = raw bytes, Content-Type = audio/mpeg|wav|ogg)
GET    /api/v1/flashcards/{flashcardID}/audio   stream audio for playback
DELETE /api/v1/flashcards/{flashcardID}/audio   remove audio (idempotent)

GET    /api/v1/decks/{deckID}/due-flashcards    cards due for study (?limit=, default 20, max 100)
POST   /api/v1/flashcards/{flashcardID}/reviews  submit a rating (again/hard/good/easy), reschedules the card
```

All routes but `/health` and `POST /api/v1/session` require the
`session_token` cookie issued by `POST /api/v1/session`. Every deck/
flashcard lookup is scoped to the authenticated user; a resource that
exists but belongs to someone else returns `404 NOT_FOUND`, not `403`, to
avoid revealing existence.

## Study configurations

By default you study as many cards as you like, in sessions of up to 20, and
each rating reschedules a card as the app always did. From the
**Configurações** page (dashboard) you can pick the built-in **"Padrão"**
configuration or create your own, **named**, choosing for each rating
("Muito difícil", "Difícil", "Fácil", "Muito fácil") how the system reacts (how
soon a card comes back and how fast its interval grows) and, optionally, a
**daily goal**: how many different cards to study per day, across all decks.
With a goal the session stops at it ("Meta do dia concluída!") and offers
"Continuar estudando" to go past it. That choice is saved on the account, so it
holds on every device for the rest of the day, and the goal applies again the
next day. A **deck** can also use a configuration of its own (chosen on the
deck's page), which then takes precedence over the general one for that deck's
cards, with its own daily count. See "Study configurations" and "Daily card
goal" in
[backend/docs/API.md](backend/docs/API.md).

## Backups

A `backup` service in the compose file takes a verified, compressed backup of
the whole database (including the audio files) once a day into `./backups/`,
keeping the last 30. See [docs/BACKUP.md](docs/BACKUP.md) for how to take one
on demand, check that it can be restored, and restore it. Copy `./backups/`
to another disk from time to time: it lives on the same disk as the data.

## Configuration

See `.env.example` for all variables. Notable defaults:

- `SESSION_SECRET` is only allowed to be empty when `APP_ENV=development`
  (an insecure placeholder is used); every other environment requires it.
- `COOKIE_SECURE` defaults to `false` in development and `true` otherwise.

## Assumptions and decisions worth knowing about

- **Module path**: the Go module is named `flashcard-backend` (no GitHub
  org/repo existed yet). Rename it in `go.mod` and update imports if this
  project is later published under a specific module path.
- **No stateful "study session" resource**: spec section 13 sketches
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
  - **Mature card threshold**: a REVIEW card is promoted to MATURE once its
  interval reaches 21 days. The spec's section 11 doesn't set this number;
  21 days is a common spaced-repetition convention (used here as a
  placeholder, not a claim about optimal learning science).
- **Session tokens**: implemented as a signed (HMAC-SHA256), stateless,
  30-day cookie — no server-side session store. This satisfies spec
  section 8's "signed, short-lived token" option and keeps auth behind the
  `session.Manager` interface so it can be swapped later.
- **Session rate limiting**: a basic in-memory, per-IP, 10-requests/minute
  limiter on `POST /api/v1/session`. It resets on process restart and does
  not coordinate across instances — adequate for a single-instance MVP,
  not for a multi-instance deployment.
- **MongoDB driver**: using `go.mongodb.org/mongo-driver` v1 rather than
  the newer v2, because v2 requires Go 1.25+ while this environment's
  installed Go is 1.22 — v1 remains fully supported and keeps the toolchain
  simple.
- **MongoDB port**: docker-compose maps MongoDB to host port 27018 (not
  the default 27017) because another MongoDB was already running locally
  during development. Containers still reach it at `mongo:27017`.
- The health endpoint now pings MongoDB and reports `"database": "ok"` or
  `"unavailable"` (spec section 21).
- **Audio upload has no multipart form**: the request body is the raw
  audio bytes; the content type comes only from the `Content-Type`
  header, never from a client-supplied filename (spec section 7 warns
  against trusting filenames — this sidesteps that entirely by never
  accepting one). GridFS's own generated file ID is the only reference
  ever stored.
- **No audio duration**: computing it server-side would need an
  audio-format parsing dependency (MP3/OGG duration isn't in a simple
  header the way WAV's is) for little MVP benefit — the browser's
  `<audio>` element already reports it once played. `DurationMs` is
  always 0.
- **Audio upload only in edit mode**: the flashcard editor only shows the
  audio upload control once a flashcard exists (i.e., after the first
  Save), since audio attaches to an existing flashcard ID via its own
  endpoint. Creating a new card with audio in one step would need a
  two-phase form; instead, save the card, then edit it again to add
  audio.
- **No oversized-file orphan window**: the GridFS adapter aborts the
  upload stream (rather than finishing it and deleting after) the moment
  it detects more than `MAX_AUDIO_SIZE_BYTES` bytes, so a rejected upload
  never briefly exists in storage.
- **Deck-name uniqueness index removed**: the spec's example index list
  (section 18) includes `decks: userId + name`, to support rejecting
  duplicate active deck names. That rule is explicitly optional (section
  6: "if this rule is selected during implementation") and was never
  implemented — no code queries decks by name — so the unused index was
  removed during the Phase 7 review, per section 18's own instruction not
  to create an index without a backing query. Add it back if duplicate-
  name rejection is implemented later.
- **Flashcard due-date indexes consolidated**: the spec's example index
  list has two separate two-field indexes (`userId + scheduling.dueAt`,
  `deckId + scheduling.dueAt`), but every real query filters by `userId`
  **and** `deckId` **and** `archivedAt` together. A single four-field
  compound index (`userId, deckId, archivedAt, scheduling.dueAt`) serves
  `ListByDeck`, `ListDue`, and `CountByDeck` — MongoDB can use a prefix of
  a compound index — so it replaced the two narrower, unused ones.
- **Structured logging includes error code and resource IDs**: the
  single access-log line per request (spec section 21) includes the
  `errorCode` when the request failed and any `deckId`/`flashcardId` from
  the URL path, not just method/path/status/duration.
- **Internationalization (Portuguese, English, Spanish, French)**: the
  frontend detects the browser's language on first visit
  (`i18next-browser-languagedetector`) and falls back to Portuguese.
  Signing in applies the language saved on the account instead, if any;
  the picker in Settings saves an explicit choice back to the account
  (`PUT /api/v1/me/language`) so it follows the user to any device,
  mirroring the existing `ActiveStudyProfileID` pattern. Every backend
  error response also carries a stable `key` (e.g. `deck.name.tooLong`,
  see `backend/docs/API.md`'s Error format section) alongside the English
  `message`; the frontend looks the key up in `src/i18n/locales/*.json`'s
  `errors` map (a flat dictionary keyed by the backend's own dotted key,
  not a nested i18next path) via `useApiErrorMessage`, falling back to a
  page-specific translated message if the key is unrecognized. `message`
  itself is never shown to the user — it's an English fallback/log string.

## Known limitations and future improvements

Per spec section 26, these are deliberately out of scope for this MVP,
not oversights:

- Passwordless magic-link authentication, OAuth — see "Session tokens"
  above for what's implemented instead.
- FSRS or any other more advanced scheduler — the current one is
  deterministic and interface-isolated (`domain/study.Scheduler`)
  specifically so it can be swapped without touching the study flow.
- Multiple exercise types (cloze deletion, translation mode,
  pronunciation practice) — only question → reveal-answer exists.
- Text-to-speech generation — audio must be uploaded, never generated.
- External object storage — audio lives in MongoDB GridFS only.
- Import/export, tags/concepts, learning analytics, notifications,
  shared decks, offline support, a mobile-native app, and more advanced
  permission models — none exist.
- The screenshot-to-flashcard importer described in
  `screenshot_to_flashcard_importer_agent.md` is a **separate, unbuilt**
  system; nothing in this backend or frontend depends on it existing.

## What's next

There are no more unimplemented product phases (1–7 are all done). Any
further work is one of: the future improvements above, the separate
screenshot importer, or ongoing maintenance (dependency updates,
production hardening like a real secrets manager for `SESSION_SECRET`,
etc.).

# API Reference

Base URL: `http://localhost:8080` (configurable via `HTTP_PORT`).

All endpoints are versioned under `/api/v1`, except `GET /health`.

## Authentication

Every route except `GET /health`, `POST /api/v1/session`, and
`DELETE /api/v1/session` requires a valid `session_token` cookie, issued
by `POST /api/v1/session`. This is **email-only access, not password-based
authentication** — see spec section 8 and the root README's security
notes. A missing or invalid cookie returns `401 UNAUTHORIZED`.

Every deck and flashcard lookup is scoped to the authenticated user. A
resource that exists but belongs to a different user returns
`404 NOT_FOUND`, not `403 FORBIDDEN` — this avoids revealing whether the
resource exists at all.

## Error format

Every non-2xx response uses the same envelope:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "key": "flashcard.question.required",
    "message": "question is required",
    "requestId": "a1b2c3d4e5f6..."
  }
}
```

`key` is a stable, machine-readable identifier for exactly which
validation or lookup failed (e.g. `deck.name.tooLong`, `flashcard.notFound`).
Clients use it to show a translated message; `message` is always the
English text and is meant as a fallback/for logs, not for display in a
localized UI.

| Code                     | HTTP status | Meaning                                   |
| ------------------------ | ----------- | ------------------------------------------ |
| `VALIDATION_ERROR`        | 400         | Request failed input validation            |
| `UNAUTHORIZED`             | 401         | Missing/invalid/expired session            |
| `FORBIDDEN`                | 403         | Reserved; not currently used (see below)   |
| `NOT_FOUND`                | 404         | Resource missing, or not owned by caller   |
| `CONFLICT`                 | 409         | Reserved for future use                    |
| `PAYLOAD_TOO_LARGE`        | 413         | Request body (e.g. audio) exceeds the limit|
| `UNSUPPORTED_MEDIA_TYPE`   | 415         | Audio content type not in the allowed set  |
| `RATE_LIMITED`             | 429         | Too many requests to a rate-limited route  |
| `INTERNAL_ERROR`           | 500         | Unexpected server error (details in logs, never in the response) |

`requestId` matches the `X-Request-Id` response header and the
`requestId` field in the corresponding server log line, for
troubleshooting.

## Health

```
GET /health
```

No authentication required. Returns application liveness and, if a
database connection is configured, connectivity:

```json
{ "status": "ok", "database": "ok" }
```

`status` is `"degraded"` and the HTTP status is `503` if the database
ping fails.

## Session (spec section 8)

```
POST /api/v1/session
```

Request:

```json
{ "email": "user@example.com" }
```

Finds or creates a user by normalized email (trimmed, lowercased) and
issues a `session_token` cookie (`HttpOnly`, `SameSite=Lax`, `Secure` when
`COOKIE_SECURE=true`). Rate-limited to 10 requests/minute per client IP.

Response:

```json
{ "user": { "id": "...", "email": "user@example.com" } }
```

```
DELETE /api/v1/session
```

Clears the session cookie. Idempotent — succeeds even with no active
session. Returns `204 No Content`.

```
GET /api/v1/me
```

Requires authentication. Returns the current user:

```json
{ "id": "...", "email": "user@example.com", "language": "pt" }
```

`language` is one of `pt`, `en`, `es`, `fr`, or absent if the user never
chose one — the client then falls back to detecting the browser's language.

```
PUT /api/v1/me/language
```

Requires authentication. Saves the caller's UI language preference so it
follows them to any device or browser.

Request:

```json
{ "language": "fr" }
```

`language` must be one of `pt`, `en`, `es`, `fr`. Returns `204 No Content`,
or `400 VALIDATION_ERROR` (key `user.language.invalid`) for an unsupported
value.

## Study configurations

A study configuration is a **named** set of choices about how the system
reacts to each rating, plus an optional daily card goal. Every user always has
the built-in **"Padrão"** configuration, which is selected until they pick
another one, and can create their own.

```
GET    /api/v1/study-profiles
POST   /api/v1/study-profiles
PUT    /api/v1/study-profiles/{profileID}
DELETE /api/v1/study-profiles/{profileID}
PUT    /api/v1/me/active-study-profile
```

A configuration looks like this (the built-in default):

```json
{
  "id": "default",
  "name": "Padrão",
  "isDefault": true,
  "dailyCardLimit": null,
  "rules": {
    "againDelayMinutes": 10,
    "hard": { "firstIntervalDays": 1, "multiplier": 1.2 },
    "good": { "firstIntervalDays": 3, "multiplier": 2.0 },
    "easy": { "firstIntervalDays": 7, "multiplier": 3.0 }
  }
}
```

`rules` says how each answer reschedules a card (in the app the buttons are
"Muito difícil" = `again`, "Difícil" = `hard`, "Fácil" = `good`,
"Muito fácil" = `easy`):

| Rule | Meaning | Allowed |
|------|---------|---------|
| `againDelayMinutes` | how soon a card comes back after "Muito difícil" | 1 to 1440 (a day) |
| `<rating>.firstIntervalDays` | wait, in days, for a card that is new or being learned again | 1 to 365 |
| `<rating>.multiplier` | factor applied to the previous interval when an already learned card is reviewed again | 1 to 5 |

A better answer never comes back sooner than a worse one: for both
`firstIntervalDays` and `multiplier`, `hard <= good <= easy`. Whatever the
values, every answer other than "Muito difícil" moves the next review at least
one day further, and no interval exceeds 365 days. A card is "Aprendido" once
its interval reaches 21 days, in every configuration.

`dailyCardLimit` is the number of **different cards** to study per day
(1 to 200), or `null` for no daily goal. See "Daily card goal" under Study.

### Listing

`GET /api/v1/study-profiles` returns the built-in default first, then the
user's own configurations (oldest first), and which one is active:

```json
{ "activeProfileId": "default", "profiles": [ { "id": "default", "...": "..." } ] }
```

### Creating and editing

`POST` (create, `201`) and `PUT .../{profileID}` (replace) take
`{ "name": "...", "dailyCardLimit": 30, "rules": { ... } }`. The name is
trimmed, required, at most 60 characters and unique per user (ignoring case):
a repeated name yields `409 CONFLICT`. A user can have at most 20
configurations. Invalid values yield `400 VALIDATION_ERROR`. Creating one does
**not** select it. The built-in default cannot be edited or deleted
(`403 FORBIDDEN`), and other users' configurations are `404 NOT_FOUND`.

### Choosing which one to use

`PUT /api/v1/me/active-study-profile` with `{ "profileId": "<id>" }` (or
`"default"`) selects the configuration used from then on; it returns `204`.
Deleting the active configuration (`DELETE`, `204`) puts the user back on the
default. Switching does not touch cards already scheduled: only how *later*
answers are handled changes.

### A configuration per deck

A deck can be studied with a configuration of its own instead of the general
(active) one, for example a deck with a different daily goal or slower reviews.

`PUT /api/v1/decks/{deckID}/study-profile` with `{ "profileId": "<id>" }`
chooses one of the user's configurations for that deck (`"default"` is the
built-in one). An empty or `null` `profileId` makes the deck follow the
general configuration again. It returns `204`; an unknown deck or
configuration (or another user's) is `404 NOT_FOUND`. The deck endpoints
(`GET /api/v1/decks` and `GET /api/v1/decks/{deckID}`) return
`studyProfileId` for a deck that has its own configuration and omit it
otherwise.

**Which one applies.** For a card the system uses the configuration of the
card's deck if that deck has one, and the general configuration if not (also
when the deck's configuration was deleted: deleting a configuration releases
the decks that used it). This decides both how answers reschedule the card
(`rules`) and the daily goal.

**Daily goals are counted per scope.** A deck with its own configuration has
its own goal counting only the cards studied today *in that deck*. The general
goal counts the cards studied today in every deck that has **no** configuration
of its own. So a deck with its own goal never uses up (or is blocked by) the
general one. See "Daily card goal" under Study for the `deckId` parameter.

### Migration

A daily goal saved before configurations existed is turned, the first time the
list is read, into a configuration named "Minha meta diária" (default review
rules plus that goal) and selected, so nobody's study changes.

## Decks

```
GET    /api/v1/decks
POST   /api/v1/decks
GET    /api/v1/decks/{deckID}
PATCH  /api/v1/decks/{deckID}
DELETE /api/v1/decks/{deckID}
```

`GET /api/v1/decks` lists the caller's **active** (non-archived) decks,
each with card counts:

```json
[
  { "id": "...", "name": "English", "description": "", "totalCards": 12, "dueCards": 3, "groupId": "..." }
]
```

`groupId` is present only when the deck is filed under a deck group (see
"Deck groups" below), and absent otherwise.

`POST`/`PATCH` request body:

```json
{ "name": "English", "description": "optional, up to 1,000 chars" }
```

`name` is required, trimmed, 1–100 characters.

`GET /api/v1/decks/{deckID}` also returns `nextDueAt` (RFC 3339): when the
earliest card that is **not yet due** comes due. It lets a screen say "your
next review is in 2 days" when nothing is due now. It is absent when no
card is scheduled in the future, and it is only on this single-deck
endpoint, not in the `GET /api/v1/decks` list.

`DELETE` **archives** the deck rather than permanently deleting it — its
flashcards and their review history are preserved, and the deck stops
appearing in `GET /api/v1/decks`. Returns `204 No Content`.

`GET /api/v1/decks?archived=true` lists the caller's **archived** decks
(most recently archived first) with the same shape and card counts as the
active list.

`POST /api/v1/decks/{deckID}/restore` un-archives a deck; its flashcards
and review history reappear exactly as they were. It is idempotent
(restoring an active deck succeeds). Returns `204 No Content`, or `404
NOT_FOUND` if the deck does not exist or belongs to another user.

`DELETE /api/v1/archived-decks/{deckID}` **permanently** deletes a deck
that is already archived, together with all of its flashcards (active and
archived), their audio files and the deck's review history. It cannot be
undone. A deck that is not archived (or does not exist / belongs to another
user) yields `404 NOT_FOUND`, so an active deck can never be deleted this
way. Returns `204 No Content`.

### Deck statistics

`GET /api/v1/decks/{deckID}/stats` returns the numbers behind the
statistics screen:

```json
{
  "deckId": "...", "name": "Français",
  "totalCards": 334, "newCards": 12, "learningCards": 23, "masteredCards": 198,
  "masteryPercent": 59,
  "studiedToday": 12, "studiedLast7Days": 45, "studiedTotal": 210,
  "dailyPerformance": [
    { "date": "2026-01-01", "reviewCount": 20, "accuracyPercent": 80, "hintCount": 3 }
  ]
}
```

`studiedToday`, `studiedLast7Days` and `studiedTotal` count **distinct
cards** reviewed at least once in the period: a card reviewed five times
counts once (`dailyPerformance[].reviewCount` counts reviews). The 7 days
include today.

The optional `tz` query parameter is an IANA time zone name (for example
`America/Manaus`). With it, "today", the 7-day window and the per-day
`dailyPerformance` entries follow the caller's local days; without it they
are UTC days. An unknown name yields `400 VALIDATION_ERROR`.

## Deck groups

A deck group is a **named folder** a user can file decks into, to organize
related collections together (e.g. an "Inglês" group holding "Phrasal
verbs", "Verbos" and "Gramática" decks). Grouping is single-level: a group
holds decks, never other groups, and a deck belongs to at most one group.

```
GET    /api/v1/deck-groups
POST   /api/v1/deck-groups
PUT    /api/v1/deck-groups/{groupID}
DELETE /api/v1/deck-groups/{groupID}
PUT    /api/v1/decks/{deckID}/group
```

`GET /api/v1/deck-groups` lists the caller's groups, oldest first:

```json
[ { "id": "...", "name": "Inglês" } ]
```

`POST`/`PUT .../deck-groups/{groupID}` request body:

```json
{ "name": "Inglês" }
```

`name` is required, trimmed, 1–60 characters, and unique per user
(ignoring case) — a repeat is `409 CONFLICT` (key `deckGroup.duplicateName`).
There is a limit of 50 groups per user (`deckGroup.limitReached`).

`DELETE /api/v1/deck-groups/{groupID}` permanently deletes the group. This
does **not** delete or archive the decks that were filed under it — they
simply lose their group and appear on their own on the dashboard again.
Returns `204 No Content`, or `404 NOT_FOUND` (key `deckGroup.notFound`) for
an unknown group or one belonging to another user.

`PUT /api/v1/decks/{deckID}/group` with `{ "groupId": "<id>" }` files a
deck under one of the user's groups; an empty or `null` `groupId` removes
it from its group. Returns `204`; an unknown deck or group (or another
user's) is `404 NOT_FOUND`.

## Flashcards

```
GET    /api/v1/decks/{deckID}/flashcards
POST   /api/v1/decks/{deckID}/flashcards
GET    /api/v1/flashcards/{flashcardID}
PATCH  /api/v1/flashcards/{flashcardID}
DELETE /api/v1/flashcards/{flashcardID}
```

Every operation under `/decks/{deckID}/...` verifies the deck belongs to
the caller before touching any flashcard.

### Listing, searching and paging

`GET /api/v1/decks/{deckID}/flashcards` returns one page of the deck's
flashcards, newest first, together with the total number of matches:

```json
{ "items": [ { "id": "...", "question": "...", "...": "..." } ], "total": 1000 }
```

Query parameters (all optional):

| Parameter  | Meaning |
|------------|---------|
| `limit`    | Page size: default 30, at most 100; must be a positive integer. |
| `offset`   | How many matches to skip; must not be negative. |
| `q`        | Search text (trimmed, at most 200 characters). Keeps the cards whose question, answer or hint contain it, ignoring case and accents (`francais` finds `Français`). Matched literally, not as a regular expression. |
| `archived` | `true` lists the archived cards instead of the active ones (most recently archived first). |

`total` is the number of matches ignoring `limit`/`offset`, so a client can
show "30 of 1000" and knows when there is nothing left to load. A `limit`
or `offset` that is not an integer (or out of range) yields `400
VALIDATION_ERROR`.

`POST`/`PATCH` request body:

```json
{
  "question": "How do you say 'rescue'?",
  "answer": "resgatar",
  "hint": "Starts with R",
  "extendedExample": {
    "text": "The firefighters managed to rescue the child.",
    "translation": "Os bombeiros conseguiram resgatar a criança."
  }
}
```

`hint` is optional: a short nudge (up to 1,000 characters, trimmed) shown
on the front of the card while studying, before the answer is revealed.
Omit it or send an empty string to have no hint (on update, that clears
it); it is left out of the response when empty.

`extendedExample` is optional; omit it entirely, or omit/empty its
`text` to clear it on update. `question` and `answer` are required,
trimmed, up to 5,000 characters each; example fields up to 10,000.

Response shape (also used by the due-cards and review endpoints):

```json
{
  "id": "...",
  "deckId": "...",
  "question": "How do you say 'rescue'?",
  "answer": "resgatar",
  "hint": "Starts with R",
  "extendedExample": { "text": "...", "translation": "..." },
  "audio": { "contentType": "audio/mpeg", "durationMs": 0, "url": "/api/v1/flashcards/.../audio" },
  "scheduling": {
    "state": "new",
    "dueAt": "2026-01-01T00:00:00Z",
    "intervalDays": 0,
    "repetitions": 0,
    "lapses": 0
  }
}
```

`extendedExample` and `audio` are omitted (not `null`) when absent.
`audio.url` is API-relative — prefix it with the API's base URL before
using it as an `<audio src>` if the frontend is served from a different
origin.

`DELETE` archives the flashcard (same semantics as deck archive) and also
detaches it from study/due-card queries. Returns `204 No Content`.

`GET /api/v1/decks/{deckID}/flashcards?archived=true` lists the deck's
archived flashcards (most recently archived first), paged like the active
list (see "Listing, searching and paging").

`POST /api/v1/flashcards/{flashcardID}/restore` un-archives a flashcard,
keeping its scheduling state and review history. It is idempotent.
Returns `204 No Content`, or `404 NOT_FOUND` if the card does not exist or
belongs to another user.

`DELETE /api/v1/archived-flashcards/{flashcardID}` **permanently** deletes a
flashcard that is already archived, together with its audio file. It cannot
be undone. A card that is not archived (or does not exist / belongs to
another user) yields `404 NOT_FOUND`. Review events already recorded for the
card are kept, as they are part of the deck's study history. Returns `204 No
Content`.

`GET /api/v1/archived-flashcards` lists the caller's archived flashcards
across all of their active decks, each with a `deckName`.

## Audio (spec section 7)

```
POST   /api/v1/flashcards/{flashcardID}/audio
GET    /api/v1/flashcards/{flashcardID}/audio
DELETE /api/v1/flashcards/{flashcardID}/audio
```

`POST` uploads or replaces a flashcard's audio. The request body is the
**raw audio bytes** — there is no multipart form and no filename field;
the content type is read only from the `Content-Type` header, one of
`audio/mpeg`, `audio/wav`, or `audio/ogg`. Body size is capped by
`MAX_AUDIO_SIZE_BYTES` (default 10 MiB); an oversized upload is rejected
with `413 PAYLOAD_TOO_LARGE` before anything is persisted. Replacing
existing audio deletes the old file after the new one is attached.
Response: the updated flashcard (see shape above).

`GET` streams the audio bytes with the stored `Content-Type`, for use
directly as an `<audio src>`. Requires authentication — the browser sends
the session cookie automatically for a same-styled resource request.

`DELETE` removes the audio (idempotent — succeeds even if there is none).
Returns `204 No Content`.

## Study (spec section 9)

```
GET  /api/v1/decks/{deckID}/due-flashcards
POST /api/v1/flashcards/{flashcardID}/reviews
```

There is no stateful "study session" resource — see the root README's
"Assumptions" section for why. A study screen is expected to:

1. `GET .../due-flashcards?limit=20` (default 20, max 100) for the next
   batch to study. A card is due the instant it's created, so this list
   naturally includes brand-new cards without special-casing.
2. Show each card's question, then reveal `answer` / `extendedExample` /
   `audio`.
3. `POST /api/v1/flashcards/{id}/reviews` with a rating to record it and
   get the card's new scheduling state back:

   ```json
   { "rating": "good", "responseTimeMs": 1500, "hintUsed": true }
   ```

   `rating` is one of `again`, `hard`, `good`, `easy`. `responseTimeMs` and
   `hintUsed` are optional and only used for the review-history record:
   `hintUsed` says the learner looked at a hint before answering. It never
   changes the scheduling; the deck statistics report it per day as
   `dailyPerformance[].hintCount`.
4. Repeat until the batch is exhausted; compute the session summary
   (cards studied, ratings breakdown) client-side from the responses.

### Daily card goal

With no goal (the default) nothing below applies. When the user's active
study configuration has a `dailyCardLimit` (see Study configurations):

- **What counts:** the number of *different* cards studied today, across
  **all** of the user's decks. A card reviewed five times today counts once.
- `GET /api/v1/decks/{deckID}/due-flashcards` serves at most as many cards
  that were **not yet studied today** as the goal still has room for. Cards
  already studied today that come due again (a card answered `again` returns
  a few minutes later) are still served: they were already counted, and
  cutting them off would leave the card half-learned.
- `POST /api/v1/study/continue-past-goal` records that the user chose to
  keep studying past the goal ("Continuar estudando"). It is saved **on the
  user's account**, so it holds on every device and browser, and it lasts
  until the end of that day (the `tz` parameter says which day); the next day
  the goal applies again. With `deckId` for a deck that has a configuration of
  its own, the choice belongs to that deck alone and does not lift the general
  goal (and the other way round). While it holds, `due-flashcards` serves every due
  card without the cap. Returns `204 No Content`.
- `?ignoreDailyLimit=true` on `due-flashcards` is a one-off override that
  serves the due cards regardless of the goal, without saving anything.
- `GET /api/v1/study/progress` reports where the user stands. With the
  optional `deckId` it is the goal of that deck (its own configuration's
  goal counting that deck only, or else the general goal); without it, the
  general goal:

  ```json
  { "dailyCardLimit": 20, "studiedToday": 12, "remaining": 8, "continuingPastGoal": false }
  ```

  `dailyCardLimit` and `remaining` are `null` without a goal; `remaining`
  never goes below 0. `studiedToday` is reported even without a goal.
  `continuingPastGoal` is `true` when the user already chose, today, to
  continue past the goal.
- "Today" is the user's own day: these endpoints accept the optional `tz`
  query parameter (an IANA name such as `America/Manaus`); without it days
  are UTC. An unknown name yields `400 VALIDATION_ERROR`.

Reviewing an archived flashcard returns `404 NOT_FOUND`. There is no
server-side guard against submitting a review twice for the same card in
quick succession — each submission is processed independently and simply
reschedules the card again.

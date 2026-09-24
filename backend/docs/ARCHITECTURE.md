# Backend Architecture

The backend follows a hexagonal / clean-architecture style: business
rules live in the center, infrastructure (MongoDB, HTTP, GridFS) sits at
the edges behind small interfaces ("ports"), and dependencies only ever
point inward.

```
cmd/api/main.go          wires everything together; the only place that
                          knows about *every* concrete adapter at once

internal/
├── domain/               entities + pure business rules, zero I/O
│   ├── user/              email validation/normalization
│   ├── deck/               name/description validation
│   ├── flashcard/          question/answer/audio validation
│   ├── review/             ReviewEvent (immutable review history)
│   └── study/              CardState, Rating, Scheduler (spaced repetition)
│
├── application/          use cases; orchestrate domain + ports, enforce
│   │                     ownership, never import MongoDB or net/http
│   ├── userservice/        email-only access flow
│   ├── deckservice/        deck CRUD + card-count aggregation
│   ├── flashcardservice/   flashcard CRUD + audio upload/playback/delete
│   └── studyservice/       due-card selection + review recording
│
├── ports/                interfaces the application layer depends on
│   ├── repositories/       User/Deck/Flashcard/ReviewEvent persistence
│   ├── audiostore/         audio binary storage (Save/Open/Delete)
│   ├── session/            session token issue/verify
│   └── clock/              injectable time, for deterministic tests
│
├── adapters/             concrete implementations of the ports above
│   ├── mongodb/            repositories, indexes, connection handling
│   ├── gridfs/             audiostore.Store backed by MongoDB GridFS
│   ├── authtoken/          session.Manager: signed, stateless HMAC cookie
│   └── http/                the HTTP API: routing, handlers, middleware
│
├── apperror/             shared error codes (VALIDATION_ERROR, NOT_FOUND,
│                         ...) used by both application and http layers
└── config/               environment-variable loading and validation
```

## Why this shape

- **`domain` has no dependencies on anything else in this module.** It's
  pure Go: structs, validation functions, the scheduler. This is what
  makes `internal/domain/study`'s scheduler tests run in microseconds
  with no database.
- **`application` depends only on `domain` and `ports`,** never on
  `adapters` or `net/http`. A service like `flashcardservice.Service`
  takes `repositories.FlashcardRepository` (an interface) as a
  constructor argument, not `*mongodb.FlashcardRepository` (a concrete
  type) — this is what lets tests substitute in-memory fakes (see e.g.
  `flashcardservice_test.go`'s `fakeFlashcardRepository`) without a real
  MongoDB.
- **`adapters` depends on `ports` (to implement them) and `domain`** (to
  convert between BSON documents and domain structs), never on
  `application`. `internal/adapters/mongodb` is the only package that
  imports the MongoDB driver; `internal/adapters/gridfs` is the only one
  that imports GridFS.
- **Only `main.go` and test files construct concrete adapters.** Everything
  else receives interfaces. This is enforced by convention, not a linter
  rule — reviewers should flag an `application` or `domain` file importing
  from `adapters`.

## Request lifecycle

```
HTTP request
  → withRequestID       (assigns/forwards X-Request-Id)
  → withAccessLog        (times the request, logs one line at the end)
  → withCORS              (only for configured origins)
  → [withRateLimit]       (POST /session only)
  → [requireAuth]         (everything except /health, POST/DELETE /session)
  → handler               (adapters/http/*_handler.go)
      → decodes/validates the request shape only (not business rules)
      → calls one application service method
      → maps the result (or apperror.Error) to a JSON response
  → application service
      → enforces ownership (deck/flashcard belongs to the caller)
      → applies domain validation
      → calls one or more repository/store ports
  → adapter (mongodb / gridfs)
      → the only place BSON documents and GridFS streams exist
```

Business rules (validation, ownership, scheduling) live in `domain` and
`application` only — HTTP handlers translate between wire format and
service calls, and MongoDB adapters translate between service calls and
BSON; neither layer makes business decisions.

## Error handling

Application services return `*apperror.Error` (a stable code + safe
message) for anything the caller should see, and wrap unexpected errors
from a port (e.g. a MongoDB failure) with `apperror.Internal(err)`, which
keeps the original error for server-side logging but never leaks it to
the client. `internal/adapters/http/errors.go`'s `writeError` is the only
place that translates an `*apperror.Error` into an HTTP status and JSON
body — see `docs/API.md` for the code table.

## Adding a new feature

A new read/write operation typically touches:

1. `domain/<entity>`: new validation function(s), if the operation
   accepts user input.
2. `ports/repositories`: a new method on the relevant repository
   interface, if a new query shape is needed.
3. `adapters/mongodb/<entity>_repository.go`: implement that method.
4. `application/<entity>service`: a new service method enforcing
   ownership and calling the validation + repository method.
5. `adapters/http/<entity>_handler.go`: a handler calling the service
   method, plus a route in `router.go`.
6. Tests at the layer(s) that gained logic: domain validation tests are
   cheapest, service tests with fakes cover ownership/business rules,
   and one HTTP integration test (real MongoDB, see
   `internal/adapters/http/*_test.go`'s `mongoTestRouter` helper) confirms
   the wiring end-to-end.

See `docs/API.md` for the resulting HTTP contract, and the root
`README.md`'s "Assumptions and decisions worth knowing about" section for
places this project deliberately diverged from the product spec.

# Agent Specification --- Flashcard Learning System

## 1. Role

You are a senior software engineer and technical lead responsible for
designing, implementing, testing, documenting, and reviewing a simple
but extensible flashcard learning system.

Your goal is to deliver a working web application for studying any
subject through question-and-answer cards, with optional audio and
examples.

You must prioritize:

-   Simplicity
-   Maintainability
-   Clear domain modeling
-   Testability
-   Security appropriate to the MVP
-   Explicit business rules
-   Good user experience
-   Incremental implementation
-   Avoidance of unnecessary abstractions

Do not add AI, external integrations, or unrelated features.

------------------------------------------------------------------------

## 2. Product vision

The system allows a user to:

1.  Enter an email to access the application.
2.  Create study subjects or decks, such as:
    -   English
    -   French
    -   Algorithms
    -   System Design
    -   Mathematics
    -   Any other subject
3.  Create flashcards associated with a deck.
4.  Store:
    -   Question
    -   Answer
    -   Optional audio
    -   Optional extended example or explanation
5.  Start a study session.
6.  View the question first.
7.  Reveal the answer.
8.  Optionally play the answer audio.
9.  Mark the card as:
    -   Again / Very difficult
    -   Difficult
    -   Easy
    -   Very easy
10. Schedule the next review automatically.
11. View cards that are due for review.
12. Continue studying until the session ends.

The first version must be intentionally simple.

------------------------------------------------------------------------

## 3. Explicit scope

### Included

-   Email-only access
-   User creation or lookup by email
-   Deck CRUD
-   Flashcard CRUD
-   Optional audio upload
-   Optional extended example
-   Study session
-   Card review rating
-   Spaced repetition scheduling
-   Review history
-   Basic dashboard
-   Responsive web UI
-   Unit and integration tests
-   Docker-based local development
-   API documentation
-   Database indexes
-   Input validation
-   Error handling
-   Logging

### Excluded

-   Passwords
-   OAuth
-   Magic links
-   Email delivery
-   AI or LLM integration
-   Automatic question generation
-   Automatic translation
-   Automatic audio generation
-   External storage providers
-   Social features
-   Shared decks
-   Gamification
-   Notifications
-   Mobile-native applications
-   Complex permissions
-   Multi-tenant organization management

If a future feature would require one of these capabilities, document it
as a future improvement instead of implementing it.

------------------------------------------------------------------------

## 4. Technology choices

### Backend

Use Go, preferably the current stable version available in the
development environment.

Recommended technologies:

-   `net/http` or a small, well-maintained HTTP router
-   Official MongoDB Go Driver
-   Standard library where practical
-   `context.Context` for request cancellation and deadlines
-   Structured logging using `log/slog`
-   `go test` for tests
-   `go vet`
-   `gofmt`
-   Static analysis when available

Avoid adopting a large framework unless there is a clear benefit.

### Frontend

Use:

-   React
-   TypeScript
-   Vite
-   A lightweight component approach
-   React Router if routing is necessary
-   Native browser audio support
-   A simple CSS strategy or a lightweight UI library

The frontend should be easy to run locally and should not require a
complex design system.

### Database

Use MongoDB.

Use GridFS for optional audio files in the MVP.

The flashcard document must store an audio reference, not the complete
audio binary.

### Local development

Provide Docker Compose for:

-   Backend
-   Frontend
-   MongoDB

Do not add Redis, Kafka, RabbitMQ, or other infrastructure.

------------------------------------------------------------------------

## 5. Architecture

Use a modular architecture inspired by Hexagonal Architecture / Clean
Architecture.

Suggested backend structure:

``` text
backend/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── domain/
│   │   ├── user/
│   │   ├── deck/
│   │   ├── flashcard/
│   │   ├── review/
│   │   └── study/
│   ├── application/
│   │   ├── userservice/
│   │   ├── deckservice/
│   │   ├── flashcardservice/
│   │   ├── studyservice/
│   │   └── reviewservice/
│   ├── ports/
│   │   ├── repositories/
│   │   ├── audio/
│   │   └── clock/
│   ├── adapters/
│   │   ├── http/
│   │   ├── mongodb/
│   │   └── gridfs/
│   └── config/
├── migrations/
├── docs/
├── Dockerfile
└── go.mod
```

Suggested frontend structure:

``` text
frontend/
├── src/
│   ├── app/
│   ├── pages/
│   │   ├── LoginPage/
│   │   ├── DashboardPage/
│   │   ├── DeckPage/
│   │   ├── FlashcardEditorPage/
│   │   └── StudyPage/
│   ├── components/
│   │   ├── DeckCard/
│   │   ├── FlashcardForm/
│   │   ├── StudyCard/
│   │   ├── AudioPlayer/
│   │   └── RatingButtons/
│   ├── services/
│   ├── hooks/
│   ├── types/
│   └── styles/
├── package.json
└── Dockerfile
```

Do not create a package or abstraction until it has a real
responsibility.

------------------------------------------------------------------------

## 6. Domain model

### User

``` text
User
- id
- email
- createdAt
- updatedAt
- lastAccessAt
```

Rules:

-   Email is required.
-   Normalize email for lookup, preferably by trimming whitespace and
    converting to lowercase.
-   Store the normalized email.
-   Create a unique index on email.
-   Do not store passwords.
-   Do not claim that email-only access is strong authentication.
-   Clearly document that this is suitable only for a low-risk MVP.

### Deck

A deck represents a study subject.

``` text
Deck
- id
- userId
- name
- description
- languageOrDomain
- createdAt
- updatedAt
- archivedAt
```

Examples:

``` text
English
French
Algorithms
System Design
Mathematics
```

Rules:

-   A deck belongs to exactly one user.
-   Name is required.
-   Name must be trimmed.
-   A user cannot create duplicate active deck names if this rule is
    selected during implementation.
-   Archived decks should not appear in the default dashboard.
-   A deck can contain zero or more flashcards.

### Flashcard

``` text
Flashcard
- id
- userId
- deckId
- question
- answer
- extendedExample
- audio
- scheduling
- createdAt
- updatedAt
- archivedAt
```

Suggested structure:

``` json
{
  "_id": "ObjectId",
  "userId": "ObjectId",
  "deckId": "ObjectId",
  "question": "Como dizer 'resgatar' em inglês?",
  "answer": "rescue",
  "extendedExample": {
    "text": "The firefighters managed to rescue the child.",
    "translation": "Os bombeiros conseguiram resgatar a criança."
  },
  "audio": {
    "gridFsFileId": "ObjectId",
    "contentType": "audio/mpeg",
    "durationMs": 1800
  },
  "scheduling": {
    "state": "new",
    "dueAt": "2026-09-14T00:00:00Z",
    "lastReviewedAt": null,
    "repetitions": 0,
    "lapses": 0,
    "intervalDays": 0,
    "easeFactor": 2.5
  },
  "createdAt": "2026-09-14T00:00:00Z",
  "updatedAt": "2026-09-14T00:00:00Z"
}
```

Rules:

-   Question is required.
-   Answer is required.
-   Extended example is optional.
-   Audio is optional.
-   A flashcard belongs to one deck.
-   The API must verify that the deck belongs to the current user.
-   Archived cards must not appear in normal study sessions.
-   Audio metadata must be validated.
-   Limit question, answer, and example sizes.
-   Do not trust client-supplied user IDs or ownership fields.

### Review event

Keep a review history separate from the current scheduling state.

``` text
ReviewEvent
- id
- userId
- flashcardId
- deckId
- rating
- previousState
- nextState
- reviewedAt
- responseTimeMs
```

The review history is useful for:

-   Debugging scheduling
-   Statistics
-   Future algorithm improvements
-   User progress
-   Auditability

------------------------------------------------------------------------

## 7. Audio storage

Use MongoDB GridFS for the MVP.

### Rules

-   Store the binary file in GridFS.
-   Store only the GridFS file ID and metadata in the flashcard
    document.
-   Validate content type.
-   Enforce a maximum file size.
-   Do not trust the original filename.
-   Generate a safe internal filename.
-   Do not expose unrestricted GridFS access.
-   Serve audio through an authenticated endpoint.
-   Verify that the audio belongs to a flashcard owned by the current
    user.
-   Delete or clean up orphaned audio files when a flashcard is deleted
    or its audio is replaced.
-   Use streaming instead of loading the entire audio file into memory.
-   Support HTTP range requests only if needed and implemented safely.

Suggested endpoint:

``` text
GET /api/v1/flashcards/{flashcardID}/audio
```

For the MVP, support a small set of formats, for example:

-   `audio/mpeg`
-   `audio/wav`
-   `audio/ogg`

Do not implement audio transcoding.

------------------------------------------------------------------------

## 8. Email-only access

The initial access flow is intentionally simple.

### Endpoint

``` text
POST /api/v1/session
```

Request:

``` json
{
  "email": "user@example.com"
}
```

Response:

``` json
{
  "user": {
    "id": "user-id",
    "email": "user@example.com"
  },
  "sessionToken": "opaque-token"
}
```

Implementation requirements:

-   Validate email format.
-   Normalize email.
-   Create or find the user.
-   Issue a session token.
-   Store the session token securely on the server or use a signed,
    short-lived token.
-   Prefer an HttpOnly, Secure, SameSite cookie for a browser
    application.
-   Do not store the raw token in logs.
-   Add basic rate limiting to the session endpoint.
-   Document that this is not password-based authentication.
-   Keep the authentication implementation replaceable.

For a local-only prototype, a simpler implementation may be accepted,
but the code must isolate authentication behind an interface.

------------------------------------------------------------------------

## 9. Study flow

### Study session behavior

1.  User opens a deck.
2.  User clicks `Start study`.
3.  Backend selects cards due for review.
4.  If there are no due cards, optionally offer new cards.
5.  The frontend shows the question.
6.  The user attempts to recall the answer.
7.  The user clicks `Show answer`.
8.  The frontend displays:
    -   Answer
    -   Optional extended example
    -   Optional audio player
9.  The user selects a rating.
10. Backend updates scheduling state and records a review event.
11. The next card is shown.
12. The session ends when the selected cards are completed.

### Card states

Use a simple state machine:

``` text
NEW
  ↓
LEARNING
  ↓
REVIEW
  ↓
MATURE
```

A failed review may move a card back to `LEARNING`.

Do not expose unnecessary internal scheduling details in the initial UI.

------------------------------------------------------------------------

## 10. Rating options

Use four ratings in the user interface:

``` text
Again
Hard
Good
Easy
```

The UI may display friendlier labels in Portuguese:

``` text
Muito difícil
Difícil
Bom
Fácil
```

The backend should use stable enum values:

``` text
again
hard
good
easy
```

Do not use free-form strings.

------------------------------------------------------------------------

## 11. Spaced repetition algorithm

Implement a deterministic, testable scheduler.

### MVP recommendation

Start with a simple interval-based algorithm inspired by Leitner/SM-2
concepts. Keep the scheduler behind an interface so it can later be
replaced by FSRS without changing the study flow.

Suggested interface:

``` go
type Scheduler interface {
    Schedule(
        state SchedulingState,
        rating Rating,
        now time.Time,
    ) SchedulingDecision
}
```

Suggested decision:

``` go
type SchedulingDecision struct {
    State           CardState
    DueAt           time.Time
    IntervalDays    int
    Repetitions     int
    Lapses          int
    EaseFactor      float64
}
```

### Initial scheduling rules

These rules are a simple MVP policy, not a claim of optimal learning
science.

#### New card

-   `again`: review again in 10 minutes
-   `hard`: review again in 1 day
-   `good`: review again in 3 days
-   `easy`: review again in 7 days

#### Learning card

-   `again`: review again in 10 minutes
-   `hard`: review again in 1 day
-   `good`: review again in 3 days
-   `easy`: review again in 7 days

#### Review card

For a review card, use the previous interval:

-   `again`: reset to learning; due in 10 minutes
-   `hard`: previous interval × 1.2, minimum 1 day
-   `good`: previous interval × 2.0, minimum 1 day
-   `easy`: previous interval × 3.0, minimum 1 day

Apply a configurable maximum interval, for example 365 days.

### Important implementation requirements

-   Use UTC timestamps internally.
-   Inject a clock into application services.
-   Make all scheduling decisions deterministic.
-   Test boundary conditions.
-   Do not calculate scheduling in HTTP handlers.
-   Do not mix scheduling logic with MongoDB code.
-   Keep the algorithm replaceable.
-   Clearly document that the initial scheduler can later be replaced by
    FSRS.

------------------------------------------------------------------------

## 12. Exercise model

The MVP has only one exercise type:

``` text
Question → Reveal answer
```

Do not implement translation mode, cloze mode, multiple choice, or
AI-generated exercises yet.

However, model the study flow so that additional exercise types can be
added later without rewriting the entire application.

A future abstraction may look like:

``` go
type Exercise interface {
    Type() string
}
```

Do not introduce this abstraction prematurely if the current
implementation does not need it.

------------------------------------------------------------------------

## 13. API design

Use REST and version the API.

### Session

``` text
POST   /api/v1/session
DELETE /api/v1/session
GET    /api/v1/me
```

### Decks

``` text
GET    /api/v1/decks
POST   /api/v1/decks
GET    /api/v1/decks/{deckID}
PATCH  /api/v1/decks/{deckID}
DELETE /api/v1/decks/{deckID}
```

### Flashcards

``` text
GET    /api/v1/decks/{deckID}/flashcards
POST   /api/v1/decks/{deckID}/flashcards
GET    /api/v1/flashcards/{flashcardID}
PATCH  /api/v1/flashcards/{flashcardID}
DELETE /api/v1/flashcards/{flashcardID}
POST   /api/v1/flashcards/{flashcardID}/audio
DELETE /api/v1/flashcards/{flashcardID}/audio
GET    /api/v1/flashcards/{flashcardID}/audio
```

### Study

``` text
POST   /api/v1/decks/{deckID}/study-sessions
GET    /api/v1/study-sessions/{sessionID}
POST   /api/v1/study-sessions/{sessionID}/reviews
POST   /api/v1/study-sessions/{sessionID}/finish
```

The exact study-session API may be simplified if the implementation can
preserve ownership, consistency, and clear state transitions.

------------------------------------------------------------------------

## 14. Frontend screens

### 14.1 Access screen

Elements:

-   Email input
-   Continue button
-   Simple explanation of the application

### 14.2 Dashboard

Elements:

-   User email
-   List of decks
-   Number of cards
-   Number of cards due
-   Create deck button
-   Start study button
-   Empty state

### 14.3 Deck details

Elements:

-   Deck name
-   Description
-   Total cards
-   New cards
-   Cards due
-   Create flashcard button
-   Start study button
-   Edit deck button
-   Archive deck button
-   List of flashcards

### 14.4 Flashcard editor

Fields:

-   Question --- required
-   Answer --- required
-   Extended example --- optional
-   Audio --- optional
-   Save button
-   Cancel button

The editor must show validation errors clearly.

### 14.5 Study screen

Initial state:

``` text
Question

[Show answer]
```

Revealed state:

``` text
Question

Answer

Optional extended example

[Play audio]

[Again] [Hard] [Good] [Easy]
```

The user should not see the rating buttons before revealing the answer.

### 14.6 Session summary

Elements:

-   Number of cards studied
-   Number of reviews
-   Number of Again ratings
-   Number of Easy ratings
-   Button to return to deck
-   Button to continue studying if cards remain

------------------------------------------------------------------------

## 15. UX principles

-   Keep the study screen distraction-free.
-   Use large readable text for questions and answers.
-   Make the answer visually distinct from the question.
-   Make the audio player obvious but unobtrusive.
-   Disable review buttons until the answer is revealed.
-   Prevent accidental double submission.
-   Show loading states.
-   Show clear error states.
-   Preserve form data when safe.
-   Support keyboard navigation.
-   Ensure adequate color contrast.
-   Do not rely only on color to communicate difficulty.
-   Make the interface usable on desktop and mobile widths.

------------------------------------------------------------------------

## 16. Security requirements

Even though authentication is intentionally minimal:

-   Validate all request bodies.
-   Normalize and validate email.
-   Never trust user IDs from the client.
-   Enforce ownership on every deck, flashcard, session, and audio
    operation.
-   Use parameterized MongoDB operations through the official driver.
-   Avoid MongoDB operator injection by validating request DTOs.
-   Limit request body sizes.
-   Limit audio file sizes.
-   Do not log email addresses unnecessarily.
-   Do not log session tokens.
-   Use secure cookie settings where applicable.
-   Add basic rate limiting to access endpoints.
-   Configure CORS narrowly.
-   Do not expose MongoDB publicly.
-   Use environment variables for configuration.
-   Return safe error messages to clients.
-   Keep detailed errors in server logs.
-   Add request IDs for troubleshooting.
-   Document the limitations of email-only access.

------------------------------------------------------------------------

## 17. Validation rules

Suggested initial limits:

-   Email: maximum 254 characters
-   Deck name: 1--100 characters
-   Deck description: maximum 1,000 characters
-   Question: 1--5,000 characters
-   Answer: 1--5,000 characters
-   Extended example: maximum 10,000 characters
-   Audio: configurable maximum, for example 10 MB

These values must be configurable and tested.

Reject:

-   Empty required fields
-   Oversized payloads
-   Unsupported audio types
-   Invalid ObjectIDs
-   Unknown rating values
-   Requests for resources belonging to another user

------------------------------------------------------------------------

## 18. MongoDB indexes

Create and document indexes such as:

``` text
users:
  unique(email)

decks:
  userId + archivedAt
  userId + name

flashcards:
  userId + deckId + archivedAt
  userId + scheduling.dueAt
  deckId + scheduling.dueAt

review_events:
  userId + reviewedAt
  flashcardId + reviewedAt
```

Review the indexes using realistic query patterns.

Do not create indexes without a query or uniqueness requirement.

------------------------------------------------------------------------

## 19. Testing strategy

### Unit tests

Cover:

-   Email normalization
-   Validation
-   Scheduler behavior
-   Rating transitions
-   New card scheduling
-   Review card scheduling
-   Lapses
-   Maximum interval
-   Clock-dependent behavior
-   Ownership rules in application services

### Integration tests

Cover:

-   MongoDB repositories
-   Deck creation and retrieval
-   Flashcard creation
-   Audio upload and retrieval through GridFS
-   Audio replacement and cleanup
-   Review persistence
-   Due-card queries
-   Unique email behavior

Use a dedicated test database or a disposable MongoDB environment.

### HTTP tests

Cover:

-   Valid requests
-   Invalid requests
-   Missing authentication
-   Ownership violations
-   Not-found behavior
-   Invalid IDs
-   Payload size limits
-   Invalid ratings
-   Duplicate review submission

### Frontend tests

At minimum, cover:

-   Access form
-   Deck creation
-   Flashcard form validation
-   Question/reveal-answer flow
-   Audio player rendering
-   Rating button behavior
-   Loading and error states

------------------------------------------------------------------------

## 20. Error model

Use a consistent API error format:

``` json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "The question is required.",
    "requestId": "request-id"
  }
}
```

Suggested codes:

``` text
VALIDATION_ERROR
UNAUTHORIZED
FORBIDDEN
NOT_FOUND
CONFLICT
PAYLOAD_TOO_LARGE
UNSUPPORTED_MEDIA_TYPE
RATE_LIMITED
INTERNAL_ERROR
```

Do not expose stack traces or database errors to the frontend.

------------------------------------------------------------------------

## 21. Observability

Implement basic structured logging:

-   Request ID
-   HTTP method
-   Route
-   Status code
-   Duration
-   Error code
-   Relevant resource IDs

Do not log:

-   Session tokens
-   Audio contents
-   Full request bodies
-   Sensitive information

A health endpoint may be added:

``` text
GET /health
```

It should indicate application availability and, if appropriate,
database connectivity.

------------------------------------------------------------------------

## 22. Configuration

Use environment variables for:

``` text
APP_ENV
HTTP_PORT
MONGODB_URI
MONGODB_DATABASE
SESSION_SECRET
COOKIE_SECURE
MAX_AUDIO_SIZE_BYTES
CORS_ALLOWED_ORIGINS
```

Provide:

-   `.env.example`
-   Configuration validation at startup
-   Safe defaults for local development
-   Clear startup errors for missing required configuration

Never commit real secrets.

------------------------------------------------------------------------

## 23. Development workflow for the agent

Work in small, verifiable increments.

### Phase 1 --- Project foundation

1.  Create repository structure.
2.  Set up Go module.
3.  Set up React/Vite frontend.
4.  Add Docker Compose.
5.  Add configuration loading.
6.  Add health endpoint.
7.  Add basic CI checks.
8.  Add README.

### Phase 2 --- Domain and persistence

1.  Define domain entities.
2.  Define repository interfaces.
3.  Implement MongoDB connection.
4.  Implement indexes.
5.  Implement user repository.
6.  Implement deck repository.
7.  Implement flashcard repository.
8.  Add repository tests.

### Phase 3 --- Access flow

1.  Implement email validation.
2.  Implement user creation or lookup.
3.  Implement session handling.
4.  Add authentication middleware.
5.  Add ownership checks.
6.  Add HTTP tests.

### Phase 4 --- Deck and flashcard management

1.  Implement deck endpoints.
2.  Implement flashcard endpoints.
3.  Implement validation.
4.  Implement frontend dashboard.
5.  Implement deck details.
6.  Implement flashcard editor.
7.  Add end-to-end manual verification.

### Phase 5 --- Audio

1.  Implement GridFS adapter.
2.  Implement upload endpoint.
3.  Validate file size and content type.
4.  Store audio metadata.
5.  Implement authenticated playback.
6.  Implement replacement and cleanup.
7.  Add integration tests.

### Phase 6 --- Study flow

1.  Implement scheduler interface.
2.  Implement initial deterministic scheduler.
3.  Implement due-card query.
4.  Implement study-session service.
5.  Implement reveal-answer UI.
6.  Implement rating buttons.
7.  Persist review events.
8.  Update scheduling state.
9.  Implement session summary.
10. Add unit and HTTP tests.

### Phase 7 --- Quality and documentation

1.  Improve error handling.
2.  Add logging.
3.  Review ownership rules.
4.  Review indexes.
5.  Add API documentation.
6.  Add architecture documentation.
7.  Add local setup instructions.
8.  Run formatting, tests, static analysis, and build.
9.  Review the implementation against this specification.
10. Document known limitations and future improvements.

------------------------------------------------------------------------

## 24. Definition of done

A feature is complete only when:

-   The domain behavior is clearly defined.
-   The implementation is separated from infrastructure concerns.
-   Validation exists.
-   Errors are handled consistently.
-   Ownership is enforced.
-   Tests cover the important behavior.
-   The frontend handles loading, success, and error states.
-   Documentation is updated.
-   The code is formatted.
-   Tests pass.
-   The application builds successfully.
-   No unrelated feature was added.

------------------------------------------------------------------------

## 25. Agent behavior rules

When implementing the project:

1.  Inspect the existing repository before making changes.
2.  Do not overwrite existing work without understanding it.
3.  Explain the implementation plan before large changes.
4.  Prefer small commits or logically separated changes.
5.  Keep business rules out of HTTP handlers.
6.  Keep MongoDB-specific details out of domain entities.
7.  Use interfaces at meaningful boundaries, not everywhere.
8.  Avoid premature abstractions.
9.  Write tests before or together with non-trivial business logic.
10. Never silently change API contracts.
11. Update documentation when behavior changes.
12. Do not introduce AI or external integrations.
13. Do not add passwords or OAuth.
14. Do not claim that email-only access provides strong identity
    verification.
15. Ask for clarification only when a decision is genuinely blocking.
16. When a reasonable default exists, document it and proceed.
17. Before finishing, run the complete verification checklist.

------------------------------------------------------------------------

## 26. Suggested future improvements

Do not implement these in the MVP, but document them:

-   Passwordless magic-link authentication
-   OAuth
-   FSRS scheduler
-   Multiple exercise types
-   Cloze deletion
-   Translation mode
-   Pronunciation practice
-   Text-to-speech generation
-   External object storage
-   Import/export
-   Tags and concepts
-   Learning analytics
-   Notifications
-   Shared decks
-   Offline support
-   Mobile application
-   More advanced permission models

------------------------------------------------------------------------

## 27. First task for the coding agent

Before writing application code:

1.  Inspect the repository.
2.  Confirm whether the repository is empty or already contains code.
3.  Create a concise implementation plan.
4.  Identify assumptions.
5.  Create the initial project skeleton.
6.  Add local development instructions.
7.  Implement the health endpoint and configuration.
8.  Add the first tests.
9.  Report exactly what was created, how to run it, and what remains.

Do not implement the entire system in one uncontrolled change.

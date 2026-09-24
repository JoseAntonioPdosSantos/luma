# Agent Specification --- Screenshot-to-Flashcard Importer

## 1. Role

You are a senior software engineer responsible for implementing an
automated import agent for the existing Flashcard Learning System.

The existing system uses:

-   Go backend
-   MongoDB
-   MongoDB GridFS for optional audio
-   React + TypeScript frontend
-   Decks and flashcards
-   Question, answer, optional extended example, and optional audio
-   Spaced-repetition scheduling

Your responsibility is to build a separate, focused ingestion workflow
that reads screenshots copied into a local folder and converts their
content into flashcards.

The importer must be reliable, repeatable, auditable, and safe against
duplicate cards.

Do not redesign the existing flashcard application unnecessarily.

------------------------------------------------------------------------

## 2. Product objective

The user will copy screenshots from a mobile application into an input
folder.

The importer must:

1.  Detect new image files.
2.  Read visible text from each image.
3.  Interpret the relevant learning content.
4.  Create a question in Portuguese.
5.  Create the answer in the selected target language:
    -   English
    -   French
    -   Another explicitly configured language
6.  Preserve useful context from the screenshot.
7.  Optionally create an extended example.
8.  Obtain or associate audio for the target-language answer when
    possible.
9.  Verify whether the image was already processed.
10. Verify whether the resulting card already exists.
11. Save the card in the existing MongoDB model.
12. Record a detailed import result.
13. Move or mark processed files so they are not repeatedly imported.
14. Produce a report containing created cards, skipped duplicates,
    failures, and items requiring review.

The importer must support batch processing.

------------------------------------------------------------------------

## 3. Important constraints

### Included

-   Local folder scanning
-   PNG, JPG, JPEG, and WEBP image support where available
-   OCR or image-text extraction
-   Screenshot interpretation
-   Portuguese question generation
-   Target-language answer generation
-   Optional extended example
-   Audio acquisition through a replaceable provider
-   Duplicate detection
-   Idempotent processing
-   MongoDB persistence
-   GridFS audio storage
-   Import history
-   Dry-run mode
-   Validation
-   Error reporting
-   Retryable failures
-   Manual review queue
-   Unit and integration tests
-   CLI execution
-   Documentation
-   Deletion of an original screenshot after it has been successfully
    turned into a flashcard (see section 9a)
-   Rate limiting of every external translation/OCR and audio (TTS)
    call, bounded at 60 calls per minute per provider, with built-in
    429 protection (see section 32)

### Excluded by default

-   Changes to the core study experience
-   Automatic modification of existing cards
-   Automatic merging of potentially similar cards
-   Automatic creation of decks without an explicit configured rule
-   Automatic guessing of the target language when confidence is low
-   Silent translation of ambiguous text
-   Silent replacement of existing audio
-   Unbounded folder monitoring
-   Sending images or text to external services without explicit
    configuration and consent

The importer may use an OCR, translation, or audio provider only if the
execution environment explicitly makes that capability available and the
provider is configured. The importer must not pretend that an
unavailable provider succeeded.

------------------------------------------------------------------------

## 4. Important interpretation rule

A screenshot may contain:

-   A word
-   A phrase
-   A dialogue
-   A definition
-   A sentence
-   A lesson explanation
-   Multiple unrelated items
-   Navigation elements
-   Advertisements
-   UI labels
-   Images without useful text
-   Text in a language other than the expected source language

The agent must distinguish learning content from irrelevant interface
text.

When the screenshot is ambiguous, the importer should create a
`needs_review` result instead of inventing content.

The importer must preserve the original OCR text and relevant source
metadata for traceability.

------------------------------------------------------------------------

## 5. Target-language behavior

The import job must explicitly specify the target language.

Example configuration:

``` yaml
sourceLanguage: auto
targetLanguage: en
questionLanguage: pt-BR
deckId: english-deck-id
```

For French:

``` yaml
sourceLanguage: auto
targetLanguage: fr
questionLanguage: pt-BR
deckId: french-deck-id
```

Supported target-language identifiers should be stable values, for
example:

``` text
en
fr
es
de
it
```

Do not infer the target language from the image alone unless the
confidence is high and the behavior is explicitly enabled.

If the target language is not configured, stop the item for manual
review.

------------------------------------------------------------------------

## 6. Expected transformation

### Example screenshot content

``` text
I have to work late today.
```

### Expected flashcard

``` text
Question:
Como dizer em inglês: "Eu tenho que trabalhar até tarde hoje"?

Answer:
I have to work late today.

Extended example:
I have to work late today because we have an important release tomorrow.
```

For French:

``` text
Question:
Como dizer em francês: "Eu tenho que trabalhar até tarde hoje"?

Answer:
Je dois travailler tard aujourd'hui.

Extended example:
Je dois travailler tard aujourd'hui parce que nous avons une mise en production importante demain.
```

The exact wording must be validated for grammatical correctness and
fidelity to the source.

The importer must not blindly translate every visible line. It must
identify the actual learning unit.

------------------------------------------------------------------------

## 7. Agent architecture

Create a separate importer module or service.

Suggested Go structure:

``` text
importer/
├── cmd/
│   └── flashcard-importer/
│       └── main.go
├── internal/
│   ├── config/
│   ├── domain/
│   │   ├── importjob/
│   │   ├── importitem/
│   │   ├── extraction/
│   │   ├── translation/
│   │   ├── duplicate/
│   │   └── audio/
│   ├── application/
│   │   ├── scanservice/
│   │   ├── importservice/
│   │   ├── duplicatecheckservice/
│   │   ├── audioservice/
│   │   └── reportservice/
│   ├── ports/
│   │   ├── imageinput/
│   │   ├── ocr/
│   │   ├── contentinterpreter/
│   │   ├── translator/
│   │   ├── audioprovider/
│   │   ├── cardrepository/
│   │   ├── importrepository/
│   │   ├── filestore/
│   │   └── clock/
│   ├── adapters/
│   │   ├── filesystem/
│   │   ├── mongodb/
│   │   ├── gridfs/
│   │   ├── ocr/
│   │   ├── translation/
│   │   └── audio/
│   └── cli/
├── docs/
├── testdata/
├── Dockerfile
└── go.mod
```

Use dependency inversion for external capabilities.

Do not hard-code OCR, translation, or audio providers into the domain
layer.

------------------------------------------------------------------------

## 8. Execution modes

The CLI must support at least these modes:

### Single batch

``` bash
flashcard-importer import --input ./incoming
```

### Dry run

``` bash
flashcard-importer import --input ./incoming --dry-run
```

Dry run must:

-   Read and analyze files
-   Detect possible duplicates
-   Generate a proposed import result
-   Not write flashcards
-   Not write audio
-   Not modify source files
-   Produce a report

### Single file

``` bash
flashcard-importer import --file ./incoming/example.png
```

### Reprocess failed files

``` bash
flashcard-importer retry --input ./failed
```

### Report

``` bash
flashcard-importer report --job-id <job-id>
```

### Validate configuration

``` bash
flashcard-importer config validate
```

------------------------------------------------------------------------

## 9. Folder lifecycle

Use a predictable folder structure:

``` text
data/
├── incoming/
├── processing/
├── processed/
├── duplicates/
├── failed/
├── review/
└── reports/
```

Rules:

-   Only supported image files in `incoming/` are candidates.
-   Move a file to `processing/` before processing when possible.
-   Use an atomic move or a lock strategy to avoid two workers
    processing the same file.
-   On successful import, create the flashcard first, confirm it was
    persisted, then delete the original image (see section 9a). Do not
    move it to `processed/` and keep the file around afterward.
-   If the image represents an already existing card, delete it the
    same way (it is now redundant) instead of moving it to
    `duplicates/`.
-   If the item is ambiguous, move it to `review/` and **do not**
    delete it — a `needs_review` item still needs the original image
    for a human (or a later run) to resolve it.
-   If processing fails, move it to `failed/` and record the error —
    **do not** delete it, for the same reason.
-   Preserve the original filename and a stable processing record even
    after the source image is gone (the record, not the file, is the
    audit trail).
-   Do not treat a moved file as a new item merely because its path
    changed.

If moving files is not possible, maintain a persistent processing
registry and use file hashes.

------------------------------------------------------------------------

## 9a. Deletion of processed screenshots

Status: **explicitly requested by the user** (2026-09-16) — this
overrides the more conservative "never delete automatically" default
that section 3 used to state. Implement it as described here, not as
an unconditional delete-everything behavior.

### When to delete

Delete the original image file **only after** the corresponding
flashcard (or duplicate confirmation) has been durably persisted —
i.e. after step 7/8 of the idempotent persistence workflow in section
18 has completed successfully, never before.

``` text
Outcome            → File action
------------------- -------------------------------------------------
created             → delete original image
source_duplicate    → delete original image (content already imported)
exact_card_duplicate→ delete original image (content already imported)
possible_duplicate  → move to review/, KEEP the image
needs_review        → move to review/, KEEP the image
failed              → move to failed/, KEEP the image
```

### Safety rules

-   Never delete before the database write is confirmed. If the
    process crashes between "flashcard persisted" and "image deleted",
    that is acceptable (a retry will find the card via its fingerprint
    and can safely delete the leftover image then) — the reverse order
    (delete-then-write) is not acceptable and must never happen.
-   Never delete a file that is still being read (finish OCR/vision
    processing and the persistence call before touching the
    filesystem).
-   Log every deletion with the import item ID, so the report in
    section 21 still shows what happened to each original file even
    though it no longer exists on disk.
-   This applies identically to the local `images/` folder flow and to
    files extracted from an uploaded ZIP (section 30) — extracted
    per-job temporary files should be deleted the same way, plus the
    whole per-job temp directory once every item in it has reached a
    terminal state.

## 10. Image identity and idempotency

Calculate a cryptographic hash of the original file, preferably SHA-256.

Store:

``` text
sourceFile
- originalFilename
- originalPath
- sha256
- fileSize
- mimeType
- discoveredAt
- processedAt
```

Create a unique index on the source file hash for import records.

The same exact image must never be processed successfully twice.

Do not use only the filename as identity because users may rename files.

If two visually identical screenshots have different binary hashes, the
importer may optionally calculate a perceptual hash. This is an
enhancement, not a mandatory MVP feature.

------------------------------------------------------------------------

## 11. OCR and extraction pipeline

If OCR/vision is provided by an external or rate-limited service, every
call goes through the shared rate limiter in section 32, the same as
translation and audio calls.

The extraction pipeline should be explicit:

``` text
Image
  ↓
Image validation
  ↓
OCR
  ↓
OCR normalization
  ↓
Content segmentation
  ↓
Learning-content selection
  ↓
Structured extraction
```

### OCR output

Store:

``` text
OCRResult
- rawText
- normalizedText
- detectedLanguage
- confidence
- boundingBoxes, if available
- provider
- extractedAt
```

### Normalization

Normalization may include:

-   Unicode normalization
-   Trimming whitespace
-   Converting repeated spaces to one space
-   Normalizing line breaks
-   Removing obvious OCR artifacts
-   Preserving punctuation that affects meaning
-   Preserving accents
-   Preserving apostrophes and contractions
-   Avoiding destructive lowercasing of the original content

Never overwrite the raw OCR text.

### Extraction result

``` text
ExtractedLearningContent
- sourceText
- sourceLanguage
- contentType
- relevantText
- contextText
- confidence
- warnings
```

Possible content types:

``` text
word
phrase
sentence
dialogue
definition
explanation
unknown
```

If no useful learning content is found, mark the item as `needs_review`.

------------------------------------------------------------------------

## 12. Content generation rules

If translation/content-interpretation is provided by an external
service, every call goes through the shared rate limiter in section 32.

The importer must generate a structured draft:

``` text
CardDraft
- question
- answer
- extendedExample
- sourceText
- sourceLanguage
- targetLanguage
- confidence
- warnings
```

### Question

The question must be in Brazilian Portuguese by default.

Examples:

``` text
Como dizer "resgatar" em inglês?
Como dizer "Eu estava trabalhando" em francês?
O que significa "rescue"?
```

The question format should reflect the content:

-   For a target-language phrase, ask how to say it in the target
    language.
-   For a source-language word, ask for the target-language equivalent.
-   For a definition, ask the user to recall the target-language term.
-   For a sentence, preserve the intended meaning.
-   For an ambiguous item, request manual review.

### Answer

The answer must be in the configured target language.

Rules:

-   Preserve meaning.
-   Preserve tense, number, person, register, and context where
    relevant.
-   Do not invent a more advanced sentence without marking it as an
    example.
-   Preserve multiple valid translations when necessary.
-   Avoid adding explanations to the answer field.
-   Keep explanations in `extendedExample` or metadata.

### Extended example

An extended example is optional.

Create one only when:

-   The source already contains a useful example, or
-   A configured content rule explicitly allows generating one, and
-   The example is faithful to the learning item.

Do not create unnecessary long text for every simple word.

If an example is generated, store:

``` text
ExtendedExample
- text
- translation
- origin: source | generated
```

The importer must make it clear whether the example came from the
screenshot or was generated by the importer.

------------------------------------------------------------------------

## 13. Audio behavior

The importer must use an audio provider interface:

``` go
type AudioProvider interface {
    GetAudio(ctx context.Context, text string, language string) (AudioAsset, error)
}
```

The provider may be implemented by:

-   A local audio file lookup
-   A local pronunciation library
-   A configured external service
-   A manually supplied audio asset

The importer must not assume that an audio provider exists.

### Audio rules

-   Audio is optional.
-   If audio is unavailable, the card may still be created if the job
    configuration permits it.
-   If audio is mandatory and unavailable, the card must go to
    `needs_review` or `failed`, according to configuration.
-   Do not fabricate an audio file.
-   Validate MIME type and file size.
-   Store audio binary data in MongoDB GridFS.
-   Store only the GridFS file ID and metadata in the flashcard.
-   Reuse an existing identical audio asset when safe.
-   Do not replace existing audio silently.
-   Store target language and source text used to obtain the audio.
-   Record provider and acquisition status.
-   Every call to the audio provider goes through the shared rate
    limiter described in section 32 — never call it in a tight loop.

Suggested audio metadata:

``` json
{
  "gridFsFileId": "ObjectId",
  "contentType": "audio/mpeg",
  "language": "en",
  "text": "I have to work late today.",
  "provider": "configured-provider",
  "sha256": "audio-hash",
  "durationMs": 2500
}
```

If the project has no permitted audio provider, implement the interface
and a clear `audio_unavailable` result. Do not add an external
integration automatically.

------------------------------------------------------------------------

## 14. Duplicate detection

Duplicate detection must happen at multiple levels.

### Level 1 --- Exact source-image duplicate

Compare the source file SHA-256 against import history.

Result:

``` text
source_duplicate
```

### Level 2 --- Exact normalized card duplicate

Build a canonical fingerprint from:

-   User ID
-   Deck ID
-   Target language
-   Canonical question
-   Canonical answer

Suggested canonicalization:

-   Unicode normalization
-   Trim
-   Collapse whitespace
-   Normalize line endings
-   Normalize equivalent apostrophe forms where safe
-   Case folding only for comparison, not for stored content
-   Do not remove meaningful punctuation
-   Do not remove accents from the stored content

Hash the canonical representation:

``` text
cardFingerprint = SHA256(
  userId +
  deckId +
  targetLanguage +
  canonicalQuestion +
  canonicalAnswer
)
```

Store the fingerprint on the flashcard and create a unique index scoped
to the user/deck as appropriate.

### Level 3 --- Semantic or near-duplicate detection

Do not automatically merge semantically similar cards in the MVP.

Examples:

``` text
I need to leave now.
I have to leave now.
```

These may be related but are not necessarily duplicates.

For near-duplicates:

-   Mark as `possible_duplicate`
-   Include the existing card IDs
-   Send to manual review
-   Do not delete or modify either card automatically

A future implementation may use more advanced similarity logic, but it
must be explicit and reviewable.

------------------------------------------------------------------------

## 15. Duplicate decision policy

Use clear outcomes:

``` text
created
source_duplicate
exact_card_duplicate
possible_duplicate
needs_review
failed
```

Suggested behavior:

  Situation                                       Action
  ----------------------------------------------- ------------------------------------------
  Same image hash already imported successfully   Skip
  Same exact card fingerprint exists              Skip and report duplicate
  Similar but not identical card exists           Manual review
  OCR confidence too low                          Manual review
  Target language missing                         Manual review
  Translation unavailable                         Failed or manual review, based on policy
  Audio unavailable but optional                  Create without audio and warn
  Audio unavailable but required                  Manual review or failed
  Database failure                                Retryable failure
  Invalid image                                   Failed

The agent must never silently discard an item.

------------------------------------------------------------------------

## 16. MongoDB model additions

Extend the existing flashcard model minimally.

### Flashcard additions

``` text
Flashcard
- source
- cardFingerprint
- targetLanguage
- importMetadata
```

Suggested structure:

``` json
{
  "source": {
    "type": "screenshot",
    "importItemId": "ObjectId",
    "sourceFileHash": "sha256",
    "originalFilename": "screenshot_001.png"
  },
  "cardFingerprint": "sha256",
  "targetLanguage": "en",
  "importMetadata": {
    "ocrProvider": "configured-provider",
    "ocrConfidence": 0.96,
    "translationProvider": "configured-provider",
    "translationConfidence": 0.91,
    "importedAt": "2026-09-14T00:00:00Z"
  }
}
```

Do not store the full screenshot inside the flashcard by default.

Store the source image path or a durable source reference only if the
application's privacy and retention policy allows it.

### ImportJob

``` text
ImportJob
- id
- userId
- inputDirectory
- targetLanguage
- questionLanguage
- deckId
- mode
- status
- startedAt
- finishedAt
- counters
- createdAt
```

Counters:

``` text
discovered
processed
created
sourceDuplicates
exactCardDuplicates
possibleDuplicates
needsReview
failed
```

### ImportItem

``` text
ImportItem
- id
- jobId
- userId
- sourceFile
- sourceHash
- status
- rawOCRText
- normalizedOCRText
- extractedContent
- cardDraft
- duplicateCandidates
- error
- retryCount
- createdFlashcardId
- createdAt
- updatedAt
- processedAt
```

Use a unique index on the source hash for the relevant ownership scope.

------------------------------------------------------------------------

## 17. MongoDB indexes

Create indexes based on actual queries.

At minimum, evaluate:

``` text
import_items:
  unique(sourceHash + userId)
  jobId + status
  userId + createdAt

flashcards:
  userId + deckId + cardFingerprint
  userId + source.sourceFileHash
  userId + targetLanguage

import_jobs:
  userId + createdAt
```

The exact uniqueness model must be implemented using MongoDB's supported
index semantics. If nullable fields are involved, use partial indexes or
another explicit strategy.

Do not rely only on application-level duplicate checks. Use database
uniqueness constraints to protect against concurrent imports.

------------------------------------------------------------------------

## 18. Idempotent persistence workflow

The import process must be safe to retry.

Recommended order:

1.  Calculate source hash.
2.  Atomically register or claim the import item.
3.  If already successfully processed, skip.
4.  Extract and build the draft.
5.  Calculate the card fingerprint.
6.  Check for an exact existing card.
7.  Persist the flashcard using a uniqueness constraint.
8.  Persist or associate audio if applicable.
9.  Mark the import item as successful.
10. Move the source file to the final folder.
11. Write the report.

Handle partial failures carefully.

If a flashcard is created but the process crashes before marking the
import item complete, a retry must detect the existing card fingerprint
and avoid creating another card.

Do not use an unsafe sequence of "check then insert" without a unique
index or equivalent atomic protection.

Consider MongoDB transactions only where they provide a real consistency
benefit and are supported by the local deployment. Do not add
transactions indiscriminately.

------------------------------------------------------------------------

## 19. CLI configuration

Support a configuration file such as:

``` yaml
inputDirectory: ./data/incoming
processingDirectory: ./data/processing
processedDirectory: ./data/processed
duplicateDirectory: ./data/duplicates
failedDirectory: ./data/failed
reviewDirectory: ./data/review
reportDirectory: ./data/reports

mongodb:
  uri: mongodb://localhost:27017
  database: flashcards

userId: ""
deckId: ""
questionLanguage: pt-BR
targetLanguage: en

audio:
  enabled: true
  required: false
  maxSizeBytes: 10485760

processing:
  dryRun: false
  maxFilesPerRun: 100
  maxRetries: 3
  minimumOCRConfidence: 0.80
  minimumContentConfidence: 0.80
  allowGeneratedExamples: false
```

Do not store secrets in the configuration file.

Use environment variables for credentials and sensitive values.

------------------------------------------------------------------------

## 20. User and deck ownership

The importer must never accept arbitrary ownership from an image or OCR
result.

The import job must run in the context of an explicitly configured user
and deck.

Before writing:

-   Verify that the user exists.
-   Verify that the deck exists.
-   Verify that the deck belongs to the user.
-   Use the authenticated or explicitly authorized execution context.
-   Never trust a user ID extracted from screenshot text.

If the user or deck is missing, fail configuration validation before
processing images.

------------------------------------------------------------------------

## 21. Reports

Generate a machine-readable JSON report and a human-readable summary.

### JSON report

Include:

``` text
jobId
startedAt
finishedAt
configuration
counters
items:
  - source filename
  - source hash
  - status
  - flashcard ID
  - duplicate candidate IDs
  - warnings
  - error code
  - error message
```

### Human-readable summary

Example:

``` text
Import completed

Discovered: 20
Created: 12
Exact duplicates: 5
Possible duplicates: 2
Needs review: 1
Failed: 0
```

Never include secrets or raw audio data in reports.

------------------------------------------------------------------------

## 22. Error handling

Use typed error categories:

``` text
INVALID_IMAGE
UNSUPPORTED_FORMAT
OCR_UNAVAILABLE
OCR_LOW_CONFIDENCE
NO_LEARNING_CONTENT
TARGET_LANGUAGE_MISSING
TRANSLATION_UNAVAILABLE
INVALID_CARD_DRAFT
AUDIO_UNAVAILABLE
AUDIO_INVALID
SOURCE_DUPLICATE
EXACT_CARD_DUPLICATE
POSSIBLE_DUPLICATE
DATABASE_ERROR
FILE_MOVE_ERROR
CONFIGURATION_ERROR
RATE_LIMITED
```

(`RATE_LIMITED` mirrors the existing `RATE_LIMITED` `apperror` code
already used by the main flashcard backend's session endpoint — reuse
that convention instead of inventing a new one.)

Rules:

-   Preserve the original error internally.
-   Return safe, useful messages in reports.
-   Mark retryable errors separately from permanent errors.
-   Do not retry invalid images indefinitely.
-   Use bounded retries with backoff for transient database or provider
    failures.
-   A `429`/`RATE_LIMITED` response from a translation, OCR, or audio
    provider is always retryable — see section 32 for the exact backoff
    strategy. Never let it fall through as a generic permanent failure.
-   Never hide a failed item.

------------------------------------------------------------------------

## 23. Observability

Log structured events for:

-   Job started
-   File discovered
-   File claimed
-   OCR completed
-   Draft created
-   Duplicate detected
-   Audio acquired
-   Card persisted
-   Item skipped
-   Item sent to review
-   Item failed
-   Job completed

Include:

-   Job ID
-   Import item ID
-   Source hash
-   Filename
-   Status
-   Duration
-   Error code

Do not log:

-   Session tokens
-   Secrets
-   Complete sensitive image content
-   Unnecessary personal data
-   Full audio binaries

------------------------------------------------------------------------

## 24. Testing strategy

### Unit tests

Cover:

-   File extension validation
-   SHA-256 calculation
-   OCR normalization
-   Canonical question normalization
-   Canonical answer normalization
-   Card fingerprint generation
-   Duplicate decision rules
-   Target-language validation
-   Draft validation
-   Folder lifecycle decisions
-   Retry classification
-   Configuration validation

### Integration tests

Cover:

-   MongoDB import-item persistence
-   Unique source-hash constraint
-   Unique card-fingerprint constraint
-   Flashcard creation
-   Retry after partial failure
-   GridFS audio persistence
-   Audio reuse
-   Import report persistence if implemented
-   Ownership validation

### End-to-end tests

Use deterministic fake adapters for:

-   OCR
-   Translation/content interpretation
-   Audio

Test:

1.  New screenshot creates one card.
2.  Same screenshot is skipped on the second run.
3.  Renamed same screenshot is skipped by hash.
4.  Different screenshot with identical learning content is skipped by
    card fingerprint.
5.  Similar content is flagged for review.
6.  Low-confidence OCR is sent to review.
7.  Optional audio failure still creates the card with a warning.
8.  Required audio failure does not create an incomplete card.
9.  Crash/retry does not create duplicates.
10. Dry run does not modify MongoDB or source folders.
11. Wrong deck ownership is rejected.
12. Invalid image is reported correctly.
13. A `created` item's source image is deleted only after the
    flashcard write is confirmed.
14. A `needs_review`, `possible_duplicate`, or `failed` item's source
    image is never deleted.
15. A fake provider that returns 429 on the Nth call causes the
    importer to back off and retry rather than fail the batch, and the
    batch never exceeds 60 calls/minute to that provider (section 32).

------------------------------------------------------------------------

## 25. Privacy and data handling

Screenshots may contain personal information or copyrighted material.

The importer must:

-   Process only explicitly configured folders.
-   Not upload images externally unless a provider is explicitly
    configured.
-   Avoid storing complete screenshots in MongoDB by default.
-   Preserve only the minimum source metadata needed for traceability.
-   Delete a screenshot once it has produced a card or was confirmed as
    a duplicate (section 9a) — deletion enabled by default per explicit
    user instruction. Provide a configuration option to switch back to
    retention for a given run, for users who prefer to keep originals.
-   Never delete a screenshot whose outcome is `possible_duplicate`,
    `needs_review`, or `failed` — those still need the original file.
-   Never expose imported content publicly.
-   Document what external providers receive, if any provider is
    enabled.
-   Fail closed when a required privacy configuration is missing.

------------------------------------------------------------------------

## 26. Implementation phases

### Phase 1 --- Foundation

1.  Inspect the existing flashcard repository.
2.  Confirm the existing MongoDB schema and repository interfaces.
3.  Create importer module.
4.  Add CLI.
5.  Add configuration.
6.  Add folder lifecycle.
7.  Add source hashing.
8.  Add import-item persistence.
9.  Add dry-run mode.

### Phase 2 --- Extraction

1.  Define OCR interface.
2.  Implement a deterministic fake OCR adapter for tests.
3.  Implement image validation.
4.  Implement OCR normalization.
5.  Implement extraction result model.
6.  Add low-confidence and empty-content handling.

### Phase 3 --- Card draft

1.  Define content interpretation interface.
2.  Define translation/content-generation interface.
3.  Implement structured card draft validation.
4.  Implement Portuguese question rules.
5.  Implement target-language validation.
6.  Preserve source text and warnings.

### Phase 4 --- Duplicate protection

1.  Implement source-hash duplicate detection.
2.  Implement canonical card fingerprint.
3.  Add MongoDB uniqueness constraints.
4.  Implement exact duplicate handling.
5.  Implement possible-duplicate review handling.
6.  Add retry-safe persistence.

### Phase 5 --- Audio

1.  Define audio provider interface.
2.  Implement fake provider for tests.
3.  Implement GridFS storage adapter.
4.  Add audio validation.
5.  Add optional/required audio policies.
6.  Add audio reuse and cleanup behavior.

### Phase 6 --- Production workflow

1.  Implement batch processing.
2.  Implement processing claims.
3.  Implement bounded retries.
4.  Implement reports.
5.  Implement processed/duplicate/review/failed folder movement.
6.  Add structured logging.
7.  Add complete integration tests.

### Phase 7 --- Verification

1.  Run formatting.
2.  Run unit tests.
3.  Run integration tests.
4.  Run static analysis.
5.  Run the CLI in dry-run mode.
6.  Run a real local batch with test images.
7.  Verify that repeated execution creates no duplicates.
8.  Verify that audio playback works in the existing frontend.
9.  Review ownership and privacy behavior.
10. Update documentation.

------------------------------------------------------------------------

## 27. Definition of done

The importer is complete only when:

-   It can scan a configured folder.
-   It can process supported image files.
-   It records source hashes.
-   It is idempotent.
-   It produces structured card drafts.
-   It generates Portuguese questions.
-   It uses an explicitly configured target language.
-   It stores optional audio through the existing GridFS model.
-   It prevents exact duplicate cards.
-   It flags possible duplicates instead of silently merging them.
-   It preserves import history.
-   It supports dry-run mode.
-   It produces useful reports.
-   It handles failures without losing traceability.
-   It verifies user and deck ownership.
-   It has unit, integration, and end-to-end tests.
-   It does not add AI or external integrations unless explicitly
    configured.
-   It does not modify existing cards automatically.
-   It does not silently invent content from ambiguous screenshots.

------------------------------------------------------------------------

## 28. Agent behavior rules

When implementing:

1.  Inspect the existing flashcard codebase before changing anything.
2.  Reuse existing domain models, repositories, authentication context,
    and GridFS abstractions when appropriate.
3.  Do not duplicate the existing flashcard persistence logic.
4.  Do not create a second incompatible Flashcard schema.
5.  Do not add external services without explicit configuration.
6.  Do not assume OCR, translation, or audio capabilities are available.
7.  Use ports/interfaces for optional capabilities.
8.  Use deterministic fake adapters in tests.
9.  Never silently overwrite existing cards.
10. Never silently ignore duplicate or failed items.
11. Prefer manual review over guessing.
12. Keep raw OCR and normalized content separate.
13. Use unique database constraints to protect against concurrency.
14. Make retries safe.
15. Do not delete source images automatically.
16. Keep the importer independently executable.
17. Keep business rules out of CLI and filesystem adapters.
18. Report assumptions and unresolved limitations.
19. Run verification before declaring completion.
20. Do not implement unrelated features.

------------------------------------------------------------------------

## 29. First task for the coding agent

Before writing code:

1.  Inspect the existing flashcard repository.
2.  Locate:
    -   Flashcard domain model
    -   Deck model
    -   User model
    -   MongoDB repositories
    -   GridFS implementation
    -   Authentication/ownership mechanism
    -   Existing API contracts
3.  Determine the safest integration point.
4.  Identify schema changes required.
5.  Produce a short implementation plan.
6.  Identify capabilities that are unavailable:
    -   OCR
    -   Translation/content interpretation
    -   Audio provider
7.  Propose deterministic adapters for unavailable capabilities.
8.  Implement only the foundation first:
    -   Configuration
    -   CLI
    -   Folder scanner
    -   SHA-256 source hashing
    -   Import-item persistence
    -   Dry-run report
9.  Run tests.
10. Report exactly what was implemented and what remains.

Do not implement the entire importer in one uncontrolled change.

------------------------------------------------------------------------

## 30. Planned enhancement --- ZIP upload from the frontend

Status: **not implemented yet**. This section specifies the work for a
future execution of this agent; do not build it until explicitly asked.

### Goal

A signed-in user must be able to, from the existing React frontend,
choose a `.zip` file containing screenshots and have the system import
them into their flashcards without touching the filesystem or a CLI.

### Backend

1.  New endpoint, following the existing router conventions in
    `internal/adapters/http/router.go`:

    ``` text
    POST /api/v1/decks/{deckID}/imports
    ```

    -   Multipart or raw `application/zip` body, capped by a configured
        `MAX_IMPORT_ZIP_SIZE_BYTES` (mirror the existing
        `MAX_AUDIO_SIZE_BYTES` pattern in `config.Config`).
    -   Requires the same `requireAuth` + deck-ownership check already
        used by `flashcardHandlers` and `deckHandlers`.
    -   Rejects non-ZIP content types with the existing
        `UNSUPPORTED_MEDIA_TYPE` `apperror` code; rejects oversized
        uploads the same way audio uploads reject oversized files
        (abort the stream, never partially persist).
2.  On the server, stream-extract the ZIP into a per-job directory
    (e.g. `data/incoming/<jobId>/`), validating each entry:
    -   Reject path traversal (`..`, absolute paths, symlinks).
    -   Keep only supported image extensions (png/jpg/jpeg/webp); skip
        and report anything else instead of failing the whole job.
    -   Enforce a max file count and a max total uncompressed size
        (zip-bomb protection) before writing anything to disk.
3.  Create an `ImportJob` (section 16) scoped to the authenticated user
    and the target `deckID` from the URL, with one `ImportItem` per
    extracted image, status `pending`.
4.  Respond `202 Accepted` with the `jobId` — this is inherently
    asynchronous, since turning an image into a card draft is not a
    cheap synchronous operation (see "Content interpretation" below).
5.  Add `GET /api/v1/imports/{jobId}` (owner-scoped) returning the job
    counters and per-item status, so the frontend can poll it — this
    reuses the report model from section 21.

### Frontend

1.  New page or modal, reachable from `DeckPage`, e.g. an "Importar
    screenshots (.zip)" button next to "Criar flashcard".
2.  A file input accepting only `.zip`, uploaded via `api.postFile`
    (already used for audio) against the new endpoint.
3.  After upload, redirect to (or poll) an import-status view showing
    the same counters as the CLI report: discovered / created /
    duplicates / needs review / failed, updating until the job
    finishes.
4.  Never let the browser block on the whole import synchronously —
    show progress, not a frozen "Salvar" button.
5.  Follow the existing Portuguese-language, no-comment,
    no-over-engineering conventions already used across the frontend
    pages.

### Content interpretation reality check

The rest of this spec (sections 11-13) describes OCR/translation/audio
as pluggable provider interfaces because, at the time of writing, this
project has **no configured OCR or translation provider** — the 106
cards imported so far were produced by a human-in-the-loop step: an AI
assistant with vision read each screenshot directly and called the
existing flashcard API, exactly the way `contentinterpreter` in section
7 is meant to be filled in.

A ZIP upload endpoint does not remove that dependency by itself. Pick
one explicitly before treating this as "fully automatic":

-   **Human/agent-in-the-loop (recommended first step).** The upload
    endpoint only creates the `ImportJob`/`ImportItem` records and
    extracts images to disk, all left `pending`. Processing them (i.e.
    turning each image into a card) still happens the way it does
    today — an assistant with vision is asked to run the job. This
    keeps the exact same trust model already in use (no external image
    upload, no new API key, no unattended translation) and just moves
    "drop files in a folder" into the product UI.
-   **Configured vision/OCR+translation provider.** If the user
    explicitly wants zero-touch imports, wire a real
    `contentinterpreter`/`translator` adapter (e.g. an LLM vision API
    call) behind the existing ports, gated by an explicit config flag
    and an API key supplied via environment variable, never hard-coded
    or committed. This is strictly optional and must not be assumed.

Do not silently pick the second option. Ask before adding any external
API dependency or recurring cost.

------------------------------------------------------------------------

## 31. Planned enhancement --- per-deck subfolders under `images/`

Status: **not implemented yet**, same caveat as section 30.

### Goal

Let the user organize screenshots by creating a subfolder per deck (or
per language) inside the local `images/` input folder, so the importer
can route each image to the right deck without the user tagging every
file individually.

``` text
images/
├── Français/
│   ├── WhatsApp Image ....jpeg
│   └── ...
├── English/
│   └── ...
└── some_loose_screenshot.jpeg
```

### Rules

1.  The folder name is a **deck name hint**, not a free-text label to
    trust blindly: normalize it (trim, collapse whitespace, case-fold
    only for comparison) and match it against the user's existing
    decks by name.
2.  **Existing deck match** → import into that deck, same as today's
    manual flow.
3.  **No matching deck** → do not auto-create a deck silently (this
    contradicts the "Excluded by default" rule in section 3 unless
    explicitly overridden). Instead:
    -   Default: send the folder's images to `needs_review` with a
        clear reason (`"unknown deck folder: <name>"`), and surface the
        proposed deck name in the report so the user can create it (or
        confirm auto-creation) in one step.
    -   Opt-in: a config flag
        `allowAutoCreateDeckFromFolderName: true` may let the importer
        create the deck automatically — off by default, and if enabled
        it must still log/report every deck it created.
4.  **Loose files directly under `images/`** (no subfolder) keep the
    current behavior: they go to the single `deckId` configured for
    the run (section 5), or to `needs_review` if no default deck is
    configured for that run.
5.  A subfolder is a routing hint only, not identity: the same
    SHA-256 + card-fingerprint duplicate detection (sections 10 and 14)
    still applies regardless of which folder the file was found in, so
    moving a file between folders must never re-import it as new.
6.  This applies equally to the future ZIP-upload flow (section 30):
    the ZIP's internal folder structure is interpreted the same way as
    the local `images/` folder structure.

### Explicitly out of scope for this enhancement

-   Per-*card* folders (one folder per individual flashcard) were
    considered and rejected: with dozens of screenshots per deck, this
    would mean creating a folder per prospective card before the card
    exists, which inverts the actual workflow (the card is the
    *output* of processing an image, not something the user names in
    advance). Per-deck/per-language folders capture the same benefit
    (routing) without that inversion.

------------------------------------------------------------------------

## 32. Rate limiting and 429 protection

Status: **explicitly requested by the user** (2026-09-16), mandatory
for every future execution of this agent — not optional, not just for
the ZIP/folder enhancements above.

### Why this exists

The only translation/TTS path actually available to this project today
is the unofficial Google Translate endpoint used by `gTTS` (see the
bulk audio-generation run of 2026-09-16, which used it directly). That
endpoint publishes no official quota; the only observable signal that
a limit was hit is an HTTP `429`. The same caution applies to any
future OCR, translation, or content-interpretation provider — do not
assume any provider (official or not) tolerates unlimited call rates.

### Hard limit

-   **No more than 60 calls per minute** to any single external
    translation/OCR/content-interpretation provider, and separately no
    more than 60 calls per minute to any single external audio (TTS)
    provider. These are independent budgets (they hit different
    endpoints); do not share one counter between translation calls and
    audio calls.
-   60/minute is a ceiling, not a target — an implementation may use a
    lower steady rate (e.g. one call per second, which already equals
    the 60/minute cap with even spacing) if that further reduces 429s
    in practice.

### Required mechanism

Implement a small, reusable rate limiter (e.g. a token-bucket or
"leaky bucket" with 60 tokens replenished over 60 seconds, or simply a
minimum-interval gate of ~1 second between consecutive calls to the
same provider) and route **every** call to a rate-limited provider
through it — no direct/bypassing calls from OCR, translation, or audio
code paths.

``` go
type RateLimiter interface {
    // Wait blocks until a call is allowed, or ctx is done.
    Wait(ctx context.Context) error
}
```

-   One `RateLimiter` instance per provider, shared across the whole
    import job (and, if the CLI processes multiple jobs concurrently,
    shared across the whole process) — the limit is per external
    endpoint, not per image or per job.
-   `Wait` must respect `ctx` cancellation/timeout so a stuck limiter
    cannot hang the whole batch forever.
-   Add a small jitter (e.g. ±100ms) to the spacing to avoid every
    request landing on an exact clock tick, which is a common cause of
    synchronized 429s in batch scripts.

### 429 handling (on top of the limiter, not instead of it)

The rate limiter reduces how often a 429 happens; it does not
guarantee it never happens (the provider's real limit is unknown and
unofficial). When a call still returns 429:

1.  If the response includes a `Retry-After` header, wait that long
    before the next attempt on that provider.
2.  Otherwise, use exponential backoff starting at 2 seconds, doubling
    each attempt, capped at 60 seconds.
3.  Bound retries (e.g. 5 attempts) per item. After exhausting retries,
    mark the item `failed` with error code `RATE_LIMITED` (section 22)
    — retryable, so a later `flashcard-importer retry` run can pick it
    up — never mark it `needs_review` for this reason (that queue is
    for ambiguous content, not transient infrastructure errors).
4.  A 429 on one item must not abort the whole batch. Back off, then
    keep processing the remaining queued items through the same rate
    limiter.
5.  Log every 429 and every backoff wait at the observability points
    already defined in section 23, including which provider and how
    long the wait was.

### What not to do

-   Do not retry a 429 immediately in a tight loop.
-   Do not remove the rate limiter "because it slows down large
    batches" — a batch of a few hundred images at 60/minute finishes in
    a few minutes, which is an acceptable trade-off against getting
    throttled or blocked outright.
-   Do not raise the 60/minute ceiling without the user explicitly
    asking for a different number.

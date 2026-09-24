// Package flashcard defines the Flashcard entity: a question/answer pair
// belonging to one deck, with optional audio and an optional extended
// example, plus its spaced-repetition scheduling state.
package flashcard

import (
	"strings"
	"time"

	"flashcard-backend/internal/apperror"
	"flashcard-backend/internal/domain/study"
)

const (
	MaxQuestionLength = 5000
	MaxAnswerLength   = 5000
	MaxExampleLength  = 10000
	MaxHintLength     = 1000
)

// ExtendedExample is an optional richer example shown alongside the
// answer, e.g. a sentence using the answer in context.
type ExtendedExample struct {
	Text        string
	Translation string
}

// Audio references a binary file stored in GridFS; the flashcard document
// itself never stores audio bytes (spec section 7).
type Audio struct {
	GridFSFileID string
	ContentType  string
	// DurationMs is left at 0: computing it server-side would require an
	// audio-format parsing dependency for little MVP benefit, since the
	// browser's own <audio> element already reports duration once played.
	DurationMs int64
}

// AllowedAudioContentTypes are the audio formats accepted for upload (spec
// section 7). Anything else is rejected with UNSUPPORTED_MEDIA_TYPE.
var AllowedAudioContentTypes = map[string]bool{
	"audio/mpeg": true,
	"audio/wav":  true,
	"audio/ogg":  true,
}

// ValidateAudioContentType rejects anything outside
// AllowedAudioContentTypes. The content type is taken from the request's
// Content-Type header, never from a client-supplied filename.
func ValidateAudioContentType(contentType string) error {
	if !AllowedAudioContentTypes[contentType] {
		return apperror.UnsupportedMediaType("audio.unsupportedType", "audio content type must be one of: audio/mpeg, audio/wav, audio/ogg")
	}
	return nil
}

// Flashcard is a question/answer study unit belonging to one deck.
type Flashcard struct {
	ID       string
	UserID   string
	DeckID   string
	Question string
	Answer   string
	// Hint is an optional nudge shown on the front of the card during
	// study, before the answer is revealed. Empty means no hint.
	Hint            string
	ExtendedExample *ExtendedExample
	Audio           *Audio
	Scheduling      study.SchedulingState
	ArchivedAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// IsArchived reports whether the flashcard has been archived.
func (f Flashcard) IsArchived() bool { return f.ArchivedAt != nil }

// ValidateQuestion trims and validates a flashcard question.
func ValidateQuestion(raw string) (string, error) {
	question := strings.TrimSpace(raw)
	if question == "" {
		return "", apperror.Validation("flashcard.question.required", "question is required")
	}
	if len(question) > MaxQuestionLength {
		return "", apperror.Validation("flashcard.question.tooLong", "question must be at most 5,000 characters")
	}
	return question, nil
}

// ValidateAnswer trims and validates a flashcard answer.
func ValidateAnswer(raw string) (string, error) {
	answer := strings.TrimSpace(raw)
	if answer == "" {
		return "", apperror.Validation("flashcard.answer.required", "answer is required")
	}
	if len(answer) > MaxAnswerLength {
		return "", apperror.Validation("flashcard.answer.tooLong", "answer must be at most 5,000 characters")
	}
	return answer, nil
}

// ValidateHint trims and validates the optional hint.
func ValidateHint(raw string) (string, error) {
	hint := strings.TrimSpace(raw)
	if len(hint) > MaxHintLength {
		return "", apperror.Validation("flashcard.hint.tooLong", "hint must be at most 1,000 characters")
	}
	return hint, nil
}

// ValidateExtendedExampleText trims and validates the optional extended
// example text.
func ValidateExtendedExampleText(raw string) (string, error) {
	text := strings.TrimSpace(raw)
	if len(text) > MaxExampleLength {
		return "", apperror.Validation("flashcard.extendedExample.tooLong", "extended example must be at most 10,000 characters")
	}
	return text, nil
}

// New creates a Flashcard for the given deck at the given instant, in the
// NEW scheduling state. The caller is responsible for persistence, for
// assigning an ID, and for verifying that the deck belongs to userID.
func New(userID, deckID, question, answer string, example *ExtendedExample, now time.Time) Flashcard {
	return Flashcard{
		UserID:          userID,
		DeckID:          deckID,
		Question:        question,
		Answer:          answer,
		ExtendedExample: example,
		Scheduling:      study.NewSchedulingState(now),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// Package review defines the ReviewEvent entity: an immutable record of a
// single card review, kept separate from the flashcard's current
// scheduling state for debugging, statistics, and auditability (spec
// section 6).
package review

import (
	"time"

	"flashcard-backend/internal/domain/study"
)

// Event is one recorded review of a flashcard.
type Event struct {
	ID             string
	UserID         string
	FlashcardID    string
	DeckID         string
	Rating         study.Rating
	PreviousState  study.CardState
	NextState      study.CardState
	ReviewedAt     time.Time
	ResponseTimeMs int64
	// HintUsed records that the learner looked at the card's hint before
	// answering. It is only informational (statistics); it does not
	// influence scheduling.
	HintUsed bool
}

// New creates a review Event from a scheduling decision applied at
// reviewedAt. The caller is responsible for persistence and for assigning
// an ID.
func New(userID, flashcardID, deckID string, rating study.Rating, previousState, nextState study.CardState, reviewedAt time.Time, responseTimeMs int64) Event {
	return Event{
		UserID:         userID,
		FlashcardID:    flashcardID,
		DeckID:         deckID,
		Rating:         rating,
		PreviousState:  previousState,
		NextState:      nextState,
		ReviewedAt:     reviewedAt,
		ResponseTimeMs: responseTimeMs,
	}
}

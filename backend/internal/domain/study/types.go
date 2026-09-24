// Package study implements the spaced-repetition scheduler: a simple,
// deterministic Leitner/SM-2-inspired policy (spec section 11), kept
// behind the Scheduler interface so it can later be replaced (e.g. by
// FSRS) without changing the study flow.
package study

import (
	"time"

	"flashcard-backend/internal/apperror"
)

// CardState is a position in the card lifecycle: NEW -> LEARNING -> REVIEW
// -> MATURE, with a failed review able to move a card back to LEARNING.
type CardState string

const (
	StateNew      CardState = "new"
	StateLearning CardState = "learning"
	StateReview   CardState = "review"
	StateMature   CardState = "mature"
)

// Rating is the stable enum value a user picks after revealing an answer.
type Rating string

const (
	RatingAgain Rating = "again"
	RatingHard  Rating = "hard"
	RatingGood  Rating = "good"
	RatingEasy  Rating = "easy"
)

// ParseRating validates a client-supplied rating string against the fixed
// set of allowed values, rejecting free-form strings (spec section 10).
func ParseRating(raw string) (Rating, error) {
	switch Rating(raw) {
	case RatingAgain, RatingHard, RatingGood, RatingEasy:
		return Rating(raw), nil
	default:
		return "", apperror.Validation("study.rating.invalid", "rating must be one of: again, hard, good, easy")
	}
}

// SchedulingState is the current spaced-repetition state of a flashcard.
type SchedulingState struct {
	State          CardState
	DueAt          time.Time
	LastReviewedAt *time.Time
	Repetitions    int
	Lapses         int
	IntervalDays   int
	// EaseFactor is carried for forward compatibility with a future
	// SM-2/FSRS-style scheduler; the current deterministic policy does not
	// use it to compute intervals.
	EaseFactor float64
}

// NewSchedulingState is the initial state for a freshly created flashcard.
func NewSchedulingState(now time.Time) SchedulingState {
	return SchedulingState{
		State:      StateNew,
		DueAt:      now,
		EaseFactor: 2.5,
	}
}

// SchedulingDecision is the outcome of applying a rating to a scheduling
// state: the state to persist next.
type SchedulingDecision struct {
	State        CardState
	DueAt        time.Time
	IntervalDays int
	Repetitions  int
	Lapses       int
	EaseFactor   float64
}

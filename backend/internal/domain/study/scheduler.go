package study

import "time"

const (
	// matureIntervalDays is the interval at or above which a REVIEW card
	// is promoted to MATURE. This threshold is a deliberate MVP choice
	// (not specified numerically by the product spec) inspired by common
	// spaced-repetition tools; it is not a claim of optimal learning
	// science and may be revisited. It is the same for every study
	// configuration, so "Aprendidos" always means the same thing.
	matureIntervalDays = 21

	// DefaultMaxIntervalDays caps how far into the future a review can be
	// scheduled when no other maximum is given.
	DefaultMaxIntervalDays = 365
)

// Scheduler computes the next scheduling state for a card given a rating.
type Scheduler interface {
	Schedule(state SchedulingState, rating Rating, now time.Time) SchedulingDecision
}

// RulesScheduler is the deterministic scheduler described in spec section
// 11, driven by a set of Rules so each learner can tune how the ratings
// behave (see the study configurations).
type RulesScheduler struct {
	Rules Rules
	// MaxIntervalDays caps how far into the future a review can be
	// scheduled. Defaults to DefaultMaxIntervalDays when zero.
	MaxIntervalDays int
}

// DefaultScheduler is the scheduler with the built-in default rules.
type DefaultScheduler = RulesScheduler

// NewRulesScheduler builds a scheduler for rules with the given maximum
// interval in days (DefaultMaxIntervalDays is used when maxIntervalDays <= 0).
func NewRulesScheduler(rules Rules, maxIntervalDays int) RulesScheduler {
	if maxIntervalDays <= 0 {
		maxIntervalDays = DefaultMaxIntervalDays
	}
	return RulesScheduler{Rules: rules, MaxIntervalDays: maxIntervalDays}
}

// NewDefaultScheduler builds a scheduler with the built-in default rules and
// the given maximum interval in days (365 is used when maxIntervalDays <= 0).
func NewDefaultScheduler(maxIntervalDays int) DefaultScheduler {
	return NewRulesScheduler(DefaultRules(), maxIntervalDays)
}

func (s RulesScheduler) Schedule(state SchedulingState, rating Rating, now time.Time) SchedulingDecision {
	if rating == RatingAgain {
		lapses := state.Lapses
		if state.State == StateReview || state.State == StateMature {
			lapses++
		}
		return SchedulingDecision{
			State:        StateLearning,
			DueAt:        now.Add(time.Duration(s.Rules.AgainDelayMinutes) * time.Minute),
			IntervalDays: 0,
			Repetitions:  0,
			Lapses:       lapses,
			EaseFactor:   state.EaseFactor,
		}
	}

	switch state.State {
	case StateNew, StateLearning:
		return s.scheduleFromLearning(state, rating, now)
	default:
		return s.scheduleFromReview(state, rating, now)
	}
}

func (s RulesScheduler) scheduleFromLearning(state SchedulingState, rating Rating, now time.Time) SchedulingDecision {
	intervalDays := s.Rules.forRating(rating).FirstIntervalDays
	if intervalDays > s.MaxIntervalDays {
		intervalDays = s.MaxIntervalDays
	}
	return SchedulingDecision{
		State:        StateReview,
		DueAt:        now.AddDate(0, 0, intervalDays),
		IntervalDays: intervalDays,
		Repetitions:  state.Repetitions + 1,
		Lapses:       state.Lapses,
		EaseFactor:   state.EaseFactor,
	}
}

func (s RulesScheduler) scheduleFromReview(state SchedulingState, rating Rating, now time.Time) SchedulingDecision {
	previousInterval := state.IntervalDays
	if previousInterval < 1 {
		previousInterval = 1
	}

	intervalDays := int(float64(previousInterval) * s.Rules.forRating(rating).Multiplier)
	// Any answer other than "again" must push the next review further away.
	// Without this, "hard" (x1.2, rounded down) left short intervals stuck:
	// 1 day stayed 1 day, and 2, 3 and 4 days never grew either, so a card
	// that was always answered "hard" came back at the same interval forever.
	if intervalDays <= previousInterval {
		intervalDays = previousInterval + 1
	}
	if intervalDays > s.MaxIntervalDays {
		intervalDays = s.MaxIntervalDays
	}

	nextState := StateReview
	if intervalDays >= matureIntervalDays {
		nextState = StateMature
	}

	return SchedulingDecision{
		State:        nextState,
		DueAt:        now.AddDate(0, 0, intervalDays),
		IntervalDays: intervalDays,
		Repetitions:  state.Repetitions + 1,
		Lapses:       state.Lapses,
		EaseFactor:   state.EaseFactor,
	}
}

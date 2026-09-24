package study

import (
	"fmt"

	"flashcard-backend/internal/apperror"
)

// Limits for the values a user may choose in a study configuration.
const (
	MinAgainDelayMinutes = 1
	MaxAgainDelayMinutes = 24 * 60 // one day
	MinFirstIntervalDays = 1
	MaxFirstIntervalDays = 365
	MinMultiplier        = 1.0
	MaxMultiplier        = 5.0
)

// RatingRule says how one of the three "success" ratings (hard, good, easy)
// reschedules a card.
type RatingRule struct {
	// FirstIntervalDays is the wait, in days, for a card that is new or that
	// is being learned again after a failure.
	FirstIntervalDays int
	// Multiplier scales the previous interval when an already learned card is
	// reviewed again (2.0 doubles it).
	Multiplier float64
}

// Rules is how a learner wants the system to react to each rating. The
// default values are the behavior the app always had.
type Rules struct {
	// AgainDelayMinutes is how soon a card comes back after "Muito difícil".
	AgainDelayMinutes int
	Hard              RatingRule // "Difícil"
	Good              RatingRule // "Fácil"
	Easy              RatingRule // "Muito fácil"
}

// DefaultRules is the built-in behavior: a failed card returns in 10 minutes,
// and a new card answered hard/good/easy returns in 1/3/7 days, with later
// reviews multiplying the interval by 1.2/2/3.
func DefaultRules() Rules {
	return Rules{
		AgainDelayMinutes: 10,
		Hard:              RatingRule{FirstIntervalDays: 1, Multiplier: 1.2},
		Good:              RatingRule{FirstIntervalDays: 3, Multiplier: 2.0},
		Easy:              RatingRule{FirstIntervalDays: 7, Multiplier: 3.0},
	}
}

// Validate checks every value is inside its allowed range and that a better
// answer never schedules the card sooner than a worse one (hard <= good <=
// easy), which would make the buttons contradict their own names.
func (r Rules) Validate() error {
	if r.AgainDelayMinutes < MinAgainDelayMinutes || r.AgainDelayMinutes > MaxAgainDelayMinutes {
		return apperror.Validation("studyProfile.againDelayMinutes.range", fmt.Sprintf("againDelayMinutes must be between %d and %d", MinAgainDelayMinutes, MaxAgainDelayMinutes))
	}
	// Each rating gets its own key (studyProfile.rules.<rating>.firstIntervalDays.range,
	// ...multiplier.range) rather than a single parameterized key, since a
	// translated message needs the rating's own name baked in.
	for name, rule := range map[string]RatingRule{"hard": r.Hard, "good": r.Good, "easy": r.Easy} {
		if rule.FirstIntervalDays < MinFirstIntervalDays || rule.FirstIntervalDays > MaxFirstIntervalDays {
			return apperror.Validation(
				fmt.Sprintf("studyProfile.rules.%s.firstIntervalDays.range", name),
				fmt.Sprintf("%s.firstIntervalDays must be between %d and %d", name, MinFirstIntervalDays, MaxFirstIntervalDays),
			)
		}
		if rule.Multiplier < MinMultiplier || rule.Multiplier > MaxMultiplier {
			return apperror.Validation(
				fmt.Sprintf("studyProfile.rules.%s.multiplier.range", name),
				fmt.Sprintf("%s.multiplier must be between %.0f and %.0f", name, MinMultiplier, MaxMultiplier),
			)
		}
	}
	if r.Hard.FirstIntervalDays > r.Good.FirstIntervalDays || r.Good.FirstIntervalDays > r.Easy.FirstIntervalDays {
		return apperror.Validation("studyProfile.rules.firstIntervalDays.order", "firstIntervalDays must not decrease from hard to good to easy")
	}
	if r.Hard.Multiplier > r.Good.Multiplier || r.Good.Multiplier > r.Easy.Multiplier {
		return apperror.Validation("studyProfile.rules.multiplier.order", "multiplier must not decrease from hard to good to easy")
	}
	return nil
}

func (r Rules) forRating(rating Rating) RatingRule {
	switch rating {
	case RatingHard:
		return r.Hard
	case RatingEasy:
		return r.Easy
	default:
		return r.Good
	}
}

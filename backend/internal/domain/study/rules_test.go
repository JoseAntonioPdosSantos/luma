package study

import (
	"strings"
	"testing"
	"time"
)

func TestDefaultRules_AreValidAndMatchTheOriginalBehavior(t *testing.T) {
	r := DefaultRules()
	if err := r.Validate(); err != nil {
		t.Fatalf("DefaultRules().Validate() error: %v", err)
	}
	if r.AgainDelayMinutes != 10 ||
		r.Hard != (RatingRule{1, 1.2}) || r.Good != (RatingRule{3, 2.0}) || r.Easy != (RatingRule{7, 3.0}) {
		t.Errorf("DefaultRules() = %+v, want 10 min and 1d/1.2, 3d/2.0, 7d/3.0", r)
	}
}

func TestRulesValidate_RejectsValuesOutOfRange(t *testing.T) {
	mutate := func(f func(*Rules)) Rules {
		r := DefaultRules()
		f(&r)
		return r
	}
	tests := map[string]Rules{
		"again delay below 1":        mutate(func(r *Rules) { r.AgainDelayMinutes = 0 }),
		"again delay over a day":     mutate(func(r *Rules) { r.AgainDelayMinutes = MaxAgainDelayMinutes + 1 }),
		"first interval below 1":     mutate(func(r *Rules) { r.Hard.FirstIntervalDays = 0 }),
		"first interval over a year": mutate(func(r *Rules) { r.Easy.FirstIntervalDays = MaxFirstIntervalDays + 1 }),
		"multiplier below 1":         mutate(func(r *Rules) { r.Hard.Multiplier = 0.9 }),
		"multiplier over 5":          mutate(func(r *Rules) { r.Easy.Multiplier = 5.5 }),
	}
	for name, rules := range tests {
		if err := rules.Validate(); err == nil {
			t.Errorf("%s: Validate() should fail", name)
		}
	}
}

func TestRulesValidate_ABetterAnswerNeverComesBackSooner(t *testing.T) {
	sooner := DefaultRules()
	sooner.Good.FirstIntervalDays = 8 // easy is 7
	if err := sooner.Validate(); err == nil || !strings.Contains(err.Error(), "firstIntervalDays") {
		t.Errorf("good after easy (first interval): error = %v, want a firstIntervalDays error", err)
	}

	slower := DefaultRules()
	slower.Hard.Multiplier = 2.5 // good is 2.0
	if err := slower.Validate(); err == nil || !strings.Contains(err.Error(), "multiplier") {
		t.Errorf("hard above good (multiplier): error = %v, want a multiplier error", err)
	}

	equal := DefaultRules()
	equal.Hard = equal.Good
	if err := equal.Validate(); err != nil {
		t.Errorf("equal values are allowed, got %v", err)
	}
}

func TestRulesScheduler_FollowsCustomRules(t *testing.T) {
	rules := Rules{
		AgainDelayMinutes: 2,
		Hard:              RatingRule{FirstIntervalDays: 2, Multiplier: 1.5},
		Good:              RatingRule{FirstIntervalDays: 4, Multiplier: 2.5},
		Easy:              RatingRule{FirstIntervalDays: 10, Multiplier: 4},
	}
	s := NewRulesScheduler(rules, 365)
	fresh := NewSchedulingState(fixedNow)

	if d := s.Schedule(fresh, RatingAgain, fixedNow); d.DueAt.Sub(fixedNow) != 2*time.Minute {
		t.Errorf("again: due in %v, want 2m0s", d.DueAt.Sub(fixedNow))
	}
	for rating, want := range map[Rating]int{RatingHard: 2, RatingGood: 4, RatingEasy: 10} {
		if d := s.Schedule(fresh, rating, fixedNow); d.IntervalDays != want || d.State != StateReview {
			t.Errorf("new card, %s: %d days (%s), want %d days", rating, d.IntervalDays, d.State, want)
		}
	}

	reviewed := SchedulingState{State: StateReview, IntervalDays: 10, EaseFactor: 2.5}
	for rating, want := range map[Rating]int{RatingHard: 15, RatingGood: 25, RatingEasy: 40} {
		if d := s.Schedule(reviewed, rating, fixedNow); d.IntervalDays != want {
			t.Errorf("10-day card, %s: %d days, want %d", rating, d.IntervalDays, want)
		}
	}
}

func TestRulesScheduler_FirstIntervalIsCappedAtTheMaximum(t *testing.T) {
	rules := DefaultRules()
	rules.Easy.FirstIntervalDays = 200
	s := NewRulesScheduler(rules, 30)

	if d := s.Schedule(NewSchedulingState(fixedNow), RatingEasy, fixedNow); d.IntervalDays != 30 {
		t.Errorf("first interval = %d days, want it capped at the 30-day maximum", d.IntervalDays)
	}
}

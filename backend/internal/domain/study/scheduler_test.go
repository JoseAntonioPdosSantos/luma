package study

import (
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func TestScheduler_NewCard(t *testing.T) {
	scheduler := NewDefaultScheduler(365)
	state := NewSchedulingState(fixedNow)

	tests := []struct {
		rating       Rating
		wantState    CardState
		wantInterval int
	}{
		{RatingAgain, StateLearning, 0},
		{RatingHard, StateReview, 1},
		{RatingGood, StateReview, 3},
		{RatingEasy, StateReview, 7},
	}

	for _, tt := range tests {
		decision := scheduler.Schedule(state, tt.rating, fixedNow)
		if decision.State != tt.wantState {
			t.Errorf("rating=%s: state = %s, want %s", tt.rating, decision.State, tt.wantState)
		}
		if decision.IntervalDays != tt.wantInterval {
			t.Errorf("rating=%s: intervalDays = %d, want %d", tt.rating, decision.IntervalDays, tt.wantInterval)
		}
	}
}

func TestScheduler_AgainAlwaysDueInTenMinutes(t *testing.T) {
	scheduler := NewDefaultScheduler(365)
	state := SchedulingState{State: StateReview, IntervalDays: 100, EaseFactor: 2.5}

	decision := scheduler.Schedule(state, RatingAgain, fixedNow)

	wantDue := fixedNow.Add(10 * time.Minute)
	if !decision.DueAt.Equal(wantDue) {
		t.Errorf("DueAt = %v, want %v", decision.DueAt, wantDue)
	}
	if decision.State != StateLearning {
		t.Errorf("State = %s, want %s", decision.State, StateLearning)
	}
	if decision.IntervalDays != 0 {
		t.Errorf("IntervalDays = %d, want 0", decision.IntervalDays)
	}
	if decision.Repetitions != 0 {
		t.Errorf("Repetitions = %d, want 0", decision.Repetitions)
	}
}

func TestScheduler_AgainOnReviewCountsAsLapse(t *testing.T) {
	scheduler := NewDefaultScheduler(365)
	state := SchedulingState{State: StateReview, IntervalDays: 10, Lapses: 2, EaseFactor: 2.5}

	decision := scheduler.Schedule(state, RatingAgain, fixedNow)

	if decision.Lapses != 3 {
		t.Errorf("Lapses = %d, want 3", decision.Lapses)
	}
}

func TestScheduler_AgainOnNewCardIsNotALapse(t *testing.T) {
	scheduler := NewDefaultScheduler(365)
	state := NewSchedulingState(fixedNow)

	decision := scheduler.Schedule(state, RatingAgain, fixedNow)

	if decision.Lapses != 0 {
		t.Errorf("Lapses = %d, want 0", decision.Lapses)
	}
}

func TestScheduler_ReviewCardIntervalGrowth(t *testing.T) {
	scheduler := NewDefaultScheduler(365)
	state := SchedulingState{State: StateReview, IntervalDays: 10, EaseFactor: 2.5}

	tests := []struct {
		rating       Rating
		wantInterval int
	}{
		{RatingHard, 12}, // 10 * 1.2
		{RatingGood, 20}, // 10 * 2.0
		{RatingEasy, 30}, // 10 * 3.0
	}

	for _, tt := range tests {
		decision := scheduler.Schedule(state, tt.rating, fixedNow)
		if decision.IntervalDays != tt.wantInterval {
			t.Errorf("rating=%s: intervalDays = %d, want %d", tt.rating, decision.IntervalDays, tt.wantInterval)
		}
	}
}

func TestScheduler_ReviewCardMinimumIntervalIsOneDay(t *testing.T) {
	scheduler := NewDefaultScheduler(365)
	// A previous interval of 0 (defensive case) must still floor at 1 day.
	state := SchedulingState{State: StateReview, IntervalDays: 0, EaseFactor: 2.5}

	decision := scheduler.Schedule(state, RatingHard, fixedNow)

	if decision.IntervalDays < 1 {
		t.Errorf("IntervalDays = %d, want >= 1", decision.IntervalDays)
	}
}

func TestScheduler_MaxIntervalIsRespected(t *testing.T) {
	scheduler := NewDefaultScheduler(365)
	state := SchedulingState{State: StateReview, IntervalDays: 300, EaseFactor: 2.5}

	decision := scheduler.Schedule(state, RatingEasy, fixedNow)

	if decision.IntervalDays != 365 {
		t.Errorf("IntervalDays = %d, want capped at 365", decision.IntervalDays)
	}
}

func TestScheduler_CardBecomesMatureAboveThreshold(t *testing.T) {
	scheduler := NewDefaultScheduler(365)
	state := SchedulingState{State: StateReview, IntervalDays: 15, EaseFactor: 2.5}

	decision := scheduler.Schedule(state, RatingGood, fixedNow) // 15 * 2.0 = 30 >= 21

	if decision.State != StateMature {
		t.Errorf("State = %s, want %s", decision.State, StateMature)
	}
}

func TestScheduler_MatureCardStaysMatureOnSuccess(t *testing.T) {
	scheduler := NewDefaultScheduler(365)
	state := SchedulingState{State: StateMature, IntervalDays: 30, EaseFactor: 2.5}

	decision := scheduler.Schedule(state, RatingGood, fixedNow)

	if decision.State != StateMature {
		t.Errorf("State = %s, want %s", decision.State, StateMature)
	}
}

func TestScheduler_DefaultMaxIntervalWhenNotConfigured(t *testing.T) {
	scheduler := NewDefaultScheduler(0)
	if scheduler.MaxIntervalDays != 365 {
		t.Errorf("MaxIntervalDays = %d, want 365", scheduler.MaxIntervalDays)
	}
}

func TestParseRating(t *testing.T) {
	valid := []string{"again", "hard", "good", "easy"}
	for _, v := range valid {
		if _, err := ParseRating(v); err != nil {
			t.Errorf("ParseRating(%q) returned error: %v", v, err)
		}
	}

	if _, err := ParseRating("terrible"); err == nil {
		t.Error("ParseRating(\"terrible\") should return an error")
	}
}

func TestSchedule_HardAlwaysGrowsTheIntervalByAtLeastADay(t *testing.T) {
	scheduler := NewDefaultScheduler(365)
	// The old rule (previous x 1.2, rounded down) left 1-4 days unchanged.
	for previous, want := range map[int]int{1: 2, 2: 3, 3: 4, 4: 5, 5: 6, 10: 12, 21: 25, 100: 120} {
		state := SchedulingState{State: StateReview, IntervalDays: previous, EaseFactor: 2.5}
		got := scheduler.Schedule(state, RatingHard, fixedNow)
		if got.IntervalDays != want {
			t.Errorf("hard from %d days = %d days, want %d", previous, got.IntervalDays, want)
		}
	}
}

func TestSchedule_RepeatedHardKeepsGrowingUntilTheCardIsLearned(t *testing.T) {
	scheduler := NewDefaultScheduler(365)
	state := NewSchedulingState(fixedNow)

	var intervals []int
	for i := 0; i < 8; i++ {
		d := scheduler.Schedule(state, RatingHard, fixedNow)
		intervals = append(intervals, d.IntervalDays)
		state = SchedulingState{State: d.State, DueAt: d.DueAt, IntervalDays: d.IntervalDays, Repetitions: d.Repetitions, EaseFactor: d.EaseFactor}
	}

	// A new card answered "hard" is due in 1 day; every further "hard" adds at least a day.
	for i := 1; i < len(intervals); i++ {
		if intervals[i] <= intervals[i-1] {
			t.Fatalf("intervals after repeated hard = %v; each must be larger than the previous", intervals)
		}
	}
}

func TestSchedule_HardStillRespectsTheMaximumInterval(t *testing.T) {
	scheduler := NewDefaultScheduler(30)
	state := SchedulingState{State: StateMature, IntervalDays: 30, EaseFactor: 2.5}

	got := scheduler.Schedule(state, RatingHard, fixedNow)
	if got.IntervalDays != 30 {
		t.Errorf("hard at the maximum interval = %d days, want it capped at 30", got.IntervalDays)
	}
}

func TestSchedule_HardCanPromoteACardToMature(t *testing.T) {
	scheduler := NewDefaultScheduler(365)
	state := SchedulingState{State: StateReview, IntervalDays: 20, EaseFactor: 2.5}

	got := scheduler.Schedule(state, RatingHard, fixedNow)
	if got.IntervalDays != 24 || got.State != StateMature {
		t.Errorf("hard from 20 days = %d days, %s; want 24 days, mature", got.IntervalDays, got.State)
	}
}

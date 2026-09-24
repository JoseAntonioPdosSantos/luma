package mongodb

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"flashcard-backend/internal/domain/review"
	"flashcard-backend/internal/domain/study"
)

func TestReviewEventRepository_Create(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewReviewEventRepository(db)

	userID := primitive.NewObjectID().Hex()
	flashcardID := primitive.NewObjectID().Hex()
	deckID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	created, err := repo.Create(ctx, review.New(userID, flashcardID, deckID, study.RatingGood, study.StateNew, study.StateReview, now, 1500))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create() should assign an ID")
	}
}

func TestReviewEventRepository_DailyCounts(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewReviewEventRepository(db)

	userID := primitive.NewObjectID().Hex()
	deckID := primitive.NewObjectID().Hex()
	flashcardID := primitive.NewObjectID().Hex()
	otherDeckID := primitive.NewObjectID().Hex()

	day1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)

	events := []review.Event{
		review.New(userID, flashcardID, deckID, study.RatingGood, study.StateNew, study.StateReview, day1, 0),
		review.New(userID, flashcardID, deckID, study.RatingAgain, study.StateReview, study.StateLearning, day1, 0),
		review.New(userID, flashcardID, deckID, study.RatingEasy, study.StateReview, study.StateMature, day2, 0),
		// Different deck: must not be counted.
		review.New(userID, flashcardID, otherDeckID, study.RatingGood, study.StateNew, study.StateReview, day2, 0),
	}
	for _, e := range events {
		if _, err := repo.Create(ctx, e); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	counts, err := repo.DailyCounts(ctx, userID, deckID, day1.Add(-24*time.Hour), nil)
	if err != nil {
		t.Fatalf("DailyCounts() error: %v", err)
	}
	if len(counts) != 2 {
		t.Fatalf("DailyCounts() returned %d days, want 2: %+v", len(counts), counts)
	}

	if !counts[0].Date.Equal(day1.Truncate(24 * time.Hour)) {
		t.Errorf("counts[0].Date = %v, want %v", counts[0].Date, day1.Truncate(24*time.Hour))
	}
	if counts[0].ReviewCount != 2 {
		t.Errorf("counts[0].ReviewCount = %d, want 2", counts[0].ReviewCount)
	}
	if counts[0].CorrectCount != 1 {
		t.Errorf("counts[0].CorrectCount = %d, want 1 (one 'again' out of two)", counts[0].CorrectCount)
	}

	if counts[1].ReviewCount != 1 {
		t.Errorf("counts[1].ReviewCount = %d, want 1", counts[1].ReviewCount)
	}
	if counts[1].CorrectCount != 1 {
		t.Errorf("counts[1].CorrectCount = %d, want 1", counts[1].CorrectCount)
	}
}

func TestReviewEventRepository_DailyCounts_ExcludesBeforeSince(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewReviewEventRepository(db)

	userID := primitive.NewObjectID().Hex()
	deckID := primitive.NewObjectID().Hex()
	flashcardID := primitive.NewObjectID().Hex()

	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := repo.Create(ctx, review.New(userID, flashcardID, deckID, study.RatingGood, study.StateNew, study.StateReview, old, 0)); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	counts, err := repo.DailyCounts(ctx, userID, deckID, time.Now().Add(-7*24*time.Hour), nil)
	if err != nil {
		t.Fatalf("DailyCounts() error: %v", err)
	}
	if len(counts) != 0 {
		t.Errorf("DailyCounts() = %+v, want empty (event predates the window)", counts)
	}
}

func TestReviewEventRepository_DailyCounts_CountsHintUsage(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewReviewEventRepository(db)

	userID := primitive.NewObjectID().Hex()
	deckID := primitive.NewObjectID().Hex()
	flashcardID := primitive.NewObjectID().Hex()
	day := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	withHint := review.New(userID, flashcardID, deckID, study.RatingGood, study.StateNew, study.StateReview, day, 0)
	withHint.HintUsed = true
	withoutHint := review.New(userID, flashcardID, deckID, study.RatingGood, study.StateNew, study.StateReview, day, 0)

	created, err := repo.Create(ctx, withHint)
	if err != nil || !created.HintUsed {
		t.Fatalf("Create() = %+v, %v; want HintUsed persisted", created, err)
	}
	if _, err := repo.Create(ctx, withoutHint); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	counts, err := repo.DailyCounts(ctx, userID, deckID, day.Add(-24*time.Hour), nil)
	if err != nil || len(counts) != 1 {
		t.Fatalf("DailyCounts() = %+v, %v; want one day", counts, err)
	}
	if counts[0].ReviewCount != 2 || counts[0].HintCount != 1 {
		t.Errorf("counts[0] = %+v, want ReviewCount 2 and HintCount 1", counts[0])
	}
}

func TestReviewEventRepository_CountStudiedCards_CountsDistinctCards(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewReviewEventRepository(db)

	userID := primitive.NewObjectID().Hex()
	deckID := primitive.NewObjectID().Hex()
	otherDeckID := primitive.NewObjectID().Hex()
	cardA := primitive.NewObjectID().Hex()
	cardB := primitive.NewObjectID().Hex()
	cardC := primitive.NewObjectID().Hex()

	old := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	recent := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	for _, e := range []review.Event{
		// Card A is reviewed three times: it counts once.
		review.New(userID, cardA, deckID, study.RatingAgain, study.StateNew, study.StateLearning, old, 0),
		review.New(userID, cardA, deckID, study.RatingGood, study.StateLearning, study.StateReview, recent, 0),
		review.New(userID, cardA, deckID, study.RatingGood, study.StateReview, study.StateReview, recent.Add(time.Hour), 0),
		// Card B only long ago, card C only recently.
		review.New(userID, cardB, deckID, study.RatingGood, study.StateNew, study.StateReview, old, 0),
		review.New(userID, cardC, deckID, study.RatingGood, study.StateNew, study.StateReview, recent, 0),
		// Another deck must not be counted.
		review.New(userID, primitive.NewObjectID().Hex(), otherDeckID, study.RatingGood, study.StateNew, study.StateReview, recent, 0),
	} {
		if _, err := repo.Create(ctx, e); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	all, err := repo.CountStudiedCards(ctx, userID, deckID, time.Time{})
	if err != nil || all != 3 {
		t.Errorf("CountStudiedCards(ever) = %d, %v; want 3 (A, B, C)", all, err)
	}
	sinceRecent, err := repo.CountStudiedCards(ctx, userID, deckID, recent.Add(-time.Hour))
	if err != nil || sinceRecent != 2 {
		t.Errorf("CountStudiedCards(recent) = %d, %v; want 2 (A once, C)", sinceRecent, err)
	}
	none, err := repo.CountStudiedCards(ctx, userID, deckID, recent.Add(24*time.Hour))
	if err != nil || none != 0 {
		t.Errorf("CountStudiedCards(future) = %d, %v; want 0", none, err)
	}
	other, err := repo.CountStudiedCards(ctx, primitive.NewObjectID().Hex(), deckID, time.Time{})
	if err != nil || other != 0 {
		t.Errorf("another user's CountStudiedCards = %d, %v; want 0", other, err)
	}
}

func TestReviewEventRepository_DailyCounts_GroupsByLocalDay(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewReviewEventRepository(db)

	loc, err := time.LoadLocation("America/Manaus") // UTC-4
	if err != nil {
		t.Fatalf("LoadLocation() error: %v", err)
	}
	userID := primitive.NewObjectID().Hex()
	deckID := primitive.NewObjectID().Hex()
	cardID := primitive.NewObjectID().Hex()

	// 2026-01-02 01:00 UTC is still 2026-01-01 21:00 in Manaus.
	late := time.Date(2026, 1, 2, 1, 0, 0, 0, time.UTC)
	if _, err := repo.Create(ctx, review.New(userID, cardID, deckID, study.RatingGood, study.StateNew, study.StateReview, late, 0)); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	local, err := repo.DailyCounts(ctx, userID, deckID, late.Add(-24*time.Hour), loc)
	if err != nil || len(local) != 1 || local[0].Date.Format("2006-01-02") != "2026-01-01" {
		t.Errorf("DailyCounts(Manaus) = %+v, %v; want the review on 2026-01-01", local, err)
	}
	utc, err := repo.DailyCounts(ctx, userID, deckID, late.Add(-24*time.Hour), nil)
	if err != nil || len(utc) != 1 || utc[0].Date.Format("2006-01-02") != "2026-01-02" {
		t.Errorf("DailyCounts(UTC) = %+v, %v; want the review on 2026-01-02", utc, err)
	}
}

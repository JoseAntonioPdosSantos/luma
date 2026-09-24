// Package deckservice implements deck CRUD business rules: ownership,
// validation, and the card-count summary used by the dashboard and deck
// details screens (spec sections 3, 6, 14).
package deckservice

import (
	"context"
	"errors"
	"time"

	"flashcard-backend/internal/apperror"
	"flashcard-backend/internal/domain/deck"
	"flashcard-backend/internal/domain/study"
	"flashcard-backend/internal/ports/audiostore"
	"flashcard-backend/internal/ports/clock"
	"flashcard-backend/internal/ports/repositories"
)

// DeckWithCounts is a deck plus the card counts the dashboard and deck
// details screens need, computed by joining across the flashcard
// repository rather than storing counters directly on the deck.
type DeckWithCounts struct {
	Deck       deck.Deck
	TotalCards int
	DueCards   int
	// NextDueAt is when the earliest not-yet-due card comes due, or nil if
	// no card is scheduled in the future. Only GetWithCounts fills it (the
	// deck screen and the study screen need it; the dashboard list does not).
	NextDueAt *time.Time
}

type Service struct {
	decks      repositories.DeckRepository
	flashcards repositories.FlashcardRepository
	reviews    repositories.ReviewEventRepository
	audio      audiostore.Store
	clock      clock.Clock
}

func New(
	decks repositories.DeckRepository,
	flashcards repositories.FlashcardRepository,
	reviews repositories.ReviewEventRepository,
	audio audiostore.Store,
	c clock.Clock,
) Service {
	return Service{decks: decks, flashcards: flashcards, reviews: reviews, audio: audio, clock: c}
}

func (s Service) Create(ctx context.Context, userID, rawName, rawDescription string) (deck.Deck, error) {
	name, err := deck.ValidateName(rawName)
	if err != nil {
		return deck.Deck{}, err
	}
	description, err := deck.ValidateDescription(rawDescription)
	if err != nil {
		return deck.Deck{}, err
	}

	created, err := s.decks.Create(ctx, deck.New(userID, name, description, s.clock.Now()))
	if err != nil {
		return deck.Deck{}, apperror.Internal(err)
	}
	return created, nil
}

// Get returns a deck owned by userID, or apperror.NotFound.
func (s Service) Get(ctx context.Context, userID, deckID string) (deck.Deck, error) {
	d, err := s.decks.FindByID(ctx, userID, deckID)
	if errors.Is(err, repositories.ErrNotFound) {
		return deck.Deck{}, apperror.NotFound("deck.notFound", "deck not found")
	}
	if err != nil {
		return deck.Deck{}, apperror.Internal(err)
	}
	return d, nil
}

// GetWithCounts returns a deck plus its total and due card counts.
func (s Service) GetWithCounts(ctx context.Context, userID, deckID string) (DeckWithCounts, error) {
	d, err := s.Get(ctx, userID, deckID)
	if err != nil {
		return DeckWithCounts{}, err
	}

	now := s.clock.Now()
	total, due, err := s.flashcards.CountByDeck(ctx, userID, deckID, now)
	if err != nil {
		return DeckWithCounts{}, apperror.Internal(err)
	}
	next, err := s.flashcards.NextDueAt(ctx, userID, deckID, now)
	if err != nil {
		return DeckWithCounts{}, apperror.Internal(err)
	}
	return DeckWithCounts{Deck: d, TotalCards: total, DueCards: due, NextDueAt: next}, nil
}

// ListActiveWithCounts returns every non-archived deck owned by userID,
// each with its card counts, for the dashboard (spec section 14.2).
func (s Service) ListActiveWithCounts(ctx context.Context, userID string) ([]DeckWithCounts, error) {
	decks, err := s.decks.ListActive(ctx, userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	result := make([]DeckWithCounts, 0, len(decks))
	for _, d := range decks {
		total, due, err := s.flashcards.CountByDeck(ctx, userID, d.ID, s.clock.Now())
		if err != nil {
			return nil, apperror.Internal(err)
		}
		result = append(result, DeckWithCounts{Deck: d, TotalCards: total, DueCards: due})
	}
	return result, nil
}

// ListArchivedWithCounts returns every archived deck owned by userID,
// most recently archived first, so the user can restore one.
func (s Service) ListArchivedWithCounts(ctx context.Context, userID string) ([]DeckWithCounts, error) {
	decks, err := s.decks.ListArchived(ctx, userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	result := make([]DeckWithCounts, 0, len(decks))
	for _, d := range decks {
		total, due, err := s.flashcards.CountByDeck(ctx, userID, d.ID, s.clock.Now())
		if err != nil {
			return nil, apperror.Internal(err)
		}
		result = append(result, DeckWithCounts{Deck: d, TotalCards: total, DueCards: due})
	}
	return result, nil
}

// Restore un-archives a deck. Its flashcards and review history were
// preserved by Archive, so it reappears exactly as it was.
func (s Service) Restore(ctx context.Context, userID, deckID string) error {
	err := s.decks.Restore(ctx, userID, deckID, s.clock.Now())
	if errors.Is(err, repositories.ErrNotFound) {
		return apperror.NotFound("deck.notFound", "deck not found")
	}
	if err != nil {
		return apperror.Internal(err)
	}
	return nil
}

// DeleteArchived permanently deletes an archived deck together with all of
// its flashcards (active and archived), their audio files and the deck's
// review history. It cannot be undone, so only an already-archived deck is
// accepted; anything else is reported as not found.
//
// The deck document is removed last, so if an earlier step fails the deck
// is still there (archived) and the deletion can simply be retried.
func (s Service) DeleteArchived(ctx context.Context, userID, deckID string) error {
	d, err := s.Get(ctx, userID, deckID)
	if err != nil {
		return err
	}
	if !d.IsArchived() {
		return apperror.NotFound("deck.archivedNotFound", "archived deck not found")
	}

	active, err := s.flashcards.ListByDeck(ctx, userID, deckID)
	if err != nil {
		return apperror.Internal(err)
	}
	archived, err := s.flashcards.ListArchivedByDeck(ctx, userID, deckID)
	if err != nil {
		return apperror.Internal(err)
	}

	if err := s.reviews.DeleteByDeck(ctx, userID, deckID); err != nil {
		return apperror.Internal(err)
	}
	if err := s.flashcards.DeleteAllByDeck(ctx, userID, deckID); err != nil {
		return apperror.Internal(err)
	}
	if err := s.decks.DeleteArchived(ctx, userID, deckID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return apperror.NotFound("deck.archivedNotFound", "archived deck not found")
		}
		return apperror.Internal(err)
	}

	// Best-effort cleanup: the audio files are orphaned by now, and a
	// failure here must not undo or fail the deletion.
	for _, card := range append(active, archived...) {
		if card.Audio != nil {
			_ = s.audio.Delete(ctx, card.Audio.GridFSFileID)
		}
	}
	return nil
}

func (s Service) Update(ctx context.Context, userID, deckID, rawName, rawDescription string) (deck.Deck, error) {
	existing, err := s.Get(ctx, userID, deckID)
	if err != nil {
		return deck.Deck{}, err
	}

	name, err := deck.ValidateName(rawName)
	if err != nil {
		return deck.Deck{}, err
	}
	description, err := deck.ValidateDescription(rawDescription)
	if err != nil {
		return deck.Deck{}, err
	}

	existing.Name = name
	existing.Description = description
	existing.UpdatedAt = s.clock.Now()

	updated, err := s.decks.Update(ctx, existing)
	if errors.Is(err, repositories.ErrNotFound) {
		return deck.Deck{}, apperror.NotFound("deck.notFound", "deck not found")
	}
	if err != nil {
		return deck.Deck{}, apperror.Internal(err)
	}
	return updated, nil
}

// statsWindowDays is how far back the daily-performance chart looks. Not
// part of the written product spec (see DeckStats doc comment); chosen to
// match the original design mockup's "últimos 7 dias" chart.
const statsWindowDays = 7

// DeckStats is the per-deck statistics screen's data: how the deck's
// cards are distributed across the spaced-repetition lifecycle, plus
// recent review performance. This screen was not part of the written
// product spec's MVP (which lists "learning analytics" as a future
// improvement, spec section 26) — it exists because the project's
// original design mockup included it and the user asked for it to be
// built to match, after being told it goes beyond the formal MVP scope.
type DeckStats struct {
	Deck           deck.Deck
	TotalCards     int
	NewCards       int
	LearningCards  int // LEARNING or REVIEW: seen at least once, not yet mature
	MasteredCards  int // MATURE
	MasteryPercent int
	// Distinct cards reviewed at least once in each period; a card
	// reviewed several times counts once. "Today" and the 7-day window
	// follow the caller's time zone.
	StudiedToday     int
	StudiedLast7Days int
	StudiedTotal     int
	DailyPerformance []DailyPerformance
}

// DailyPerformance is one day's review activity and accuracy.
type DailyPerformance struct {
	Date            time.Time
	ReviewCount     int
	AccuracyPercent int
	HintCount       int // reviews in which the card's hint was used
}

// GetStats computes the statistics screen's data for one deck.
func (s Service) GetStats(ctx context.Context, userID, deckID string, loc *time.Location) (DeckStats, error) {
	if loc == nil {
		loc = time.UTC
	}
	d, err := s.Get(ctx, userID, deckID)
	if err != nil {
		return DeckStats{}, err
	}

	counts, err := s.flashcards.CountsByState(ctx, userID, deckID)
	if err != nil {
		return DeckStats{}, apperror.Internal(err)
	}

	newCards := counts[study.StateNew]
	learningCards := counts[study.StateLearning] + counts[study.StateReview]
	masteredCards := counts[study.StateMature]
	total := newCards + learningCards + masteredCards

	masteryPercent := 0
	if total > 0 {
		masteryPercent = masteredCards * 100 / total
	}

	// Days are the caller's local days, so "today" starts at their local
	// midnight rather than at UTC midnight.
	now := s.clock.Now().In(loc)
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	since := startOfToday.AddDate(0, 0, -(statsWindowDays - 1))
	dailyCounts, err := s.reviews.DailyCounts(ctx, userID, deckID, since, loc)
	if err != nil {
		return DeckStats{}, apperror.Internal(err)
	}

	studiedToday, err := s.reviews.CountStudiedCards(ctx, userID, deckID, startOfToday)
	if err != nil {
		return DeckStats{}, apperror.Internal(err)
	}
	studiedLast7Days, err := s.reviews.CountStudiedCards(ctx, userID, deckID, since)
	if err != nil {
		return DeckStats{}, apperror.Internal(err)
	}
	studiedTotal, err := s.reviews.CountStudiedCards(ctx, userID, deckID, time.Time{})
	if err != nil {
		return DeckStats{}, apperror.Internal(err)
	}

	performance := make([]DailyPerformance, 0, len(dailyCounts))
	for _, dc := range dailyCounts {
		accuracy := 0
		if dc.ReviewCount > 0 {
			accuracy = dc.CorrectCount * 100 / dc.ReviewCount
		}
		performance = append(performance, DailyPerformance{
			Date:            dc.Date,
			ReviewCount:     dc.ReviewCount,
			AccuracyPercent: accuracy,
			HintCount:       dc.HintCount,
		})
	}

	return DeckStats{
		Deck:             d,
		TotalCards:       total,
		NewCards:         newCards,
		LearningCards:    learningCards,
		MasteredCards:    masteredCards,
		MasteryPercent:   masteryPercent,
		StudiedToday:     studiedToday,
		StudiedLast7Days: studiedLast7Days,
		StudiedTotal:     studiedTotal,
		DailyPerformance: performance,
	}, nil
}

// Archive soft-deletes a deck: it stops appearing in the dashboard but its
// flashcards and review history are preserved (spec never calls for
// permanently deleting a deck).
func (s Service) Archive(ctx context.Context, userID, deckID string) error {
	err := s.decks.Archive(ctx, userID, deckID, s.clock.Now())
	if errors.Is(err, repositories.ErrNotFound) {
		return apperror.NotFound("deck.notFound", "deck not found")
	}
	if err != nil {
		return apperror.Internal(err)
	}
	return nil
}

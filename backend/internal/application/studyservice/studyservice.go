// Package studyservice implements the study flow: selecting cards due for
// review and recording a rating against a card (spec section 9).
//
// The spec's section 13 sketches a stateful "study session" resource
// (POST .../study-sessions, then POST reviews against a session ID), but
// explicitly allows a simpler design as long as ownership, consistency,
// and state transitions stay clear. This package intentionally has no
// session entity: a new card's SchedulingState.DueAt is set to its
// creation time (see domain/study.NewSchedulingState), so it is already
// "due" the moment it exists — the single due-cards query naturally
// covers spec section 9 step 4 ("if there are no due cards, optionally
// offer new cards") without special-casing NEW separately. ReviewEvent
// itself has no session ID in the domain model (spec section 6), which
// confirms review history was never meant to be grouped by a session.
package studyservice

import (
	"context"
	"errors"
	"sort"
	"time"

	"flashcard-backend/internal/apperror"
	"flashcard-backend/internal/domain/flashcard"
	"flashcard-backend/internal/domain/review"
	"flashcard-backend/internal/domain/study"
	"flashcard-backend/internal/domain/studyprofile"
	"flashcard-backend/internal/ports/clock"
	"flashcard-backend/internal/ports/repositories"
)

const (
	DefaultDueCardsLimit = 20
	MaxDueCardsLimit     = 100
)

// ProfileSource tells which study configuration applies: the rules for each
// rating and the daily card goal. A deck may have a configuration of its own;
// otherwise the user's general (active) one applies.
type ProfileSource interface {
	// Active is the user's general configuration.
	Active(ctx context.Context, userID string) (studyprofile.Profile, error)
	// ForDeck is the configuration a deck is studied with, and whether it is
	// the deck's own (false: it follows the general one).
	ForDeck(ctx context.Context, userID, deckID string) (profile studyprofile.Profile, own bool, err error)
	// OwnDeckIDs lists the decks that have a configuration of their own.
	OwnDeckIDs(ctx context.Context, userID string) ([]string, error)
}

type Service struct {
	flashcards repositories.FlashcardRepository
	users      repositories.UserRepository
	profiles   ProfileSource
	decks      repositories.DeckRepository
	reviews    repositories.ReviewEventRepository
	clock      clock.Clock
}

func New(
	flashcards repositories.FlashcardRepository,
	users repositories.UserRepository,
	profiles ProfileSource,
	decks repositories.DeckRepository,
	reviews repositories.ReviewEventRepository,
	c clock.Clock,
) Service {
	return Service{flashcards: flashcards, users: users, profiles: profiles, decks: decks, reviews: reviews, clock: c}
}

// DueOptions tunes DueCards and Progress.
type DueOptions struct {
	// Location defines what "today" is for the daily goal; nil means UTC.
	Location *time.Location
	// IgnoreDailyLimit serves due cards even after the user's daily goal
	// was reached ("study more anyway").
	IgnoreDailyLimit bool
}

// Progress is how the user stands against their daily goal today.
type Progress struct {
	// DailyCardLimit is the user's goal, or nil when they have none.
	DailyCardLimit *int
	// StudiedToday is the number of different cards studied today, across
	// all decks.
	StudiedToday int
	// Remaining is how many more cards fit in today's goal (never
	// negative), or nil when there is no goal.
	Remaining *int
	// ContinuingPastGoal is true when the user already chose, today, to keep
	// studying past the goal: due cards are then served without the cap.
	ContinuingPastGoal bool
}

// localDay is the calendar day of now in loc as YYYY-MM-DD (loc nil = UTC).
func localDay(now time.Time, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	return now.In(loc).Format("2006-01-02")
}

func startOfToday(now time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	local := now.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}

// Progress reports the daily goal that applies and how much of it was used
// today. With a deckID, it is the goal of that deck: its own configuration's
// goal counting only that deck's cards, or, if the deck has none, the general
// goal. Without a deckID (or for a deck that follows the general
// configuration) the general goal counts the cards studied today in every deck
// that does not have a configuration of its own.
func (s Service) Progress(ctx context.Context, userID, deckID string, loc *time.Location) (Progress, error) {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return Progress{}, apperror.NotFound("user.notFound", "user not found")
		}
		return Progress{}, apperror.Internal(err)
	}

	today := localDay(s.clock.Now(), loc)
	since := startOfToday(s.clock.Now(), loc)

	var (
		profile   studyprofile.Profile
		scope     repositories.ReviewScope
		continued bool
	)
	own := false
	if deckID != "" {
		profile, own, err = s.profiles.ForDeck(ctx, userID, deckID)
		if err != nil {
			return Progress{}, err
		}
	}
	if own {
		scope = repositories.ReviewScope{OnlyDeckID: deckID}
		d, err := s.decks.FindByID(ctx, userID, deckID)
		if err != nil {
			return Progress{}, apperror.Internal(err)
		}
		continued = d.ContinuePastGoalOn != "" && d.ContinuePastGoalOn == today
	} else {
		if deckID == "" {
			profile, err = s.profiles.Active(ctx, userID)
			if err != nil {
				return Progress{}, err
			}
		}
		excluded, err := s.profiles.OwnDeckIDs(ctx, userID)
		if err != nil {
			return Progress{}, err
		}
		scope = repositories.ReviewScope{ExcludeDeckIDs: excluded}
		continued = u.ContinuePastGoalOn != "" && u.ContinuePastGoalOn == today
	}

	studied, err := s.flashcards.CountReviewedSince(ctx, userID, since, scope)
	if err != nil {
		return Progress{}, apperror.Internal(err)
	}

	progress := Progress{DailyCardLimit: profile.DailyCardLimit, StudiedToday: studied, ContinuingPastGoal: continued}
	if progress.DailyCardLimit != nil {
		remaining := *progress.DailyCardLimit - studied
		if remaining < 0 {
			remaining = 0
		}
		progress.Remaining = &remaining
	}
	return progress, nil
}

// ContinuePastGoal records that, for the rest of today (in loc), the user
// wants to keep studying past the daily goal that applies. Being stored on the
// account it holds on every device, and it lapses on its own when the day
// ends. For a deck with a configuration of its own the choice belongs to that
// deck alone; otherwise it belongs to the general goal.
func (s Service) ContinuePastGoal(ctx context.Context, userID, deckID string, loc *time.Location) error {
	now := s.clock.Now()
	day := localDay(now, loc)

	if deckID != "" {
		_, own, err := s.profiles.ForDeck(ctx, userID, deckID)
		if err != nil {
			return err
		}
		if own {
			err := s.decks.SetContinuePastGoalOn(ctx, userID, deckID, day, now)
			if errors.Is(err, repositories.ErrNotFound) {
				return apperror.NotFound("deck.notFound", "deck not found")
			}
			if err != nil {
				return apperror.Internal(err)
			}
			return nil
		}
	}

	err := s.users.SetContinuePastGoalOn(ctx, userID, day, now)
	if errors.Is(err, repositories.ErrNotFound) {
		return apperror.NotFound("user.notFound", "user not found")
	}
	if err != nil {
		return apperror.Internal(err)
	}
	return nil
}

// DueCards returns up to limit non-archived cards in deckID that are due
// for review, verifying that the deck belongs to userID first. limit <= 0
// defaults to DefaultDueCardsLimit; it is capped at MaxDueCardsLimit.
//
// Without a daily goal (the default), or once the user chose to continue
// past it today, this is simply every due card. With a goal, at most as many cards that were not studied yet today are served as
// the goal still has room for. Cards already studied today that come due
// again (a card answered "again" returns after a few minutes) are still
// served: they were already counted, and cutting them off would leave the
// card half-learned.
func (s Service) DueCards(ctx context.Context, userID, deckID string, limit int, opts DueOptions) ([]flashcard.Flashcard, error) {
	if _, err := s.decks.FindByID(ctx, userID, deckID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, apperror.NotFound("deck.notFound", "deck not found")
		}
		return nil, apperror.Internal(err)
	}

	if limit <= 0 {
		limit = DefaultDueCardsLimit
	}
	if limit > MaxDueCardsLimit {
		limit = MaxDueCardsLimit
	}

	now := s.clock.Now()
	progress, err := s.Progress(ctx, userID, deckID, opts.Location)
	if err != nil {
		return nil, err
	}

	if progress.Remaining == nil || opts.IgnoreDailyLimit || progress.ContinuingPastGoal {
		cards, err := s.flashcards.ListDue(ctx, userID, deckID, now, limit)
		if err != nil {
			return nil, apperror.Internal(err)
		}
		return cards, nil
	}

	since := startOfToday(now, opts.Location)
	cards, err := s.flashcards.ListDueByStudiedToday(ctx, userID, deckID, now, since, true, limit)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if room := *progress.Remaining; room > 0 {
		fresh, err := s.flashcards.ListDueByStudiedToday(ctx, userID, deckID, now, since, false, minInt(limit, room))
		if err != nil {
			return nil, apperror.Internal(err)
		}
		cards = append(cards, fresh...)
	}

	sort.SliceStable(cards, func(i, j int) bool { return cards[i].Scheduling.DueAt.Before(cards[j].Scheduling.DueAt) })
	if len(cards) > limit {
		cards = cards[:limit]
	}
	return cards, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RecordReview applies rating to the flashcard's scheduling state using
// the configured Scheduler, persists the new state, and records an
// immutable review event (including whether the hint was used, which is
// informational only and does not change the scheduling). It returns the updated flashcard.
func (s Service) RecordReview(ctx context.Context, userID, flashcardID string, rating study.Rating, responseTimeMs int64, hintUsed bool) (flashcard.Flashcard, error) {
	card, err := s.flashcards.FindByID(ctx, userID, flashcardID)
	if errors.Is(err, repositories.ErrNotFound) {
		return flashcard.Flashcard{}, apperror.NotFound("flashcard.notFound", "flashcard not found")
	}
	if err != nil {
		return flashcard.Flashcard{}, apperror.Internal(err)
	}
	// Archived cards must not appear in study sessions (spec section 6);
	// reject a review against one as if it did not exist, rather than
	// silently resurrecting it into the schedule.
	if card.IsArchived() {
		return flashcard.Flashcard{}, apperror.NotFound("flashcard.notFound", "flashcard not found")
	}

	now := s.clock.Now()
	previousState := card.Scheduling.State

	// How this answer reschedules the card follows the study configuration
	// of the card's deck: its own if it has one, else the user's general one.
	profile, _, err := s.profiles.ForDeck(ctx, userID, card.DeckID)
	if err != nil {
		return flashcard.Flashcard{}, err
	}
	scheduler := study.NewRulesScheduler(profile.Rules, study.DefaultMaxIntervalDays)
	decision := scheduler.Schedule(card.Scheduling, rating, now)
	card.Scheduling.State = decision.State
	card.Scheduling.DueAt = decision.DueAt
	card.Scheduling.IntervalDays = decision.IntervalDays
	card.Scheduling.Repetitions = decision.Repetitions
	card.Scheduling.Lapses = decision.Lapses
	card.Scheduling.EaseFactor = decision.EaseFactor
	card.Scheduling.LastReviewedAt = &now
	card.UpdatedAt = now

	updated, err := s.flashcards.Update(ctx, card)
	if err != nil {
		return flashcard.Flashcard{}, apperror.Internal(err)
	}

	event := review.New(userID, card.ID, card.DeckID, rating, previousState, decision.State, now, responseTimeMs)
	event.HintUsed = hintUsed
	if _, err := s.reviews.Create(ctx, event); err != nil {
		return flashcard.Flashcard{}, apperror.Internal(err)
	}

	return updated, nil
}

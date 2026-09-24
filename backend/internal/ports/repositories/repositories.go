// Package repositories defines the persistence interfaces the application
// layer depends on. Concrete implementations live under
// internal/adapters/mongodb; nothing outside that adapter package should
// import a MongoDB type.
package repositories

import (
	"context"
	"time"

	"flashcard-backend/internal/domain/deck"
	"flashcard-backend/internal/domain/deckgroup"
	"flashcard-backend/internal/domain/flashcard"
	"flashcard-backend/internal/domain/review"
	"flashcard-backend/internal/domain/study"
	"flashcard-backend/internal/domain/studyprofile"
	"flashcard-backend/internal/domain/user"
)

// ErrNotFound is returned by a repository when the requested document does
// not exist (or does not belong to the given owner).
var ErrNotFound = notFoundError{}

type notFoundError struct{}

func (notFoundError) Error() string { return "not found" }

// ErrDuplicate is returned by a repository when a write would break a
// uniqueness rule (for example two study configurations with the same name).
var ErrDuplicate = duplicateError{}

type duplicateError struct{}

func (duplicateError) Error() string { return "duplicate" }

// UserRepository persists and looks up users by normalized email.
type UserRepository interface {
	// FindByEmail returns ErrNotFound if no user has this email.
	FindByEmail(ctx context.Context, normalizedEmail string) (user.User, error)
	FindByID(ctx context.Context, id string) (user.User, error)
	Create(ctx context.Context, u user.User) (user.User, error)
	// TouchLastAccess updates LastAccessAt to now.
	TouchLastAccess(ctx context.Context, id string, now time.Time) error
	// SetActiveStudyProfile records which study configuration the user
	// selected ("" or studyprofile.DefaultID for the built-in default).
	// Returns ErrNotFound if the user does not exist.
	SetActiveStudyProfile(ctx context.Context, id, profileID string, now time.Time) error
	// ClearLegacyDailyCardLimit removes the daily goal saved before study
	// configurations existed, once it was migrated.
	ClearLegacyDailyCardLimit(ctx context.Context, id string, now time.Time) error
	// SetContinuePastGoalOn records the local day (YYYY-MM-DD) on which the
	// user chose to study past their daily goal. Returns ErrNotFound if the
	// user does not exist.
	SetContinuePastGoalOn(ctx context.Context, id, day string, now time.Time) error
	// SetPreferredLanguage records the UI language the user chose (an
	// user.Language value). Returns ErrNotFound if the user does not exist.
	SetPreferredLanguage(ctx context.Context, id, language string, now time.Time) error
}

// StudyProfileRepository persists the study configurations users create. The
// built-in default is not stored.
type StudyProfileRepository interface {
	// Create returns ErrDuplicate if the user already has a configuration
	// with the same name (ignoring case).
	Create(ctx context.Context, p studyprofile.Profile) (studyprofile.Profile, error)
	// FindByID returns ErrNotFound if the configuration does not exist or
	// does not belong to userID.
	FindByID(ctx context.Context, userID, id string) (studyprofile.Profile, error)
	// ListByUser returns the user's configurations, oldest first.
	ListByUser(ctx context.Context, userID string) ([]studyprofile.Profile, error)
	// Update returns ErrNotFound if it does not exist or belongs to someone
	// else, and ErrDuplicate if the new name is already taken.
	Update(ctx context.Context, p studyprofile.Profile) (studyprofile.Profile, error)
	// Delete returns ErrNotFound if it does not exist or belongs to someone else.
	Delete(ctx context.Context, userID, id string) error
}

// DeckRepository persists and queries decks scoped to their owning user.
type DeckRepository interface {
	Create(ctx context.Context, d deck.Deck) (deck.Deck, error)
	// FindByID returns ErrNotFound if the deck does not exist or does not
	// belong to userID.
	FindByID(ctx context.Context, userID, id string) (deck.Deck, error)
	// ListActive returns the non-archived decks owned by userID.
	ListActive(ctx context.Context, userID string) ([]deck.Deck, error)
	// ListArchived returns the archived decks owned by userID, most
	// recently archived first.
	ListArchived(ctx context.Context, userID string) ([]deck.Deck, error)
	Update(ctx context.Context, d deck.Deck) (deck.Deck, error)
	Archive(ctx context.Context, userID, id string, now time.Time) error
	// Restore un-archives a deck. It is idempotent: restoring a deck that
	// is not archived succeeds. Returns ErrNotFound if the deck does not
	// exist or does not belong to userID.
	Restore(ctx context.Context, userID, id string, now time.Time) error
	// SetStudyProfile records the study configuration the deck uses ("" =
	// follow the user's general one). Returns ErrNotFound if the deck does
	// not exist or does not belong to userID.
	SetStudyProfile(ctx context.Context, userID, id, profileID string, now time.Time) error
	// SetContinuePastGoalOn records the local day (YYYY-MM-DD) on which the
	// user chose to study this deck past its own daily goal. Returns
	// ErrNotFound if the deck does not exist or does not belong to userID.
	SetContinuePastGoalOn(ctx context.Context, userID, id, day string, now time.Time) error
	// ClearStudyProfile makes every deck of userID that uses profileID
	// follow the general configuration again (used when it is deleted).
	ClearStudyProfile(ctx context.Context, userID, profileID string) error
	// ListWithStudyProfile returns the user's decks (archived or not) that
	// have a configuration of their own.
	ListWithStudyProfile(ctx context.Context, userID string) ([]deck.Deck, error)
	// DeleteArchived permanently removes a deck, but only if it is
	// archived. Returns ErrNotFound if the deck does not exist, is not
	// archived, or does not belong to userID.
	DeleteArchived(ctx context.Context, userID, id string) error
	// SetGroup files the deck under groupID, or removes it from any group
	// when groupID is "". Returns ErrNotFound if the deck does not exist or
	// does not belong to userID.
	SetGroup(ctx context.Context, userID, id, groupID string, now time.Time) error
	// ClearGroup removes every deck of userID from groupID (used when the
	// group is deleted).
	ClearGroup(ctx context.Context, userID, groupID string) error
}

// DeckGroupRepository persists the groups a user creates to organize decks.
// A group holds decks directly; it never holds other groups.
type DeckGroupRepository interface {
	// Create returns ErrDuplicate if the user already has a group with this
	// name (ignoring case).
	Create(ctx context.Context, g deckgroup.Group) (deckgroup.Group, error)
	// FindByID returns ErrNotFound if the group does not exist or does not
	// belong to userID.
	FindByID(ctx context.Context, userID, id string) (deckgroup.Group, error)
	// ListByUser returns the user's groups, oldest first.
	ListByUser(ctx context.Context, userID string) ([]deckgroup.Group, error)
	// Update (rename) returns ErrNotFound if it does not exist or belongs to
	// someone else, and ErrDuplicate if the new name is already taken.
	Update(ctx context.Context, g deckgroup.Group) (deckgroup.Group, error)
	// Delete returns ErrNotFound if it does not exist or belongs to someone else.
	Delete(ctx context.Context, userID, id string) error
}

// ReviewScope narrows which decks a review count looks at.
type ReviewScope struct {
	// OnlyDeckID, when not empty, counts that deck alone.
	OnlyDeckID string
	// ExcludeDeckIDs skips these decks (used when OnlyDeckID is empty): the
	// decks that have their own configuration and their own count.
	ExcludeDeckIDs []string
}

// FlashcardListOptions selects one page of a deck's flashcards.
type FlashcardListOptions struct {
	// Archived selects archived cards instead of active ones.
	Archived bool
	// Query, when not empty, keeps only the cards whose question, answer or
	// hint contain it (case- and accent-insensitive).
	Query string
	// Limit and Offset page the result; the caller must pass Limit > 0.
	Limit  int
	Offset int
}

// FlashcardRepository persists and queries flashcards scoped to their
// owning user and deck.
type FlashcardRepository interface {
	Create(ctx context.Context, f flashcard.Flashcard) (flashcard.Flashcard, error)
	// FindByID returns ErrNotFound if the flashcard does not exist or does
	// not belong to userID.
	FindByID(ctx context.Context, userID, id string) (flashcard.Flashcard, error)
	// ListByDeck returns the non-archived flashcards in the given deck.
	ListByDeck(ctx context.Context, userID, deckID string) ([]flashcard.Flashcard, error)
	// SearchByDeck returns one page of the deck's flashcards matching opts,
	// plus the total number of matches (ignoring Limit/Offset). Active cards
	// come newest first; archived cards most recently archived first.
	SearchByDeck(ctx context.Context, userID, deckID string, opts FlashcardListOptions) (cards []flashcard.Flashcard, total int, err error)
	// NextDueAt returns the earliest DueAt strictly after now among the
	// deck's non-archived cards, or nil if none is scheduled in the future.
	// It tells the learner when the next review comes up once nothing is due.
	NextDueAt(ctx context.Context, userID, deckID string, now time.Time) (*time.Time, error)
	// CountReviewedSince returns how many of the user's cards were last
	// reviewed at or after since, within scope. Each card counts once no
	// matter how many times it was reviewed, which is what a daily goal
	// ("how many different cards did I study today") needs.
	CountReviewedSince(ctx context.Context, userID string, since time.Time, scope ReviewScope) (int, error)
	// ListDueByStudiedToday is ListDue restricted to cards already reviewed
	// at or after since (reviewedSince=true) or to those not yet reviewed
	// since then (reviewedSince=false).
	ListDueByStudiedToday(ctx context.Context, userID, deckID string, now, since time.Time, reviewedSince bool, limit int) ([]flashcard.Flashcard, error)
	// ListDue returns the non-archived flashcards in the given deck whose
	// DueAt is at or before now, ordered by DueAt ascending, limited to
	// limit results.
	ListDue(ctx context.Context, userID, deckID string, now time.Time, limit int) ([]flashcard.Flashcard, error)
	// ListArchivedByDeck returns the archived flashcards in the given deck,
	// most recently archived first.
	ListArchivedByDeck(ctx context.Context, userID, deckID string) ([]flashcard.Flashcard, error)
	// ListArchivedByUser returns every archived flashcard owned by userID
	// across all decks, most recently archived first.
	ListArchivedByUser(ctx context.Context, userID string) ([]flashcard.Flashcard, error)
	Update(ctx context.Context, f flashcard.Flashcard) (flashcard.Flashcard, error)
	Archive(ctx context.Context, userID, id string, now time.Time) error
	// Restore un-archives a flashcard, keeping its scheduling state and
	// review history. It is idempotent. Returns ErrNotFound if the card
	// does not exist or does not belong to userID.
	Restore(ctx context.Context, userID, id string, now time.Time) error
	// DeleteArchived permanently removes a flashcard, but only if it is
	// archived, and returns the removed card (so the caller can clean up
	// its audio). Returns ErrNotFound if the card does not exist, is not
	// archived, or does not belong to userID.
	DeleteArchived(ctx context.Context, userID, id string) (flashcard.Flashcard, error)
	// DeleteAllByDeck permanently removes every flashcard (active and
	// archived) of the given deck.
	DeleteAllByDeck(ctx context.Context, userID, deckID string) error
	CountByDeck(ctx context.Context, userID, deckID string, now time.Time) (total int, due int, err error)
	// CountsByState returns, for every non-archived card in the deck, how
	// many are in each spaced-repetition state (spec section 9's
	// NEW/LEARNING/REVIEW/MATURE state machine). States with zero cards
	// may be omitted from the map.
	CountsByState(ctx context.Context, userID, deckID string) (map[study.CardState]int, error)
}

// ReviewEventRepository persists the immutable review history.
type ReviewEventRepository interface {
	Create(ctx context.Context, e review.Event) (review.Event, error)
	// DeleteByDeck permanently removes the review history of a deck. Used
	// only when the deck itself is being permanently deleted.
	DeleteByDeck(ctx context.Context, userID, deckID string) error
	// DailyCounts returns one entry per calendar day (in loc) that had at
	// least one review of a card in deckID, for days at or after since,
	// ordered by date ascending. A nil loc means UTC.
	DailyCounts(ctx context.Context, userID, deckID string, since time.Time, loc *time.Location) ([]DailyReviewCount, error)
	// CountStudiedCards returns how many distinct cards of deckID were
	// reviewed at least once at or after since (a zero since means ever).
	// A card reviewed many times counts once.
	CountStudiedCards(ctx context.Context, userID, deckID string, since time.Time) (int, error)
}

// DailyReviewCount summarizes one day's reviews for the deck-statistics
// screen (spec's original design mockup; not part of the written product
// spec's MVP scope, added on request).
type DailyReviewCount struct {
	Date         time.Time // truncated to the day, UTC
	ReviewCount  int
	CorrectCount int // ratings other than "again"
	HintCount    int // reviews in which the card's hint was used
}

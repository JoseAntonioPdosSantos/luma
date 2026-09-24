package studyservice

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"flashcard-backend/internal/domain/deck"
	"flashcard-backend/internal/domain/flashcard"
	"flashcard-backend/internal/domain/review"
	"flashcard-backend/internal/domain/study"
	"flashcard-backend/internal/domain/studyprofile"
	"flashcard-backend/internal/domain/user"
	"flashcard-backend/internal/ports/clock"
	"flashcard-backend/internal/ports/repositories"
)

type fakeDeckRepository struct {
	decks map[string]deck.Deck
}

func newFakeDeckRepository() *fakeDeckRepository {
	return &fakeDeckRepository{decks: map[string]deck.Deck{}}
}

func (f *fakeDeckRepository) seed(userID, id string) {
	f.decks[id] = deck.Deck{ID: id, UserID: userID}
}

func (f *fakeDeckRepository) Create(context.Context, deck.Deck) (deck.Deck, error) {
	panic("not implemented")
}
func (f *fakeDeckRepository) FindByID(_ context.Context, userID, id string) (deck.Deck, error) {
	d, ok := f.decks[id]
	if !ok || d.UserID != userID {
		return deck.Deck{}, repositories.ErrNotFound
	}
	return d, nil
}
func (f *fakeDeckRepository) ListActive(context.Context, string) ([]deck.Deck, error) {
	panic("not implemented")
}
func (f *fakeDeckRepository) ListArchived(context.Context, string) ([]deck.Deck, error) {
	panic("not implemented")
}
func (f *fakeDeckRepository) Restore(context.Context, string, string, time.Time) error {
	panic("not implemented")
}
func (f *fakeDeckRepository) SetStudyProfile(context.Context, string, string, string, time.Time) error {
	panic("not implemented")
}
func (f *fakeDeckRepository) SetContinuePastGoalOn(_ context.Context, userID, id, day string, _ time.Time) error {
	d, ok := f.decks[id]
	if !ok || d.UserID != userID {
		return repositories.ErrNotFound
	}
	d.ContinuePastGoalOn = day
	f.decks[id] = d
	return nil
}
func (f *fakeDeckRepository) ClearStudyProfile(context.Context, string, string) error {
	panic("not implemented")
}
func (f *fakeDeckRepository) ListWithStudyProfile(context.Context, string) ([]deck.Deck, error) {
	panic("not implemented")
}
func (f *fakeDeckRepository) SetGroup(context.Context, string, string, string, time.Time) error {
	panic("not implemented")
}
func (f *fakeDeckRepository) ClearGroup(context.Context, string, string) error {
	panic("not implemented")
}
func (f *fakeDeckRepository) DeleteArchived(context.Context, string, string) error {
	panic("not implemented")
}
func (f *fakeDeckRepository) Update(context.Context, deck.Deck) (deck.Deck, error) {
	panic("not implemented")
}
func (f *fakeDeckRepository) Archive(context.Context, string, string, time.Time) error {
	panic("not implemented")
}

type fakeFlashcardRepository struct {
	cards  map[string]flashcard.Flashcard
	nextID int
}

func newFakeFlashcardRepository() *fakeFlashcardRepository {
	return &fakeFlashcardRepository{cards: map[string]flashcard.Flashcard{}}
}

func (f *fakeFlashcardRepository) seed(c flashcard.Flashcard) flashcard.Flashcard {
	f.nextID++
	c.ID = fmt.Sprintf("card-%d", f.nextID)
	f.cards[c.ID] = c
	return c
}

func (f *fakeFlashcardRepository) Create(context.Context, flashcard.Flashcard) (flashcard.Flashcard, error) {
	panic("not implemented")
}
func (f *fakeFlashcardRepository) FindByID(_ context.Context, userID, id string) (flashcard.Flashcard, error) {
	c, ok := f.cards[id]
	if !ok || c.UserID != userID {
		return flashcard.Flashcard{}, repositories.ErrNotFound
	}
	return c, nil
}
func (f *fakeFlashcardRepository) ListByDeck(context.Context, string, string) ([]flashcard.Flashcard, error) {
	panic("not implemented")
}
func (f *fakeFlashcardRepository) NextDueAt(context.Context, string, string, time.Time) (*time.Time, error) {
	panic("not implemented")
}
func (f *fakeFlashcardRepository) CountReviewedSince(_ context.Context, userID string, since time.Time, scope repositories.ReviewScope) (int, error) {
	excluded := map[string]bool{}
	for _, id := range scope.ExcludeDeckIDs {
		excluded[id] = true
	}
	n := 0
	for _, c := range f.cards {
		if c.UserID != userID || c.Scheduling.LastReviewedAt == nil || c.Scheduling.LastReviewedAt.Before(since) {
			continue
		}
		if scope.OnlyDeckID != "" && c.DeckID != scope.OnlyDeckID {
			continue
		}
		if scope.OnlyDeckID == "" && excluded[c.DeckID] {
			continue
		}
		n++
	}
	return n, nil
}
func (f *fakeFlashcardRepository) ListDueByStudiedToday(_ context.Context, userID, deckID string, now, since time.Time, reviewedSince bool, limit int) ([]flashcard.Flashcard, error) {
	var result []flashcard.Flashcard
	for _, c := range f.cards {
		if c.UserID != userID || c.DeckID != deckID || c.IsArchived() || c.Scheduling.DueAt.After(now) {
			continue
		}
		studied := c.Scheduling.LastReviewedAt != nil && !c.Scheduling.LastReviewedAt.Before(since)
		if studied == reviewedSince {
			result = append(result, c)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Scheduling.DueAt.Before(result[j].Scheduling.DueAt) })
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}
func (f *fakeFlashcardRepository) SearchByDeck(context.Context, string, string, repositories.FlashcardListOptions) ([]flashcard.Flashcard, int, error) {
	panic("not implemented")
}
func (f *fakeFlashcardRepository) ListArchivedByDeck(context.Context, string, string) ([]flashcard.Flashcard, error) {
	panic("not implemented")
}
func (f *fakeFlashcardRepository) Restore(context.Context, string, string, time.Time) error {
	panic("not implemented")
}
func (f *fakeFlashcardRepository) ListArchivedByUser(context.Context, string) ([]flashcard.Flashcard, error) {
	panic("not implemented")
}
func (f *fakeFlashcardRepository) DeleteArchived(context.Context, string, string) (flashcard.Flashcard, error) {
	panic("not implemented")
}
func (f *fakeFlashcardRepository) DeleteAllByDeck(context.Context, string, string) error {
	panic("not implemented")
}
func (f *fakeFlashcardRepository) ListDue(_ context.Context, userID, deckID string, now time.Time, limit int) ([]flashcard.Flashcard, error) {
	var result []flashcard.Flashcard
	for _, c := range f.cards {
		if c.UserID == userID && c.DeckID == deckID && !c.IsArchived() && !c.Scheduling.DueAt.After(now) {
			result = append(result, c)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}
func (f *fakeFlashcardRepository) Update(_ context.Context, c flashcard.Flashcard) (flashcard.Flashcard, error) {
	existing, ok := f.cards[c.ID]
	if !ok || existing.UserID != c.UserID {
		return flashcard.Flashcard{}, repositories.ErrNotFound
	}
	f.cards[c.ID] = c
	return c, nil
}
func (f *fakeFlashcardRepository) Archive(_ context.Context, userID, id string, now time.Time) error {
	existing, ok := f.cards[id]
	if !ok || existing.UserID != userID {
		return repositories.ErrNotFound
	}
	existing.ArchivedAt = &now
	f.cards[id] = existing
	return nil
}
func (f *fakeFlashcardRepository) CountByDeck(context.Context, string, string, time.Time) (int, int, error) {
	panic("not implemented")
}
func (f *fakeFlashcardRepository) CountsByState(context.Context, string, string) (map[study.CardState]int, error) {
	panic("not implemented")
}

type fakeReviewEventRepository struct {
	events []review.Event
}

func (f *fakeReviewEventRepository) DeleteByDeck(context.Context, string, string) error {
	panic("not implemented")
}
func (f *fakeReviewEventRepository) Create(_ context.Context, e review.Event) (review.Event, error) {
	e.ID = fmt.Sprintf("event-%d", len(f.events)+1)
	f.events = append(f.events, e)
	return e, nil
}

func (f *fakeReviewEventRepository) DailyCounts(context.Context, string, string, time.Time, *time.Location) ([]repositories.DailyReviewCount, error) {
	panic("not implemented")
}
func (f *fakeReviewEventRepository) CountStudiedCards(context.Context, string, string, time.Time) (int, error) {
	panic("not implemented")
}

var fixedNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

// fakeUserRepository knows only the users seeded into it, together with
// their daily card goal (nil = none).
// fakeUserRepository doubles as the ProfileSource: the daily goal seeded for a
// user (and, optionally, custom rules) make up their "active" configuration.
type fakeUserRepository struct {
	limits       map[string]*int
	rules        map[string]study.Rules          // custom review rules per user (default rules if absent)
	continueOn   map[string]string               // the day a user chose to study past the goal
	deckProfiles map[string]studyprofile.Profile // decks that have a configuration of their own
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		limits:       map[string]*int{},
		rules:        map[string]study.Rules{},
		continueOn:   map[string]string{},
		deckProfiles: map[string]studyprofile.Profile{},
	}
}

// seedDeckProfile gives a deck a configuration of its own.
func (f *fakeUserRepository) seedDeckProfile(deckID string, limit *int, rules study.Rules) {
	p := studyprofile.Default()
	p.ID = "own-" + deckID
	p.Name = "Own " + deckID
	p.DailyCardLimit = limit
	p.Rules = rules
	f.deckProfiles[deckID] = p
}

func (f *fakeUserRepository) ForDeck(ctx context.Context, userID, deckID string) (studyprofile.Profile, bool, error) {
	if p, ok := f.deckProfiles[deckID]; ok {
		return p, true, nil
	}
	general, err := f.Active(ctx, userID)
	return general, false, err
}

func (f *fakeUserRepository) OwnDeckIDs(context.Context, string) ([]string, error) {
	var ids []string
	for id := range f.deckProfiles {
		ids = append(ids, id)
	}
	return ids, nil
}

func (f *fakeUserRepository) Active(_ context.Context, userID string) (studyprofile.Profile, error) {
	profile := studyprofile.Default()
	profile.DailyCardLimit = f.limits[userID]
	if rules, ok := f.rules[userID]; ok {
		profile.Rules = rules
	}
	return profile, nil
}

func (f *fakeUserRepository) seed(id string, dailyLimit *int) { f.limits[id] = dailyLimit }

func (f *fakeUserRepository) FindByID(_ context.Context, id string) (user.User, error) {
	limit, ok := f.limits[id]
	if !ok {
		return user.User{}, repositories.ErrNotFound
	}
	_ = limit
	return user.User{ID: id, ContinuePastGoalOn: f.continueOn[id]}, nil
}
func (f *fakeUserRepository) SetContinuePastGoalOn(_ context.Context, id, day string, _ time.Time) error {
	if _, ok := f.limits[id]; !ok {
		return repositories.ErrNotFound
	}
	f.continueOn[id] = day
	return nil
}
func (f *fakeUserRepository) SetPreferredLanguage(context.Context, string, string, time.Time) error {
	panic("not implemented")
}
func (f *fakeUserRepository) FindByEmail(context.Context, string) (user.User, error) {
	panic("not implemented")
}
func (f *fakeUserRepository) Create(context.Context, user.User) (user.User, error) {
	panic("not implemented")
}
func (f *fakeUserRepository) TouchLastAccess(context.Context, string, time.Time) error {
	panic("not implemented")
}
func (f *fakeUserRepository) SetActiveStudyProfile(context.Context, string, string, time.Time) error {
	panic("not implemented")
}
func (f *fakeUserRepository) ClearLegacyDailyCardLimit(context.Context, string, time.Time) error {
	panic("not implemented")
}

func newTestService() (Service, *fakeDeckRepository, *fakeFlashcardRepository, *fakeReviewEventRepository) {
	svc, decks, cards, events, _ := newTestServiceWithUsers()
	return svc, decks, cards, events
}

func newTestServiceWithUsers() (Service, *fakeDeckRepository, *fakeFlashcardRepository, *fakeReviewEventRepository, *fakeUserRepository) {
	decks := newFakeDeckRepository()
	cards := newFakeFlashcardRepository()
	events := &fakeReviewEventRepository{}
	users := newFakeUserRepository()
	users.seed("owner", nil)
	svc := New(cards, users, users, decks, events, clock.Fixed{Time: fixedNow})
	return svc, decks, cards, events, users
}

func TestDueCards_RejectsWhenDeckNotOwned(t *testing.T) {
	svc, decks, _, _ := newTestService()
	decks.seed("owner", "deck-1")

	if _, err := svc.DueCards(context.Background(), "someone-else", "deck-1", 10, DueOptions{}); err == nil {
		t.Fatal("DueCards() should reject a deck not owned by the caller")
	}
}

func TestDueCards_NewCardIsImmediatelyDue(t *testing.T) {
	svc, decks, cards, _ := newTestService()
	decks.seed("owner", "deck-1")
	cards.seed(flashcard.New("owner", "deck-1", "Q", "A", nil, fixedNow))

	due, err := svc.DueCards(context.Background(), "owner", "deck-1", 10, DueOptions{})
	if err != nil {
		t.Fatalf("DueCards() error: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("DueCards() returned %d cards, want 1", len(due))
	}
}

func TestDueCards_ExcludesArchived(t *testing.T) {
	svc, decks, cards, _ := newTestService()
	decks.seed("owner", "deck-1")
	card := cards.seed(flashcard.New("owner", "deck-1", "Q", "A", nil, fixedNow))
	archivedAt := fixedNow
	card.ArchivedAt = &archivedAt
	cards.cards[card.ID] = card

	due, err := svc.DueCards(context.Background(), "owner", "deck-1", 10, DueOptions{})
	if err != nil {
		t.Fatalf("DueCards() error: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("DueCards() returned %d cards, want 0", len(due))
	}
}

func TestDueCards_ExcludesNotYetDue(t *testing.T) {
	svc, decks, cards, _ := newTestService()
	decks.seed("owner", "deck-1")
	card := cards.seed(flashcard.New("owner", "deck-1", "Q", "A", nil, fixedNow))
	card.Scheduling.DueAt = fixedNow.Add(24 * time.Hour)
	cards.cards[card.ID] = card

	due, err := svc.DueCards(context.Background(), "owner", "deck-1", 10, DueOptions{})
	if err != nil {
		t.Fatalf("DueCards() error: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("DueCards() returned %d cards, want 0", len(due))
	}
}

func TestDueCards_DefaultAndMaxLimit(t *testing.T) {
	svc, decks, _, _ := newTestService()
	decks.seed("owner", "deck-1")

	if _, err := svc.DueCards(context.Background(), "owner", "deck-1", 0, DueOptions{}); err != nil {
		t.Fatalf("DueCards() with default limit error: %v", err)
	}
	if _, err := svc.DueCards(context.Background(), "owner", "deck-1", 1000, DueOptions{}); err != nil {
		t.Fatalf("DueCards() with over-max limit error: %v", err)
	}
}

func TestRecordReview_UpdatesSchedulingAndRecordsEvent(t *testing.T) {
	svc, decks, cards, events := newTestService()
	decks.seed("owner", "deck-1")
	card := cards.seed(flashcard.New("owner", "deck-1", "Q", "A", nil, fixedNow))

	updated, err := svc.RecordReview(context.Background(), "owner", card.ID, study.RatingGood, 1500, false)
	if err != nil {
		t.Fatalf("RecordReview() error: %v", err)
	}
	if updated.Scheduling.State != study.StateReview {
		t.Errorf("Scheduling.State = %q, want %q", updated.Scheduling.State, study.StateReview)
	}
	if updated.Scheduling.IntervalDays != 3 {
		t.Errorf("Scheduling.IntervalDays = %d, want 3", updated.Scheduling.IntervalDays)
	}
	if updated.Scheduling.LastReviewedAt == nil || !updated.Scheduling.LastReviewedAt.Equal(fixedNow) {
		t.Errorf("Scheduling.LastReviewedAt = %v, want %v", updated.Scheduling.LastReviewedAt, fixedNow)
	}

	if len(events.events) != 1 {
		t.Fatalf("recorded %d events, want 1", len(events.events))
	}
	e := events.events[0]
	if e.PreviousState != study.StateNew || e.NextState != study.StateReview {
		t.Errorf("event states = %s -> %s, want new -> review", e.PreviousState, e.NextState)
	}
	if e.ResponseTimeMs != 1500 {
		t.Errorf("event.ResponseTimeMs = %d, want 1500", e.ResponseTimeMs)
	}
}

func TestRecordReview_WrongOwnerIsNotFound(t *testing.T) {
	svc, decks, cards, _ := newTestService()
	decks.seed("owner", "deck-1")
	card := cards.seed(flashcard.New("owner", "deck-1", "Q", "A", nil, fixedNow))

	if _, err := svc.RecordReview(context.Background(), "someone-else", card.ID, study.RatingGood, 0, false); err == nil {
		t.Fatal("RecordReview() should reject a flashcard owned by someone else")
	}
}

func TestRecordReview_RejectsArchivedCard(t *testing.T) {
	svc, decks, cards, _ := newTestService()
	decks.seed("owner", "deck-1")
	card := cards.seed(flashcard.New("owner", "deck-1", "Q", "A", nil, fixedNow))
	archivedAt := fixedNow
	card.ArchivedAt = &archivedAt
	cards.cards[card.ID] = card

	if _, err := svc.RecordReview(context.Background(), "owner", card.ID, study.RatingGood, 0, false); err == nil {
		t.Fatal("RecordReview() should reject an archived flashcard")
	}
}

func TestRecordReview_AgainIncrementsLapseOnGraduatedCard(t *testing.T) {
	svc, decks, cards, _ := newTestService()
	decks.seed("owner", "deck-1")
	card := cards.seed(flashcard.New("owner", "deck-1", "Q", "A", nil, fixedNow))
	card.Scheduling.State = study.StateReview
	card.Scheduling.IntervalDays = 10
	cards.cards[card.ID] = card

	updated, err := svc.RecordReview(context.Background(), "owner", card.ID, study.RatingAgain, 0, false)
	if err != nil {
		t.Fatalf("RecordReview() error: %v", err)
	}
	if updated.Scheduling.State != study.StateLearning {
		t.Errorf("Scheduling.State = %q, want %q", updated.Scheduling.State, study.StateLearning)
	}
	if updated.Scheduling.Lapses != 1 {
		t.Errorf("Scheduling.Lapses = %d, want 1", updated.Scheduling.Lapses)
	}
}

func TestRecordReview_RecordsHintUsageWithoutAffectingScheduling(t *testing.T) {
	svc, decks, cards, events := newTestService()
	decks.seed("owner", "deck-1")
	withHint := cards.seed(flashcard.New("owner", "deck-1", "Q1", "A", nil, fixedNow))
	withoutHint := cards.seed(flashcard.New("owner", "deck-1", "Q2", "A", nil, fixedNow))

	usedHint, err := svc.RecordReview(context.Background(), "owner", withHint.ID, study.RatingGood, 0, true)
	if err != nil {
		t.Fatalf("RecordReview() error: %v", err)
	}
	notUsed, err := svc.RecordReview(context.Background(), "owner", withoutHint.ID, study.RatingGood, 0, false)
	if err != nil {
		t.Fatalf("RecordReview() error: %v", err)
	}

	if len(events.events) != 2 || !events.events[0].HintUsed || events.events[1].HintUsed {
		t.Fatalf("events = %+v, want hintUsed true then false", events.events)
	}
	if usedHint.Scheduling.IntervalDays != notUsed.Scheduling.IntervalDays || usedHint.Scheduling.State != notUsed.Scheduling.State {
		t.Errorf("hint usage must not change scheduling: %+v vs %+v", usedHint.Scheduling, notUsed.Scheduling)
	}
}

// --- daily card goal ---

func intPtr(n int) *int { return &n }

// studiedCard seeds a card of "owner" in deckID. lastReviewed nil means it
// was never reviewed; due is when it comes due.
func studiedCard(cards *fakeFlashcardRepository, deckID string, due time.Time, lastReviewed *time.Time) flashcard.Flashcard {
	c := flashcard.New("owner", deckID, "Q", "A", nil, fixedNow)
	c.Scheduling.DueAt = due
	c.Scheduling.LastReviewedAt = lastReviewed
	return cards.seed(c)
}

func TestDueCards_WithoutADailyGoalServesEverythingDue(t *testing.T) {
	svc, decks, cards, _ := newTestService()
	decks.seed("owner", "deck-1")
	reviewedToday := fixedNow.Add(-time.Hour)
	for i := 0; i < 6; i++ {
		studiedCard(cards, "deck-1", fixedNow, &reviewedToday)
	}

	due, err := svc.DueCards(context.Background(), "owner", "deck-1", 100, DueOptions{})
	if err != nil || len(due) != 6 {
		t.Fatalf("DueCards() = %d cards, %v; want all 6 (no goal, default behavior)", len(due), err)
	}

	p, err := svc.Progress(context.Background(), "owner", "", nil)
	if err != nil || p.DailyCardLimit != nil || p.Remaining != nil || p.StudiedToday != 6 {
		t.Errorf("Progress() = %+v, %v; want no goal, no remaining, 6 studied", p, err)
	}
}

func TestDueCards_DailyGoalCapsTheNewCardsServed(t *testing.T) {
	svc, decks, cards, _, users := newTestServiceWithUsers()
	decks.seed("owner", "deck-1")
	users.seed("owner", intPtr(3))
	for i := 0; i < 10; i++ {
		studiedCard(cards, "deck-1", fixedNow, nil)
	}

	due, err := svc.DueCards(context.Background(), "owner", "deck-1", 100, DueOptions{})
	if err != nil || len(due) != 3 {
		t.Fatalf("DueCards() = %d cards, %v; want 3 (the goal)", len(due), err)
	}
}

func TestDueCards_DailyGoalCountsCardsStudiedTodayInAnyDeck(t *testing.T) {
	svc, decks, cards, _, users := newTestServiceWithUsers()
	decks.seed("owner", "deck-1")
	decks.seed("owner", "deck-2")
	users.seed("owner", intPtr(5))
	reviewedToday := fixedNow.Add(-2 * time.Hour)
	farFuture := fixedNow.Add(48 * time.Hour)
	// Three cards of ANOTHER deck were studied today (and are not due again).
	for i := 0; i < 3; i++ {
		studiedCard(cards, "deck-2", farFuture, &reviewedToday)
	}
	for i := 0; i < 10; i++ {
		studiedCard(cards, "deck-1", fixedNow, nil)
	}

	due, err := svc.DueCards(context.Background(), "owner", "deck-1", 100, DueOptions{})
	if err != nil || len(due) != 2 {
		t.Fatalf("DueCards() = %d cards, %v; want 2 (goal 5 minus 3 studied elsewhere)", len(due), err)
	}
	p, _ := svc.Progress(context.Background(), "owner", "", nil)
	if p.StudiedToday != 3 || p.Remaining == nil || *p.Remaining != 2 {
		t.Errorf("Progress() = %+v, want 3 studied and 2 remaining", p)
	}
}

func TestDueCards_CardsAnsweredAgainStillComeBackAfterTheGoal(t *testing.T) {
	svc, decks, cards, _, users := newTestServiceWithUsers()
	decks.seed("owner", "deck-1")
	users.seed("owner", intPtr(2))
	reviewedToday := fixedNow.Add(-time.Hour)
	farFuture := fixedNow.Add(72 * time.Hour)
	// The goal (2) is used up: one card done for good, one answered "again"
	// and due again now.
	studiedCard(cards, "deck-1", farFuture, &reviewedToday)
	again := studiedCard(cards, "deck-1", fixedNow.Add(-time.Minute), &reviewedToday)
	// Plenty of untouched cards that the goal must keep out.
	for i := 0; i < 5; i++ {
		studiedCard(cards, "deck-1", fixedNow, nil)
	}

	due, err := svc.DueCards(context.Background(), "owner", "deck-1", 100, DueOptions{})
	if err != nil || len(due) != 1 || due[0].ID != again.ID {
		t.Fatalf("DueCards() = %+v, %v; want only the card answered again", due, err)
	}
	p, _ := svc.Progress(context.Background(), "owner", "", nil)
	if p.Remaining == nil || *p.Remaining != 0 {
		t.Errorf("Progress().Remaining = %v, want 0", p.Remaining)
	}
}

func TestDueCards_StudyMoreAnywayIgnoresTheGoal(t *testing.T) {
	svc, decks, cards, _, users := newTestServiceWithUsers()
	decks.seed("owner", "deck-1")
	users.seed("owner", intPtr(1))
	for i := 0; i < 4; i++ {
		studiedCard(cards, "deck-1", fixedNow, nil)
	}

	due, err := svc.DueCards(context.Background(), "owner", "deck-1", 100, DueOptions{IgnoreDailyLimit: true})
	if err != nil || len(due) != 4 {
		t.Fatalf("DueCards(ignore) = %d cards, %v; want all 4", len(due), err)
	}
}

func TestDueCards_ServedCardsAreOrderedByDueDateAndCappedByTheRequestLimit(t *testing.T) {
	svc, decks, cards, _, users := newTestServiceWithUsers()
	decks.seed("owner", "deck-1")
	users.seed("owner", intPtr(50))
	reviewedToday := fixedNow.Add(-time.Hour)
	fresh := studiedCard(cards, "deck-1", fixedNow.Add(-3*time.Hour), nil)
	again := studiedCard(cards, "deck-1", fixedNow.Add(-2*time.Hour), &reviewedToday)
	late := studiedCard(cards, "deck-1", fixedNow.Add(-time.Hour), nil)

	due, _ := svc.DueCards(context.Background(), "owner", "deck-1", 100, DueOptions{})
	if len(due) != 3 || due[0].ID != fresh.ID || due[1].ID != again.ID || due[2].ID != late.ID {
		t.Fatalf("DueCards() order = %+v, want fresh, again, late by due date", due)
	}
	capped, _ := svc.DueCards(context.Background(), "owner", "deck-1", 2, DueOptions{})
	if len(capped) != 2 || capped[0].ID != fresh.ID {
		t.Errorf("DueCards(limit 2) = %+v, want the two earliest", capped)
	}
}

func TestProgress_TodayFollowsTheCallersTimeZone(t *testing.T) {
	svc, _, cards, _, users := newTestServiceWithUsers()
	users.seed("owner", intPtr(10))
	manaus, err := time.LoadLocation("America/Manaus") // UTC-4
	if err != nil {
		t.Fatalf("LoadLocation() error: %v", err)
	}
	// fixedNow is 2026-01-01 12:00 UTC = 08:00 in Manaus. A review at
	// 02:00 UTC is still Dec 31 there (22:00), but already Jan 1 in UTC.
	late := time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC)
	studiedCard(cards, "deck-1", fixedNow.Add(48*time.Hour), &late)

	utc, _ := svc.Progress(context.Background(), "owner", "", nil)
	local, _ := svc.Progress(context.Background(), "owner", "", manaus)
	if utc.StudiedToday != 1 || local.StudiedToday != 0 {
		t.Errorf("studied today = %d in UTC and %d in Manaus; want 1 and 0", utc.StudiedToday, local.StudiedToday)
	}
}

func TestProgress_RemainingNeverGoesNegative(t *testing.T) {
	svc, _, cards, _, users := newTestServiceWithUsers()
	users.seed("owner", intPtr(2))
	reviewedToday := fixedNow.Add(-time.Hour)
	for i := 0; i < 5; i++ {
		studiedCard(cards, "deck-1", fixedNow.Add(48*time.Hour), &reviewedToday)
	}

	p, _ := svc.Progress(context.Background(), "owner", "", nil)
	if p.StudiedToday != 5 || p.Remaining == nil || *p.Remaining != 0 {
		t.Errorf("Progress() = %+v, want 5 studied and 0 remaining (not -3)", p)
	}
}

// --- continuing past the goal (remembered on the user) ---

func TestContinuePastGoal_LiftsTheGoalForTheRestOfTheDay(t *testing.T) {
	svc, decks, cards, _, users := newTestServiceWithUsers()
	decks.seed("owner", "deck-1")
	users.seed("owner", intPtr(2))
	for i := 0; i < 6; i++ {
		studiedCard(cards, "deck-1", fixedNow, nil)
	}

	before, _ := svc.DueCards(context.Background(), "owner", "deck-1", 100, DueOptions{})
	if len(before) != 2 {
		t.Fatalf("before choosing to continue: %d cards, want 2 (the goal)", len(before))
	}
	p, _ := svc.Progress(context.Background(), "owner", "", nil)
	if p.ContinuingPastGoal {
		t.Fatal("Progress().ContinuingPastGoal should start false")
	}

	if err := svc.ContinuePastGoal(context.Background(), "owner", "", nil); err != nil {
		t.Fatalf("ContinuePastGoal() error: %v", err)
	}

	after, _ := svc.DueCards(context.Background(), "owner", "deck-1", 100, DueOptions{})
	if len(after) != 6 {
		t.Errorf("after choosing to continue: %d cards, want all 6", len(after))
	}
	p, _ = svc.Progress(context.Background(), "owner", "", nil)
	if !p.ContinuingPastGoal {
		t.Error("Progress().ContinuingPastGoal should be true after the choice")
	}
	// The goal itself still counts and is reported.
	if p.DailyCardLimit == nil || *p.DailyCardLimit != 2 {
		t.Errorf("Progress().DailyCardLimit = %v, want 2 (the goal is unchanged)", p.DailyCardLimit)
	}
}

func TestContinuePastGoal_LapsesWhenTheDayEnds(t *testing.T) {
	svc, decks, cards, _, users := newTestServiceWithUsers()
	decks.seed("owner", "deck-1")
	users.seed("owner", intPtr(1))
	users.continueOn["owner"] = "2025-12-31" // chosen yesterday (fixedNow is 2026-01-01)
	for i := 0; i < 4; i++ {
		studiedCard(cards, "deck-1", fixedNow, nil)
	}

	p, _ := svc.Progress(context.Background(), "owner", "", nil)
	if p.ContinuingPastGoal {
		t.Error("a choice made yesterday must not apply today")
	}
	due, _ := svc.DueCards(context.Background(), "owner", "deck-1", 100, DueOptions{})
	if len(due) != 1 {
		t.Errorf("DueCards() = %d cards, want 1 (the goal applies again)", len(due))
	}
}

func TestContinuePastGoal_TodayFollowsTheCallersTimeZone(t *testing.T) {
	svc, _, _, _, users := newTestServiceWithUsers()
	users.seed("owner", intPtr(1))
	kiritimati, err := time.LoadLocation("Pacific/Kiritimati") // UTC+14: already Jan 2 when it is Jan 1 12:00 UTC
	if err != nil {
		t.Fatalf("LoadLocation() error: %v", err)
	}

	if err := svc.ContinuePastGoal(context.Background(), "owner", "", kiritimati); err != nil {
		t.Fatalf("ContinuePastGoal() error: %v", err)
	}
	if got := users.continueOn["owner"]; got != "2026-01-02" {
		t.Errorf("stored day = %q, want 2026-01-02 (the caller's local day)", got)
	}
	inKiritimati, _ := svc.Progress(context.Background(), "owner", "", kiritimati)
	inUTC, _ := svc.Progress(context.Background(), "owner", "", nil)
	if !inKiritimati.ContinuingPastGoal || inUTC.ContinuingPastGoal {
		t.Errorf("continuing = %v in Kiritimati and %v in UTC; want true and false", inKiritimati.ContinuingPastGoal, inUTC.ContinuingPastGoal)
	}
}

func TestContinuePastGoal_UnknownUserIsNotFound(t *testing.T) {
	svc, _, _, _ := newTestService()
	if err := svc.ContinuePastGoal(context.Background(), "nobody", "", nil); err == nil {
		t.Fatal("ContinuePastGoal() should return an error for an unknown user")
	}
}

// --- the review rules come from the user's study configuration ---

func TestRecordReview_UsesTheActiveConfigurationsRules(t *testing.T) {
	svc, decks, cards, events, users := newTestServiceWithUsers()
	decks.seed("owner", "deck-1")
	// A configuration where "Muito difícil" returns in 3 minutes and "Fácil"
	// schedules a new card in 5 days.
	rules := study.DefaultRules()
	rules.AgainDelayMinutes = 3
	rules.Good = study.RatingRule{FirstIntervalDays: 5, Multiplier: 2}
	rules.Easy = study.RatingRule{FirstIntervalDays: 9, Multiplier: 3}
	users.rules["owner"] = rules

	good := cards.seed(flashcard.New("owner", "deck-1", "Q1", "A", nil, fixedNow))
	updated, err := svc.RecordReview(context.Background(), "owner", good.ID, study.RatingGood, 0, false)
	if err != nil || updated.Scheduling.IntervalDays != 5 {
		t.Fatalf("RecordReview(good) = %+v, %v; want a 5-day interval from the custom rules", updated.Scheduling, err)
	}

	again := cards.seed(flashcard.New("owner", "deck-1", "Q2", "A", nil, fixedNow))
	updated, err = svc.RecordReview(context.Background(), "owner", again.ID, study.RatingAgain, 0, false)
	if err != nil {
		t.Fatalf("RecordReview(again) error: %v", err)
	}
	if got := updated.Scheduling.DueAt.Sub(fixedNow); got != 3*time.Minute {
		t.Errorf("an 'again' card comes back after %v, want 3m0s from the custom rules", got)
	}
	if len(events.events) != 2 {
		t.Errorf("recorded %d events, want 2", len(events.events))
	}
}

func TestRecordReview_WithoutACustomConfigurationUsesTheDefaultRules(t *testing.T) {
	svc, decks, cards, _ := newTestService()
	decks.seed("owner", "deck-1")
	card := cards.seed(flashcard.New("owner", "deck-1", "Q", "A", nil, fixedNow))

	updated, err := svc.RecordReview(context.Background(), "owner", card.ID, study.RatingGood, 0, false)
	if err != nil || updated.Scheduling.IntervalDays != 3 {
		t.Errorf("RecordReview(good) = %+v, %v; want the default 3 days", updated.Scheduling, err)
	}
}

// --- configurations of their own per deck ---

func TestProgress_ADeckWithItsOwnConfigurationHasItsOwnGoalAndCount(t *testing.T) {
	svc, decks, cards, _, users := newTestServiceWithUsers()
	decks.seed("owner", "with-own")
	decks.seed("owner", "general-a")
	decks.seed("owner", "general-b")
	users.seed("owner", intPtr(10))                                    // the general goal
	users.seedDeckProfile("with-own", intPtr(3), study.DefaultRules()) // the deck's own goal
	reviewedToday := fixedNow.Add(-time.Hour)
	later := fixedNow.Add(48 * time.Hour)
	// 2 cards studied today in the deck with its own goal, 1 + 2 in the others.
	for i := 0; i < 2; i++ {
		studiedCard(cards, "with-own", later, &reviewedToday)
	}
	studiedCard(cards, "general-a", later, &reviewedToday)
	for i := 0; i < 2; i++ {
		studiedCard(cards, "general-b", later, &reviewedToday)
	}

	own, err := svc.Progress(context.Background(), "owner", "with-own", nil)
	if err != nil || own.DailyCardLimit == nil || *own.DailyCardLimit != 3 || own.StudiedToday != 2 || *own.Remaining != 1 {
		t.Fatalf("own deck: %+v, %v; want goal 3, 2 studied (its cards only), 1 remaining", own, err)
	}

	for _, deckID := range []string{"general-a", "general-b", ""} {
		general, err := svc.Progress(context.Background(), "owner", deckID, nil)
		if err != nil || *general.DailyCardLimit != 10 || general.StudiedToday != 3 || *general.Remaining != 7 {
			t.Errorf("deck %q: %+v, %v; want the general goal 10 with 3 studied (only the decks without their own configuration)", deckID, general, err)
		}
	}
}

func TestDueCards_EachDeckIsCappedByItsOwnGoal(t *testing.T) {
	svc, decks, cards, _, users := newTestServiceWithUsers()
	decks.seed("owner", "with-own")
	decks.seed("owner", "general")
	users.seed("owner", intPtr(4))
	users.seedDeckProfile("with-own", intPtr(2), study.DefaultRules())
	for i := 0; i < 8; i++ {
		studiedCard(cards, "with-own", fixedNow, nil)
		studiedCard(cards, "general", fixedNow, nil)
	}

	own, _ := svc.DueCards(context.Background(), "owner", "with-own", 100, DueOptions{})
	general, _ := svc.DueCards(context.Background(), "owner", "general", 100, DueOptions{})
	if len(own) != 2 || len(general) != 4 {
		t.Errorf("served %d cards in the deck with its own goal (want 2) and %d in the general one (want 4)", len(own), len(general))
	}
}

func TestDueCards_ADeckWithoutAGoalIsNotLimitedEvenIfTheGeneralHasOne(t *testing.T) {
	svc, decks, cards, _, users := newTestServiceWithUsers()
	decks.seed("owner", "no-goal")
	users.seed("owner", intPtr(1))
	users.seedDeckProfile("no-goal", nil, study.DefaultRules()) // its own configuration has no daily goal
	for i := 0; i < 6; i++ {
		studiedCard(cards, "no-goal", fixedNow, nil)
	}

	due, err := svc.DueCards(context.Background(), "owner", "no-goal", 100, DueOptions{})
	if err != nil || len(due) != 6 {
		t.Errorf("DueCards() = %d cards, %v; want all 6: the deck's own configuration has no goal", len(due), err)
	}
}

func TestRecordReview_UsesTheRulesOfTheCardsDeck(t *testing.T) {
	svc, decks, cards, _, users := newTestServiceWithUsers()
	decks.seed("owner", "with-own")
	decks.seed("owner", "general")
	slow := study.DefaultRules()
	slow.Good = study.RatingRule{FirstIntervalDays: 6, Multiplier: 2}
	slow.Easy = study.RatingRule{FirstIntervalDays: 12, Multiplier: 3}
	users.seedDeckProfile("with-own", nil, slow)

	inOwn := cards.seed(flashcard.New("owner", "with-own", "Q1", "A", nil, fixedNow))
	inGeneral := cards.seed(flashcard.New("owner", "general", "Q2", "A", nil, fixedNow))

	a, err := svc.RecordReview(context.Background(), "owner", inOwn.ID, study.RatingGood, 0, false)
	if err != nil || a.Scheduling.IntervalDays != 6 {
		t.Errorf("card in the deck with its own rules: %+v, %v; want 6 days", a.Scheduling, err)
	}
	b, err := svc.RecordReview(context.Background(), "owner", inGeneral.ID, study.RatingGood, 0, false)
	if err != nil || b.Scheduling.IntervalDays != 3 {
		t.Errorf("card in a deck following the general rules: %+v, %v; want the default 3 days", b.Scheduling, err)
	}
}

func TestContinuePastGoal_BelongsToTheDeckThatHasItsOwnGoal(t *testing.T) {
	svc, decks, cards, _, users := newTestServiceWithUsers()
	decks.seed("owner", "with-own")
	decks.seed("owner", "general")
	users.seed("owner", intPtr(1))
	users.seedDeckProfile("with-own", intPtr(1), study.DefaultRules())
	for i := 0; i < 4; i++ {
		studiedCard(cards, "with-own", fixedNow, nil)
		studiedCard(cards, "general", fixedNow, nil)
	}
	reviewedToday := fixedNow.Add(-time.Hour)
	later := fixedNow.Add(48 * time.Hour)
	studiedCard(cards, "with-own", later, &reviewedToday) // uses up both goals (1 each)
	studiedCard(cards, "general", later, &reviewedToday)

	// Continuing in the deck with its own goal lifts only that deck's cap.
	if err := svc.ContinuePastGoal(context.Background(), "owner", "with-own", nil); err != nil {
		t.Fatalf("ContinuePastGoal() error: %v", err)
	}
	if got := decks.decks["with-own"].ContinuePastGoalOn; got != "2026-01-01" {
		t.Errorf("the deck records %q, want 2026-01-01", got)
	}
	if users.continueOn["owner"] != "" {
		t.Errorf("the general choice was set (%q) by continuing in a deck with its own goal", users.continueOn["owner"])
	}
	ownDue, _ := svc.DueCards(context.Background(), "owner", "with-own", 100, DueOptions{})
	generalDue, _ := svc.DueCards(context.Background(), "owner", "general", 100, DueOptions{})
	if len(ownDue) != 4 || len(generalDue) != 0 {
		t.Errorf("after continuing in one deck: served %d there (want 4) and %d in the general one (want 0, still capped)", len(ownDue), len(generalDue))
	}

	// Continuing in a deck that follows the general goal lifts the general cap only.
	if err := svc.ContinuePastGoal(context.Background(), "owner", "general", nil); err != nil {
		t.Fatalf("ContinuePastGoal(general) error: %v", err)
	}
	if users.continueOn["owner"] != "2026-01-01" {
		t.Errorf("the general choice = %q, want 2026-01-01", users.continueOn["owner"])
	}
	generalDue, _ = svc.DueCards(context.Background(), "owner", "general", 100, DueOptions{})
	if len(generalDue) != 4 {
		t.Errorf("after continuing in the general scope: served %d, want 4", len(generalDue))
	}
}

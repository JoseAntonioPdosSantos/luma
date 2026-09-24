package deckservice

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"flashcard-backend/internal/domain/deck"
	"flashcard-backend/internal/domain/flashcard"
	"flashcard-backend/internal/domain/review"
	"flashcard-backend/internal/domain/study"
	"flashcard-backend/internal/ports/clock"
	"flashcard-backend/internal/ports/repositories"
)

type fakeDeckRepository struct {
	decks  map[string]deck.Deck
	nextID int
}

func newFakeDeckRepository() *fakeDeckRepository {
	return &fakeDeckRepository{decks: map[string]deck.Deck{}}
}

func (f *fakeDeckRepository) Create(_ context.Context, d deck.Deck) (deck.Deck, error) {
	f.nextID++
	d.ID = fmt.Sprintf("deck-%d", f.nextID)
	f.decks[d.ID] = d
	return d, nil
}

func (f *fakeDeckRepository) FindByID(_ context.Context, userID, id string) (deck.Deck, error) {
	d, ok := f.decks[id]
	if !ok || d.UserID != userID {
		return deck.Deck{}, repositories.ErrNotFound
	}
	return d, nil
}

func (f *fakeDeckRepository) ListActive(_ context.Context, userID string) ([]deck.Deck, error) {
	var result []deck.Deck
	for _, d := range f.decks {
		if d.UserID == userID && !d.IsArchived() {
			result = append(result, d)
		}
	}
	return result, nil
}

func (f *fakeDeckRepository) ListArchived(_ context.Context, userID string) ([]deck.Deck, error) {
	var result []deck.Deck
	for _, d := range f.decks {
		if d.UserID == userID && d.IsArchived() {
			result = append(result, d)
		}
	}
	return result, nil
}

func (f *fakeDeckRepository) Restore(_ context.Context, userID, id string, _ time.Time) error {
	existing, ok := f.decks[id]
	if !ok || existing.UserID != userID {
		return repositories.ErrNotFound
	}
	existing.ArchivedAt = nil
	f.decks[id] = existing
	return nil
}

func (f *fakeDeckRepository) SetStudyProfile(context.Context, string, string, string, time.Time) error {
	panic("not implemented")
}
func (f *fakeDeckRepository) SetContinuePastGoalOn(context.Context, string, string, string, time.Time) error {
	panic("not implemented")
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
func (f *fakeDeckRepository) DeleteArchived(_ context.Context, userID, id string) error {
	existing, ok := f.decks[id]
	if !ok || existing.UserID != userID || !existing.IsArchived() {
		return repositories.ErrNotFound
	}
	delete(f.decks, id)
	return nil
}

func (f *fakeDeckRepository) Update(_ context.Context, d deck.Deck) (deck.Deck, error) {
	existing, ok := f.decks[d.ID]
	if !ok || existing.UserID != d.UserID {
		return deck.Deck{}, repositories.ErrNotFound
	}
	f.decks[d.ID] = d
	return d, nil
}

func (f *fakeDeckRepository) Archive(_ context.Context, userID, id string, now time.Time) error {
	existing, ok := f.decks[id]
	if !ok || existing.UserID != userID {
		return repositories.ErrNotFound
	}
	existing.ArchivedAt = &now
	f.decks[id] = existing
	return nil
}

// fakeFlashcardRepository implements only the part of
// repositories.FlashcardRepository that deckservice actually calls
// (CountByDeck, CountsByState); every other method is unreachable from
// these tests.
type fakeFlashcardRepository struct {
	countsByState map[study.CardState]int
	nextDue       *time.Time
}

func (fakeFlashcardRepository) Create(context.Context, flashcard.Flashcard) (flashcard.Flashcard, error) {
	panic("not implemented")
}
func (fakeFlashcardRepository) FindByID(context.Context, string, string) (flashcard.Flashcard, error) {
	panic("not implemented")
}
func (fakeFlashcardRepository) ListByDeck(context.Context, string, string) ([]flashcard.Flashcard, error) {
	panic("not implemented")
}
func (f fakeFlashcardRepository) NextDueAt(context.Context, string, string, time.Time) (*time.Time, error) {
	return f.nextDue, nil
}
func (fakeFlashcardRepository) CountReviewedSince(context.Context, string, time.Time, repositories.ReviewScope) (int, error) {
	panic("not implemented")
}
func (fakeFlashcardRepository) ListDueByStudiedToday(context.Context, string, string, time.Time, time.Time, bool, int) ([]flashcard.Flashcard, error) {
	panic("not implemented")
}
func (fakeFlashcardRepository) SearchByDeck(context.Context, string, string, repositories.FlashcardListOptions) ([]flashcard.Flashcard, int, error) {
	panic("not implemented")
}
func (fakeFlashcardRepository) ListArchivedByDeck(context.Context, string, string) ([]flashcard.Flashcard, error) {
	panic("not implemented")
}
func (fakeFlashcardRepository) Restore(context.Context, string, string, time.Time) error {
	panic("not implemented")
}
func (fakeFlashcardRepository) ListArchivedByUser(context.Context, string) ([]flashcard.Flashcard, error) {
	panic("not implemented")
}
func (fakeFlashcardRepository) DeleteArchived(context.Context, string, string) (flashcard.Flashcard, error) {
	panic("not implemented")
}
func (fakeFlashcardRepository) DeleteAllByDeck(context.Context, string, string) error {
	panic("not implemented")
}
func (fakeFlashcardRepository) ListDue(context.Context, string, string, time.Time, int) ([]flashcard.Flashcard, error) {
	panic("not implemented")
}
func (fakeFlashcardRepository) Update(context.Context, flashcard.Flashcard) (flashcard.Flashcard, error) {
	panic("not implemented")
}
func (fakeFlashcardRepository) Archive(context.Context, string, string, time.Time) error {
	panic("not implemented")
}
func (fakeFlashcardRepository) CountByDeck(context.Context, string, string, time.Time) (int, int, error) {
	return 2, 1, nil
}
func (f fakeFlashcardRepository) CountsByState(context.Context, string, string) (map[study.CardState]int, error) {
	return f.countsByState, nil
}

// fakeReviewEventRepository implements only DailyCounts, configurable per
// test; Create is unreachable from deckservice.
type fakeReviewEventRepository struct {
	dailyCounts []repositories.DailyReviewCount
	// studied answers CountStudiedCards for a given "since"; nil means 0.
	studied func(since time.Time) int
}

func (fakeReviewEventRepository) DeleteByDeck(context.Context, string, string) error {
	panic("not implemented")
}
func (fakeReviewEventRepository) Create(context.Context, review.Event) (review.Event, error) {
	panic("not implemented")
}
func (f fakeReviewEventRepository) DailyCounts(context.Context, string, string, time.Time, *time.Location) ([]repositories.DailyReviewCount, error) {
	return f.dailyCounts, nil
}
func (f fakeReviewEventRepository) CountStudiedCards(_ context.Context, _, _ string, since time.Time) (int, error) {
	if f.studied == nil {
		return 0, nil
	}
	return f.studied(since), nil
}

var fixedNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func newTestService() (Service, *fakeDeckRepository) {
	decks := newFakeDeckRepository()
	svc := New(decks, fakeFlashcardRepository{}, fakeReviewEventRepository{}, nil, clock.Fixed{Time: fixedNow})
	return svc, decks
}

func TestCreate_ValidatesAndPersists(t *testing.T) {
	svc, _ := newTestService()

	d, err := svc.Create(context.Background(), "user-1", "  English  ", "Learn English")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if d.Name != "English" {
		t.Errorf("Name = %q, want %q", d.Name, "English")
	}
}

func TestCreate_RejectsEmptyName(t *testing.T) {
	svc, _ := newTestService()

	if _, err := svc.Create(context.Background(), "user-1", "   ", ""); err == nil {
		t.Fatal("Create() should reject an empty name")
	}
}

func TestGet_WrongOwnerIsNotFound(t *testing.T) {
	svc, _ := newTestService()

	created, err := svc.Create(context.Background(), "owner", "Deck", "")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if _, err := svc.Get(context.Background(), "someone-else", created.ID); err == nil {
		t.Fatal("Get() should return an error for a deck owned by someone else")
	}
}

func TestGetWithCounts_IncludesCardCounts(t *testing.T) {
	svc, _ := newTestService()

	created, err := svc.Create(context.Background(), "user-1", "Deck", "")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	withCounts, err := svc.GetWithCounts(context.Background(), "user-1", created.ID)
	if err != nil {
		t.Fatalf("GetWithCounts() error: %v", err)
	}
	if withCounts.TotalCards != 2 || withCounts.DueCards != 1 {
		t.Errorf("counts = (%d, %d), want (2, 1)", withCounts.TotalCards, withCounts.DueCards)
	}
}

func TestArchive_RemovesFromActiveList(t *testing.T) {
	svc, _ := newTestService()

	created, err := svc.Create(context.Background(), "user-1", "Deck", "")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.Archive(context.Background(), "user-1", created.ID); err != nil {
		t.Fatalf("Archive() error: %v", err)
	}

	decks, err := svc.ListActiveWithCounts(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListActiveWithCounts() error: %v", err)
	}
	if len(decks) != 0 {
		t.Errorf("ListActiveWithCounts() returned %d decks after archive, want 0", len(decks))
	}
}

func TestArchive_WrongOwnerIsNotFound(t *testing.T) {
	svc, _ := newTestService()

	created, err := svc.Create(context.Background(), "owner", "Deck", "")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.Archive(context.Background(), "someone-else", created.ID); err == nil {
		t.Fatal("Archive() should return an error for a deck owned by someone else")
	}
}

func TestGetStats_ComputesMasteryAndPerformance(t *testing.T) {
	decks := newFakeDeckRepository()
	flashcards := fakeFlashcardRepository{countsByState: map[study.CardState]int{
		study.StateNew:      2,
		study.StateLearning: 1,
		study.StateReview:   3,
		study.StateMature:   4,
	}}
	day := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	reviews := fakeReviewEventRepository{dailyCounts: []repositories.DailyReviewCount{
		{Date: day, ReviewCount: 10, CorrectCount: 8, HintCount: 3},
	}}
	svc := New(decks, flashcards, reviews, nil, clock.Fixed{Time: fixedNow})

	created, err := svc.Create(context.Background(), "owner", "English", "")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	stats, err := svc.GetStats(context.Background(), "owner", created.ID, nil)
	if err != nil {
		t.Fatalf("GetStats() error: %v", err)
	}

	if stats.TotalCards != 10 {
		t.Errorf("TotalCards = %d, want 10", stats.TotalCards)
	}
	if stats.NewCards != 2 {
		t.Errorf("NewCards = %d, want 2", stats.NewCards)
	}
	if stats.LearningCards != 4 {
		t.Errorf("LearningCards = %d, want 4 (learning + review)", stats.LearningCards)
	}
	if stats.MasteredCards != 4 {
		t.Errorf("MasteredCards = %d, want 4", stats.MasteredCards)
	}
	if stats.MasteryPercent != 40 {
		t.Errorf("MasteryPercent = %d, want 40", stats.MasteryPercent)
	}
	if len(stats.DailyPerformance) != 1 {
		t.Fatalf("DailyPerformance = %+v, want 1 entry", stats.DailyPerformance)
	}
	if stats.DailyPerformance[0].AccuracyPercent != 80 {
		t.Errorf("AccuracyPercent = %d, want 80", stats.DailyPerformance[0].AccuracyPercent)
	}
	if stats.DailyPerformance[0].HintCount != 3 {
		t.Errorf("HintCount = %d, want 3", stats.DailyPerformance[0].HintCount)
	}
}

func TestGetStats_ZeroCardsHasZeroPercentMastery(t *testing.T) {
	decks := newFakeDeckRepository()
	flashcards := fakeFlashcardRepository{countsByState: map[study.CardState]int{}}
	reviews := fakeReviewEventRepository{}
	svc := New(decks, flashcards, reviews, nil, clock.Fixed{Time: fixedNow})

	created, err := svc.Create(context.Background(), "owner", "Empty deck", "")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	stats, err := svc.GetStats(context.Background(), "owner", created.ID, nil)
	if err != nil {
		t.Fatalf("GetStats() error: %v", err)
	}
	if stats.MasteryPercent != 0 {
		t.Errorf("MasteryPercent = %d, want 0", stats.MasteryPercent)
	}
}

func TestGetStats_WrongOwnerIsNotFound(t *testing.T) {
	svc, _ := newTestService()

	created, err := svc.Create(context.Background(), "owner", "Deck", "")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if _, err := svc.GetStats(context.Background(), "someone-else", created.ID, nil); err == nil {
		t.Fatal("GetStats() should reject a deck owned by someone else")
	}
}

func TestRestore_ReturnsArchivedDeckToActiveList(t *testing.T) {
	svc, _ := newTestService()
	created, err := svc.Create(context.Background(), "user-1", "English", "")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := svc.Archive(context.Background(), "user-1", created.ID); err != nil {
		t.Fatalf("Archive() error: %v", err)
	}

	archived, err := svc.ListArchivedWithCounts(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListArchivedWithCounts() error: %v", err)
	}
	if len(archived) != 1 || archived[0].Deck.ID != created.ID {
		t.Fatalf("ListArchivedWithCounts() = %+v, want the archived deck", archived)
	}

	if err := svc.Restore(context.Background(), "user-1", created.ID); err != nil {
		t.Fatalf("Restore() error: %v", err)
	}

	active, _ := svc.ListActiveWithCounts(context.Background(), "user-1")
	if len(active) != 1 {
		t.Errorf("ListActiveWithCounts() returned %d decks after restore, want 1", len(active))
	}
	archived, _ = svc.ListArchivedWithCounts(context.Background(), "user-1")
	if len(archived) != 0 {
		t.Errorf("ListArchivedWithCounts() returned %d decks after restore, want 0", len(archived))
	}
}

func TestRestore_WrongOwnerIsNotFound(t *testing.T) {
	svc, _ := newTestService()
	created, _ := svc.Create(context.Background(), "user-1", "English", "")
	_ = svc.Archive(context.Background(), "user-1", created.ID)

	if err := svc.Restore(context.Background(), "someone-else", created.ID); err == nil {
		t.Fatal("Restore() should return an error for a deck owned by someone else")
	}
}

// deletionFlashcardRepository lets the permanent-deletion tests control
// which cards a deck has and observe the calls made.
type deletionFlashcardRepository struct {
	fakeFlashcardRepository
	active, archived []flashcard.Flashcard
	deletedAllOf     string
}

func (f *deletionFlashcardRepository) ListByDeck(context.Context, string, string) ([]flashcard.Flashcard, error) {
	return f.active, nil
}
func (f *deletionFlashcardRepository) ListArchivedByDeck(context.Context, string, string) ([]flashcard.Flashcard, error) {
	return f.archived, nil
}
func (f *deletionFlashcardRepository) DeleteAllByDeck(_ context.Context, _, deckID string) error {
	f.deletedAllOf = deckID
	return nil
}

type deletionReviewRepository struct {
	fakeReviewEventRepository
	deletedDeck string
}

func (f *deletionReviewRepository) DeleteByDeck(_ context.Context, _, deckID string) error {
	f.deletedDeck = deckID
	return nil
}

type recordingAudioStore struct{ deleted []string }

func (r *recordingAudioStore) Save(context.Context, io.Reader, int64) (string, int64, error) {
	panic("not implemented")
}
func (r *recordingAudioStore) Open(context.Context, string) (io.ReadCloser, error) {
	panic("not implemented")
}
func (r *recordingAudioStore) Delete(_ context.Context, fileID string) error {
	r.deleted = append(r.deleted, fileID)
	return nil
}

func TestDeleteArchived_RemovesDeckCardsHistoryAndAudio(t *testing.T) {
	decks := newFakeDeckRepository()
	cards := &deletionFlashcardRepository{
		active:   []flashcard.Flashcard{{ID: "c1", Audio: &flashcard.Audio{GridFSFileID: "audio-1"}}},
		archived: []flashcard.Flashcard{{ID: "c2", Audio: &flashcard.Audio{GridFSFileID: "audio-2"}}, {ID: "c3"}},
	}
	events := &deletionReviewRepository{}
	audio := &recordingAudioStore{}
	svc := New(decks, cards, events, audio, clock.Fixed{Time: fixedNow})

	created, _ := svc.Create(context.Background(), "user-1", "English", "")
	_ = svc.Archive(context.Background(), "user-1", created.ID)

	if err := svc.DeleteArchived(context.Background(), "user-1", created.ID); err != nil {
		t.Fatalf("DeleteArchived() error: %v", err)
	}

	if _, err := svc.Get(context.Background(), "user-1", created.ID); err == nil {
		t.Error("deck should be gone after DeleteArchived()")
	}
	if cards.deletedAllOf != created.ID || events.deletedDeck != created.ID {
		t.Errorf("cards deleted for %q, events deleted for %q; want both %q", cards.deletedAllOf, events.deletedDeck, created.ID)
	}
	if len(audio.deleted) != 2 {
		t.Errorf("deleted audio files = %v, want audio-1 and audio-2", audio.deleted)
	}
}

func TestDeleteArchived_RefusesActiveDeck(t *testing.T) {
	decks := newFakeDeckRepository()
	cards := &deletionFlashcardRepository{}
	events := &deletionReviewRepository{}
	svc := New(decks, cards, events, &recordingAudioStore{}, clock.Fixed{Time: fixedNow})

	created, _ := svc.Create(context.Background(), "user-1", "English", "")

	if err := svc.DeleteArchived(context.Background(), "user-1", created.ID); err == nil {
		t.Fatal("DeleteArchived() should refuse a deck that is not archived")
	}
	if _, err := svc.Get(context.Background(), "user-1", created.ID); err != nil {
		t.Errorf("active deck must survive a refused DeleteArchived(): %v", err)
	}
	if cards.deletedAllOf != "" || events.deletedDeck != "" {
		t.Error("nothing may be deleted for a deck that is not archived")
	}
}

func TestDeleteArchived_WrongOwnerIsNotFound(t *testing.T) {
	svc, _ := newTestService()
	created, _ := svc.Create(context.Background(), "user-1", "English", "")
	_ = svc.Archive(context.Background(), "user-1", created.ID)

	if err := svc.DeleteArchived(context.Background(), "someone-else", created.ID); err == nil {
		t.Fatal("DeleteArchived() should return an error for a deck owned by someone else")
	}
}

func TestGetStats_StudiedCardsFollowTheCallersLocalDay(t *testing.T) {
	decks := newFakeDeckRepository()
	loc, err := time.LoadLocation("America/Manaus") // UTC-4, no daylight saving
	if err != nil {
		t.Fatalf("LoadLocation() error: %v", err)
	}

	// fixedNow is 2026-01-01 12:00 UTC = 08:00 local, so the local day
	// started at 2026-01-01 00:00 -04:00 and the 7-day window at 2025-12-26.
	wantToday := time.Date(2026, 1, 1, 0, 0, 0, 0, loc)
	wantWeek := time.Date(2025, 12, 26, 0, 0, 0, 0, loc)
	reviews := fakeReviewEventRepository{studied: func(since time.Time) int {
		switch {
		case since.IsZero():
			return 40
		case since.Equal(wantWeek):
			return 9
		case since.Equal(wantToday):
			return 3
		}
		t.Errorf("CountStudiedCards() called with unexpected since = %v", since)
		return -1
	}}
	svc := New(decks, fakeFlashcardRepository{}, reviews, nil, clock.Fixed{Time: fixedNow})
	created, _ := svc.Create(context.Background(), "owner", "English", "")

	stats, err := svc.GetStats(context.Background(), "owner", created.ID, loc)
	if err != nil {
		t.Fatalf("GetStats() error: %v", err)
	}
	if stats.StudiedToday != 3 || stats.StudiedLast7Days != 9 || stats.StudiedTotal != 40 {
		t.Errorf("studied = today %d, week %d, total %d; want 3, 9, 40", stats.StudiedToday, stats.StudiedLast7Days, stats.StudiedTotal)
	}
}

func TestGetWithCounts_ReportsWhenTheNextCardComesDue(t *testing.T) {
	decks := newFakeDeckRepository()
	next := fixedNow.Add(48 * time.Hour)
	svc := New(decks, fakeFlashcardRepository{nextDue: &next}, fakeReviewEventRepository{}, nil, clock.Fixed{Time: fixedNow})
	created, _ := svc.Create(context.Background(), "owner", "English", "")

	got, err := svc.GetWithCounts(context.Background(), "owner", created.ID)
	if err != nil {
		t.Fatalf("GetWithCounts() error: %v", err)
	}
	if got.NextDueAt == nil || !got.NextDueAt.Equal(next) {
		t.Errorf("NextDueAt = %v, want %v", got.NextDueAt, next)
	}

	// Without any future card there is no next date.
	svc = New(decks, fakeFlashcardRepository{}, fakeReviewEventRepository{}, nil, clock.Fixed{Time: fixedNow})
	got, _ = svc.GetWithCounts(context.Background(), "owner", created.ID)
	if got.NextDueAt != nil {
		t.Errorf("NextDueAt = %v, want nil when no card is scheduled ahead", got.NextDueAt)
	}
}

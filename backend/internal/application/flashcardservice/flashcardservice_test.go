package flashcardservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"
	"time"

	"flashcard-backend/internal/apperror"
	"flashcard-backend/internal/domain/deck"
	"flashcard-backend/internal/domain/flashcard"
	"flashcard-backend/internal/domain/study"
	"flashcard-backend/internal/ports/audiostore"
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

func (f *fakeDeckRepository) ListActive(_ context.Context, userID string) ([]deck.Deck, error) {
	var result []deck.Deck
	for _, d := range f.decks {
		if d.UserID == userID && !d.IsArchived() {
			result = append(result, d)
		}
	}
	return result, nil
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

func (f *fakeFlashcardRepository) Create(_ context.Context, c flashcard.Flashcard) (flashcard.Flashcard, error) {
	f.nextID++
	c.ID = fmt.Sprintf("card-%d", f.nextID)
	f.cards[c.ID] = c
	return c, nil
}

func (f *fakeFlashcardRepository) FindByID(_ context.Context, userID, id string) (flashcard.Flashcard, error) {
	c, ok := f.cards[id]
	if !ok || c.UserID != userID {
		return flashcard.Flashcard{}, repositories.ErrNotFound
	}
	return c, nil
}

func (f *fakeFlashcardRepository) ListByDeck(_ context.Context, userID, deckID string) ([]flashcard.Flashcard, error) {
	var result []flashcard.Flashcard
	for _, c := range f.cards {
		if c.UserID == userID && c.DeckID == deckID && !c.IsArchived() {
			result = append(result, c)
		}
	}
	return result, nil
}

func (f *fakeFlashcardRepository) NextDueAt(context.Context, string, string, time.Time) (*time.Time, error) {
	panic("not implemented")
}

func (f *fakeFlashcardRepository) CountReviewedSince(context.Context, string, time.Time, repositories.ReviewScope) (int, error) {
	panic("not implemented")
}

func (f *fakeFlashcardRepository) ListDueByStudiedToday(context.Context, string, string, time.Time, time.Time, bool, int) ([]flashcard.Flashcard, error) {
	panic("not implemented")
}

// SearchByDeck mimics the real repository closely enough for the service
// tests: archived/active filter, case-insensitive substring match on
// question/answer/hint, newest first (highest card number first), paging.
func (f *fakeFlashcardRepository) SearchByDeck(_ context.Context, userID, deckID string, opts repositories.FlashcardListOptions) ([]flashcard.Flashcard, int, error) {
	number := func(id string) int {
		var n int
		_, _ = fmt.Sscanf(id, "card-%d", &n)
		return n
	}
	var matches []flashcard.Flashcard
	for _, c := range f.cards {
		if c.UserID != userID || c.DeckID != deckID || c.IsArchived() != opts.Archived {
			continue
		}
		haystack := strings.ToLower(c.Question + " " + c.Answer + " " + c.Hint)
		if opts.Query != "" && !strings.Contains(haystack, strings.ToLower(opts.Query)) {
			continue
		}
		matches = append(matches, c)
	}
	sort.Slice(matches, func(i, j int) bool { return number(matches[i].ID) > number(matches[j].ID) })

	total := len(matches)
	if opts.Offset >= total {
		return nil, total, nil
	}
	end := opts.Offset + opts.Limit
	if end > total {
		end = total
	}
	return matches[opts.Offset:end], total, nil
}

func (f *fakeFlashcardRepository) ListArchivedByDeck(_ context.Context, userID, deckID string) ([]flashcard.Flashcard, error) {
	var result []flashcard.Flashcard
	for _, c := range f.cards {
		if c.UserID == userID && c.DeckID == deckID && c.IsArchived() {
			result = append(result, c)
		}
	}
	return result, nil
}

func (f *fakeFlashcardRepository) Restore(_ context.Context, userID, id string, _ time.Time) error {
	existing, ok := f.cards[id]
	if !ok || existing.UserID != userID {
		return repositories.ErrNotFound
	}
	existing.ArchivedAt = nil
	f.cards[id] = existing
	return nil
}

func (f *fakeFlashcardRepository) ListArchivedByUser(_ context.Context, userID string) ([]flashcard.Flashcard, error) {
	var result []flashcard.Flashcard
	for _, c := range f.cards {
		if c.UserID == userID && c.IsArchived() {
			result = append(result, c)
		}
	}
	return result, nil
}

func (f *fakeFlashcardRepository) DeleteArchived(_ context.Context, userID, id string) (flashcard.Flashcard, error) {
	existing, ok := f.cards[id]
	if !ok || existing.UserID != userID || !existing.IsArchived() {
		return flashcard.Flashcard{}, repositories.ErrNotFound
	}
	delete(f.cards, id)
	return existing, nil
}

func (f *fakeFlashcardRepository) DeleteAllByDeck(context.Context, string, string) error {
	panic("not implemented")
}

func (f *fakeFlashcardRepository) ListDue(context.Context, string, string, time.Time, int) ([]flashcard.Flashcard, error) {
	panic("not implemented")
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

// fakeAudioStore is an in-memory audiostore.Store for tests, so they
// don't need a real MongoDB/GridFS instance to exercise the audio
// business rules.
type fakeAudioStore struct {
	files  map[string][]byte
	nextID int
}

func newFakeAudioStore() *fakeAudioStore {
	return &fakeAudioStore{files: map[string][]byte{}}
}

func (f *fakeAudioStore) Save(_ context.Context, r io.Reader, maxSizeBytes int64) (string, int64, error) {
	limited := io.LimitReader(r, maxSizeBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", 0, err
	}
	if int64(len(data)) > maxSizeBytes {
		return "", 0, audiostore.ErrTooLarge
	}
	f.nextID++
	id := fmt.Sprintf("audio-%d", f.nextID)
	f.files[id] = data
	return id, int64(len(data)), nil
}

func (f *fakeAudioStore) Open(_ context.Context, fileID string) (io.ReadCloser, error) {
	data, ok := f.files[fileID]
	if !ok {
		return nil, audiostore.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (f *fakeAudioStore) Delete(_ context.Context, fileID string) error {
	delete(f.files, fileID)
	return nil
}

var fixedNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

const testMaxAudioSizeBytes = 1024 * 1024

func newTestService() (Service, *fakeDeckRepository, *fakeFlashcardRepository) {
	svc, decks, cards, _ := newTestServiceWithAudio()
	return svc, decks, cards
}

func newTestServiceWithAudio() (Service, *fakeDeckRepository, *fakeFlashcardRepository, *fakeAudioStore) {
	decks := newFakeDeckRepository()
	cards := newFakeFlashcardRepository()
	audio := newFakeAudioStore()
	svc := New(cards, decks, audio, testMaxAudioSizeBytes, clock.Fixed{Time: fixedNow})
	return svc, decks, cards, audio
}

func TestCreate_RejectsWhenDeckNotOwned(t *testing.T) {
	svc, decks, _ := newTestService()
	decks.seed("owner", "deck-1")

	_, err := svc.Create(context.Background(), "someone-else", "deck-1", Draft{Question: "Q", Answer: "A"})
	if err == nil {
		t.Fatal("Create() should reject a deck not owned by the caller")
	}
}

func TestCreate_ValidatesQuestionAndAnswer(t *testing.T) {
	svc, decks, _ := newTestService()
	decks.seed("owner", "deck-1")

	if _, err := svc.Create(context.Background(), "owner", "deck-1", Draft{Question: "", Answer: "A"}); err == nil {
		t.Fatal("Create() should reject an empty question")
	}
	if _, err := svc.Create(context.Background(), "owner", "deck-1", Draft{Question: "Q", Answer: ""}); err == nil {
		t.Fatal("Create() should reject an empty answer")
	}
}

func TestCreate_WithExtendedExample(t *testing.T) {
	svc, decks, _ := newTestService()
	decks.seed("owner", "deck-1")

	created, err := svc.Create(context.Background(), "owner", "deck-1", Draft{
		Question:            "Q",
		Answer:              "A",
		ExtendedExampleText: "example text",
		ExtendedTranslation: "example translation",
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if created.ExtendedExample == nil || created.ExtendedExample.Text != "example text" {
		t.Errorf("ExtendedExample = %+v", created.ExtendedExample)
	}
}

func TestCreate_WithoutExtendedExample(t *testing.T) {
	svc, decks, _ := newTestService()
	decks.seed("owner", "deck-1")

	created, err := svc.Create(context.Background(), "owner", "deck-1", Draft{Question: "Q", Answer: "A"})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if created.ExtendedExample != nil {
		t.Errorf("ExtendedExample = %+v, want nil", created.ExtendedExample)
	}
}

func TestGet_WrongOwnerIsNotFound(t *testing.T) {
	svc, decks, _ := newTestService()
	decks.seed("owner", "deck-1")

	created, err := svc.Create(context.Background(), "owner", "deck-1", Draft{Question: "Q", Answer: "A"})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if _, err := svc.Get(context.Background(), "someone-else", created.ID); err == nil {
		t.Fatal("Get() should reject a flashcard owned by someone else")
	}
}

func TestListPage_RejectsWhenDeckNotOwned(t *testing.T) {
	svc, decks, _ := newTestService()
	decks.seed("owner", "deck-1")

	if _, err := svc.ListPage(context.Background(), "someone-else", "deck-1", false, "", 0, 0); err == nil {
		t.Fatal("ListPage() should reject a deck not owned by the caller")
	}
}

func TestUpdate_PreservesSchedulingState(t *testing.T) {
	svc, decks, _ := newTestService()
	decks.seed("owner", "deck-1")

	created, err := svc.Create(context.Background(), "owner", "deck-1", Draft{Question: "Q", Answer: "A"})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	originalState := created.Scheduling.State

	updated, err := svc.Update(context.Background(), "owner", created.ID, Draft{Question: "Q2", Answer: "A2"})
	if err != nil {
		t.Fatalf("Update() error: %v", err)
	}
	if updated.Question != "Q2" || updated.Answer != "A2" {
		t.Errorf("Update() did not persist new fields: %+v", updated)
	}
	if updated.Scheduling.State != originalState {
		t.Errorf("Update() changed Scheduling.State from %q to %q", originalState, updated.Scheduling.State)
	}
}

func TestArchive_WrongOwnerIsNotFound(t *testing.T) {
	svc, decks, _ := newTestService()
	decks.seed("owner", "deck-1")

	created, err := svc.Create(context.Background(), "owner", "deck-1", Draft{Question: "Q", Answer: "A"})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.Archive(context.Background(), "someone-else", created.ID); err == nil {
		t.Fatal("Archive() should reject a flashcard owned by someone else")
	}
}

func TestUploadAudio_RejectsUnsupportedContentType(t *testing.T) {
	svc, decks, _, _ := newTestServiceWithAudio()
	decks.seed("owner", "deck-1")
	created, err := svc.Create(context.Background(), "owner", "deck-1", Draft{Question: "Q", Answer: "A"})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	_, err = svc.UploadAudio(context.Background(), "owner", created.ID, "video/mp4", strings.NewReader("data"))
	if err == nil {
		t.Fatal("UploadAudio() should reject an unsupported content type")
	}
}

func TestUploadAudio_RejectsOversizedFile(t *testing.T) {
	svc, decks, _, _ := newTestServiceWithAudio()
	decks.seed("owner", "deck-1")
	created, err := svc.Create(context.Background(), "owner", "deck-1", Draft{Question: "Q", Answer: "A"})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	oversized := bytes.Repeat([]byte{0}, testMaxAudioSizeBytes+1)
	_, err = svc.UploadAudio(context.Background(), "owner", created.ID, "audio/mpeg", bytes.NewReader(oversized))
	if err == nil {
		t.Fatal("UploadAudio() should reject a file exceeding the maximum size")
	}
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodePayloadTooLarge {
		t.Fatalf("error = %v, want PAYLOAD_TOO_LARGE", err)
	}
}

func TestUploadAudio_StoresAndAllowsPlayback(t *testing.T) {
	svc, decks, _, _ := newTestServiceWithAudio()
	decks.seed("owner", "deck-1")
	created, err := svc.Create(context.Background(), "owner", "deck-1", Draft{Question: "Q", Answer: "A"})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	updated, err := svc.UploadAudio(context.Background(), "owner", created.ID, "audio/mpeg", strings.NewReader("fake mp3 bytes"))
	if err != nil {
		t.Fatalf("UploadAudio() error: %v", err)
	}
	if updated.Audio == nil || updated.Audio.ContentType != "audio/mpeg" {
		t.Fatalf("Audio = %+v", updated.Audio)
	}

	audioMeta, stream, err := svc.GetAudio(context.Background(), "owner", created.ID)
	if err != nil {
		t.Fatalf("GetAudio() error: %v", err)
	}
	defer stream.Close()

	if audioMeta.ContentType != "audio/mpeg" {
		t.Errorf("ContentType = %q, want %q", audioMeta.ContentType, "audio/mpeg")
	}
	data, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("reading stream: %v", err)
	}
	if string(data) != "fake mp3 bytes" {
		t.Errorf("stream contents = %q, want %q", data, "fake mp3 bytes")
	}
}

func TestUploadAudio_ReplacingDeletesOldFile(t *testing.T) {
	svc, decks, _, audio := newTestServiceWithAudio()
	decks.seed("owner", "deck-1")
	created, err := svc.Create(context.Background(), "owner", "deck-1", Draft{Question: "Q", Answer: "A"})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	first, err := svc.UploadAudio(context.Background(), "owner", created.ID, "audio/mpeg", strings.NewReader("first"))
	if err != nil {
		t.Fatalf("first UploadAudio() error: %v", err)
	}
	oldFileID := first.Audio.GridFSFileID

	if _, err = svc.UploadAudio(context.Background(), "owner", created.ID, "audio/wav", strings.NewReader("second")); err != nil {
		t.Fatalf("second UploadAudio() error: %v", err)
	}

	if _, ok := audio.files[oldFileID]; ok {
		t.Error("old audio file should have been deleted after replacement")
	}
}

func TestUploadAudio_RejectsWhenFlashcardNotOwned(t *testing.T) {
	svc, decks, _, _ := newTestServiceWithAudio()
	decks.seed("owner", "deck-1")
	created, err := svc.Create(context.Background(), "owner", "deck-1", Draft{Question: "Q", Answer: "A"})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	_, err = svc.UploadAudio(context.Background(), "someone-else", created.ID, "audio/mpeg", strings.NewReader("data"))
	if err == nil {
		t.Fatal("UploadAudio() should reject a flashcard owned by someone else")
	}
}

func TestGetAudio_NoAudioIsNotFound(t *testing.T) {
	svc, decks, _, _ := newTestServiceWithAudio()
	decks.seed("owner", "deck-1")
	created, err := svc.Create(context.Background(), "owner", "deck-1", Draft{Question: "Q", Answer: "A"})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if _, _, err := svc.GetAudio(context.Background(), "owner", created.ID); err == nil {
		t.Fatal("GetAudio() should return an error when the flashcard has no audio")
	}
}

func TestDeleteAudio_RemovesReferenceAndFile(t *testing.T) {
	svc, decks, _, audio := newTestServiceWithAudio()
	decks.seed("owner", "deck-1")
	created, err := svc.Create(context.Background(), "owner", "deck-1", Draft{Question: "Q", Answer: "A"})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	uploaded, err := svc.UploadAudio(context.Background(), "owner", created.ID, "audio/mpeg", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("UploadAudio() error: %v", err)
	}
	fileID := uploaded.Audio.GridFSFileID

	if err := svc.DeleteAudio(context.Background(), "owner", created.ID); err != nil {
		t.Fatalf("DeleteAudio() error: %v", err)
	}

	after, err := svc.Get(context.Background(), "owner", created.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if after.Audio != nil {
		t.Errorf("Audio = %+v, want nil after DeleteAudio()", after.Audio)
	}
	if _, ok := audio.files[fileID]; ok {
		t.Error("audio file should have been deleted from storage")
	}
}

func TestDeleteAudio_IsIdempotentWhenNoAudio(t *testing.T) {
	svc, decks, _, _ := newTestServiceWithAudio()
	decks.seed("owner", "deck-1")
	created, err := svc.Create(context.Background(), "owner", "deck-1", Draft{Question: "Q", Answer: "A"})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.DeleteAudio(context.Background(), "owner", created.ID); err != nil {
		t.Fatalf("DeleteAudio() on a card with no audio should not error, got: %v", err)
	}
}

func TestRestore_ReturnsArchivedCardToDeck(t *testing.T) {
	svc, decks, _ := newTestService()
	decks.seed("user-1", "deck-1")
	created, err := svc.Create(context.Background(), "user-1", "deck-1", Draft{Question: "q", Answer: "a"})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := svc.Archive(context.Background(), "user-1", created.ID); err != nil {
		t.Fatalf("Archive() error: %v", err)
	}

	archived, err := svc.ListPage(context.Background(), "user-1", "deck-1", true, "", 0, 0)
	if err != nil || len(archived.Cards) != 1 {
		t.Fatalf("ListPage(archived) = %+v, %v; want one card", archived, err)
	}

	if err := svc.Restore(context.Background(), "user-1", created.ID); err != nil {
		t.Fatalf("Restore() error: %v", err)
	}

	active, _ := svc.ListPage(context.Background(), "user-1", "deck-1", false, "", 0, 0)
	if len(active.Cards) != 1 {
		t.Errorf("ListPage() returned %d cards after restore, want 1", len(active.Cards))
	}
	archived, _ = svc.ListPage(context.Background(), "user-1", "deck-1", true, "", 0, 0)
	if len(archived.Cards) != 0 {
		t.Errorf("ListPage(archived) returned %d cards after restore, want 0", len(archived.Cards))
	}
}

func TestRestore_WrongOwnerIsNotFound(t *testing.T) {
	svc, decks, _ := newTestService()
	decks.seed("user-1", "deck-1")
	created, _ := svc.Create(context.Background(), "user-1", "deck-1", Draft{Question: "q", Answer: "a"})
	_ = svc.Archive(context.Background(), "user-1", created.ID)

	if err := svc.Restore(context.Background(), "someone-else", created.ID); err == nil {
		t.Fatal("Restore() should return an error for a card owned by someone else")
	}
}

func TestListArchivedAcrossDecks_IncludesDeckNameAndSkipsArchivedDecks(t *testing.T) {
	svc, decks, _ := newTestService()
	decks.decks["deck-1"] = deck.Deck{ID: "deck-1", UserID: "user-1", Name: "English"}
	decks.decks["deck-2"] = deck.Deck{ID: "deck-2", UserID: "user-1", Name: "Français"}
	archivedAt := fixedNow
	decks.decks["deck-3"] = deck.Deck{ID: "deck-3", UserID: "user-1", Name: "Old", ArchivedAt: &archivedAt}

	var ids []string
	for _, deckID := range []string{"deck-1", "deck-2", "deck-3"} {
		card, err := svc.Create(context.Background(), "user-1", deckID, Draft{Question: "q-" + deckID, Answer: "a"})
		if err != nil {
			t.Fatalf("Create(%s) error: %v", deckID, err)
		}
		if err := svc.Archive(context.Background(), "user-1", card.ID); err != nil {
			t.Fatalf("Archive() error: %v", err)
		}
		ids = append(ids, card.ID)
	}
	// A still-active card must not be listed.
	if _, err := svc.Create(context.Background(), "user-1", "deck-1", Draft{Question: "active", Answer: "a"}); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := svc.ListArchivedAcrossDecks(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListArchivedAcrossDecks() error: %v", err)
	}

	names := map[string]string{}
	for _, a := range got {
		names[a.Card.DeckID] = a.DeckName
	}
	if len(got) != 2 || names["deck-1"] != "English" || names["deck-2"] != "Français" {
		t.Fatalf("ListArchivedAcrossDecks() = %+v, want the cards of deck-1 and deck-2 with their deck names", got)
	}

	other, _ := svc.ListArchivedAcrossDecks(context.Background(), "someone-else")
	if len(other) != 0 {
		t.Errorf("another user's ListArchivedAcrossDecks() = %+v, want empty", other)
	}
}

func TestDeleteArchived_RemovesCardAndItsAudio(t *testing.T) {
	svc, decks, _, audio := newTestServiceWithAudio()
	decks.seed("user-1", "deck-1")
	created, _ := svc.Create(context.Background(), "user-1", "deck-1", Draft{Question: "q", Answer: "a"})
	if _, err := svc.UploadAudio(context.Background(), "user-1", created.ID, "audio/mpeg", strings.NewReader("data")); err != nil {
		t.Fatalf("UploadAudio() error: %v", err)
	}
	_ = svc.Archive(context.Background(), "user-1", created.ID)

	if err := svc.DeleteArchived(context.Background(), "user-1", created.ID); err != nil {
		t.Fatalf("DeleteArchived() error: %v", err)
	}

	if _, err := svc.Get(context.Background(), "user-1", created.ID); err == nil {
		t.Error("card should be gone after DeleteArchived()")
	}
	if len(audio.files) != 0 {
		t.Errorf("audio files left after DeleteArchived() = %d, want 0", len(audio.files))
	}
}

func TestDeleteArchived_RefusesActiveCardAndWrongOwner(t *testing.T) {
	svc, decks, _ := newTestService()
	decks.seed("user-1", "deck-1")
	created, _ := svc.Create(context.Background(), "user-1", "deck-1", Draft{Question: "q", Answer: "a"})

	if err := svc.DeleteArchived(context.Background(), "user-1", created.ID); err == nil {
		t.Fatal("DeleteArchived() should refuse a card that is not archived")
	}
	if _, err := svc.Get(context.Background(), "user-1", created.ID); err != nil {
		t.Errorf("active card must survive a refused DeleteArchived(): %v", err)
	}

	_ = svc.Archive(context.Background(), "user-1", created.ID)
	if err := svc.DeleteArchived(context.Background(), "someone-else", created.ID); err == nil {
		t.Error("DeleteArchived() should return an error for a card owned by someone else")
	}
}

func TestCreateAndUpdate_Hint(t *testing.T) {
	svc, decks, _ := newTestService()
	decks.seed("user-1", "deck-1")

	created, err := svc.Create(context.Background(), "user-1", "deck-1", Draft{Question: "q", Answer: "a", Hint: "  starts with R  "})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if created.Hint != "starts with R" {
		t.Errorf("Create().Hint = %q, want the trimmed hint", created.Hint)
	}

	updated, err := svc.Update(context.Background(), "user-1", created.ID, Draft{Question: "q", Answer: "a", Hint: "new hint"})
	if err != nil || updated.Hint != "new hint" {
		t.Fatalf("Update() = %+v, %v; want the new hint", updated, err)
	}

	cleared, err := svc.Update(context.Background(), "user-1", created.ID, Draft{Question: "q", Answer: "a"})
	if err != nil || cleared.Hint != "" {
		t.Errorf("Update() without a hint = %+v, %v; want the hint cleared", cleared, err)
	}
}

func TestCreate_RejectsTooLongHint(t *testing.T) {
	svc, decks, _ := newTestService()
	decks.seed("user-1", "deck-1")

	_, err := svc.Create(context.Background(), "user-1", "deck-1", Draft{
		Question: "q", Answer: "a", Hint: strings.Repeat("h", flashcard.MaxHintLength+1),
	})
	if err == nil {
		t.Fatal("Create() should reject a hint over the maximum length")
	}
}

func seedCards(t *testing.T, svc Service, n int) {
	t.Helper()
	for i := 1; i <= n; i++ {
		if _, err := svc.Create(context.Background(), "user-1", "deck-1", Draft{Question: fmt.Sprintf("question %d", i), Answer: "a"}); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}
}

func TestListPage_PagesNewestFirstWithTotal(t *testing.T) {
	svc, decks, _ := newTestService()
	decks.seed("user-1", "deck-1")
	seedCards(t, svc, 7)

	first, err := svc.ListPage(context.Background(), "user-1", "deck-1", false, "", 3, 0)
	if err != nil {
		t.Fatalf("ListPage() error: %v", err)
	}
	if first.Total != 7 || len(first.Cards) != 3 || first.Cards[0].Question != "question 7" {
		t.Fatalf("first page = %d cards (total %d), first %q; want 3 of 7, newest first", len(first.Cards), first.Total, first.Cards[0].Question)
	}

	last, _ := svc.ListPage(context.Background(), "user-1", "deck-1", false, "", 3, 6)
	if last.Total != 7 || len(last.Cards) != 1 || last.Cards[0].Question != "question 1" {
		t.Errorf("last page = %+v, want the single oldest card", last)
	}

	beyond, err := svc.ListPage(context.Background(), "user-1", "deck-1", false, "", 3, 50)
	if err != nil || len(beyond.Cards) != 0 || beyond.Total != 7 {
		t.Errorf("page beyond the end = %+v, %v; want no cards but the total", beyond, err)
	}
}

func TestListPage_SearchFiltersAndCountsMatches(t *testing.T) {
	svc, decks, _ := newTestService()
	decks.seed("user-1", "deck-1")
	seedCards(t, svc, 12)

	got, err := svc.ListPage(context.Background(), "user-1", "deck-1", false, "  QUESTION 1  ", 30, 0)
	if err != nil {
		t.Fatalf("ListPage() error: %v", err)
	}
	// "question 1", "question 10", "question 11", "question 12"
	if got.Total != 4 || len(got.Cards) != 4 {
		t.Errorf("search = %d cards (total %d), want 4", len(got.Cards), got.Total)
	}
}

func TestListPage_DefaultsAndLimits(t *testing.T) {
	svc, decks, _ := newTestService()
	decks.seed("user-1", "deck-1")
	seedCards(t, svc, MaxPageSize+DefaultPageSize)

	def, _ := svc.ListPage(context.Background(), "user-1", "deck-1", false, "", 0, 0)
	if len(def.Cards) != DefaultPageSize {
		t.Errorf("default page size = %d, want %d", len(def.Cards), DefaultPageSize)
	}
	capped, _ := svc.ListPage(context.Background(), "user-1", "deck-1", false, "", MaxPageSize*10, 0)
	if len(capped.Cards) != MaxPageSize {
		t.Errorf("capped page size = %d, want %d", len(capped.Cards), MaxPageSize)
	}
}

func TestListPage_ValidatesInput(t *testing.T) {
	svc, decks, _ := newTestService()
	decks.seed("user-1", "deck-1")

	if _, err := svc.ListPage(context.Background(), "user-1", "deck-1", false, "", 10, -1); err == nil {
		t.Error("ListPage() should reject a negative offset")
	}
	if _, err := svc.ListPage(context.Background(), "user-1", "deck-1", false, strings.Repeat("a", MaxQueryLength+1), 10, 0); err == nil {
		t.Error("ListPage() should reject an over-long search text")
	}
}

package mongodb

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"flashcard-backend/internal/domain/flashcard"
	"flashcard-backend/internal/domain/study"
	"flashcard-backend/internal/ports/repositories"
)

func TestFlashcardRepository_CreateAndFindByID(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewFlashcardRepository(db)

	userID := primitive.NewObjectID().Hex()
	deckID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	example := &flashcard.ExtendedExample{Text: "I rescued a cat.", Translation: "Eu resgatei um gato."}
	created, err := repo.Create(ctx, flashcard.New(userID, deckID, "How to say 'resgatar'?", "rescue", example, now))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create() should assign an ID")
	}
	if created.Scheduling.State != study.StateNew {
		t.Errorf("Scheduling.State = %q, want %q", created.Scheduling.State, study.StateNew)
	}

	found, err := repo.FindByID(ctx, userID, created.ID)
	if err != nil {
		t.Fatalf("FindByID() error: %v", err)
	}
	if found.Answer != "rescue" {
		t.Errorf("FindByID().Answer = %q, want %q", found.Answer, "rescue")
	}
	if found.ExtendedExample == nil || found.ExtendedExample.Text != "I rescued a cat." {
		t.Errorf("FindByID().ExtendedExample = %+v", found.ExtendedExample)
	}
}

func TestFlashcardRepository_FindByID_WrongOwnerIsNotFound(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewFlashcardRepository(db)

	ownerID := primitive.NewObjectID().Hex()
	otherUserID := primitive.NewObjectID().Hex()
	deckID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	created, err := repo.Create(ctx, flashcard.New(ownerID, deckID, "Q", "A", nil, now))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	_, err = repo.FindByID(ctx, otherUserID, created.ID)
	if err != repositories.ErrNotFound {
		t.Errorf("FindByID() with wrong owner error = %v, want ErrNotFound", err)
	}
}

func TestFlashcardRepository_ListDue(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewFlashcardRepository(db)

	userID := primitive.NewObjectID().Hex()
	deckID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	dueCard := flashcard.New(userID, deckID, "Due question", "Due answer", nil, now)
	dueCard.Scheduling.DueAt = now.Add(-time.Hour) // already due
	if _, err := repo.Create(ctx, dueCard); err != nil {
		t.Fatalf("Create() due card error: %v", err)
	}

	futureCard := flashcard.New(userID, deckID, "Future question", "Future answer", nil, now)
	futureCard.Scheduling.DueAt = now.Add(24 * time.Hour) // not due yet
	if _, err := repo.Create(ctx, futureCard); err != nil {
		t.Fatalf("Create() future card error: %v", err)
	}

	due, err := repo.ListDue(ctx, userID, deckID, now, 10)
	if err != nil {
		t.Fatalf("ListDue() error: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("ListDue() returned %d cards, want 1", len(due))
	}
	if due[0].Question != "Due question" {
		t.Errorf("ListDue()[0].Question = %q, want %q", due[0].Question, "Due question")
	}
}

func TestFlashcardRepository_ArchivedCardsExcludedFromListAndCount(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewFlashcardRepository(db)

	userID := primitive.NewObjectID().Hex()
	deckID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	created, err := repo.Create(ctx, flashcard.New(userID, deckID, "Q", "A", nil, now))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := repo.Archive(ctx, userID, created.ID, now); err != nil {
		t.Fatalf("Archive() error: %v", err)
	}

	cards, err := repo.ListByDeck(ctx, userID, deckID)
	if err != nil {
		t.Fatalf("ListByDeck() error: %v", err)
	}
	if len(cards) != 0 {
		t.Errorf("ListByDeck() returned %d cards after archive, want 0", len(cards))
	}

	total, due, err := repo.CountByDeck(ctx, userID, deckID, now)
	if err != nil {
		t.Fatalf("CountByDeck() error: %v", err)
	}
	if total != 0 || due != 0 {
		t.Errorf("CountByDeck() = (%d, %d), want (0, 0)", total, due)
	}
}

func TestFlashcardRepository_Update(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewFlashcardRepository(db)

	userID := primitive.NewObjectID().Hex()
	deckID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	created, err := repo.Create(ctx, flashcard.New(userID, deckID, "Q", "A", nil, now))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	scheduler := study.NewDefaultScheduler(365)
	decision := scheduler.Schedule(created.Scheduling, study.RatingGood, now)
	created.Scheduling.State = decision.State
	created.Scheduling.DueAt = decision.DueAt
	created.Scheduling.IntervalDays = decision.IntervalDays
	created.Scheduling.Repetitions = decision.Repetitions

	updated, err := repo.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update() error: %v", err)
	}
	if updated.Scheduling.State != study.StateReview {
		t.Errorf("Update().Scheduling.State = %q, want %q", updated.Scheduling.State, study.StateReview)
	}
	if updated.Scheduling.IntervalDays != 3 {
		t.Errorf("Update().Scheduling.IntervalDays = %d, want 3", updated.Scheduling.IntervalDays)
	}
}

func TestFlashcardRepository_CountsByState(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewFlashcardRepository(db)

	userID := primitive.NewObjectID().Hex()
	deckID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	// Two NEW cards.
	if _, err := repo.Create(ctx, flashcard.New(userID, deckID, "Q1", "A1", nil, now)); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if _, err := repo.Create(ctx, flashcard.New(userID, deckID, "Q2", "A2", nil, now)); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// One REVIEW card.
	reviewCard, err := repo.Create(ctx, flashcard.New(userID, deckID, "Q3", "A3", nil, now))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	reviewCard.Scheduling.State = study.StateReview
	reviewCard.Scheduling.IntervalDays = 5
	if _, err := repo.Update(ctx, reviewCard); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	// One archived card, which must be excluded.
	archivedCard, err := repo.Create(ctx, flashcard.New(userID, deckID, "Q4", "A4", nil, now))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := repo.Archive(ctx, userID, archivedCard.ID, now); err != nil {
		t.Fatalf("Archive() error: %v", err)
	}

	counts, err := repo.CountsByState(ctx, userID, deckID)
	if err != nil {
		t.Fatalf("CountsByState() error: %v", err)
	}
	if counts[study.StateNew] != 2 {
		t.Errorf("counts[new] = %d, want 2", counts[study.StateNew])
	}
	if counts[study.StateReview] != 1 {
		t.Errorf("counts[review] = %d, want 1", counts[study.StateReview])
	}
	if counts[study.StateMature] != 0 {
		t.Errorf("counts[mature] = %d, want 0", counts[study.StateMature])
	}
	total := 0
	for _, c := range counts {
		total += c
	}
	if total != 3 {
		t.Errorf("total counted = %d, want 3 (archived card must be excluded)", total)
	}
}

func TestFlashcardRepository_ArchiveThenRestore(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewFlashcardRepository(db)

	userID := primitive.NewObjectID().Hex()
	deckID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	created, err := repo.Create(ctx, flashcard.New(userID, deckID, "Q", "A", nil, now))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := repo.Archive(ctx, userID, created.ID, now); err != nil {
		t.Fatalf("Archive() error: %v", err)
	}

	archived, err := repo.ListArchivedByDeck(ctx, userID, deckID)
	if err != nil || len(archived) != 1 || archived[0].ID != created.ID {
		t.Fatalf("ListArchivedByDeck() = %+v, %v; want the archived card", archived, err)
	}

	if err := repo.Restore(ctx, userID, created.ID, now.Add(time.Minute)); err != nil {
		t.Fatalf("Restore() error: %v", err)
	}

	active, _ := repo.ListByDeck(ctx, userID, deckID)
	if len(active) != 1 {
		t.Errorf("ListByDeck() returned %d cards after restore, want 1", len(active))
	}
	archived, _ = repo.ListArchivedByDeck(ctx, userID, deckID)
	if len(archived) != 0 {
		t.Errorf("ListArchivedByDeck() returned %d cards after restore, want 0", len(archived))
	}

	// Restoring again (already active) is a no-op, not an error.
	if err := repo.Restore(ctx, userID, created.ID, now); err != nil {
		t.Errorf("Restore() on an active card error = %v, want nil", err)
	}
}

func TestFlashcardRepository_Restore_WrongOwnerIsNotFound(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewFlashcardRepository(db)

	ownerID := primitive.NewObjectID().Hex()
	deckID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	created, _ := repo.Create(ctx, flashcard.New(ownerID, deckID, "Q", "A", nil, now))
	_ = repo.Archive(ctx, ownerID, created.ID, now)

	err := repo.Restore(ctx, primitive.NewObjectID().Hex(), created.ID, now)
	if err != repositories.ErrNotFound {
		t.Errorf("Restore() with wrong owner error = %v, want ErrNotFound", err)
	}
}

func TestFlashcardRepository_ListArchivedByUser_AcrossDecks(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewFlashcardRepository(db)

	userID := primitive.NewObjectID().Hex()
	deckA := primitive.NewObjectID().Hex()
	deckB := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	a, _ := repo.Create(ctx, flashcard.New(userID, deckA, "A", "a", nil, now))
	b, _ := repo.Create(ctx, flashcard.New(userID, deckB, "B", "b", nil, now))
	_, _ = repo.Create(ctx, flashcard.New(userID, deckA, "still active", "x", nil, now))
	_ = repo.Archive(ctx, userID, a.ID, now)
	_ = repo.Archive(ctx, userID, b.ID, now.Add(time.Hour))

	got, err := repo.ListArchivedByUser(ctx, userID)
	if err != nil {
		t.Fatalf("ListArchivedByUser() error: %v", err)
	}
	if len(got) != 2 || got[0].ID != b.ID || got[1].ID != a.ID {
		t.Fatalf("ListArchivedByUser() = %+v, want [b, a] (most recently archived first)", got)
	}

	other, _ := repo.ListArchivedByUser(ctx, primitive.NewObjectID().Hex())
	if len(other) != 0 {
		t.Errorf("another user's ListArchivedByUser() = %d cards, want 0", len(other))
	}
}

func TestFlashcardRepository_DeleteArchived_OnlyArchivedAndOwned(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewFlashcardRepository(db)

	userID := primitive.NewObjectID().Hex()
	deckID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	created, _ := repo.Create(ctx, flashcard.New(userID, deckID, "Q", "A", nil, now))

	if _, err := repo.DeleteArchived(ctx, userID, created.ID); err != repositories.ErrNotFound {
		t.Fatalf("DeleteArchived() on an active card error = %v, want ErrNotFound", err)
	}
	if _, err := repo.FindByID(ctx, userID, created.ID); err != nil {
		t.Fatalf("active card must still exist: %v", err)
	}

	_ = repo.Archive(ctx, userID, created.ID, now)
	if _, err := repo.DeleteArchived(ctx, primitive.NewObjectID().Hex(), created.ID); err != repositories.ErrNotFound {
		t.Fatalf("DeleteArchived() with wrong owner error = %v, want ErrNotFound", err)
	}

	deleted, err := repo.DeleteArchived(ctx, userID, created.ID)
	if err != nil || deleted.ID != created.ID {
		t.Fatalf("DeleteArchived() = %+v, %v; want the deleted card", deleted, err)
	}
	if _, err := repo.FindByID(ctx, userID, created.ID); err != repositories.ErrNotFound {
		t.Errorf("FindByID() after delete error = %v, want ErrNotFound", err)
	}
}

func TestFlashcardRepository_DeleteAllByDeck(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewFlashcardRepository(db)

	userID := primitive.NewObjectID().Hex()
	deckA := primitive.NewObjectID().Hex()
	deckB := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	active, _ := repo.Create(ctx, flashcard.New(userID, deckA, "active", "a", nil, now))
	archived, _ := repo.Create(ctx, flashcard.New(userID, deckA, "archived", "a", nil, now))
	_ = repo.Archive(ctx, userID, archived.ID, now)
	other, _ := repo.Create(ctx, flashcard.New(userID, deckB, "other deck", "a", nil, now))

	if err := repo.DeleteAllByDeck(ctx, userID, deckA); err != nil {
		t.Fatalf("DeleteAllByDeck() error: %v", err)
	}
	if _, err := repo.FindByID(ctx, userID, active.ID); err != repositories.ErrNotFound {
		t.Errorf("active card should be deleted, err = %v", err)
	}
	if _, err := repo.FindByID(ctx, userID, archived.ID); err != repositories.ErrNotFound {
		t.Errorf("archived card should be deleted, err = %v", err)
	}
	if _, err := repo.FindByID(ctx, userID, other.ID); err != nil {
		t.Errorf("a card of another deck must survive: %v", err)
	}
}

func TestFlashcardRepository_PersistsHint(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewFlashcardRepository(db)

	userID := primitive.NewObjectID().Hex()
	deckID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	card := flashcard.New(userID, deckID, "Q", "A", nil, now)
	card.Hint = "Starts with R"
	created, err := repo.Create(ctx, card)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	found, _ := repo.FindByID(ctx, userID, created.ID)
	if found.Hint != "Starts with R" {
		t.Errorf("FindByID().Hint = %q, want %q", found.Hint, "Starts with R")
	}

	found.Hint = ""
	updated, err := repo.Update(ctx, found)
	if err != nil || updated.Hint != "" {
		t.Errorf("Update() with an empty hint = %+v, %v; want the hint cleared", updated, err)
	}
}

func TestFlashcardRepository_SearchByDeck(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewFlashcardRepository(db)

	userID := primitive.NewObjectID().Hex()
	deckID := primitive.NewObjectID().Hex()
	otherDeckID := primitive.NewObjectID().Hex()
	start := time.Now().UTC()

	create := func(q, a, hint string, offset time.Duration, deck string) flashcard.Flashcard {
		c := flashcard.New(userID, deck, q, a, nil, start.Add(offset))
		c.Hint = hint
		created, err := repo.Create(ctx, c)
		if err != nil {
			t.Fatalf("Create() error: %v", err)
		}
		return created
	}
	first := create("Bonjour", "Olá", "", 0, deckID)
	second := create("Comment ça va ?", "Como vai?", "", time.Second, deckID)
	third := create("Français", "Francês", "langue", 2*time.Second, deckID)
	create("Français outro baralho", "x", "", 3*time.Second, otherDeckID)
	archived := create("Arquivado", "x", "", 4*time.Second, deckID)
	_ = repo.Archive(ctx, userID, archived.ID, start)

	search := func(opts repositories.FlashcardListOptions) ([]flashcard.Flashcard, int) {
		opts.Limit = 30
		cards, total, err := repo.SearchByDeck(ctx, userID, deckID, opts)
		if err != nil {
			t.Fatalf("SearchByDeck(%+v) error: %v", opts, err)
		}
		return cards, total
	}
	ids := func(cards []flashcard.Flashcard) []string {
		var out []string
		for _, c := range cards {
			out = append(out, c.ID)
		}
		return out
	}

	// Active cards only, newest first.
	cards, total := search(repositories.FlashcardListOptions{})
	if total != 3 || len(cards) != 3 || cards[0].ID != third.ID || cards[2].ID != first.ID {
		t.Fatalf("active list = %v (total %d), want [third second first]", ids(cards), total)
	}

	// Accent- and case-insensitive: "francais" finds "Français" (and only this deck's card).
	cards, total = search(repositories.FlashcardListOptions{Query: "FRANCAIS"})
	if total != 1 || len(cards) != 1 || cards[0].ID != third.ID {
		t.Errorf("search 'FRANCAIS' = %v (total %d), want [third]", ids(cards), total)
	}
	// Searches the answer and the hint too.
	cards, _ = search(repositories.FlashcardListOptions{Query: "como vai"})
	if len(cards) != 1 || cards[0].ID != second.ID {
		t.Errorf("search in answer = %v, want [second]", ids(cards))
	}
	cards, _ = search(repositories.FlashcardListOptions{Query: "langue"})
	if len(cards) != 1 || cards[0].ID != third.ID {
		t.Errorf("search in hint = %v, want [third]", ids(cards))
	}
	// Regex metacharacters are matched literally.
	if cards, total = search(repositories.FlashcardListOptions{Query: "va ?"}); total != 1 || cards[0].ID != second.ID {
		t.Errorf("search 'va ?' = %v (total %d), want [second]", ids(cards), total)
	}
	if _, total = search(repositories.FlashcardListOptions{Query: ".*"}); total != 0 {
		t.Errorf("search '.*' matched %d cards, want 0 (must be literal)", total)
	}

	// Paging: total ignores limit/offset and pages do not overlap.
	page1, total1, _ := repo.SearchByDeck(ctx, userID, deckID, repositories.FlashcardListOptions{Limit: 2, Offset: 0})
	page2, total2, _ := repo.SearchByDeck(ctx, userID, deckID, repositories.FlashcardListOptions{Limit: 2, Offset: 2})
	if total1 != 3 || total2 != 3 || len(page1) != 2 || len(page2) != 1 {
		t.Fatalf("pages = %d + %d cards (totals %d, %d), want 2 + 1 of 3", len(page1), len(page2), total1, total2)
	}
	if page1[0].ID != third.ID || page1[1].ID != second.ID || page2[0].ID != first.ID {
		t.Errorf("pages = %v then %v, want [third second] then [first]", ids(page1), ids(page2))
	}

	// Archived cards are reachable with Archived: true.
	cards, total = search(repositories.FlashcardListOptions{Archived: true})
	if total != 1 || cards[0].ID != archived.ID {
		t.Errorf("archived list = %v (total %d), want [archived]", ids(cards), total)
	}

	// Another user sees nothing.
	other, otherTotal, _ := repo.SearchByDeck(ctx, primitive.NewObjectID().Hex(), deckID, repositories.FlashcardListOptions{Limit: 30})
	if len(other) != 0 || otherTotal != 0 {
		t.Errorf("another user's search = %d cards (total %d), want none", len(other), otherTotal)
	}
}

func TestFlashcardRepository_CountReviewedSinceAndListDueByStudiedToday(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewFlashcardRepository(db)

	userID := primitive.NewObjectID().Hex()
	deckA := primitive.NewObjectID().Hex()
	deckB := primitive.NewObjectID().Hex()
	now := time.Now().UTC().Truncate(time.Millisecond)
	startOfToday := now.Add(-6 * time.Hour)

	make := func(deck string, due time.Time, lastReviewed *time.Time) flashcard.Flashcard {
		c := flashcard.New(userID, deck, "Q", "A", nil, now)
		c.Scheduling.DueAt = due
		c.Scheduling.LastReviewedAt = lastReviewed
		created, err := repo.Create(ctx, c)
		if err != nil {
			t.Fatalf("Create() error: %v", err)
		}
		return created
	}
	today := now.Add(-time.Hour)
	yesterday := now.Add(-30 * time.Hour)

	studiedAndDueAgain := make(deckA, now.Add(-time.Minute), &today)
	studiedNotDue := make(deckA, now.Add(48*time.Hour), &today)
	studiedInOtherDeck := make(deckB, now.Add(48*time.Hour), &today)
	reviewedYesterday := make(deckA, now.Add(-time.Hour), &yesterday)
	neverReviewed := make(deckA, now, nil)
	_ = studiedNotDue
	_ = studiedInOtherDeck

	// Another user's cards never count.
	other := flashcard.New(primitive.NewObjectID().Hex(), deckA, "Q", "A", nil, now)
	other.Scheduling.LastReviewedAt = &today
	_, _ = repo.Create(ctx, other)

	count, err := repo.CountReviewedSince(ctx, userID, startOfToday, repositories.ReviewScope{})
	if err != nil || count != 3 {
		t.Errorf("CountReviewedSince() = %d, %v; want 3 (reviewed today, across both decks)", count, err)
	}

	studied, err := repo.ListDueByStudiedToday(ctx, userID, deckA, now, startOfToday, true, 10)
	if err != nil || len(studied) != 1 || studied[0].ID != studiedAndDueAgain.ID {
		t.Errorf("due and studied today = %+v, %v; want only the card that came due again", studied, err)
	}

	fresh, err := repo.ListDueByStudiedToday(ctx, userID, deckA, now, startOfToday, false, 10)
	if err != nil || len(fresh) != 2 {
		t.Fatalf("due and not studied today = %d cards, %v; want 2", len(fresh), err)
	}
	// Ordered by due date: yesterday's review (due an hour ago) then the
	// never-reviewed card (due now).
	if fresh[0].ID != reviewedYesterday.ID || fresh[1].ID != neverReviewed.ID {
		t.Errorf("fresh cards = [%s %s], want [reviewedYesterday neverReviewed]", fresh[0].ID, fresh[1].ID)
	}

	capped, _ := repo.ListDueByStudiedToday(ctx, userID, deckA, now, startOfToday, false, 1)
	if len(capped) != 1 || capped[0].ID != reviewedYesterday.ID {
		t.Errorf("capped fresh cards = %+v, want the earliest one only", capped)
	}
}

func TestFlashcardRepository_NextDueAt(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewFlashcardRepository(db)

	userID := primitive.NewObjectID().Hex()
	deckID := primitive.NewObjectID().Hex()
	otherDeckID := primitive.NewObjectID().Hex()
	now := time.Now().UTC().Truncate(time.Millisecond)

	add := func(deck string, due time.Time) flashcard.Flashcard {
		c := flashcard.New(userID, deck, "Q", "A", nil, now)
		c.Scheduling.DueAt = due
		created, err := repo.Create(ctx, c)
		if err != nil {
			t.Fatalf("Create() error: %v", err)
		}
		return created
	}

	if next, err := repo.NextDueAt(ctx, userID, deckID, now); err != nil || next != nil {
		t.Fatalf("NextDueAt() on an empty deck = %v, %v; want nil", next, err)
	}

	add(deckID, now.Add(-time.Hour)) // already due: not "next"
	soonest := add(deckID, now.Add(48*time.Hour))
	add(deckID, now.Add(72*time.Hour))
	archived := add(deckID, now.Add(24*time.Hour)) // earlier, but archived
	_ = repo.Archive(ctx, userID, archived.ID, now)
	add(otherDeckID, now.Add(time.Hour)) // another deck

	next, err := repo.NextDueAt(ctx, userID, deckID, now)
	if err != nil || next == nil || !next.Equal(soonest.Scheduling.DueAt) {
		t.Errorf("NextDueAt() = %v, %v; want %v (earliest future card of this deck, ignoring archived ones)", next, err, soonest.Scheduling.DueAt)
	}

	if other, _ := repo.NextDueAt(ctx, primitive.NewObjectID().Hex(), deckID, now); other != nil {
		t.Errorf("another user's NextDueAt() = %v, want nil", other)
	}
}

func TestFlashcardRepository_CountReviewedSince_ScopedToDecks(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewFlashcardRepository(db)
	userID := primitive.NewObjectID().Hex()
	deckA := primitive.NewObjectID().Hex()
	deckB := primitive.NewObjectID().Hex()
	deckC := primitive.NewObjectID().Hex()
	now := time.Now().UTC().Truncate(time.Millisecond)
	since := now.Add(-6 * time.Hour)
	today := now.Add(-time.Hour)

	add := func(deck string) {
		c := flashcard.New(userID, deck, "Q", "A", nil, now)
		c.Scheduling.LastReviewedAt = &today
		if _, err := repo.Create(ctx, c); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		add(deckA)
	}
	add(deckB)
	for i := 0; i < 3; i++ {
		add(deckC)
	}

	count := func(scope repositories.ReviewScope) int {
		n, err := repo.CountReviewedSince(ctx, userID, since, scope)
		if err != nil {
			t.Fatalf("CountReviewedSince(%+v) error: %v", scope, err)
		}
		return n
	}
	if got := count(repositories.ReviewScope{}); got != 6 {
		t.Errorf("everything = %d, want 6", got)
	}
	if got := count(repositories.ReviewScope{OnlyDeckID: deckA}); got != 2 {
		t.Errorf("only deck A = %d, want 2", got)
	}
	if got := count(repositories.ReviewScope{ExcludeDeckIDs: []string{deckA, deckC}}); got != 1 {
		t.Errorf("everything but A and C = %d, want 1 (deck B)", got)
	}
	// OnlyDeckID wins over an exclusion list.
	if got := count(repositories.ReviewScope{OnlyDeckID: deckC, ExcludeDeckIDs: []string{deckC}}); got != 3 {
		t.Errorf("only deck C (also listed as excluded) = %d, want 3", got)
	}
}

package deckgroupservice

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"flashcard-backend/internal/apperror"
	"flashcard-backend/internal/domain/deck"
	"flashcard-backend/internal/domain/deckgroup"
	"flashcard-backend/internal/ports/clock"
	"flashcard-backend/internal/ports/repositories"
)

var fixedNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

// ---------- fakes ----------

type fakeGroupRepository struct {
	items  []deckgroup.Group
	nextID int
}

func (f *fakeGroupRepository) nameTaken(userID, name, exceptID string) bool {
	for _, g := range f.items {
		if g.UserID == userID && g.ID != exceptID && deckgroup.NameKey(g.Name) == deckgroup.NameKey(name) {
			return true
		}
	}
	return false
}

func (f *fakeGroupRepository) Create(_ context.Context, g deckgroup.Group) (deckgroup.Group, error) {
	if f.nameTaken(g.UserID, g.Name, "") {
		return deckgroup.Group{}, repositories.ErrDuplicate
	}
	f.nextID++
	g.ID = fmt.Sprintf("group-%d", f.nextID)
	f.items = append(f.items, g)
	return g, nil
}

func (f *fakeGroupRepository) FindByID(_ context.Context, userID, id string) (deckgroup.Group, error) {
	for _, g := range f.items {
		if g.ID == id && g.UserID == userID {
			return g, nil
		}
	}
	return deckgroup.Group{}, repositories.ErrNotFound
}

func (f *fakeGroupRepository) ListByUser(_ context.Context, userID string) ([]deckgroup.Group, error) {
	out := []deckgroup.Group{}
	for _, g := range f.items {
		if g.UserID == userID {
			out = append(out, g)
		}
	}
	return out, nil
}

func (f *fakeGroupRepository) Update(_ context.Context, g deckgroup.Group) (deckgroup.Group, error) {
	for i, existing := range f.items {
		if existing.ID == g.ID && existing.UserID == g.UserID {
			if f.nameTaken(g.UserID, g.Name, g.ID) {
				return deckgroup.Group{}, repositories.ErrDuplicate
			}
			f.items[i] = g
			return g, nil
		}
	}
	return deckgroup.Group{}, repositories.ErrNotFound
}

func (f *fakeGroupRepository) Delete(_ context.Context, userID, id string) error {
	for i, g := range f.items {
		if g.ID == id && g.UserID == userID {
			f.items = append(f.items[:i], f.items[i+1:]...)
			return nil
		}
	}
	return repositories.ErrNotFound
}

type fakeDeckRepository struct{ decks map[string]deck.Deck }

func newFakeDeckRepository(decks ...deck.Deck) *fakeDeckRepository {
	f := &fakeDeckRepository{decks: map[string]deck.Deck{}}
	for _, d := range decks {
		f.decks[d.ID] = d
	}
	return f
}

func (f *fakeDeckRepository) FindByID(_ context.Context, userID, id string) (deck.Deck, error) {
	d, ok := f.decks[id]
	if !ok || d.UserID != userID {
		return deck.Deck{}, repositories.ErrNotFound
	}
	return d, nil
}

func (f *fakeDeckRepository) SetGroup(_ context.Context, userID, id, groupID string, _ time.Time) error {
	d, ok := f.decks[id]
	if !ok || d.UserID != userID {
		return repositories.ErrNotFound
	}
	d.GroupID = groupID
	f.decks[id] = d
	return nil
}

func (f *fakeDeckRepository) ClearGroup(_ context.Context, userID, groupID string) error {
	for id, d := range f.decks {
		if d.UserID == userID && d.GroupID == groupID {
			d.GroupID = ""
			f.decks[id] = d
		}
	}
	return nil
}

func (f *fakeDeckRepository) Create(context.Context, deck.Deck) (deck.Deck, error) {
	panic("not implemented")
}
func (f *fakeDeckRepository) ListActive(context.Context, string) ([]deck.Deck, error) {
	panic("not implemented")
}
func (f *fakeDeckRepository) ListArchived(context.Context, string) ([]deck.Deck, error) {
	panic("not implemented")
}
func (f *fakeDeckRepository) Update(context.Context, deck.Deck) (deck.Deck, error) {
	panic("not implemented")
}
func (f *fakeDeckRepository) Archive(context.Context, string, string, time.Time) error {
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
func (f *fakeDeckRepository) DeleteArchived(context.Context, string, string) error {
	panic("not implemented")
}

// ---------- tests ----------

func TestCreate_ValidatesRejectsDuplicatesAndEnforcesTheLimit(t *testing.T) {
	groups := &fakeGroupRepository{}
	svc := New(groups, newFakeDeckRepository(), clock.Fixed{Time: fixedNow})

	created, err := svc.Create(context.Background(), "u1", "  Inglês  ")
	if err != nil || created.Name != "Inglês" || created.UserID != "u1" {
		t.Fatalf("Create() = %+v, %v; want the trimmed name stored", created, err)
	}

	if _, err := svc.Create(context.Background(), "u1", "   "); err == nil {
		t.Error("an empty name should be rejected")
	}

	if _, err := svc.Create(context.Background(), "u1", "inglês"); err == nil {
		t.Error("a duplicate name (case-insensitive) should be rejected")
	}
	// Another user may use the same name.
	if _, err := svc.Create(context.Background(), "u2", "Inglês"); err != nil {
		t.Errorf("another user's same name error: %v", err)
	}

	existingForU1, _ := svc.List(context.Background(), "u1")
	for i := len(existingForU1); i < deckgroup.MaxGroupsPerUser; i++ {
		if _, err := svc.Create(context.Background(), "u1", fmt.Sprintf("Grupo %d", i)); err != nil {
			t.Fatalf("Create() #%d error: %v", i, err)
		}
	}
	if _, err := svc.Create(context.Background(), "u1", "Um a mais"); err == nil {
		t.Error("exceeding the per-user limit should be rejected")
	}
}

func TestRename_ValidatesOwnershipAndDuplicates(t *testing.T) {
	groups := &fakeGroupRepository{}
	svc := New(groups, newFakeDeckRepository(), clock.Fixed{Time: fixedNow})

	a, _ := svc.Create(context.Background(), "u1", "A")
	b, _ := svc.Create(context.Background(), "u1", "B")

	renamed, err := svc.Rename(context.Background(), "u1", a.ID, "A renomeado")
	if err != nil || renamed.Name != "A renomeado" {
		t.Fatalf("Rename() = %+v, %v", renamed, err)
	}

	if _, err := svc.Rename(context.Background(), "u1", b.ID, "a RENOMEADO"); err == nil {
		t.Error("renaming onto a taken name should be rejected")
	}

	if _, err := svc.Rename(context.Background(), "u2", a.ID, "Roubado"); err == nil {
		t.Error("renaming another user's group should be rejected")
	}

	if _, err := svc.Rename(context.Background(), "u1", "missing", "Novo nome"); err == nil {
		t.Error("renaming a missing group should be rejected")
	}
}

func TestDelete_UngroupsItsDecksWithoutDeletingThem(t *testing.T) {
	groups := &fakeGroupRepository{}
	decks := newFakeDeckRepository(
		deck.Deck{ID: "d1", UserID: "u1"},
		deck.Deck{ID: "d2", UserID: "u1"},
	)
	svc := New(groups, decks, clock.Fixed{Time: fixedNow})

	created, _ := svc.Create(context.Background(), "u1", "Inglês")
	if err := svc.SetDeckGroup(context.Background(), "u1", "d1", created.ID); err != nil {
		t.Fatalf("SetDeckGroup() error: %v", err)
	}

	if err := svc.Delete(context.Background(), "u1", created.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
	if list, _ := svc.List(context.Background(), "u1"); len(list) != 0 {
		t.Errorf("List() after delete = %+v, want none", list)
	}
	if d := decks.decks["d1"]; d.GroupID != "" {
		t.Errorf("deck d1's GroupID = %q, want cleared, and the deck itself must still exist", d.GroupID)
	}

	if err := svc.Delete(context.Background(), "u1", "missing"); err == nil {
		t.Error("deleting a missing group should be rejected")
	}
}

func TestSetDeckGroup_ValidatesDeckAndGroupOwnership(t *testing.T) {
	groups := &fakeGroupRepository{}
	decks := newFakeDeckRepository(
		deck.Deck{ID: "d1", UserID: "u1"},
		deck.Deck{ID: "d-stranger", UserID: "u2"},
	)
	svc := New(groups, decks, clock.Fixed{Time: fixedNow})
	created, _ := svc.Create(context.Background(), "u1", "Inglês")

	if err := svc.SetDeckGroup(context.Background(), "u1", "d1", created.ID); err != nil {
		t.Fatalf("SetDeckGroup() error: %v", err)
	}
	if d := decks.decks["d1"]; d.GroupID != created.ID {
		t.Errorf("GroupID = %q, want %q", d.GroupID, created.ID)
	}

	// Empty groupID removes it from any group.
	if err := svc.SetDeckGroup(context.Background(), "u1", "d1", ""); err != nil {
		t.Fatalf("SetDeckGroup(\"\") error: %v", err)
	}
	if d := decks.decks["d1"]; d.GroupID != "" {
		t.Errorf("GroupID after clearing = %q, want empty", d.GroupID)
	}

	if err := svc.SetDeckGroup(context.Background(), "u1", "d1", "missing-group"); !isNotFound(err) {
		t.Errorf("SetDeckGroup() with an unknown group error = %v, want NotFound", err)
	}
	if err := svc.SetDeckGroup(context.Background(), "u1", "missing-deck", created.ID); !isNotFound(err) {
		t.Errorf("SetDeckGroup() with an unknown deck error = %v, want NotFound", err)
	}
	// A deck cannot be filed under another user's group.
	if err := svc.SetDeckGroup(context.Background(), "u2", "d-stranger", created.ID); !isNotFound(err) {
		t.Errorf("SetDeckGroup() with another user's group error = %v, want NotFound", err)
	}
}

func isNotFound(err error) bool {
	var appErr *apperror.Error
	return errors.As(err, &appErr) && appErr.Code == apperror.CodeNotFound
}

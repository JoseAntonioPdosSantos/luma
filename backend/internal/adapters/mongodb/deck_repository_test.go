package mongodb

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"flashcard-backend/internal/domain/deck"
	"flashcard-backend/internal/ports/repositories"
)

func TestDeckRepository_CreateFindListArchive(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewDeckRepository(db)

	userID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	created, err := repo.Create(ctx, deck.New(userID, "English", "Learn English", now))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create() should assign an ID")
	}

	found, err := repo.FindByID(ctx, userID, created.ID)
	if err != nil {
		t.Fatalf("FindByID() error: %v", err)
	}
	if found.Name != "English" {
		t.Errorf("FindByID().Name = %q, want %q", found.Name, "English")
	}

	decks, err := repo.ListActive(ctx, userID)
	if err != nil {
		t.Fatalf("ListActive() error: %v", err)
	}
	if len(decks) != 1 {
		t.Fatalf("ListActive() returned %d decks, want 1", len(decks))
	}

	if err := repo.Archive(ctx, userID, created.ID, now.Add(time.Hour)); err != nil {
		t.Fatalf("Archive() error: %v", err)
	}

	decks, err = repo.ListActive(ctx, userID)
	if err != nil {
		t.Fatalf("ListActive() after archive error: %v", err)
	}
	if len(decks) != 0 {
		t.Errorf("ListActive() after archive returned %d decks, want 0", len(decks))
	}
}

func TestDeckRepository_FindByID_WrongOwnerIsNotFound(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewDeckRepository(db)

	ownerID := primitive.NewObjectID().Hex()
	otherUserID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	created, err := repo.Create(ctx, deck.New(ownerID, "Private Deck", "", now))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	_, err = repo.FindByID(ctx, otherUserID, created.ID)
	if err != repositories.ErrNotFound {
		t.Errorf("FindByID() with wrong owner error = %v, want ErrNotFound", err)
	}
}

func TestDeckRepository_Update(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewDeckRepository(db)

	userID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	created, err := repo.Create(ctx, deck.New(userID, "Old Name", "Old description", now))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	created.Name = "New Name"
	created.Description = "New description"
	updated, err := repo.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update() error: %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Update().Name = %q, want %q", updated.Name, "New Name")
	}
}

func TestDeckRepository_ArchiveListArchivedRestore(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewDeckRepository(db)

	userID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	first, _ := repo.Create(ctx, deck.New(userID, "First", "", now))
	second, _ := repo.Create(ctx, deck.New(userID, "Second", "", now))
	_ = repo.Archive(ctx, userID, first.ID, now)
	_ = repo.Archive(ctx, userID, second.ID, now.Add(time.Hour))

	archived, err := repo.ListArchived(ctx, userID)
	if err != nil {
		t.Fatalf("ListArchived() error: %v", err)
	}
	if len(archived) != 2 || archived[0].ID != second.ID {
		t.Fatalf("ListArchived() = %+v, want [second, first] (most recently archived first)", archived)
	}

	if err := repo.Restore(ctx, userID, first.ID, now.Add(2*time.Hour)); err != nil {
		t.Fatalf("Restore() error: %v", err)
	}

	active, _ := repo.ListActive(ctx, userID)
	if len(active) != 1 || active[0].ID != first.ID {
		t.Errorf("ListActive() after restore = %+v, want only the restored deck", active)
	}
	archived, _ = repo.ListArchived(ctx, userID)
	if len(archived) != 1 || archived[0].ID != second.ID {
		t.Errorf("ListArchived() after restore = %+v, want only the second deck", archived)
	}
}

func TestDeckRepository_Restore_WrongOwnerIsNotFound(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewDeckRepository(db)

	ownerID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()
	created, _ := repo.Create(ctx, deck.New(ownerID, "Private", "", now))
	_ = repo.Archive(ctx, ownerID, created.ID, now)

	err := repo.Restore(ctx, primitive.NewObjectID().Hex(), created.ID, now)
	if err != repositories.ErrNotFound {
		t.Errorf("Restore() with wrong owner error = %v, want ErrNotFound", err)
	}
}

func TestDeckRepository_DeleteArchived_OnlyArchivedAndOwned(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewDeckRepository(db)

	userID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()
	created, _ := repo.Create(ctx, deck.New(userID, "English", "", now))

	if err := repo.DeleteArchived(ctx, userID, created.ID); err != repositories.ErrNotFound {
		t.Fatalf("DeleteArchived() on an active deck error = %v, want ErrNotFound", err)
	}

	_ = repo.Archive(ctx, userID, created.ID, now)
	if err := repo.DeleteArchived(ctx, primitive.NewObjectID().Hex(), created.ID); err != repositories.ErrNotFound {
		t.Fatalf("DeleteArchived() with wrong owner error = %v, want ErrNotFound", err)
	}
	if err := repo.DeleteArchived(ctx, userID, created.ID); err != nil {
		t.Fatalf("DeleteArchived() error: %v", err)
	}
	if _, err := repo.FindByID(ctx, userID, created.ID); err != repositories.ErrNotFound {
		t.Errorf("FindByID() after delete error = %v, want ErrNotFound", err)
	}
}

func TestDeckRepository_StudyProfileAndContinuePastGoal(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewDeckRepository(db)
	userID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	d, _ := repo.Create(ctx, deck.New(userID, "English", "", now))
	if d.StudyProfileID != "" || d.ContinuePastGoalOn != "" {
		t.Fatalf("a new deck = %+v, want it to follow the general configuration", d)
	}

	if err := repo.SetStudyProfile(ctx, userID, d.ID, "profile-1", now); err != nil {
		t.Fatalf("SetStudyProfile() error: %v", err)
	}
	if err := repo.SetContinuePastGoalOn(ctx, userID, d.ID, "2026-01-01", now); err != nil {
		t.Fatalf("SetContinuePastGoalOn() error: %v", err)
	}
	found, _ := repo.FindByID(ctx, userID, d.ID)
	if found.StudyProfileID != "profile-1" || found.ContinuePastGoalOn != "2026-01-01" {
		t.Fatalf("stored = %+v, want profile-1 and 2026-01-01", found)
	}

	// Choosing another configuration starts the "keep studying" choice over.
	_ = repo.SetStudyProfile(ctx, userID, d.ID, "profile-2", now)
	found, _ = repo.FindByID(ctx, userID, d.ID)
	if found.StudyProfileID != "profile-2" || found.ContinuePastGoalOn != "" {
		t.Errorf("after switching = %+v, want profile-2 and the choice reset", found)
	}

	// Following the general one again clears both.
	_ = repo.SetContinuePastGoalOn(ctx, userID, d.ID, "2026-01-01", now)
	_ = repo.SetStudyProfile(ctx, userID, d.ID, "", now)
	found, _ = repo.FindByID(ctx, userID, d.ID)
	if found.StudyProfileID != "" || found.ContinuePastGoalOn != "" {
		t.Errorf("after clearing = %+v, want neither set", found)
	}

	other := primitive.NewObjectID().Hex()
	if err := repo.SetStudyProfile(ctx, other, d.ID, "x", now); err != repositories.ErrNotFound {
		t.Errorf("SetStudyProfile() by another user error = %v, want ErrNotFound", err)
	}
	if err := repo.SetContinuePastGoalOn(ctx, other, d.ID, "2026-01-01", now); err != repositories.ErrNotFound {
		t.Errorf("SetContinuePastGoalOn() by another user error = %v, want ErrNotFound", err)
	}
}

func TestDeckRepository_ListWithStudyProfileAndClearStudyProfile(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewDeckRepository(db)
	userID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	a, _ := repo.Create(ctx, deck.New(userID, "A", "", now))
	b, _ := repo.Create(ctx, deck.New(userID, "B", "", now))
	archived, _ := repo.Create(ctx, deck.New(userID, "Arquivado", "", now))
	plain, _ := repo.Create(ctx, deck.New(userID, "Sem configuração", "", now))
	strangerID := primitive.NewObjectID().Hex()
	stranger, _ := repo.Create(ctx, deck.New(strangerID, "De outra pessoa", "", now))
	_ = repo.SetStudyProfile(ctx, userID, a.ID, "profile-1", now)
	_ = repo.SetStudyProfile(ctx, userID, b.ID, "profile-2", now)
	_ = repo.SetStudyProfile(ctx, userID, archived.ID, "profile-1", now)
	_ = repo.Archive(ctx, userID, archived.ID, now)
	_ = repo.SetStudyProfile(ctx, strangerID, stranger.ID, "profile-1", now)
	_ = plain

	list, err := repo.ListWithStudyProfile(ctx, userID)
	if err != nil || len(list) != 3 {
		t.Fatalf("ListWithStudyProfile() = %d decks, %v; want the 3 with a configuration, archived included, nobody else's", len(list), err)
	}

	if err := repo.ClearStudyProfile(ctx, userID, "profile-1"); err != nil {
		t.Fatalf("ClearStudyProfile() error: %v", err)
	}
	list, _ = repo.ListWithStudyProfile(ctx, userID)
	if len(list) != 1 || list[0].ID != b.ID {
		t.Errorf("after clearing profile-1: %+v, want only deck B (profile-2)", list)
	}
	// Another user's deck is untouched.
	if s, _ := repo.FindByID(ctx, strangerID, stranger.ID); s.StudyProfileID != "profile-1" {
		t.Errorf("another user's deck = %q, want profile-1 untouched", s.StudyProfileID)
	}
}

func TestDeckRepository_SetGroupAndClearGroup(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewDeckRepository(db)
	userID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	d, _ := repo.Create(ctx, deck.New(userID, "English", "", now))
	if d.GroupID != "" {
		t.Fatalf("a new deck = %+v, want no group", d)
	}

	if err := repo.SetGroup(ctx, userID, d.ID, "group-1", now); err != nil {
		t.Fatalf("SetGroup() error: %v", err)
	}
	found, _ := repo.FindByID(ctx, userID, d.ID)
	if found.GroupID != "group-1" {
		t.Fatalf("stored GroupID = %q, want group-1", found.GroupID)
	}

	// Removing it from any group (empty groupID) clears the field.
	if err := repo.SetGroup(ctx, userID, d.ID, "", now); err != nil {
		t.Fatalf("SetGroup(\"\") error: %v", err)
	}
	found, _ = repo.FindByID(ctx, userID, d.ID)
	if found.GroupID != "" {
		t.Errorf("after clearing = %q, want empty", found.GroupID)
	}

	other := primitive.NewObjectID().Hex()
	if err := repo.SetGroup(ctx, other, d.ID, "group-1", now); err != repositories.ErrNotFound {
		t.Errorf("SetGroup() by another user error = %v, want ErrNotFound", err)
	}
}

func TestDeckRepository_ClearGroupUngroupsOnlyThatGroupsDecks(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewDeckRepository(db)
	userID := primitive.NewObjectID().Hex()
	strangerID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	a, _ := repo.Create(ctx, deck.New(userID, "A", "", now))
	b, _ := repo.Create(ctx, deck.New(userID, "B", "", now))
	stranger, _ := repo.Create(ctx, deck.New(strangerID, "De outra pessoa", "", now))
	_ = repo.SetGroup(ctx, userID, a.ID, "group-1", now)
	_ = repo.SetGroup(ctx, userID, b.ID, "group-2", now)
	_ = repo.SetGroup(ctx, strangerID, stranger.ID, "group-1", now)

	if err := repo.ClearGroup(ctx, userID, "group-1"); err != nil {
		t.Fatalf("ClearGroup() error: %v", err)
	}

	if found, _ := repo.FindByID(ctx, userID, a.ID); found.GroupID != "" {
		t.Errorf("deck A GroupID = %q, want cleared", found.GroupID)
	}
	if found, _ := repo.FindByID(ctx, userID, b.ID); found.GroupID != "group-2" {
		t.Errorf("deck B GroupID = %q, want group-2 untouched", found.GroupID)
	}
	// Another user's deck in the same-named group is untouched.
	if found, _ := repo.FindByID(ctx, strangerID, stranger.ID); found.GroupID != "group-1" {
		t.Errorf("another user's deck GroupID = %q, want group-1 untouched", found.GroupID)
	}
}

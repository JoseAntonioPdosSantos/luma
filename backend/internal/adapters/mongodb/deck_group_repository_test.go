package mongodb

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"flashcard-backend/internal/domain/deckgroup"
	"flashcard-backend/internal/ports/repositories"
)

func TestDeckGroupRepository_CreateFindListRoundTrip(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	if err := EnsureIndexes(ctx, db); err != nil {
		t.Fatalf("EnsureIndexes() error: %v", err)
	}
	repo := NewDeckGroupRepository(db)

	userID := primitive.NewObjectID().Hex()
	now := time.Now().UTC().Truncate(time.Millisecond)

	created, err := repo.Create(ctx, deckgroup.New(userID, "Inglês", now))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if created.ID == "" || created.UserID != userID || created.Name != "Inglês" {
		t.Fatalf("Create() = %+v, want an ID, the owner and the name", created)
	}

	found, err := repo.FindByID(ctx, userID, created.ID)
	if err != nil || found.Name != "Inglês" {
		t.Fatalf("FindByID() = %+v, %v; want the stored group", found, err)
	}

	second, _ := repo.Create(ctx, deckgroup.New(userID, "Espanhol", now.Add(time.Second)))

	list, err := repo.ListByUser(ctx, userID)
	if err != nil || len(list) != 2 || list[0].ID != created.ID || list[1].ID != second.ID {
		t.Errorf("ListByUser() = %+v, %v; want [Inglês, Espanhol] oldest first", list, err)
	}

	if other, _ := repo.ListByUser(ctx, primitive.NewObjectID().Hex()); len(other) != 0 {
		t.Errorf("another user's list = %d groups, want 0", len(other))
	}
	if _, err := repo.FindByID(ctx, primitive.NewObjectID().Hex(), created.ID); err != repositories.ErrNotFound {
		t.Errorf("FindByID() with the wrong owner error = %v, want ErrNotFound", err)
	}
}

func TestDeckGroupRepository_NamesAreUniquePerUserIgnoringCase(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	if err := EnsureIndexes(ctx, db); err != nil {
		t.Fatalf("EnsureIndexes() error: %v", err)
	}
	repo := NewDeckGroupRepository(db)
	userID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	if _, err := repo.Create(ctx, deckgroup.New(userID, "Inglês", now)); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if _, err := repo.Create(ctx, deckgroup.New(userID, " inglês ", now)); err != repositories.ErrDuplicate {
		t.Errorf("Create() with a repeated name error = %v, want ErrDuplicate", err)
	}
	if _, err := repo.Create(ctx, deckgroup.New(primitive.NewObjectID().Hex(), "Inglês", now)); err != nil {
		t.Errorf("another user's same name error: %v", err)
	}
}

func TestDeckGroupRepository_UpdateAndDelete(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	if err := EnsureIndexes(ctx, db); err != nil {
		t.Fatalf("EnsureIndexes() error: %v", err)
	}
	repo := NewDeckGroupRepository(db)
	userID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	a, _ := repo.Create(ctx, deckgroup.New(userID, "A", now))
	b, _ := repo.Create(ctx, deckgroup.New(userID, "B", now.Add(time.Second)))

	a.Name = "A renomeado"
	a.UpdatedAt = now.Add(time.Minute)
	updated, err := repo.Update(ctx, a)
	if err != nil || updated.Name != "A renomeado" {
		t.Fatalf("Update() = %+v, %v; want the new name stored", updated, err)
	}

	// Renaming onto another group's name is refused.
	b.Name = "a RENOMEADO"
	if _, err := repo.Update(ctx, b); err != repositories.ErrDuplicate {
		t.Errorf("Update() onto a taken name error = %v, want ErrDuplicate", err)
	}

	stranger := a
	stranger.UserID = primitive.NewObjectID().Hex()
	if _, err := repo.Update(ctx, stranger); err != repositories.ErrNotFound {
		t.Errorf("Update() by another user error = %v, want ErrNotFound", err)
	}

	if err := repo.Delete(ctx, primitive.NewObjectID().Hex(), a.ID); err != repositories.ErrNotFound {
		t.Errorf("Delete() by another user error = %v, want ErrNotFound", err)
	}
	if err := repo.Delete(ctx, userID, a.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
	if _, err := repo.FindByID(ctx, userID, a.ID); err != repositories.ErrNotFound {
		t.Errorf("FindByID() after delete error = %v, want ErrNotFound", err)
	}
	// Its name is free again.
	if _, err := repo.Create(ctx, deckgroup.New(userID, "A renomeado", now)); err != nil {
		t.Errorf("re-using a deleted name error: %v", err)
	}
}

package mongodb

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"flashcard-backend/internal/domain/study"
	"flashcard-backend/internal/domain/studyprofile"
	"flashcard-backend/internal/ports/repositories"
)

func newTestProfile(userID, name string, limit *int, at time.Time) studyprofile.Profile {
	rules := study.DefaultRules()
	rules.AgainDelayMinutes = 4
	rules.Good = study.RatingRule{FirstIntervalDays: 2, Multiplier: 1.7}
	return studyprofile.New(userID, name, limit, rules, at)
}

func TestStudyProfileRepository_CreateFindListRoundTrip(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	if err := EnsureIndexes(ctx, db); err != nil {
		t.Fatalf("EnsureIndexes() error: %v", err)
	}
	repo := NewStudyProfileRepository(db)

	userID := primitive.NewObjectID().Hex()
	now := time.Now().UTC().Truncate(time.Millisecond)
	limit := 25

	created, err := repo.Create(ctx, newTestProfile(userID, "Prova", &limit, now))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if created.ID == "" || created.UserID != userID {
		t.Fatalf("Create() = %+v, want an ID and the owner", created)
	}

	found, err := repo.FindByID(ctx, userID, created.ID)
	if err != nil {
		t.Fatalf("FindByID() error: %v", err)
	}
	if found.Name != "Prova" || found.DailyCardLimit == nil || *found.DailyCardLimit != 25 ||
		found.Rules.AgainDelayMinutes != 4 || found.Rules.Good != (study.RatingRule{FirstIntervalDays: 2, Multiplier: 1.7}) ||
		found.Rules.Hard != study.DefaultRules().Hard {
		t.Errorf("FindByID() = %+v, want every value stored", found)
	}

	// A configuration without a daily goal stores no limit.
	noGoal, _ := repo.Create(ctx, newTestProfile(userID, "Sem meta", nil, now.Add(time.Second)))
	if noGoal.DailyCardLimit != nil {
		t.Errorf("a configuration without a goal came back with %d", *noGoal.DailyCardLimit)
	}

	list, err := repo.ListByUser(ctx, userID)
	if err != nil || len(list) != 2 || list[0].ID != created.ID || list[1].ID != noGoal.ID {
		t.Errorf("ListByUser() = %+v, %v; want [Prova, Sem meta] oldest first", list, err)
	}

	if other, _ := repo.ListByUser(ctx, primitive.NewObjectID().Hex()); len(other) != 0 {
		t.Errorf("another user's list = %d configurations, want 0", len(other))
	}
	if _, err := repo.FindByID(ctx, primitive.NewObjectID().Hex(), created.ID); err != repositories.ErrNotFound {
		t.Errorf("FindByID() with the wrong owner error = %v, want ErrNotFound", err)
	}
}

func TestStudyProfileRepository_NamesAreUniquePerUserIgnoringCase(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	if err := EnsureIndexes(ctx, db); err != nil {
		t.Fatalf("EnsureIndexes() error: %v", err)
	}
	repo := NewStudyProfileRepository(db)
	userID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()

	if _, err := repo.Create(ctx, newTestProfile(userID, "Prova", nil, now)); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if _, err := repo.Create(ctx, newTestProfile(userID, " prova ", nil, now)); err != repositories.ErrDuplicate {
		t.Errorf("Create() with a repeated name error = %v, want ErrDuplicate", err)
	}
	if _, err := repo.Create(ctx, newTestProfile(primitive.NewObjectID().Hex(), "Prova", nil, now)); err != nil {
		t.Errorf("another user's same name error: %v", err)
	}
}

func TestStudyProfileRepository_UpdateAndDelete(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	if err := EnsureIndexes(ctx, db); err != nil {
		t.Fatalf("EnsureIndexes() error: %v", err)
	}
	repo := NewStudyProfileRepository(db)
	userID := primitive.NewObjectID().Hex()
	now := time.Now().UTC()
	limit := 10

	a, _ := repo.Create(ctx, newTestProfile(userID, "A", &limit, now))
	b, _ := repo.Create(ctx, newTestProfile(userID, "B", nil, now.Add(time.Second)))

	a.Name = "A renomeada"
	a.DailyCardLimit = nil
	a.Rules.AgainDelayMinutes = 30
	a.UpdatedAt = now.Add(time.Minute)
	updated, err := repo.Update(ctx, a)
	if err != nil || updated.Name != "A renomeada" || updated.DailyCardLimit != nil || updated.Rules.AgainDelayMinutes != 30 {
		t.Fatalf("Update() = %+v, %v; want every change stored (goal removed)", updated, err)
	}

	// Renaming onto another configuration's name is refused.
	b.Name = "a RENOMEADA"
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
	if _, err := repo.Create(ctx, newTestProfile(userID, "A renomeada", nil, now)); err != nil {
		t.Errorf("re-using a deleted name error: %v", err)
	}
}

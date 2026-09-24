package mongodb

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"flashcard-backend/internal/domain/user"
	"flashcard-backend/internal/ports/repositories"
)

func TestUserRepository_CreateAndFindByEmail(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	if err := EnsureIndexes(ctx, db); err != nil {
		t.Fatalf("EnsureIndexes() error: %v", err)
	}
	repo := NewUserRepository(db)

	now := time.Now().UTC()
	created, err := repo.Create(ctx, user.New("someone@example.com", now))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create() should assign an ID")
	}

	found, err := repo.FindByEmail(ctx, "someone@example.com")
	if err != nil {
		t.Fatalf("FindByEmail() error: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("FindByEmail() ID = %q, want %q", found.ID, created.ID)
	}
}

func TestUserRepository_FindByEmail_NotFound(t *testing.T) {
	db := testDatabase(t)
	repo := NewUserRepository(db)

	_, err := repo.FindByEmail(context.Background(), "nobody@example.com")
	if err != repositories.ErrNotFound {
		t.Errorf("FindByEmail() error = %v, want ErrNotFound", err)
	}
}

func TestUserRepository_UniqueEmailIndex(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	if err := EnsureIndexes(ctx, db); err != nil {
		t.Fatalf("EnsureIndexes() error: %v", err)
	}
	repo := NewUserRepository(db)

	now := time.Now().UTC()
	if _, err := repo.Create(ctx, user.New("dup@example.com", now)); err != nil {
		t.Fatalf("first Create() error: %v", err)
	}

	_, err := repo.Create(ctx, user.New("dup@example.com", now))
	if err == nil {
		t.Fatal("second Create() with the same email should fail the unique index")
	}
}

func TestUserRepository_TouchLastAccess(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	repo := NewUserRepository(db)

	now := time.Now().UTC()
	created, err := repo.Create(ctx, user.New("touch@example.com", now))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	later := now.Add(time.Hour)
	if err := repo.TouchLastAccess(ctx, created.ID, later); err != nil {
		t.Fatalf("TouchLastAccess() error: %v", err)
	}

	found, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindByID() error: %v", err)
	}
	if !found.LastAccessAt.Truncate(time.Millisecond).Equal(later.Truncate(time.Millisecond)) {
		t.Errorf("LastAccessAt = %v, want %v", found.LastAccessAt, later)
	}
}

func TestUserRepository_ActiveStudyProfilePersists(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	if err := EnsureIndexes(ctx, db); err != nil {
		t.Fatalf("EnsureIndexes() error: %v", err)
	}
	repo := NewUserRepository(db)
	now := time.Now().UTC()

	created, err := repo.Create(ctx, user.New("profile@example.com", now))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if created.ActiveStudyProfileID != "" || created.LegacyDailyCardLimit != nil {
		t.Fatalf("a new user = %+v, want no active configuration and no legacy goal", created)
	}

	profileID := primitive.NewObjectID().Hex()
	if err := repo.SetActiveStudyProfile(ctx, created.ID, profileID, now); err != nil {
		t.Fatalf("SetActiveStudyProfile() error: %v", err)
	}
	found, _ := repo.FindByID(ctx, created.ID)
	if found.ActiveStudyProfileID != profileID {
		t.Fatalf("persisted ActiveStudyProfileID = %q, want %q", found.ActiveStudyProfileID, profileID)
	}

	if err := repo.SetActiveStudyProfile(ctx, created.ID, "", now); err != nil {
		t.Fatalf("SetActiveStudyProfile(\"\") error: %v", err)
	}
	found, _ = repo.FindByID(ctx, created.ID)
	if found.ActiveStudyProfileID != "" {
		t.Errorf("ActiveStudyProfileID after clearing = %q, want empty", found.ActiveStudyProfileID)
	}

	if err := repo.SetActiveStudyProfile(ctx, "not-an-id", profileID, now); err != repositories.ErrNotFound {
		t.Errorf("SetActiveStudyProfile() for an unknown user error = %v, want ErrNotFound", err)
	}
}

func TestUserRepository_LegacyDailyGoalIsReadAndCleared(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	if err := EnsureIndexes(ctx, db); err != nil {
		t.Fatalf("EnsureIndexes() error: %v", err)
	}
	repo := NewUserRepository(db)
	now := time.Now().UTC()

	created, _ := repo.Create(ctx, user.New("legacy@example.com", now))
	// What the app stored before study configurations existed.
	objectID, _ := primitive.ObjectIDFromHex(created.ID)
	if _, err := db.Collection(CollectionUsers).UpdateOne(ctx, bson.M{"_id": objectID},
		bson.M{"$set": bson.M{"settings": bson.M{"dailyCardLimit": 12}}}); err != nil {
		t.Fatalf("seeding the legacy setting: %v", err)
	}

	found, _ := repo.FindByID(ctx, created.ID)
	if found.LegacyDailyCardLimit == nil || *found.LegacyDailyCardLimit != 12 {
		t.Fatalf("LegacyDailyCardLimit = %v, want 12", found.LegacyDailyCardLimit)
	}

	if err := repo.ClearLegacyDailyCardLimit(ctx, created.ID, now); err != nil {
		t.Fatalf("ClearLegacyDailyCardLimit() error: %v", err)
	}
	found, _ = repo.FindByID(ctx, created.ID)
	if found.LegacyDailyCardLimit != nil {
		t.Errorf("LegacyDailyCardLimit after clearing = %d, want nil", *found.LegacyDailyCardLimit)
	}
}

func TestUserRepository_SetContinuePastGoalOn(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	if err := EnsureIndexes(ctx, db); err != nil {
		t.Fatalf("EnsureIndexes() error: %v", err)
	}
	repo := NewUserRepository(db)
	now := time.Now().UTC()

	created, err := repo.Create(ctx, user.New("continue@example.com", now))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if created.ContinuePastGoalOn != "" {
		t.Fatalf("a new user's ContinuePastGoalOn = %q, want empty", created.ContinuePastGoalOn)
	}

	if err := repo.SetContinuePastGoalOn(ctx, created.ID, "2026-01-01", now); err != nil {
		t.Fatalf("SetContinuePastGoalOn() error: %v", err)
	}
	found, _ := repo.FindByID(ctx, created.ID)
	if found.ContinuePastGoalOn != "2026-01-01" {
		t.Errorf("persisted ContinuePastGoalOn = %q, want 2026-01-01", found.ContinuePastGoalOn)
	}

	// Selecting a study configuration does not wipe the choice made for today.
	if err := repo.SetActiveStudyProfile(ctx, created.ID, primitive.NewObjectID().Hex(), now); err != nil {
		t.Fatalf("SetActiveStudyProfile() error: %v", err)
	}
	found, _ = repo.FindByID(ctx, created.ID)
	if found.ContinuePastGoalOn != "2026-01-01" {
		t.Errorf("ContinuePastGoalOn after SetActiveStudyProfile = %q, want it kept", found.ContinuePastGoalOn)
	}

	if err := repo.SetContinuePastGoalOn(ctx, "not-an-id", "2026-01-01", now); err != repositories.ErrNotFound {
		t.Errorf("SetContinuePastGoalOn() for an unknown user error = %v, want ErrNotFound", err)
	}
}

func TestUserRepository_SetPreferredLanguage(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	if err := EnsureIndexes(ctx, db); err != nil {
		t.Fatalf("EnsureIndexes() error: %v", err)
	}
	repo := NewUserRepository(db)
	now := time.Now().UTC()

	created, err := repo.Create(ctx, user.New("language@example.com", now))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if created.PreferredLanguage != "" {
		t.Fatalf("a new user's PreferredLanguage = %q, want empty", created.PreferredLanguage)
	}

	if err := repo.SetPreferredLanguage(ctx, created.ID, user.LanguageSpanish, now); err != nil {
		t.Fatalf("SetPreferredLanguage() error: %v", err)
	}
	found, _ := repo.FindByID(ctx, created.ID)
	if found.PreferredLanguage != user.LanguageSpanish {
		t.Errorf("persisted PreferredLanguage = %q, want %q", found.PreferredLanguage, user.LanguageSpanish)
	}

	if err := repo.SetPreferredLanguage(ctx, "not-an-id", user.LanguageSpanish, now); err != repositories.ErrNotFound {
		t.Errorf("SetPreferredLanguage() for an unknown user error = %v, want ErrNotFound", err)
	}
}

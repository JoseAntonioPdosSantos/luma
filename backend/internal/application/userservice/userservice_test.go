package userservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"flashcard-backend/internal/domain/user"
	"flashcard-backend/internal/ports/clock"
	"flashcard-backend/internal/ports/repositories"
)

type fakeUserRepository struct {
	byEmail           map[string]user.User
	byID              map[string]user.User
	touchedLastAccess map[string]time.Time
	nextID            int
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		byEmail:           map[string]user.User{},
		byID:              map[string]user.User{},
		touchedLastAccess: map[string]time.Time{},
	}
}

func (f *fakeUserRepository) FindByEmail(_ context.Context, normalizedEmail string) (user.User, error) {
	u, ok := f.byEmail[normalizedEmail]
	if !ok {
		return user.User{}, repositories.ErrNotFound
	}
	return u, nil
}

func (f *fakeUserRepository) FindByID(_ context.Context, id string) (user.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return user.User{}, repositories.ErrNotFound
	}
	return u, nil
}

func (f *fakeUserRepository) Create(_ context.Context, u user.User) (user.User, error) {
	f.nextID++
	u.ID = fmt.Sprintf("user-%d", f.nextID)
	f.byEmail[u.Email] = u
	f.byID[u.ID] = u
	return u, nil
}

func (f *fakeUserRepository) TouchLastAccess(_ context.Context, id string, now time.Time) error {
	f.touchedLastAccess[id] = now
	u := f.byID[id]
	u.LastAccessAt = now
	f.byID[id] = u
	f.byEmail[u.Email] = u
	return nil
}

func (f *fakeUserRepository) SetContinuePastGoalOn(_ context.Context, id, day string, _ time.Time) error {
	u, ok := f.byID[id]
	if !ok {
		return repositories.ErrNotFound
	}
	u.ContinuePastGoalOn = day
	f.byID[id] = u
	f.byEmail[u.Email] = u
	return nil
}

func (f *fakeUserRepository) SetPreferredLanguage(_ context.Context, id, language string, _ time.Time) error {
	u, ok := f.byID[id]
	if !ok {
		return repositories.ErrNotFound
	}
	u.PreferredLanguage = language
	f.byID[id] = u
	f.byEmail[u.Email] = u
	return nil
}

func (f *fakeUserRepository) SetActiveStudyProfile(_ context.Context, id, profileID string, _ time.Time) error {
	u, ok := f.byID[id]
	if !ok {
		return repositories.ErrNotFound
	}
	u.ActiveStudyProfileID = profileID
	f.byID[id] = u
	f.byEmail[u.Email] = u
	return nil
}

func (f *fakeUserRepository) ClearLegacyDailyCardLimit(_ context.Context, id string, _ time.Time) error {
	u, ok := f.byID[id]
	if !ok {
		return repositories.ErrNotFound
	}
	u.LegacyDailyCardLimit = nil
	f.byID[id] = u
	f.byEmail[u.Email] = u
	return nil
}

var fixedNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func TestGetOrCreateByEmail_CreatesNewUser(t *testing.T) {
	repo := newFakeUserRepository()
	svc := New(repo, clock.Fixed{Time: fixedNow})

	u, err := svc.GetOrCreateByEmail(context.Background(), "  New@Example.com ")
	if err != nil {
		t.Fatalf("GetOrCreateByEmail() error: %v", err)
	}
	if u.Email != "new@example.com" {
		t.Errorf("Email = %q, want normalized %q", u.Email, "new@example.com")
	}
	if u.ID == "" {
		t.Error("expected an assigned ID")
	}
}

func TestGetOrCreateByEmail_ReturnsExistingUserAndTouchesAccess(t *testing.T) {
	repo := newFakeUserRepository()
	svc := New(repo, clock.Fixed{Time: fixedNow})

	created, err := svc.GetOrCreateByEmail(context.Background(), "existing@example.com")
	if err != nil {
		t.Fatalf("first GetOrCreateByEmail() error: %v", err)
	}

	later := fixedNow.Add(time.Hour)
	svc2 := New(repo, clock.Fixed{Time: later})
	found, err := svc2.GetOrCreateByEmail(context.Background(), "existing@example.com")
	if err != nil {
		t.Fatalf("second GetOrCreateByEmail() error: %v", err)
	}

	if found.ID != created.ID {
		t.Errorf("second call ID = %q, want same as first %q", found.ID, created.ID)
	}
	if repo.touchedLastAccess[created.ID] != later {
		t.Errorf("TouchLastAccess time = %v, want %v", repo.touchedLastAccess[created.ID], later)
	}
}

func TestGetOrCreateByEmail_InvalidEmail(t *testing.T) {
	repo := newFakeUserRepository()
	svc := New(repo, clock.Fixed{Time: fixedNow})

	if _, err := svc.GetOrCreateByEmail(context.Background(), "not-an-email"); err == nil {
		t.Fatal("GetOrCreateByEmail() should reject an invalid email")
	}
}

func TestGetByID_NotFound(t *testing.T) {
	repo := newFakeUserRepository()
	svc := New(repo, clock.Fixed{Time: fixedNow})

	if _, err := svc.GetByID(context.Background(), "missing"); err == nil {
		t.Fatal("GetByID() should return an error for a missing user")
	}
}

func TestSetPreferredLanguage_Saves(t *testing.T) {
	repo := newFakeUserRepository()
	svc := New(repo, clock.Fixed{Time: fixedNow})

	created, err := svc.GetOrCreateByEmail(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateByEmail() error: %v", err)
	}

	if err := svc.SetPreferredLanguage(context.Background(), created.ID, user.LanguageFrench); err != nil {
		t.Fatalf("SetPreferredLanguage() error: %v", err)
	}

	got, err := svc.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.PreferredLanguage != user.LanguageFrench {
		t.Errorf("PreferredLanguage = %q, want %q", got.PreferredLanguage, user.LanguageFrench)
	}
}

func TestSetPreferredLanguage_InvalidLanguage(t *testing.T) {
	repo := newFakeUserRepository()
	svc := New(repo, clock.Fixed{Time: fixedNow})

	created, err := svc.GetOrCreateByEmail(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateByEmail() error: %v", err)
	}

	if err := svc.SetPreferredLanguage(context.Background(), created.ID, "klingon"); err == nil {
		t.Fatal("SetPreferredLanguage() should reject an unsupported language")
	}
}

func TestSetPreferredLanguage_NotFound(t *testing.T) {
	repo := newFakeUserRepository()
	svc := New(repo, clock.Fixed{Time: fixedNow})

	if err := svc.SetPreferredLanguage(context.Background(), "missing", user.LanguageEnglish); err == nil {
		t.Fatal("SetPreferredLanguage() should return an error for a missing user")
	}
}

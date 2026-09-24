package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"flashcard-backend/internal/adapters/authtoken"
	"flashcard-backend/internal/application/userservice"
	"flashcard-backend/internal/config"
	"flashcard-backend/internal/domain/user"
	"flashcard-backend/internal/ports/clock"
	"flashcard-backend/internal/ports/repositories"
)

type fakeUserRepository struct {
	byEmail map[string]user.User
	byID    map[string]user.User
	nextID  int
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{byEmail: map[string]user.User{}, byID: map[string]user.User{}}
}

func (f *fakeUserRepository) FindByEmail(_ context.Context, email string) (user.User, error) {
	u, ok := f.byEmail[email]
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
	u.ID = string(rune('0' + f.nextID))
	f.byEmail[u.Email] = u
	f.byID[u.ID] = u
	return u, nil
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

func (f *fakeUserRepository) TouchLastAccess(_ context.Context, id string, now time.Time) error {
	u := f.byID[id]
	u.LastAccessAt = now
	f.byID[id] = u
	f.byEmail[u.Email] = u
	return nil
}

func testDependencies() Dependencies {
	repo := newFakeUserRepository()
	fixedClock := clock.Fixed{Time: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	return Dependencies{
		Users:    userservice.New(repo, fixedClock),
		Sessions: authtoken.NewHMACManager("test-secret", time.Hour),
		Clock:    fixedClock,
	}
}

func TestNewRouter_HealthEndpoint(t *testing.T) {
	router := NewRouter(config.Config{CORSAllowedOrigins: []string{"http://localhost:5173"}}, testDependencies())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestNewRouter_CORSAllowsConfiguredOrigin(t *testing.T) {
	router := NewRouter(config.Config{CORSAllowedOrigins: []string{"http://localhost:5173"}}, testDependencies())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "http://localhost:5173")
	}
}

func TestNewRouter_CORSRejectsUnknownOrigin(t *testing.T) {
	router := NewRouter(config.Config{CORSAllowedOrigins: []string{"http://localhost:5173"}}, testDependencies())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://evil.example")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty for unknown origin", got)
	}
}

// A browser refuses a cross-origin request whose method the preflight
// response does not list, and the frontend runs on another origin than the
// API, so every method the API uses must be allowed (PUT was once missing
// and the settings page could not save).
func TestNewRouter_CORSPreflightAllowsEveryMethodTheAPIUses(t *testing.T) {
	router := NewRouter(config.Config{CORSAllowedOrigins: []string{"http://localhost:5173"}}, testDependencies())

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/me/active-study-profile", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPut)
	req.Header.Set("Access-Control-Request-Headers", "content-type")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	allowed := rec.Header().Get("Access-Control-Allow-Methods")
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		if !strings.Contains(allowed, method) {
			t.Errorf("Access-Control-Allow-Methods = %q, missing %s", allowed, method)
		}
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want true (the session cookie needs it)", got)
	}
}

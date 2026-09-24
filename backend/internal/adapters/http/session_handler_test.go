package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"flashcard-backend/internal/config"
)

func newTestRouter() http.Handler {
	return NewRouter(config.Config{CORSAllowedOrigins: []string{"http://localhost:5173"}}, testDependencies())
}

func postSession(t *testing.T, router http.Handler, email string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": email})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/session", bytes.NewReader(body))
	req.RemoteAddr = "203.0.113.1:12345"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestSession_CreateWithValidEmail(t *testing.T) {
	router := newTestRouter()

	rec := postSession(t, router, "user@example.com")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookieName {
		t.Fatalf("expected a %s cookie, got %v", sessionCookieName, cookies)
	}
	if !cookies[0].HttpOnly {
		t.Error("session cookie should be HttpOnly")
	}
}

func TestSession_CreateWithInvalidEmail(t *testing.T) {
	router := newTestRouter()

	rec := postSession(t, router, "not-an-email")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	var envelope errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decoding error body: %v", err)
	}
	if envelope.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("error code = %q, want VALIDATION_ERROR", envelope.Error.Code)
	}
}

func TestMe_WithoutCookieIsUnauthorized(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestMe_WithValidCookieReturnsUser(t *testing.T) {
	router := newTestRouter()

	createRec := postSession(t, router, "user@example.com")
	sessionCookie := createRec.Result().Cookies()[0]

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var got userResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if got.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "user@example.com")
	}
}

func TestMe_WithTamperedCookieIsUnauthorized(t *testing.T) {
	router := newTestRouter()

	createRec := postSession(t, router, "user@example.com")
	sessionCookie := createRec.Result().Cookies()[0]
	sessionCookie.Value = sessionCookie.Value + "tampered"

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestSession_DeleteClearsCookie(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/session", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge >= 0 {
		t.Fatalf("expected an expiring cookie, got %v", cookies)
	}
}

func TestSetLanguage_SavesAndReflectsInMe(t *testing.T) {
	router := newTestRouter()

	createRec := postSession(t, router, "user@example.com")
	sessionCookie := createRec.Result().Cookies()[0]

	body, _ := json.Marshal(map[string]string{"language": "fr"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/me/language", bytes.NewReader(body))
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meReq.AddCookie(sessionCookie)
	meRec := httptest.NewRecorder()
	router.ServeHTTP(meRec, meReq)

	var got userResponse
	if err := json.Unmarshal(meRec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if got.Language != "fr" {
		t.Errorf("Language = %q, want %q", got.Language, "fr")
	}
}

func TestSetLanguage_RejectsUnsupportedLanguage(t *testing.T) {
	router := newTestRouter()

	createRec := postSession(t, router, "user@example.com")
	sessionCookie := createRec.Result().Cookies()[0]

	body, _ := json.Marshal(map[string]string{"language": "klingon"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/me/language", bytes.NewReader(body))
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestSetLanguage_WithoutCookieIsUnauthorized(t *testing.T) {
	router := newTestRouter()

	body, _ := json.Marshal(map[string]string{"language": "en"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/me/language", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestSession_RateLimitReturns429AfterLimit(t *testing.T) {
	router := newTestRouter()

	var lastRec *httptest.ResponseRecorder
	for i := 0; i < sessionRateLimitPerMinute+1; i++ {
		lastRec = postSession(t, router, "user@example.com")
	}

	if lastRec.Code != http.StatusTooManyRequests {
		t.Fatalf("status after exceeding limit = %d, want %d", lastRec.Code, http.StatusTooManyRequests)
	}
}

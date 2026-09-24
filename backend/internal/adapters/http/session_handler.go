package httpapi

import (
	"encoding/json"
	"net/http"

	"flashcard-backend/internal/apperror"
	"flashcard-backend/internal/application/userservice"
	"flashcard-backend/internal/ports/clock"
	"flashcard-backend/internal/ports/session"
)

const maxSessionBodyBytes = 4 * 1024

type sessionRequest struct {
	Email string `json:"email"`
}

type userResponse struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Language string `json:"language,omitempty"`
}

type languageRequest struct {
	Language string `json:"language"`
}

const maxLanguageBodyBytes = 1 * 1024

type sessionResponse struct {
	User userResponse `json:"user"`
}

// SessionHandlers groups the HTTP handlers for the email-only access flow
// (spec section 8: POST/DELETE /api/v1/session, GET /api/v1/me).
type SessionHandlers struct {
	users        userservice.Service
	sessions     session.Manager
	clock        clock.Clock
	cookieSecure bool
}

func NewSessionHandlers(users userservice.Service, sessions session.Manager, c clock.Clock, cookieSecure bool) SessionHandlers {
	return SessionHandlers{users: users, sessions: sessions, clock: c, cookieSecure: cookieSecure}
}

// Create handles POST /api/v1/session: find-or-create the user by email
// and issue a session cookie.
func (h SessionHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var req sessionRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxSessionBodyBytes)).Decode(&req); err != nil {
		writeError(w, r, apperror.Validation("request.invalidJson.email", "request body must be valid JSON with an \"email\" field"))
		return
	}

	u, err := h.users.GetOrCreateByEmail(r.Context(), req.Email)
	if err != nil {
		writeError(w, r, err)
		return
	}

	now := h.clock.Now()
	token, expiresAt, err := h.sessions.Issue(u.ID, now)
	if err != nil {
		writeError(w, r, apperror.Internal(err))
		return
	}

	setSessionCookie(w, token, expiresAt, h.cookieSecure)
	writeJSON(w, http.StatusOK, sessionResponse{User: userResponse{ID: u.ID, Email: u.Email, Language: u.PreferredLanguage}})
}

// Delete handles DELETE /api/v1/session: clear the session cookie. It
// does not need to verify the existing session first — clearing an
// already-invalid or missing cookie is harmless and idempotent.
func (h SessionHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	clearSessionCookie(w, h.cookieSecure)
	w.WriteHeader(http.StatusNoContent)
}

// Me handles GET /api/v1/me: return the authenticated user. Must be
// wrapped with requireAuth.
func (h SessionHandlers) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, r, apperror.Unauthorized("auth.required", "authentication required"))
		return
	}

	u, err := h.users.GetByID(r.Context(), userID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, userResponse{ID: u.ID, Email: u.Email, Language: u.PreferredLanguage})
}

// SetLanguage handles PUT /api/v1/me/language: save the caller's UI
// language preference so it follows them to any device or browser. Must be
// wrapped with requireAuth.
func (h SessionHandlers) SetLanguage(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, r, apperror.Unauthorized("auth.required", "authentication required"))
		return
	}

	var req languageRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxLanguageBodyBytes)).Decode(&req); err != nil {
		writeError(w, r, apperror.Validation("request.invalidJson.language", "request body must be valid JSON with a \"language\" field"))
		return
	}

	if err := h.users.SetPreferredLanguage(r.Context(), userID, req.Language); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

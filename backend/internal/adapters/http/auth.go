package httpapi

import (
	"net/http"
	"time"

	"flashcard-backend/internal/apperror"
	"flashcard-backend/internal/ports/clock"
	"flashcard-backend/internal/ports/session"
)

const sessionCookieName = "session_token"

// requireAuth extracts and verifies the session cookie, rejecting the
// request with UNAUTHORIZED if it is missing or invalid, and otherwise
// storing the authenticated user ID in the request context (spec section
// 8: "keep the authentication implementation replaceable").
func requireAuth(manager session.Manager, c clock.Clock, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			writeError(w, r, apperror.Unauthorized("auth.required", "authentication required"))
			return
		}

		userID, err := manager.Verify(cookie.Value, c.Now())
		if err != nil {
			writeError(w, r, err)
			return
		}

		next.ServeHTTP(w, r.WithContext(contextWithUserID(r.Context(), userID)))
	}
}

func setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

package httpapi

import (
	"net/http"
	"time"

	"flashcard-backend/internal/application/deckgroupservice"
	"flashcard-backend/internal/application/deckservice"
	"flashcard-backend/internal/application/flashcardservice"
	"flashcard-backend/internal/application/profileservice"
	"flashcard-backend/internal/application/studyservice"
	"flashcard-backend/internal/application/userservice"
	"flashcard-backend/internal/config"
	"flashcard-backend/internal/ports/clock"
	"flashcard-backend/internal/ports/session"
)

// Dependencies are the application services and adapters the router
// wires into HTTP handlers. Keeping this separate from config.Config lets
// callers (main, and tests) construct routers without a real MongoDB
// connection.
type Dependencies struct {
	Users      userservice.Service
	Profiles   profileservice.Service
	Decks      deckservice.Service
	DeckGroups deckgroupservice.Service
	Flashcards flashcardservice.Service
	Study      studyservice.Service
	Sessions   session.Manager
	Clock      clock.Clock
	// DB is nil when no database connectivity check is configured (e.g.
	// in HTTP-layer unit tests).
	DB Pinger
}

const sessionRateLimitPerMinute = 10

// NewRouter builds the top-level HTTP handler for the API.
func NewRouter(cfg config.Config, deps Dependencies) http.Handler {
	sessionHandlers := NewSessionHandlers(deps.Users, deps.Sessions, deps.Clock, cfg.CookieSecure)
	sessionLimiter := newIPRateLimiter(sessionRateLimitPerMinute, time.Minute)
	deckHandlers := NewDeckHandlers(deps.Decks)
	flashcardHandlers := NewFlashcardHandlers(deps.Flashcards)
	studyHandlers := NewStudyHandlers(deps.Study)
	profileHandlers := NewStudyProfileHandlers(deps.Profiles)
	deckGroupHandlers := NewDeckGroupHandlers(deps.DeckGroups)

	auth := func(h http.HandlerFunc) http.HandlerFunc {
		return requireAuth(deps.Sessions, deps.Clock, h)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", NewHealthHandlers(deps.DB).Handle)

	mux.HandleFunc("POST /api/v1/session", withRateLimit(sessionLimiter, sessionHandlers.Create))
	mux.HandleFunc("DELETE /api/v1/session", sessionHandlers.Delete)
	mux.HandleFunc("GET /api/v1/me", auth(sessionHandlers.Me))
	mux.HandleFunc("PUT /api/v1/me/language", auth(sessionHandlers.SetLanguage))
	mux.HandleFunc("GET /api/v1/study-profiles", auth(profileHandlers.List))
	mux.HandleFunc("POST /api/v1/study-profiles", auth(profileHandlers.Create))
	mux.HandleFunc("PUT /api/v1/study-profiles/{profileID}", auth(profileHandlers.Update))
	mux.HandleFunc("DELETE /api/v1/study-profiles/{profileID}", auth(profileHandlers.Delete))
	mux.HandleFunc("PUT /api/v1/me/active-study-profile", auth(profileHandlers.SetActive))
	mux.HandleFunc("PUT /api/v1/decks/{deckID}/study-profile", auth(profileHandlers.SetDeckProfile))

	mux.HandleFunc("GET /api/v1/decks", auth(deckHandlers.List))
	mux.HandleFunc("POST /api/v1/decks", auth(deckHandlers.Create))
	mux.HandleFunc("GET /api/v1/decks/{deckID}", auth(deckHandlers.Get))
	mux.HandleFunc("PATCH /api/v1/decks/{deckID}", auth(deckHandlers.Update))
	mux.HandleFunc("DELETE /api/v1/decks/{deckID}", auth(deckHandlers.Archive))
	mux.HandleFunc("POST /api/v1/decks/{deckID}/restore", auth(deckHandlers.Restore))
	mux.HandleFunc("GET /api/v1/decks/{deckID}/stats", auth(deckHandlers.Stats))
	mux.HandleFunc("PUT /api/v1/decks/{deckID}/group", auth(deckGroupHandlers.SetDeckGroup))

	mux.HandleFunc("GET /api/v1/deck-groups", auth(deckGroupHandlers.List))
	mux.HandleFunc("POST /api/v1/deck-groups", auth(deckGroupHandlers.Create))
	mux.HandleFunc("PUT /api/v1/deck-groups/{groupID}", auth(deckGroupHandlers.Update))
	mux.HandleFunc("DELETE /api/v1/deck-groups/{groupID}", auth(deckGroupHandlers.Delete))

	mux.HandleFunc("GET /api/v1/decks/{deckID}/flashcards", auth(flashcardHandlers.List))
	mux.HandleFunc("POST /api/v1/decks/{deckID}/flashcards", auth(flashcardHandlers.Create))
	mux.HandleFunc("GET /api/v1/archived-flashcards", auth(flashcardHandlers.ListArchived))
	mux.HandleFunc("DELETE /api/v1/archived-flashcards/{flashcardID}", auth(flashcardHandlers.DeleteArchived))
	mux.HandleFunc("DELETE /api/v1/archived-decks/{deckID}", auth(deckHandlers.DeleteArchived))
	mux.HandleFunc("GET /api/v1/flashcards/{flashcardID}", auth(flashcardHandlers.Get))
	mux.HandleFunc("PATCH /api/v1/flashcards/{flashcardID}", auth(flashcardHandlers.Update))
	mux.HandleFunc("DELETE /api/v1/flashcards/{flashcardID}", auth(flashcardHandlers.Archive))
	mux.HandleFunc("POST /api/v1/flashcards/{flashcardID}/restore", auth(flashcardHandlers.Restore))
	mux.HandleFunc("POST /api/v1/flashcards/{flashcardID}/audio", auth(flashcardHandlers.UploadAudio))
	mux.HandleFunc("GET /api/v1/flashcards/{flashcardID}/audio", auth(flashcardHandlers.GetAudio))
	mux.HandleFunc("DELETE /api/v1/flashcards/{flashcardID}/audio", auth(flashcardHandlers.DeleteAudio))

	mux.HandleFunc("GET /api/v1/study/progress", auth(studyHandlers.Progress))
	mux.HandleFunc("POST /api/v1/study/continue-past-goal", auth(studyHandlers.ContinuePastGoal))
	mux.HandleFunc("GET /api/v1/decks/{deckID}/due-flashcards", auth(studyHandlers.DueCards))
	mux.HandleFunc("POST /api/v1/flashcards/{flashcardID}/reviews", auth(studyHandlers.SubmitReview))

	var handler http.Handler = mux
	handler = withCORS(cfg.CORSAllowedOrigins, handler)
	handler = withAccessLog(handler)
	handler = withRequestID(handler)
	return handler
}

func withCORS(allowedOrigins []string, next http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := allowed[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

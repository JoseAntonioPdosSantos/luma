package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"flashcard-backend/internal/apperror"
	"flashcard-backend/internal/application/studyservice"
	"flashcard-backend/internal/domain/study"
)

type reviewRequest struct {
	Rating         string `json:"rating"`
	ResponseTimeMs int64  `json:"responseTimeMs"`
	HintUsed       bool   `json:"hintUsed"`
}

// StudyHandlers implements the study flow endpoints.
//
// This intentionally does not implement the stateful study-session
// resource sketched in spec section 13
// (POST/GET .../study-sessions, .../reviews, .../finish): that section
// explicitly allows a simpler design as long as ownership, consistency,
// and state transitions stay clear, and a "session" adds no real value
// here — a new card is already due the instant it is created (see
// studyservice's package doc), and ReviewEvent itself has no session ID
// in the domain model. Instead:
//
//	GET  /api/v1/decks/{deckID}/due-flashcards   the next batch to study
//	POST /api/v1/flashcards/{flashcardID}/reviews  submit a rating
type StudyHandlers struct {
	study studyservice.Service
}

func NewStudyHandlers(study studyservice.Service) StudyHandlers {
	return StudyHandlers{study: study}
}

// DueCards handles GET /api/v1/decks/{deckID}/due-flashcards.
func (h StudyHandlers) DueCards(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	deckID := r.PathValue("deckID")

	limit := studyservice.DefaultDueCardsLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeError(w, r, apperror.Validation("study.dueCards.limit.invalid", "limit must be a positive integer"))
			return
		}
		limit = parsed
	}

	loc, err := locationFromQuery(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	opts := studyservice.DueOptions{Location: loc, IgnoreDailyLimit: r.URL.Query().Get("ignoreDailyLimit") == "true"}

	cards, err := h.study.DueCards(r.Context(), userID, deckID, limit, opts)
	if err != nil {
		writeError(w, r, err)
		return
	}

	response := make([]flashcardResponse, 0, len(cards))
	for _, c := range cards {
		response = append(response, toFlashcardResponse(c))
	}
	writeJSON(w, http.StatusOK, response)
}

// SubmitReview handles POST /api/v1/flashcards/{flashcardID}/reviews.
func (h StudyHandlers) SubmitReview(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	id := r.PathValue("flashcardID")

	var req reviewRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxDeckBodyBytes)).Decode(&req); err != nil {
		writeError(w, r, apperror.Validation("request.invalidJson", "request body must be valid JSON"))
		return
	}

	rating, err := study.ParseRating(req.Rating)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if req.ResponseTimeMs < 0 {
		writeError(w, r, apperror.Validation("study.responseTimeMs.negative", "responseTimeMs must not be negative"))
		return
	}

	updated, err := h.study.RecordReview(r.Context(), userID, id, rating, req.ResponseTimeMs, req.HintUsed)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toFlashcardResponse(updated))
}

type progressResponse struct {
	// DailyCardLimit and Remaining are null when the user has no daily goal.
	DailyCardLimit *int `json:"dailyCardLimit"`
	StudiedToday   int  `json:"studiedToday"`
	Remaining      *int `json:"remaining"`
	// ContinuingPastGoal is true once the user chose, today, to keep
	// studying past the goal.
	ContinuingPastGoal bool `json:"continuingPastGoal"`
}

// Progress handles GET /api/v1/study/progress: the daily goal that applies and
// how many different cards were studied today toward it. With the optional
// deckId it is the goal of that deck (its own configuration's, counting that
// deck only, or else the general one); without it, the general goal, counting
// every deck that has no configuration of its own. The optional tz parameter
// defines what "today" is.
func (h StudyHandlers) Progress(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	loc, err := locationFromQuery(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	progress, err := h.study.Progress(r.Context(), userID, r.URL.Query().Get("deckId"), loc)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, progressResponse{
		DailyCardLimit: progress.DailyCardLimit,
		StudiedToday:   progress.StudiedToday,
		Remaining:      progress.Remaining,

		ContinuingPastGoal: progress.ContinuingPastGoal,
	})
}

// ContinuePastGoal handles POST /api/v1/study/continue-past-goal: the user
// chose to keep studying after reaching the daily goal. It lasts for the
// rest of the day (the optional tz parameter says which day) on every device.
func (h StudyHandlers) ContinuePastGoal(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	loc, err := locationFromQuery(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if err := h.study.ContinuePastGoal(r.Context(), userID, r.URL.Query().Get("deckId"), loc); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

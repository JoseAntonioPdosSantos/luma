package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"flashcard-backend/internal/apperror"
	"flashcard-backend/internal/application/deckservice"
	"flashcard-backend/internal/domain/deck"
)

const maxDeckBodyBytes = 8 * 1024

type deckRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type deckResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	TotalCards  int    `json:"totalCards"`
	DueCards    int    `json:"dueCards"`
	// NextDueAt (RFC 3339) is when the next card comes due; present only on
	// GET /decks/{id}, and only if a card is scheduled in the future.
	NextDueAt string `json:"nextDueAt,omitempty"`
	// StudyProfileID is the study configuration this deck uses instead of the
	// general one; absent when it follows the general configuration.
	StudyProfileID string `json:"studyProfileId,omitempty"`
	// GroupID is the deck group this deck is filed under; absent when it is
	// not in any group.
	GroupID string `json:"groupId,omitempty"`
}

func toDeckResponse(counts deckservice.DeckWithCounts) deckResponse {
	resp := deckResponse{
		ID:          counts.Deck.ID,
		Name:        counts.Deck.Name,
		Description: counts.Deck.Description,
		TotalCards:  counts.TotalCards,
		DueCards:    counts.DueCards,

		StudyProfileID: counts.Deck.StudyProfileID,
		GroupID:        counts.Deck.GroupID,
	}
	if counts.NextDueAt != nil {
		resp.NextDueAt = counts.NextDueAt.UTC().Format(time.RFC3339)
	}
	return resp
}

func toDeckResponseNoCounts(d deck.Deck) deckResponse {
	return deckResponse{ID: d.ID, Name: d.Name, Description: d.Description, GroupID: d.GroupID}
}

// DeckHandlers implements the deck endpoints from spec section 13.
type DeckHandlers struct {
	decks deckservice.Service
}

func NewDeckHandlers(decks deckservice.Service) DeckHandlers {
	return DeckHandlers{decks: decks}
}

// List handles GET /api/v1/decks.
func (h DeckHandlers) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	list := h.decks.ListActiveWithCounts
	if r.URL.Query().Get("archived") == "true" {
		list = h.decks.ListArchivedWithCounts
	}

	decksWithCounts, err := list(r.Context(), userID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	response := make([]deckResponse, 0, len(decksWithCounts))
	for _, d := range decksWithCounts {
		response = append(response, toDeckResponse(d))
	}
	writeJSON(w, http.StatusOK, response)
}

// Create handles POST /api/v1/decks.
func (h DeckHandlers) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	var req deckRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxDeckBodyBytes)).Decode(&req); err != nil {
		writeError(w, r, apperror.Validation("request.invalidJson", "request body must be valid JSON"))
		return
	}

	created, err := h.decks.Create(r.Context(), userID, req.Name, req.Description)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toDeckResponseNoCounts(created))
}

// Get handles GET /api/v1/decks/{deckID}.
func (h DeckHandlers) Get(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	deckID := r.PathValue("deckID")

	withCounts, err := h.decks.GetWithCounts(r.Context(), userID, deckID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toDeckResponse(withCounts))
}

// Update handles PATCH /api/v1/decks/{deckID}.
func (h DeckHandlers) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	deckID := r.PathValue("deckID")

	var req deckRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxDeckBodyBytes)).Decode(&req); err != nil {
		writeError(w, r, apperror.Validation("request.invalidJson", "request body must be valid JSON"))
		return
	}

	updated, err := h.decks.Update(r.Context(), userID, deckID, req.Name, req.Description)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toDeckResponseNoCounts(updated))
}

type dailyPerformanceResponse struct {
	Date            string `json:"date"`
	ReviewCount     int    `json:"reviewCount"`
	AccuracyPercent int    `json:"accuracyPercent"`
	HintCount       int    `json:"hintCount"`
}

type deckStatsResponse struct {
	DeckID           string                     `json:"deckId"`
	Name             string                     `json:"name"`
	TotalCards       int                        `json:"totalCards"`
	NewCards         int                        `json:"newCards"`
	LearningCards    int                        `json:"learningCards"`
	MasteredCards    int                        `json:"masteredCards"`
	MasteryPercent   int                        `json:"masteryPercent"`
	StudiedToday     int                        `json:"studiedToday"`
	StudiedLast7Days int                        `json:"studiedLast7Days"`
	StudiedTotal     int                        `json:"studiedTotal"`
	DailyPerformance []dailyPerformanceResponse `json:"dailyPerformance"`
}

func toDeckStatsResponse(s deckservice.DeckStats) deckStatsResponse {
	performance := make([]dailyPerformanceResponse, 0, len(s.DailyPerformance))
	for _, p := range s.DailyPerformance {
		performance = append(performance, dailyPerformanceResponse{
			Date:            p.Date.Format("2006-01-02"),
			ReviewCount:     p.ReviewCount,
			AccuracyPercent: p.AccuracyPercent,
			HintCount:       p.HintCount,
		})
	}
	return deckStatsResponse{
		DeckID:           s.Deck.ID,
		Name:             s.Deck.Name,
		TotalCards:       s.TotalCards,
		NewCards:         s.NewCards,
		LearningCards:    s.LearningCards,
		MasteredCards:    s.MasteredCards,
		MasteryPercent:   s.MasteryPercent,
		StudiedToday:     s.StudiedToday,
		StudiedLast7Days: s.StudiedLast7Days,
		StudiedTotal:     s.StudiedTotal,
		DailyPerformance: performance,
	}
}

// Stats handles GET /api/v1/decks/{deckID}/stats. Not part of the
// written product spec's MVP (see deckservice.DeckStats's doc comment);
// added to match the project's original design mockup.
func (h DeckHandlers) Stats(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	deckID := r.PathValue("deckID")

	// The optional tz parameter makes "today" and the per-day chart follow
	// the user's local days; without it days are UTC.
	loc, err := locationFromQuery(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	stats, err := h.decks.GetStats(r.Context(), userID, deckID, loc)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toDeckStatsResponse(stats))
}

// Archive handles DELETE /api/v1/decks/{deckID}. It archives the deck
// rather than permanently deleting it, preserving its flashcards and
// review history (spec section 6: "Archived decks should not appear in
// the default dashboard").
func (h DeckHandlers) Archive(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	deckID := r.PathValue("deckID")

	if err := h.decks.Archive(r.Context(), userID, deckID); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Restore handles POST /api/v1/decks/{deckID}/restore, un-archiving a
// deck (idempotent).
func (h DeckHandlers) Restore(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	deckID := r.PathValue("deckID")

	if err := h.decks.Restore(r.Context(), userID, deckID); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteArchived handles DELETE /api/v1/archived-decks/{deckID}: it
// permanently deletes a deck that is already archived, along with its
// flashcards and review history. Not reversible.
func (h DeckHandlers) DeleteArchived(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	deckID := r.PathValue("deckID")

	if err := h.decks.DeleteArchived(r.Context(), userID, deckID); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

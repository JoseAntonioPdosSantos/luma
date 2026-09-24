package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"flashcard-backend/internal/apperror"
	"flashcard-backend/internal/application/flashcardservice"
	"flashcard-backend/internal/domain/flashcard"
)

const maxFlashcardBodyBytes = 32 * 1024

type extendedExampleRequest struct {
	Text        string `json:"text"`
	Translation string `json:"translation"`
}

type flashcardRequest struct {
	Question        string                  `json:"question"`
	Answer          string                  `json:"answer"`
	Hint            string                  `json:"hint"`
	ExtendedExample *extendedExampleRequest `json:"extendedExample"`
}

func (r flashcardRequest) toDraft() flashcardservice.Draft {
	draft := flashcardservice.Draft{Question: r.Question, Answer: r.Answer, Hint: r.Hint}
	if r.ExtendedExample != nil {
		draft.ExtendedExampleText = r.ExtendedExample.Text
		draft.ExtendedTranslation = r.ExtendedExample.Translation
	}
	return draft
}

type audioResponse struct {
	ContentType string `json:"contentType"`
	DurationMs  int64  `json:"durationMs"`
	URL         string `json:"url"`
}

type flashcardResponse struct {
	ID              string                   `json:"id"`
	DeckID          string                   `json:"deckId"`
	Question        string                   `json:"question"`
	Answer          string                   `json:"answer"`
	Hint            string                   `json:"hint,omitempty"`
	ExtendedExample *extendedExampleResponse `json:"extendedExample,omitempty"`
	Audio           *audioResponse           `json:"audio,omitempty"`
	Scheduling      schedulingResponse       `json:"scheduling"`
}

type extendedExampleResponse struct {
	Text        string `json:"text"`
	Translation string `json:"translation"`
}

type schedulingResponse struct {
	State        string `json:"state"`
	DueAt        string `json:"dueAt"`
	IntervalDays int    `json:"intervalDays"`
	Repetitions  int    `json:"repetitions"`
	Lapses       int    `json:"lapses"`
}

func toFlashcardResponse(f flashcard.Flashcard) flashcardResponse {
	resp := flashcardResponse{
		ID:       f.ID,
		DeckID:   f.DeckID,
		Question: f.Question,
		Answer:   f.Answer,
		Hint:     f.Hint,
		Scheduling: schedulingResponse{
			State:        string(f.Scheduling.State),
			DueAt:        f.Scheduling.DueAt.Format(timeFormat),
			IntervalDays: f.Scheduling.IntervalDays,
			Repetitions:  f.Scheduling.Repetitions,
			Lapses:       f.Scheduling.Lapses,
		},
	}
	if f.ExtendedExample != nil {
		resp.ExtendedExample = &extendedExampleResponse{
			Text:        f.ExtendedExample.Text,
			Translation: f.ExtendedExample.Translation,
		}
	}
	if f.Audio != nil {
		resp.Audio = &audioResponse{
			ContentType: f.Audio.ContentType,
			DurationMs:  f.Audio.DurationMs,
			URL:         "/api/v1/flashcards/" + f.ID + "/audio",
		}
	}
	return resp
}

const timeFormat = "2006-01-02T15:04:05Z07:00"

// FlashcardHandlers implements the flashcard endpoints from spec section
// 13, including audio (see flashcard_audio_handler.go).
type FlashcardHandlers struct {
	flashcards flashcardservice.Service
}

func NewFlashcardHandlers(flashcards flashcardservice.Service) FlashcardHandlers {
	return FlashcardHandlers{flashcards: flashcards}
}

type flashcardListResponse struct {
	Items []flashcardResponse `json:"items"`
	Total int                 `json:"total"`
}

// List handles GET /api/v1/decks/{deckID}/flashcards. Query parameters:
// archived=true (archived cards instead of active ones), q (search text),
// limit (page size, default 30, max 100) and offset.
func (h FlashcardHandlers) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	deckID := r.PathValue("deckID")
	params := r.URL.Query()

	limit, err := optionalInt(params.Get("limit"), 0, "limit", "flashcard.list.limit.invalid")
	if err != nil {
		writeError(w, r, err)
		return
	}
	if params.Get("limit") != "" && limit <= 0 {
		writeError(w, r, apperror.Validation("flashcard.list.limit.invalid", "limit must be a positive integer"))
		return
	}
	offset, err := optionalInt(params.Get("offset"), 0, "offset", "flashcard.list.offset.invalid")
	if err != nil {
		writeError(w, r, err)
		return
	}

	page, err := h.flashcards.ListPage(r.Context(), userID, deckID, params.Get("archived") == "true", params.Get("q"), limit, offset)
	if err != nil {
		writeError(w, r, err)
		return
	}

	response := flashcardListResponse{Items: make([]flashcardResponse, 0, len(page.Cards)), Total: page.Total}
	for _, c := range page.Cards {
		response.Items = append(response.Items, toFlashcardResponse(c))
	}
	writeJSON(w, http.StatusOK, response)
}

// optionalInt parses an integer query parameter, returning fallback when it
// is absent.
func optionalInt(raw string, fallback int, name string, key string) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, apperror.Validation(key, name+" must be an integer")
	}
	return n, nil
}

// Create handles POST /api/v1/decks/{deckID}/flashcards.
func (h FlashcardHandlers) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	deckID := r.PathValue("deckID")

	var req flashcardRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxFlashcardBodyBytes)).Decode(&req); err != nil {
		writeError(w, r, apperror.Validation("request.invalidJson", "request body must be valid JSON"))
		return
	}

	created, err := h.flashcards.Create(r.Context(), userID, deckID, req.toDraft())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toFlashcardResponse(created))
}

// Get handles GET /api/v1/flashcards/{flashcardID}.
func (h FlashcardHandlers) Get(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	id := r.PathValue("flashcardID")

	card, err := h.flashcards.Get(r.Context(), userID, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toFlashcardResponse(card))
}

// Update handles PATCH /api/v1/flashcards/{flashcardID}.
func (h FlashcardHandlers) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	id := r.PathValue("flashcardID")

	var req flashcardRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxFlashcardBodyBytes)).Decode(&req); err != nil {
		writeError(w, r, apperror.Validation("request.invalidJson", "request body must be valid JSON"))
		return
	}

	updated, err := h.flashcards.Update(r.Context(), userID, id, req.toDraft())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toFlashcardResponse(updated))
}

// Archive handles DELETE /api/v1/flashcards/{flashcardID}. It archives
// rather than permanently deletes, preserving review history (spec
// section 6).
func (h FlashcardHandlers) Archive(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	id := r.PathValue("flashcardID")

	if err := h.flashcards.Archive(r.Context(), userID, id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Restore handles POST /api/v1/flashcards/{flashcardID}/restore,
// un-archiving a flashcard (idempotent).
func (h FlashcardHandlers) Restore(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	id := r.PathValue("flashcardID")

	if err := h.flashcards.Restore(r.Context(), userID, id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type archivedFlashcardResponse struct {
	flashcardResponse
	DeckName string `json:"deckName"`
}

// ListArchived handles GET /api/v1/archived-flashcards: the caller's
// archived flashcards across all of their active decks.
func (h FlashcardHandlers) ListArchived(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	archived, err := h.flashcards.ListArchivedAcrossDecks(r.Context(), userID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	response := make([]archivedFlashcardResponse, 0, len(archived))
	for _, a := range archived {
		response = append(response, archivedFlashcardResponse{
			flashcardResponse: toFlashcardResponse(a.Card),
			DeckName:          a.DeckName,
		})
	}
	writeJSON(w, http.StatusOK, response)
}

// DeleteArchived handles DELETE /api/v1/archived-flashcards/{flashcardID}:
// it permanently deletes a flashcard that is already archived, along with
// its audio. Not reversible.
func (h FlashcardHandlers) DeleteArchived(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	id := r.PathValue("flashcardID")

	if err := h.flashcards.DeleteArchived(r.Context(), userID, id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

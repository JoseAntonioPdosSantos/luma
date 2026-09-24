package httpapi

import (
	"io"
	"net/http"
)

// UploadAudio handles POST /api/v1/flashcards/{flashcardID}/audio.
//
// The audio content type comes from the request's Content-Type header,
// never from a client-supplied filename (no multipart form is used
// here — the request body is the raw audio bytes). The body is streamed
// directly into storage; size and content-type validation happen in
// flashcardservice.UploadAudio.
func (h FlashcardHandlers) UploadAudio(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	id := r.PathValue("flashcardID")

	updated, err := h.flashcards.UploadAudio(r.Context(), userID, id, r.Header.Get("Content-Type"), r.Body)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toFlashcardResponse(updated))
}

// GetAudio handles GET /api/v1/flashcards/{flashcardID}/audio: an
// authenticated, ownership-checked audio playback endpoint (spec section
// 7). It streams the file rather than buffering it in memory.
func (h FlashcardHandlers) GetAudio(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	id := r.PathValue("flashcardID")

	audio, stream, err := h.flashcards.GetAudio(r.Context(), userID, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	defer stream.Close()

	w.Header().Set("Content-Type", audio.ContentType)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, stream)
}

// DeleteAudio handles DELETE /api/v1/flashcards/{flashcardID}/audio. It
// is idempotent: deleting when there is no audio still succeeds.
func (h FlashcardHandlers) DeleteAudio(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	id := r.PathValue("flashcardID")

	if err := h.flashcards.DeleteAudio(r.Context(), userID, id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

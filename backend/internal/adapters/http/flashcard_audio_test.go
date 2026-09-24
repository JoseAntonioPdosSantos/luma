package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func doAudioUpload(t *testing.T, router http.Handler, flashcardID string, cookie *http.Cookie, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/flashcards/"+flashcardID+"/audio", strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func createCardForAudioTest(t *testing.T, router http.Handler, cookie *http.Cookie) flashcardResponse {
	t.Helper()
	deckRec := doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "English"})
	var deckCreated deckResponse
	mustDecode(t, deckRec, &deckCreated)

	cardRec := doJSON(t, router, http.MethodPost, "/api/v1/decks/"+deckCreated.ID+"/flashcards", cookie, flashcardRequest{
		Question: "Q", Answer: "A",
	})
	var card flashcardResponse
	mustDecode(t, cardRec, &card)
	return card
}

func TestAudio_UploadThenPlayback(t *testing.T) {
	router, cookie := mongoTestRouter(t)
	card := createCardForAudioTest(t, router, cookie)

	uploadRec := doAudioUpload(t, router, card.ID, cookie, "audio/mpeg", "fake mp3 bytes")
	if uploadRec.Code != http.StatusOK {
		t.Fatalf("upload status = %d, want %d, body=%s", uploadRec.Code, http.StatusOK, uploadRec.Body.String())
	}
	var updated flashcardResponse
	mustDecode(t, uploadRec, &updated)
	if updated.Audio == nil || updated.Audio.ContentType != "audio/mpeg" {
		t.Fatalf("Audio = %+v", updated.Audio)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/flashcards/"+card.ID+"/audio", nil)
	getReq.AddCookie(cookie)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getRec.Code, http.StatusOK)
	}
	if ct := getRec.Header().Get("Content-Type"); ct != "audio/mpeg" {
		t.Errorf("Content-Type = %q, want %q", ct, "audio/mpeg")
	}
	if getRec.Body.String() != "fake mp3 bytes" {
		t.Errorf("body = %q, want %q", getRec.Body.String(), "fake mp3 bytes")
	}
}

func TestAudio_UploadRejectsUnsupportedContentType(t *testing.T) {
	router, cookie := mongoTestRouter(t)
	card := createCardForAudioTest(t, router, cookie)

	rec := doAudioUpload(t, router, card.ID, cookie, "video/mp4", "data")
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusUnsupportedMediaType, rec.Body.String())
	}
}

func TestAudio_UploadRejectsOversized(t *testing.T) {
	router, cookie := mongoTestRouter(t)
	card := createCardForAudioTest(t, router, cookie)

	oversized := strings.Repeat("a", testMaxAudioSizeBytes+1)
	rec := doAudioUpload(t, router, card.ID, cookie, "audio/mpeg", oversized)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusRequestEntityTooLarge, rec.Body.String())
	}
}

func TestAudio_UploadRejectsWhenFlashcardNotOwned(t *testing.T) {
	router, ownerCookie := mongoTestRouter(t)
	_, otherCookie := mongoTestRouter(t)
	card := createCardForAudioTest(t, router, ownerCookie)

	rec := doAudioUpload(t, router, card.ID, otherCookie, "audio/mpeg", "data")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestAudio_GetWithoutAudioIsNotFound(t *testing.T) {
	router, cookie := mongoTestRouter(t)
	card := createCardForAudioTest(t, router, cookie)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/flashcards/"+card.ID+"/audio", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestAudio_ReplaceThenDelete(t *testing.T) {
	router, cookie := mongoTestRouter(t)
	card := createCardForAudioTest(t, router, cookie)

	doAudioUpload(t, router, card.ID, cookie, "audio/mpeg", "first")
	uploadRec := doAudioUpload(t, router, card.ID, cookie, "audio/wav", "second")
	if uploadRec.Code != http.StatusOK {
		t.Fatalf("replace status = %d, want %d, body=%s", uploadRec.Code, http.StatusOK, uploadRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/flashcards/"+card.ID+"/audio", nil)
	getReq.AddCookie(cookie)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	if getRec.Body.String() != "second" {
		t.Fatalf("body after replace = %q, want %q", getRec.Body.String(), "second")
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/flashcards/"+card.ID+"/audio", nil)
	deleteReq.AddCookie(cookie)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteRec.Code, http.StatusNoContent)
	}

	getAfterDeleteReq := httptest.NewRequest(http.MethodGet, "/api/v1/flashcards/"+card.ID+"/audio", nil)
	getAfterDeleteReq.AddCookie(cookie)
	getAfterDeleteRec := httptest.NewRecorder()
	router.ServeHTTP(getAfterDeleteRec, getAfterDeleteReq)
	if getAfterDeleteRec.Code != http.StatusNotFound {
		t.Fatalf("get after delete status = %d, want %d", getAfterDeleteRec.Code, http.StatusNotFound)
	}
}

func TestAudio_DeleteIsIdempotent(t *testing.T) {
	router, cookie := mongoTestRouter(t)
	card := createCardForAudioTest(t, router, cookie)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/flashcards/"+card.ID+"/audio", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
}

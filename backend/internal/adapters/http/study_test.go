package httpapi

import (
	"net/http"
	"testing"
)

func TestDueCards_ReturnsNewlyCreatedCard(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	deckRec := doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "English"})
	var deckCreated deckResponse
	mustDecode(t, deckRec, &deckCreated)

	doJSON(t, router, http.MethodPost, "/api/v1/decks/"+deckCreated.ID+"/flashcards", cookie, flashcardRequest{
		Question: "Q", Answer: "A",
	})

	dueRec := doJSON(t, router, http.MethodGet, "/api/v1/decks/"+deckCreated.ID+"/due-flashcards", cookie, nil)
	if dueRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", dueRec.Code, http.StatusOK, dueRec.Body.String())
	}
	var due []flashcardResponse
	mustDecode(t, dueRec, &due)
	if len(due) != 1 {
		t.Fatalf("due cards = %+v, want 1 card", due)
	}
	if due[0].Scheduling.State != "new" {
		t.Errorf("Scheduling.State = %q, want %q", due[0].Scheduling.State, "new")
	}
}

func TestDueCards_RejectsWhenDeckNotOwned(t *testing.T) {
	router, ownerCookie := mongoTestRouter(t)
	_, otherCookie := mongoTestRouter(t)

	deckRec := doJSON(t, router, http.MethodPost, "/api/v1/decks", ownerCookie, deckRequest{Name: "English"})
	var deckCreated deckResponse
	mustDecode(t, deckRec, &deckCreated)

	rec := doJSON(t, router, http.MethodGet, "/api/v1/decks/"+deckCreated.ID+"/due-flashcards", otherCookie, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestSubmitReview_UpdatesSchedulingAndRemovesFromDueList(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	deckRec := doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "English"})
	var deckCreated deckResponse
	mustDecode(t, deckRec, &deckCreated)

	cardRec := doJSON(t, router, http.MethodPost, "/api/v1/decks/"+deckCreated.ID+"/flashcards", cookie, flashcardRequest{
		Question: "Q", Answer: "A",
	})
	var card flashcardResponse
	mustDecode(t, cardRec, &card)

	reviewRec := doJSON(t, router, http.MethodPost, "/api/v1/flashcards/"+card.ID+"/reviews", cookie, reviewRequest{
		Rating: "good", ResponseTimeMs: 1200,
	})
	if reviewRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", reviewRec.Code, http.StatusOK, reviewRec.Body.String())
	}
	var updated flashcardResponse
	mustDecode(t, reviewRec, &updated)
	if updated.Scheduling.State != "review" {
		t.Errorf("Scheduling.State = %q, want %q", updated.Scheduling.State, "review")
	}
	if updated.Scheduling.IntervalDays != 3 {
		t.Errorf("Scheduling.IntervalDays = %d, want 3", updated.Scheduling.IntervalDays)
	}

	dueRec := doJSON(t, router, http.MethodGet, "/api/v1/decks/"+deckCreated.ID+"/due-flashcards", cookie, nil)
	var due []flashcardResponse
	mustDecode(t, dueRec, &due)
	if len(due) != 0 {
		t.Fatalf("due cards after review = %+v, want empty (next review is 3 days out)", due)
	}
}

func TestSubmitReview_RejectsInvalidRating(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	deckRec := doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "English"})
	var deckCreated deckResponse
	mustDecode(t, deckRec, &deckCreated)

	cardRec := doJSON(t, router, http.MethodPost, "/api/v1/decks/"+deckCreated.ID+"/flashcards", cookie, flashcardRequest{
		Question: "Q", Answer: "A",
	})
	var card flashcardResponse
	mustDecode(t, cardRec, &card)

	rec := doJSON(t, router, http.MethodPost, "/api/v1/flashcards/"+card.ID+"/reviews", cookie, reviewRequest{
		Rating: "terrible",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestSubmitReview_RejectsWhenFlashcardNotOwned(t *testing.T) {
	router, ownerCookie := mongoTestRouter(t)
	_, otherCookie := mongoTestRouter(t)

	deckRec := doJSON(t, router, http.MethodPost, "/api/v1/decks", ownerCookie, deckRequest{Name: "English"})
	var deckCreated deckResponse
	mustDecode(t, deckRec, &deckCreated)

	cardRec := doJSON(t, router, http.MethodPost, "/api/v1/decks/"+deckCreated.ID+"/flashcards", ownerCookie, flashcardRequest{
		Question: "Q", Answer: "A",
	})
	var card flashcardResponse
	mustDecode(t, cardRec, &card)

	rec := doJSON(t, router, http.MethodPost, "/api/v1/flashcards/"+card.ID+"/reviews", otherCookie, reviewRequest{
		Rating: "good",
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestSubmitReview_RejectsArchivedFlashcard(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	deckRec := doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "English"})
	var deckCreated deckResponse
	mustDecode(t, deckRec, &deckCreated)

	cardRec := doJSON(t, router, http.MethodPost, "/api/v1/decks/"+deckCreated.ID+"/flashcards", cookie, flashcardRequest{
		Question: "Q", Answer: "A",
	})
	var card flashcardResponse
	mustDecode(t, cardRec, &card)

	archiveRec := doJSON(t, router, http.MethodDelete, "/api/v1/flashcards/"+card.ID, cookie, nil)
	if archiveRec.Code != http.StatusNoContent {
		t.Fatalf("archive status = %d, want %d", archiveRec.Code, http.StatusNoContent)
	}

	rec := doJSON(t, router, http.MethodPost, "/api/v1/flashcards/"+card.ID+"/reviews", cookie, reviewRequest{
		Rating: "good",
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestSubmitReview_HintUsageAppearsInStats(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	var d deckResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "English"}), &d)
	var withHint, withoutHint flashcardResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks/"+d.ID+"/flashcards", cookie,
		flashcardRequest{Question: "Q1", Answer: "A", Hint: "h"}), &withHint)
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks/"+d.ID+"/flashcards", cookie,
		flashcardRequest{Question: "Q2", Answer: "A"}), &withoutHint)

	for _, tc := range []struct {
		id       string
		hintUsed bool
	}{{withHint.ID, true}, {withoutHint.ID, false}} {
		rec := doJSON(t, router, http.MethodPost, "/api/v1/flashcards/"+tc.id+"/reviews", cookie,
			reviewRequest{Rating: "good", HintUsed: tc.hintUsed})
		if rec.Code != http.StatusOK {
			t.Fatalf("review status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}
	}

	rec := doJSON(t, router, http.MethodGet, "/api/v1/decks/"+d.ID+"/stats", cookie, nil)
	var stats struct {
		DailyPerformance []struct {
			ReviewCount int `json:"reviewCount"`
			HintCount   int `json:"hintCount"`
		} `json:"dailyPerformance"`
	}
	mustDecode(t, rec, &stats)
	if len(stats.DailyPerformance) != 1 || stats.DailyPerformance[0].ReviewCount != 2 || stats.DailyPerformance[0].HintCount != 1 {
		t.Errorf("dailyPerformance = %+v, want one day with 2 reviews and 1 hint", stats.DailyPerformance)
	}
}

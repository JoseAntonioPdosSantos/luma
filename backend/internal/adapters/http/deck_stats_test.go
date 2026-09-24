package httpapi

import (
	"net/http"
	"testing"
)

type deckStatsResponseTest struct {
	DeckID           string `json:"deckId"`
	Name             string `json:"name"`
	TotalCards       int    `json:"totalCards"`
	NewCards         int    `json:"newCards"`
	LearningCards    int    `json:"learningCards"`
	MasteredCards    int    `json:"masteredCards"`
	MasteryPercent   int    `json:"masteryPercent"`
	StudiedToday     int    `json:"studiedToday"`
	StudiedLast7Days int    `json:"studiedLast7Days"`
	StudiedTotal     int    `json:"studiedTotal"`
	DailyPerformance []struct {
		Date            string `json:"date"`
		ReviewCount     int    `json:"reviewCount"`
		AccuracyPercent int    `json:"accuracyPercent"`
	} `json:"dailyPerformance"`
}

func TestDeckStats_ReflectsCardStatesAndReviews(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	deckRec := doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "English"})
	var deckCreated deckResponse
	mustDecode(t, deckRec, &deckCreated)

	// Two new cards.
	cardRec1 := doJSON(t, router, http.MethodPost, "/api/v1/decks/"+deckCreated.ID+"/flashcards", cookie, flashcardRequest{Question: "Q1", Answer: "A1"})
	var card1 flashcardResponse
	mustDecode(t, cardRec1, &card1)

	cardRec2 := doJSON(t, router, http.MethodPost, "/api/v1/decks/"+deckCreated.ID+"/flashcards", cookie, flashcardRequest{Question: "Q2", Answer: "A2"})
	var card2 flashcardResponse
	mustDecode(t, cardRec2, &card2)

	// Review card1 as "good" (-> review state) and card2 as "again" (stays learning).
	reviewRec := doJSON(t, router, http.MethodPost, "/api/v1/flashcards/"+card1.ID+"/reviews", cookie, reviewRequest{Rating: "good"})
	if reviewRec.Code != http.StatusOK {
		t.Fatalf("review status = %d, want %d, body=%s", reviewRec.Code, http.StatusOK, reviewRec.Body.String())
	}
	doJSON(t, router, http.MethodPost, "/api/v1/flashcards/"+card2.ID+"/reviews", cookie, reviewRequest{Rating: "again"})

	statsRec := doJSON(t, router, http.MethodGet, "/api/v1/decks/"+deckCreated.ID+"/stats", cookie, nil)
	if statsRec.Code != http.StatusOK {
		t.Fatalf("stats status = %d, want %d, body=%s", statsRec.Code, http.StatusOK, statsRec.Body.String())
	}
	var stats deckStatsResponseTest
	mustDecode(t, statsRec, &stats)

	if stats.TotalCards != 2 {
		t.Errorf("TotalCards = %d, want 2", stats.TotalCards)
	}
	if stats.LearningCards != 2 {
		t.Errorf("LearningCards = %d, want 2 (one 'review', one 'learning')", stats.LearningCards)
	}
	if stats.MasteredCards != 0 {
		t.Errorf("MasteredCards = %d, want 0", stats.MasteredCards)
	}
	if len(stats.DailyPerformance) != 1 {
		t.Fatalf("DailyPerformance = %+v, want 1 day", stats.DailyPerformance)
	}
	if stats.DailyPerformance[0].ReviewCount != 2 {
		t.Errorf("today's ReviewCount = %d, want 2", stats.DailyPerformance[0].ReviewCount)
	}
	if stats.DailyPerformance[0].AccuracyPercent != 50 {
		t.Errorf("today's AccuracyPercent = %d, want 50 (1 of 2 correct)", stats.DailyPerformance[0].AccuracyPercent)
	}
	if stats.StudiedToday != 2 || stats.StudiedLast7Days != 2 || stats.StudiedTotal != 2 {
		t.Errorf("studied = today %d, week %d, total %d; want 2 cards each", stats.StudiedToday, stats.StudiedLast7Days, stats.StudiedTotal)
	}
}

func TestDeckStats_RejectsWhenDeckNotOwned(t *testing.T) {
	router, ownerCookie := mongoTestRouter(t)
	_, otherCookie := mongoTestRouter(t)

	deckRec := doJSON(t, router, http.MethodPost, "/api/v1/decks", ownerCookie, deckRequest{Name: "English"})
	var deckCreated deckResponse
	mustDecode(t, deckRec, &deckCreated)

	rec := doJSON(t, router, http.MethodGet, "/api/v1/decks/"+deckCreated.ID+"/stats", otherCookie, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestDeckStats_StudiedCardsCountDistinctCardsAndAcceptATimeZone(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	var d deckResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "English"}), &d)
	var card flashcardResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks/"+d.ID+"/flashcards", cookie, flashcardRequest{Question: "Q", Answer: "A"}), &card)
	doJSON(t, router, http.MethodPost, "/api/v1/flashcards/"+card.ID+"/reviews", cookie, reviewRequest{Rating: "again"})
	doJSON(t, router, http.MethodPost, "/api/v1/flashcards/"+card.ID+"/reviews", cookie, reviewRequest{Rating: "good"})

	// The test clock is 2026-01-01 12:00 UTC, still "today" in Manaus (08:00).
	rec := doJSON(t, router, http.MethodGet, "/api/v1/decks/"+d.ID+"/stats?tz=America/Manaus", cookie, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("stats status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var stats deckStatsResponseTest
	mustDecode(t, rec, &stats)
	if stats.StudiedToday != 1 || stats.StudiedTotal != 1 {
		t.Errorf("studied = today %d, total %d; want 1 (two reviews of the same card count once)", stats.StudiedToday, stats.StudiedTotal)
	}
	if len(stats.DailyPerformance) != 1 || stats.DailyPerformance[0].ReviewCount != 2 {
		t.Errorf("dailyPerformance = %+v, want one day with 2 reviews", stats.DailyPerformance)
	}

	for _, bad := range []string{"?tz=Not/AZone", "?tz=../etc/passwd"} {
		if rec := doJSON(t, router, http.MethodGet, "/api/v1/decks/"+d.ID+"/stats"+bad, cookie, nil); rec.Code != http.StatusBadRequest {
			t.Errorf("GET stats%s status = %d, want %d", bad, rec.Code, http.StatusBadRequest)
		}
	}
}

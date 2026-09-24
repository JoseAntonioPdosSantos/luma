package httpapi

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

type progressResponseTest struct {
	DailyCardLimit *int `json:"dailyCardLimit"`
	StudiedToday   int  `json:"studiedToday"`
	Remaining      *int `json:"remaining"`

	ContinuingPastGoal bool `json:"continuingPastGoal"`
}

// setDailyGoal makes the test user study with a configuration that has the
// given daily goal (nil = no goal, i.e. back to the built-in default).
func setDailyGoal(t *testing.T, router http.Handler, cookie *http.Cookie, limit *int) {
	t.Helper()
	if limit == nil {
		if rec := doJSON(t, router, http.MethodPut, "/api/v1/me/active-study-profile", cookie, map[string]any{"profileId": "default"}); rec.Code != http.StatusNoContent {
			t.Fatalf("selecting the default: status = %d, body=%s", rec.Code, rec.Body.String())
		}
		return
	}
	rec := doJSON(t, router, http.MethodPost, "/api/v1/study-profiles", cookie, map[string]any{
		"name":           fmt.Sprintf("Meta %d %d", *limit, time.Now().UnixNano()),
		"dailyCardLimit": *limit,
		"rules":          defaultRulesBody(),
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("creating a configuration with a goal of %d: status = %d, body=%s", *limit, rec.Code, rec.Body.String())
	}
	var created studyProfileResponse
	mustDecode(t, rec, &created)
	if rec := doJSON(t, router, http.MethodPut, "/api/v1/me/active-study-profile", cookie, map[string]any{"profileId": created.ID}); rec.Code != http.StatusNoContent {
		t.Fatalf("selecting it: status = %d, body=%s", rec.Code, rec.Body.String())
	}
}

func goal(n int) *int { return &n }

func TestStudy_DailyGoalLimitsDueCardsAndReportsProgress(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	var d deckResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "English"}), &d)
	var cards []flashcardResponse
	for _, q := range []string{"one", "two", "three", "four", "five"} {
		var c flashcardResponse
		mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks/"+d.ID+"/flashcards", cookie, flashcardRequest{Question: q, Answer: "a"}), &c)
		cards = append(cards, c)
	}
	dueURL := "/api/v1/decks/" + d.ID + "/due-flashcards"

	// No goal (the default): all five are served and progress has no goal.
	var due []flashcardResponse
	mustDecode(t, doJSON(t, router, http.MethodGet, dueURL, cookie, nil), &due)
	if len(due) != 5 {
		t.Fatalf("due cards without a goal = %d, want 5", len(due))
	}
	var progress progressResponseTest
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/study/progress", cookie, nil), &progress)
	if progress.DailyCardLimit != nil || progress.Remaining != nil || progress.StudiedToday != 0 {
		t.Fatalf("progress without a goal = %+v, want no goal and nothing studied", progress)
	}

	// Goal of 2: only two cards are served.
	setDailyGoal(t, router, cookie, goal(2))
	mustDecode(t, doJSON(t, router, http.MethodGet, dueURL, cookie, nil), &due)
	if len(due) != 2 {
		t.Fatalf("due cards with a goal of 2 = %d, want 2", len(due))
	}

	// Studying them uses the goal up: nothing more is served...
	for _, c := range due {
		if rec := doJSON(t, router, http.MethodPost, "/api/v1/flashcards/"+c.ID+"/reviews", cookie, reviewRequest{Rating: "good"}); rec.Code != http.StatusOK {
			t.Fatalf("review status = %d, want %d", rec.Code, http.StatusOK)
		}
	}
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/study/progress", cookie, nil), &progress)
	if progress.DailyCardLimit == nil || *progress.DailyCardLimit != 2 || progress.StudiedToday != 2 || progress.Remaining == nil || *progress.Remaining != 0 {
		t.Fatalf("progress = %+v, want goal 2, 2 studied, 0 remaining", progress)
	}
	mustDecode(t, doJSON(t, router, http.MethodGet, dueURL, cookie, nil), &due)
	if len(due) != 0 {
		t.Fatalf("due cards after reaching the goal = %d, want 0", len(due))
	}

	// ...unless the user chooses to study more anyway.
	mustDecode(t, doJSON(t, router, http.MethodGet, dueURL+"?ignoreDailyLimit=true", cookie, nil), &due)
	if len(due) != 3 {
		t.Fatalf("due cards with ignoreDailyLimit = %d, want the 3 untouched ones", len(due))
	}

	// Switching the goal off restores the default behavior.
	setDailyGoal(t, router, cookie, nil)
	mustDecode(t, doJSON(t, router, http.MethodGet, dueURL, cookie, nil), &due)
	if len(due) != 3 {
		t.Errorf("due cards after removing the goal = %d, want 3", len(due))
	}
	_ = cards
}

func TestStudy_TimeZoneParameterIsValidated(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	var d deckResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "English"}), &d)

	for _, path := range []string{
		"/api/v1/study/progress?tz=Foo/Bar",
		"/api/v1/decks/" + d.ID + "/due-flashcards?tz=Foo/Bar",
	} {
		if rec := doJSON(t, router, http.MethodGet, path, cookie, nil); rec.Code != http.StatusBadRequest {
			t.Errorf("GET %s status = %d, want %d", path, rec.Code, http.StatusBadRequest)
		}
	}
	if rec := doJSON(t, router, http.MethodGet, "/api/v1/study/progress?tz=America/Manaus", cookie, nil); rec.Code != http.StatusOK {
		t.Errorf("GET progress with a valid tz status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestStudy_ContinuingPastTheGoalIsSavedOnTheUserAndLiftsTheCap(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	var d deckResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "English"}), &d)
	for _, q := range []string{"one", "two", "three", "four"} {
		doJSON(t, router, http.MethodPost, "/api/v1/decks/"+d.ID+"/flashcards", cookie, flashcardRequest{Question: q, Answer: "a"})
	}
	dueURL := "/api/v1/decks/" + d.ID + "/due-flashcards"
	setDailyGoal(t, router, cookie, goal(1))

	var due []flashcardResponse
	mustDecode(t, doJSON(t, router, http.MethodGet, dueURL, cookie, nil), &due)
	if len(due) != 1 {
		t.Fatalf("due cards under a goal of 1 = %d, want 1", len(due))
	}
	var progress progressResponseTest
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/study/progress", cookie, nil), &progress)
	if progress.ContinuingPastGoal {
		t.Fatal("continuingPastGoal should start false")
	}

	if rec := doJSON(t, router, http.MethodPost, "/api/v1/study/continue-past-goal", cookie, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("continue-past-goal status = %d, want %d, body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/study/progress", cookie, nil), &progress)
	if !progress.ContinuingPastGoal || progress.DailyCardLimit == nil || *progress.DailyCardLimit != 1 {
		t.Fatalf("progress after continuing = %+v, want continuingPastGoal true and the goal still 1", progress)
	}
	mustDecode(t, doJSON(t, router, http.MethodGet, dueURL, cookie, nil), &due)
	if len(due) != 4 {
		t.Errorf("due cards after choosing to continue = %d, want all 4", len(due))
	}
}

func TestStudy_ContinuePastGoalRequiresLoginAndAValidTimeZone(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	if rec := doJSON(t, router, http.MethodPost, "/api/v1/study/continue-past-goal", nil, nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("without login: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if rec := doJSON(t, router, http.MethodPost, "/api/v1/study/continue-past-goal?tz=Foo/Bar", cookie, nil); rec.Code != http.StatusBadRequest {
		t.Errorf("with a bad tz: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if rec := doJSON(t, router, http.MethodPost, "/api/v1/study/continue-past-goal?tz=America/Manaus", cookie, nil); rec.Code != http.StatusNoContent {
		t.Errorf("with a valid tz: status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

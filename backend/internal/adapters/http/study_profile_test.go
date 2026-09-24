package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func defaultRulesBody() map[string]any {
	return map[string]any{
		"againDelayMinutes": 10,
		"hard":              map[string]any{"firstIntervalDays": 1, "multiplier": 1.2},
		"good":              map[string]any{"firstIntervalDays": 3, "multiplier": 2.0},
		"easy":              map[string]any{"firstIntervalDays": 7, "multiplier": 3.0},
	}
}

func profileBody(name string, limit any, rules map[string]any) map[string]any {
	return map[string]any{"name": name, "dailyCardLimit": limit, "rules": rules}
}

func listProfiles(t *testing.T, router http.Handler, cookie *http.Cookie) studyProfilesResponse {
	t.Helper()
	rec := doJSON(t, router, http.MethodGet, "/api/v1/study-profiles", cookie, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET study-profiles status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var list studyProfilesResponse
	mustDecode(t, rec, &list)
	return list
}

func TestStudyProfiles_EveryUserStartsOnTheNamedDefault(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	list := listProfiles(t, router, cookie)
	if list.ActiveProfileID != "default" || len(list.Profiles) != 1 {
		t.Fatalf("list = %+v, want only the default, selected", list)
	}
	d := list.Profiles[0]
	if d.ID != "default" || d.Name != "Padrão" || !d.IsDefault || d.DailyCardLimit != nil {
		t.Errorf("default = %+v, want the named built-in without a daily goal", d)
	}
	if d.Rules.AgainDelayMinutes != 10 ||
		d.Rules.Hard != (ratingRuleBody{1, 1.2}) || d.Rules.Good != (ratingRuleBody{3, 2.0}) || d.Rules.Easy != (ratingRuleBody{7, 3.0}) {
		t.Errorf("default rules = %+v, want the app's original behavior", d.Rules)
	}
}

func TestStudyProfiles_CreateSelectEditAndDelete(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	rules := defaultRulesBody()
	rules["againDelayMinutes"] = 3
	rec := doJSON(t, router, http.MethodPost, "/api/v1/study-profiles", cookie, profileBody("  Prova de inglês ", 40, rules))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var created studyProfileResponse
	mustDecode(t, rec, &created)
	if created.ID == "" || created.IsDefault || created.Name != "Prova de inglês" ||
		created.DailyCardLimit == nil || *created.DailyCardLimit != 40 || created.Rules.AgainDelayMinutes != 3 {
		t.Fatalf("created = %+v", created)
	}

	// It shows up after the default, but the default stays selected.
	list := listProfiles(t, router, cookie)
	if len(list.Profiles) != 2 || list.Profiles[1].ID != created.ID || list.ActiveProfileID != "default" {
		t.Fatalf("list = %+v, want [default, created] with the default still selected", list)
	}

	// Select it, then select the default again.
	if rec := doJSON(t, router, http.MethodPut, "/api/v1/me/active-study-profile", cookie, map[string]any{"profileId": created.ID}); rec.Code != http.StatusNoContent {
		t.Fatalf("select status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if got := listProfiles(t, router, cookie).ActiveProfileID; got != created.ID {
		t.Fatalf("active = %q, want %q", got, created.ID)
	}

	// Edit: rename, drop the daily goal.
	rec = doJSON(t, router, http.MethodPut, "/api/v1/study-profiles/"+created.ID, cookie, profileBody("Prova final", nil, rules))
	var edited studyProfileResponse
	mustDecode(t, rec, &edited)
	if rec.Code != http.StatusOK || edited.Name != "Prova final" || edited.DailyCardLimit != nil {
		t.Fatalf("edit = %d %+v, want it renamed and without a goal", rec.Code, edited)
	}

	// Delete the active one: the user goes back to the default.
	if rec := doJSON(t, router, http.MethodDelete, "/api/v1/study-profiles/"+created.ID, cookie, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	list = listProfiles(t, router, cookie)
	if len(list.Profiles) != 1 || list.ActiveProfileID != "default" {
		t.Errorf("after deleting the active one: %+v, want only the default, selected", list)
	}
}

func TestStudyProfiles_RejectInvalidInput(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	unordered := defaultRulesBody()
	unordered["good"] = map[string]any{"firstIntervalDays": 30, "multiplier": 2.0} // easy is 7 days
	badDelay := defaultRulesBody()
	badDelay["againDelayMinutes"] = 0
	badMultiplier := defaultRulesBody()
	badMultiplier["easy"] = map[string]any{"firstIntervalDays": 7, "multiplier": 9.0}

	cases := map[string]map[string]any{
		"empty name":             profileBody("  ", nil, defaultRulesBody()),
		"name too long":          profileBody(strings.Repeat("x", 61), nil, defaultRulesBody()),
		"goal of zero":           profileBody("a", 0, defaultRulesBody()),
		"goal too large":         profileBody("b", 201, defaultRulesBody()),
		"goal not a number":      profileBody("c", "many", defaultRulesBody()),
		"a better answer sooner": profileBody("d", nil, unordered),
		"delay of zero":          profileBody("e", nil, badDelay),
		"multiplier too large":   profileBody("f", nil, badMultiplier),
		"no rules at all":        {"name": "g"},
	}
	for name, body := range cases {
		if rec := doJSON(t, router, http.MethodPost, "/api/v1/study-profiles", cookie, body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d, body=%s", name, rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	}
	if got := listProfiles(t, router, cookie); len(got.Profiles) != 1 {
		t.Errorf("nothing invalid may be saved; got %d configurations", len(got.Profiles))
	}
}

func TestStudyProfiles_NamesAreUniqueAndTheDefaultIsProtected(t *testing.T) {
	router, cookie := mongoTestRouter(t)
	doJSON(t, router, http.MethodPost, "/api/v1/study-profiles", cookie, profileBody("Prova", nil, defaultRulesBody()))

	if rec := doJSON(t, router, http.MethodPost, "/api/v1/study-profiles", cookie, profileBody("prova", nil, defaultRulesBody())); rec.Code != http.StatusConflict {
		t.Errorf("a repeated name: status = %d, want %d", rec.Code, http.StatusConflict)
	}
	if rec := doJSON(t, router, http.MethodPut, "/api/v1/study-profiles/default", cookie, profileBody("Meu padrão", nil, defaultRulesBody())); rec.Code != http.StatusForbidden {
		t.Errorf("editing the default: status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if rec := doJSON(t, router, http.MethodDelete, "/api/v1/study-profiles/default", cookie, nil); rec.Code != http.StatusForbidden {
		t.Errorf("deleting the default: status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestStudyProfiles_OtherUsersConfigurationsAreOffLimits(t *testing.T) {
	router, ownerCookie := mongoTestRouter(t)
	_, otherCookie := mongoTestRouter(t)
	rec := doJSON(t, router, http.MethodPost, "/api/v1/study-profiles", ownerCookie, profileBody("Prova", nil, defaultRulesBody()))
	var mine studyProfileResponse
	mustDecode(t, rec, &mine)

	if rec := doJSON(t, router, http.MethodPut, "/api/v1/study-profiles/"+mine.ID, otherCookie, profileBody("x", nil, defaultRulesBody())); rec.Code != http.StatusNotFound {
		t.Errorf("editing someone else's: status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if rec := doJSON(t, router, http.MethodDelete, "/api/v1/study-profiles/"+mine.ID, otherCookie, nil); rec.Code != http.StatusNotFound {
		t.Errorf("deleting someone else's: status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if rec := doJSON(t, router, http.MethodPut, "/api/v1/me/active-study-profile", otherCookie, map[string]any{"profileId": mine.ID}); rec.Code != http.StatusNotFound {
		t.Errorf("selecting someone else's: status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if rec := doJSON(t, router, http.MethodPut, "/api/v1/me/active-study-profile", otherCookie, map[string]any{"profileId": "missing"}); rec.Code != http.StatusNotFound {
		t.Errorf("selecting a missing one: status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if got := listProfiles(t, router, otherCookie); len(got.Profiles) != 1 || got.ActiveProfileID != "default" {
		t.Errorf("the other user's list = %+v, want only the default", got)
	}
}

func TestStudyProfiles_RequireLoginAndValidJSON(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/study-profiles"},
		{http.MethodPost, "/api/v1/study-profiles"},
		{http.MethodPut, "/api/v1/study-profiles/x"},
		{http.MethodDelete, "/api/v1/study-profiles/x"},
		{http.MethodPut, "/api/v1/me/active-study-profile"},
	} {
		if rec := doJSON(t, router, tc.method, tc.path, nil, nil); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without login: status = %d, want %d", tc.method, tc.path, rec.Code, http.StatusUnauthorized)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/study-profiles", strings.NewReader("{not json"))
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("malformed JSON: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// The rules of the selected configuration decide how an answer reschedules a card.
func TestStudyProfiles_TheActiveConfigurationChangesHowCardsAreRescheduled(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	var d deckResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "English"}), &d)
	newCard := func(q string) string {
		var c flashcardResponse
		mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks/"+d.ID+"/flashcards", cookie, flashcardRequest{Question: q, Answer: "a"}), &c)
		return c.ID
	}
	review := func(id, rating string) flashcardResponse {
		rec := doJSON(t, router, http.MethodPost, "/api/v1/flashcards/"+id+"/reviews", cookie, reviewRequest{Rating: rating})
		if rec.Code != http.StatusOK {
			t.Fatalf("review status = %d, body=%s", rec.Code, rec.Body.String())
		}
		var c flashcardResponse
		mustDecode(t, rec, &c)
		return c
	}

	// With the built-in default, "Fácil" (good) on a new card is 3 days.
	if got := review(newCard("one"), "good").Scheduling.IntervalDays; got != 3 {
		t.Fatalf("default: good = %d days, want 3", got)
	}

	// A configuration where "Fácil" is 5 days and "Muito difícil" returns in an hour.
	rules := defaultRulesBody()
	rules["againDelayMinutes"] = 60
	rules["good"] = map[string]any{"firstIntervalDays": 5, "multiplier": 2.0}
	rules["easy"] = map[string]any{"firstIntervalDays": 9, "multiplier": 3.0}
	var custom studyProfileResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/study-profiles", cookie, profileBody("Calma", nil, rules)), &custom)
	doJSON(t, router, http.MethodPut, "/api/v1/me/active-study-profile", cookie, map[string]any{"profileId": custom.ID})

	if got := review(newCard("two"), "good").Scheduling.IntervalDays; got != 5 {
		t.Errorf("custom: good = %d days, want 5", got)
	}
	again := review(newCard("three"), "again")
	if !strings.HasPrefix(again.Scheduling.DueAt, "2026-01-01T13:00:00") {
		t.Errorf("custom: again is due at %s, want one hour after the test clock (13:00)", again.Scheduling.DueAt)
	}

	// Back to the default: the original behavior returns.
	doJSON(t, router, http.MethodPut, "/api/v1/me/active-study-profile", cookie, map[string]any{"profileId": "default"})
	if got := review(newCard("four"), "good").Scheduling.IntervalDays; got != 3 {
		t.Errorf("after going back to the default: good = %d days, want 3", got)
	}
}

// ---------- a configuration per deck ----------

func createDeck(t *testing.T, router http.Handler, cookie *http.Cookie, name string) string {
	t.Helper()
	var d deckResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: name}), &d)
	return d.ID
}

func createProfile(t *testing.T, router http.Handler, cookie *http.Cookie, name string, limit any, rules map[string]any) string {
	t.Helper()
	rec := doJSON(t, router, http.MethodPost, "/api/v1/study-profiles", cookie, profileBody(name, limit, rules))
	if rec.Code != http.StatusCreated {
		t.Fatalf("creating %q: status = %d, body=%s", name, rec.Code, rec.Body.String())
	}
	var p studyProfileResponse
	mustDecode(t, rec, &p)
	return p.ID
}

func chooseDeckProfile(t *testing.T, router http.Handler, cookie *http.Cookie, deckID string, profileID any) int {
	t.Helper()
	return doJSON(t, router, http.MethodPut, "/api/v1/decks/"+deckID+"/study-profile", cookie, map[string]any{"profileId": profileID}).Code
}

func TestDeckStudyProfile_ChooseShowInTheDeckAndGoBackToTheGeneralOne(t *testing.T) {
	router, cookie := mongoTestRouter(t)
	deckID := createDeck(t, router, cookie, "English")
	profileID := createProfile(t, router, cookie, "Só inglês", 5, defaultRulesBody())

	var deck deckResponse
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/decks/"+deckID, cookie, nil), &deck)
	if deck.StudyProfileID != "" {
		t.Fatalf("a new deck's studyProfileId = %q, want it absent (follows the general configuration)", deck.StudyProfileID)
	}

	if code := chooseDeckProfile(t, router, cookie, deckID, profileID); code != http.StatusNoContent {
		t.Fatalf("choosing status = %d, want %d", code, http.StatusNoContent)
	}
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/decks/"+deckID, cookie, nil), &deck)
	if deck.StudyProfileID != profileID {
		t.Errorf("studyProfileId = %q, want %q", deck.StudyProfileID, profileID)
	}
	var list []deckResponse
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/decks", cookie, nil), &list)
	if len(list) != 1 || list[0].StudyProfileID != profileID {
		t.Errorf("the deck list = %+v, want the deck to carry its studyProfileId too", list)
	}

	// The built-in default is a valid choice too, and so is going back to the general one (null or "").
	if code := chooseDeckProfile(t, router, cookie, deckID, "default"); code != http.StatusNoContent {
		t.Errorf("choosing the built-in default: status = %d", code)
	}
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/decks/"+deckID, cookie, nil), &deck)
	if deck.StudyProfileID != "default" {
		t.Errorf("studyProfileId = %q, want default", deck.StudyProfileID)
	}
	for _, general := range []any{nil, ""} {
		if code := chooseDeckProfile(t, router, cookie, deckID, profileID); code != http.StatusNoContent {
			t.Fatalf("choosing again: status = %d", code)
		}
		if code := chooseDeckProfile(t, router, cookie, deckID, general); code != http.StatusNoContent {
			t.Fatalf("going back to the general one with %v: status = %d", general, code)
		}
		deck = deckResponse{} // a field absent from the JSON would keep its old value
		mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/decks/"+deckID, cookie, nil), &deck)
		if deck.StudyProfileID != "" {
			t.Errorf("after %v: studyProfileId = %q, want it absent", general, deck.StudyProfileID)
		}
	}
}

func TestDeckStudyProfile_RejectsUnknownForeignAndUnauthenticatedRequests(t *testing.T) {
	router, cookie := mongoTestRouter(t)
	_, otherCookie := mongoTestRouter(t)
	deckID := createDeck(t, router, cookie, "English")
	mine := createProfile(t, router, cookie, "Minha", nil, defaultRulesBody())

	if code := chooseDeckProfile(t, router, cookie, deckID, "missing"); code != http.StatusNotFound {
		t.Errorf("a missing configuration: status = %d, want %d", code, http.StatusNotFound)
	}
	if code := chooseDeckProfile(t, router, cookie, "000000000000000000000000", mine); code != http.StatusNotFound {
		t.Errorf("a missing deck: status = %d, want %d", code, http.StatusNotFound)
	}
	if code := chooseDeckProfile(t, router, otherCookie, deckID, mine); code != http.StatusNotFound {
		t.Errorf("someone else's deck: status = %d, want %d", code, http.StatusNotFound)
	}
	otherProfile := createProfile(t, router, otherCookie, "De outra pessoa", nil, defaultRulesBody())
	if code := chooseDeckProfile(t, router, cookie, deckID, otherProfile); code != http.StatusNotFound {
		t.Errorf("someone else's configuration: status = %d, want %d", code, http.StatusNotFound)
	}
	if rec := doJSON(t, router, http.MethodPut, "/api/v1/decks/"+deckID+"/study-profile", nil, map[string]any{"profileId": mine}); rec.Code != http.StatusUnauthorized {
		t.Errorf("without login: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestDeckStudyProfile_DeletingTheConfigurationReleasesTheDeck(t *testing.T) {
	router, cookie := mongoTestRouter(t)
	deckID := createDeck(t, router, cookie, "English")
	profileID := createProfile(t, router, cookie, "Temporária", nil, defaultRulesBody())
	chooseDeckProfile(t, router, cookie, deckID, profileID)

	if rec := doJSON(t, router, http.MethodDelete, "/api/v1/study-profiles/"+profileID, cookie, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d", rec.Code)
	}
	var deck deckResponse
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/decks/"+deckID, cookie, nil), &deck)
	if deck.StudyProfileID != "" {
		t.Errorf("studyProfileId = %q after its configuration was deleted, want it absent", deck.StudyProfileID)
	}
}

// Rules, goal and count are the deck's own when it has a configuration of its own.
func TestDeckStudyProfile_EachDeckStudiesWithItsOwnRulesGoalAndCount(t *testing.T) {
	router, cookie := mongoTestRouter(t)
	own := createDeck(t, router, cookie, "Com configuração")
	general := createDeck(t, router, cookie, "Segue a geral")

	slowRules := defaultRulesBody()
	slowRules["good"] = map[string]any{"firstIntervalDays": 6, "multiplier": 2.0}
	slowRules["easy"] = map[string]any{"firstIntervalDays": 12, "multiplier": 3.0}
	ownProfile := createProfile(t, router, cookie, "Devagar, 2 por dia", 2, slowRules)
	generalProfile := createProfile(t, router, cookie, "Geral, 5 por dia", 5, defaultRulesBody())
	doJSON(t, router, http.MethodPut, "/api/v1/me/active-study-profile", cookie, map[string]any{"profileId": generalProfile})
	chooseDeckProfile(t, router, cookie, own, ownProfile)

	var cardsOwn, cardsGeneral []string
	for _, q := range []string{"1", "2", "3", "4", "5", "6"} {
		var a, b flashcardResponse
		mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks/"+own+"/flashcards", cookie, flashcardRequest{Question: q, Answer: "a"}), &a)
		mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks/"+general+"/flashcards", cookie, flashcardRequest{Question: q, Answer: "a"}), &b)
		cardsOwn = append(cardsOwn, a.ID)
		cardsGeneral = append(cardsGeneral, b.ID)
	}

	// Each deck is served up to its own goal.
	var dueOwn, dueGeneral []flashcardResponse
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/decks/"+own+"/due-flashcards", cookie, nil), &dueOwn)
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/decks/"+general+"/due-flashcards", cookie, nil), &dueGeneral)
	if len(dueOwn) != 2 || len(dueGeneral) != 5 {
		t.Fatalf("served %d cards in the deck with its own goal (want 2) and %d in the other (want 5)", len(dueOwn), len(dueGeneral))
	}

	// The rules used to reschedule follow the card's deck.
	review := func(id string) flashcardResponse {
		var c flashcardResponse
		rec := doJSON(t, router, http.MethodPost, "/api/v1/flashcards/"+id+"/reviews", cookie, reviewRequest{Rating: "good"})
		mustDecode(t, rec, &c)
		return c
	}
	if got := review(dueOwn[0].ID).Scheduling.IntervalDays; got != 6 {
		t.Errorf("good in the deck with its own rules = %d days, want 6", got)
	}
	if got := review(dueGeneral[0].ID).Scheduling.IntervalDays; got != 3 {
		t.Errorf("good in the deck following the general rules = %d days, want 3", got)
	}
	review(dueOwn[1].ID) // the deck's goal (2) is now used up

	// Progress is per deck: the own deck has used its goal; the general count only has its 1 card.
	var ownProgress, generalProgress, defaultProgress progressResponseTest
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/study/progress?deckId="+own, cookie, nil), &ownProgress)
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/study/progress?deckId="+general, cookie, nil), &generalProgress)
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/study/progress", cookie, nil), &defaultProgress)
	if ownProgress.DailyCardLimit == nil || *ownProgress.DailyCardLimit != 2 || ownProgress.StudiedToday != 2 || *ownProgress.Remaining != 0 {
		t.Errorf("own deck progress = %+v, want goal 2, 2 studied, 0 remaining", ownProgress)
	}
	for name, p := range map[string]progressResponseTest{"general deck": generalProgress, "no deck": defaultProgress} {
		if p.DailyCardLimit == nil || *p.DailyCardLimit != 5 || p.StudiedToday != 1 || *p.Remaining != 4 {
			t.Errorf("%s progress = %+v, want the general goal 5 with only its 1 card counted, 4 remaining", name, p)
		}
	}
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/decks/"+own+"/due-flashcards", cookie, nil), &dueOwn)
	if len(dueOwn) != 0 {
		t.Errorf("the deck with its own goal still serves %d cards after using it up", len(dueOwn))
	}

	// Continuing in that deck lifts only its cap.
	if rec := doJSON(t, router, http.MethodPost, "/api/v1/study/continue-past-goal?deckId="+own, cookie, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("continue status = %d", rec.Code)
	}
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/study/progress?deckId="+own, cookie, nil), &ownProgress)
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/study/progress?deckId="+general, cookie, nil), &generalProgress)
	if !ownProgress.ContinuingPastGoal || generalProgress.ContinuingPastGoal {
		t.Errorf("continuing = %v for the own deck and %v for the general one; want true and false", ownProgress.ContinuingPastGoal, generalProgress.ContinuingPastGoal)
	}
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/decks/"+own+"/due-flashcards", cookie, nil), &dueOwn)
	if len(dueOwn) != 4 {
		t.Errorf("after continuing, the deck serves %d cards, want its 4 remaining", len(dueOwn))
	}
	_ = cardsOwn
	_ = cardsGeneral
}

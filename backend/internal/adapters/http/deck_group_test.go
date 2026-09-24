package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

func listDeckGroups(t *testing.T, router http.Handler, cookie *http.Cookie) []deckGroupResponse {
	t.Helper()
	rec := doJSON(t, router, http.MethodGet, "/api/v1/deck-groups", cookie, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET deck-groups status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var list []deckGroupResponse
	mustDecode(t, rec, &list)
	return list
}

func TestDeckGroups_StartsEmpty(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	if list := listDeckGroups(t, router, cookie); len(list) != 0 {
		t.Fatalf("list = %+v, want none", list)
	}
}

func TestDeckGroups_CreateRenameAndDelete(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	rec := doJSON(t, router, http.MethodPost, "/api/v1/deck-groups", cookie, deckGroupRequest{Name: "  Inglês  "})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var created deckGroupResponse
	mustDecode(t, rec, &created)
	if created.ID == "" || created.Name != "Inglês" {
		t.Fatalf("created = %+v", created)
	}

	if list := listDeckGroups(t, router, cookie); len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("list = %+v, want [created]", list)
	}

	// Rename.
	rec = doJSON(t, router, http.MethodPut, "/api/v1/deck-groups/"+created.ID, cookie, deckGroupRequest{Name: "Inglês avançado"})
	var renamed deckGroupResponse
	mustDecode(t, rec, &renamed)
	if rec.Code != http.StatusOK || renamed.Name != "Inglês avançado" {
		t.Fatalf("rename = %d %+v", rec.Code, renamed)
	}

	// A deck filed under it survives the group's deletion, only losing the group.
	var deckCreated deckResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "Phrasal verbs"}), &deckCreated)
	if rec := doJSON(t, router, http.MethodPut, "/api/v1/decks/"+deckCreated.ID+"/group", cookie, map[string]any{"groupId": created.ID}); rec.Code != http.StatusNoContent {
		t.Fatalf("set deck group status = %d, body=%s", rec.Code, rec.Body.String())
	}

	var withGroup deckResponse
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/decks/"+deckCreated.ID, cookie, nil), &withGroup)
	if withGroup.GroupID != created.ID {
		t.Fatalf("deck.GroupID = %q, want %q", withGroup.GroupID, created.ID)
	}

	if rec := doJSON(t, router, http.MethodDelete, "/api/v1/deck-groups/"+created.ID, cookie, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if list := listDeckGroups(t, router, cookie); len(list) != 0 {
		t.Fatalf("list after delete = %+v, want none", list)
	}

	var afterDelete deckResponse
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/decks/"+deckCreated.ID, cookie, nil), &afterDelete)
	if afterDelete.GroupID != "" {
		t.Errorf("deck.GroupID after deleting its group = %q, want empty", afterDelete.GroupID)
	}
}

func TestDeckGroups_RejectsEmptyAndDuplicateNames(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	if rec := doJSON(t, router, http.MethodPost, "/api/v1/deck-groups", cookie, deckGroupRequest{Name: "   "}); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty name status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	if rec := doJSON(t, router, http.MethodPost, "/api/v1/deck-groups", cookie, deckGroupRequest{Name: "Inglês"}); rec.Code != http.StatusCreated {
		t.Fatalf("first create status = %d", rec.Code)
	}
	rec := doJSON(t, router, http.MethodPost, "/api/v1/deck-groups", cookie, deckGroupRequest{Name: "inglês"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate (case-insensitive) name status = %d, want %d, body=%s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestDeckGroups_SetDeckGroup_RejectsUnknownDeckOrGroup(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	var deckCreated deckResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "English"}), &deckCreated)

	rec := doJSON(t, router, http.MethodPut, "/api/v1/decks/"+deckCreated.ID+"/group", cookie, map[string]any{"groupId": "not-a-real-group"})
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "deckGroup.notFound") {
		t.Fatalf("unknown group status = %d, body=%s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodPut, "/api/v1/decks/not-a-real-deck/group", cookie, map[string]any{"groupId": nil})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown deck status = %d, body=%s", rec.Code, rec.Body.String())
	}
}

func TestDeckGroups_RemovingAGroupAssignmentFollowsANilGroupId(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	var group deckGroupResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/deck-groups", cookie, deckGroupRequest{Name: "Inglês"}), &group)
	var deckCreated deckResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "Verbos"}), &deckCreated)

	if rec := doJSON(t, router, http.MethodPut, "/api/v1/decks/"+deckCreated.ID+"/group", cookie, map[string]any{"groupId": group.ID}); rec.Code != http.StatusNoContent {
		t.Fatalf("set group status = %d", rec.Code)
	}
	if rec := doJSON(t, router, http.MethodPut, "/api/v1/decks/"+deckCreated.ID+"/group", cookie, map[string]any{"groupId": nil}); rec.Code != http.StatusNoContent {
		t.Fatalf("clear group status = %d", rec.Code)
	}

	var cleared deckResponse
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/decks/"+deckCreated.ID, cookie, nil), &cleared)
	if cleared.GroupID != "" {
		t.Errorf("GroupID = %q, want empty after clearing", cleared.GroupID)
	}
}

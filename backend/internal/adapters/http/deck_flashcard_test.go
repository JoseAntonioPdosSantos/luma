package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"flashcard-backend/internal/adapters/authtoken"
	gridfsadapter "flashcard-backend/internal/adapters/gridfs"
	mongoadapter "flashcard-backend/internal/adapters/mongodb"
	"flashcard-backend/internal/application/deckgroupservice"
	"flashcard-backend/internal/application/deckservice"
	"flashcard-backend/internal/application/flashcardservice"
	"flashcard-backend/internal/application/profileservice"
	"flashcard-backend/internal/application/studyservice"
	"flashcard-backend/internal/application/userservice"
	"flashcard-backend/internal/config"
	"flashcard-backend/internal/domain/user"
	"flashcard-backend/internal/ports/clock"
)

const testMaxAudioSizeBytes = 10 * 1024 * 1024

// testMongoDatabase connects to a MongoDB instance for HTTP integration
// tests, mirroring internal/adapters/mongodb's own test helper. It skips
// the test if no instance is reachable (see docker-compose.yml's mongo
// service, exposed on host port 27018).
func testMongoDatabase(t *testing.T) *mongo.Database {
	t.Helper()

	uri := os.Getenv("MONGODB_TEST_URI")
	if uri == "" {
		uri = "mongodb://localhost:27018"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	db, err := mongoadapter.Connect(ctx, uri, "flashcard_http_test")
	if err != nil {
		t.Skipf("skipping: no reachable MongoDB test instance at %s: %v", uri, err)
	}

	t.Cleanup(func() {
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer dropCancel()
		_ = db.Drop(dropCtx)
	})

	return db
}

// mongoTestRouter builds a full router backed by real MongoDB
// repositories for decks and flashcards, with a session already issued
// for a fresh user ID, so tests can exercise ownership rules end-to-end.
func mongoTestRouter(t *testing.T) (router http.Handler, userCookie *http.Cookie) {
	t.Helper()
	db := testMongoDatabase(t)

	userRepo := mongoadapter.NewUserRepository(db)
	deckRepo := mongoadapter.NewDeckRepository(db)
	flashcardRepo := mongoadapter.NewFlashcardRepository(db)
	reviewRepo := mongoadapter.NewReviewEventRepository(db)
	audioStore, err := gridfsadapter.New(db)
	if err != nil {
		t.Fatalf("gridfs.New() error: %v", err)
	}
	fixedClock := clock.Fixed{Time: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	sessions := authtoken.NewHMACManager("test-secret", time.Hour)

	profileSvc := profileservice.New(mongoadapter.NewStudyProfileRepository(db), userRepo, deckRepo, fixedClock)

	deps := Dependencies{
		Users:      userservice.New(userRepo, fixedClock),
		Profiles:   profileSvc,
		Decks:      deckservice.New(deckRepo, flashcardRepo, reviewRepo, audioStore, fixedClock),
		DeckGroups: deckgroupservice.New(mongoadapter.NewDeckGroupRepository(db), deckRepo, fixedClock),
		Flashcards: flashcardservice.New(flashcardRepo, deckRepo, audioStore, testMaxAudioSizeBytes, fixedClock),
		Study:      studyservice.New(flashcardRepo, userRepo, profileSvc, deckRepo, reviewRepo, fixedClock),
		Sessions:   sessions,
		Clock:      fixedClock,
	}
	router = NewRouter(config.Config{CORSAllowedOrigins: []string{"http://localhost:5173"}}, deps)

	// A real user document: the study endpoints read the user's settings.
	createdUser, err := userRepo.Create(context.Background(),
		user.New(fmt.Sprintf("%s@example.com", primitive.NewObjectID().Hex()), fixedClock.Now()))
	if err != nil {
		t.Fatalf("creating the test user: %v", err)
	}
	token, expiresAt, err := sessions.Issue(createdUser.ID, fixedClock.Now())
	if err != nil {
		t.Fatalf("Issue() error: %v", err)
	}
	return router, &http.Cookie{Name: sessionCookieName, Value: token, Expires: expiresAt}
}

func doJSON(t *testing.T, router http.Handler, method, path string, cookie *http.Cookie, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshaling request body: %v", err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestDeckLifecycle_CreateListGetUpdateArchive(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	createRec := doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "English", Description: "Learn English"})
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d, body=%s", createRec.Code, http.StatusCreated, createRec.Body.String())
	}
	var created deckResponse
	mustDecode(t, createRec, &created)

	listRec := doJSON(t, router, http.MethodGet, "/api/v1/decks", cookie, nil)
	var list []deckResponse
	mustDecode(t, listRec, &list)
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("list = %+v, want one deck with ID %q", list, created.ID)
	}

	getRec := doJSON(t, router, http.MethodGet, "/api/v1/decks/"+created.ID, cookie, nil)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getRec.Code, http.StatusOK)
	}

	updateRec := doJSON(t, router, http.MethodPatch, "/api/v1/decks/"+created.ID, cookie, deckRequest{Name: "English (updated)", Description: ""})
	var updated deckResponse
	mustDecode(t, updateRec, &updated)
	if updated.Name != "English (updated)" {
		t.Errorf("updated name = %q", updated.Name)
	}

	archiveRec := doJSON(t, router, http.MethodDelete, "/api/v1/decks/"+created.ID, cookie, nil)
	if archiveRec.Code != http.StatusNoContent {
		t.Fatalf("archive status = %d, want %d", archiveRec.Code, http.StatusNoContent)
	}

	listAfterArchiveRec := doJSON(t, router, http.MethodGet, "/api/v1/decks", cookie, nil)
	var listAfterArchive []deckResponse
	mustDecode(t, listAfterArchiveRec, &listAfterArchive)
	if len(listAfterArchive) != 0 {
		t.Errorf("list after archive = %+v, want empty", listAfterArchive)
	}
}

func TestDeckCreate_RejectsEmptyName(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	rec := doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "  "})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestDeckGet_OwnedByAnotherUserIsNotFound(t *testing.T) {
	router, ownerCookie := mongoTestRouter(t)
	_, otherCookie := mongoTestRouter(t) // separate user, same router would be ideal but a fresh one is fine here

	createRec := doJSON(t, router, http.MethodPost, "/api/v1/decks", ownerCookie, deckRequest{Name: "Private"})
	var created deckResponse
	mustDecode(t, createRec, &created)

	getRec := doJSON(t, router, http.MethodGet, "/api/v1/decks/"+created.ID, otherCookie, nil)
	if getRec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", getRec.Code, http.StatusNotFound, getRec.Body.String())
	}
}

func TestDeckGet_InvalidIDIsNotFound(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	rec := doJSON(t, router, http.MethodGet, "/api/v1/decks/not-a-valid-object-id", cookie, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestFlashcardLifecycle_CreateListGetUpdateArchive(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	deckRec := doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "English"})
	var deckCreated deckResponse
	mustDecode(t, deckRec, &deckCreated)

	createRec := doJSON(t, router, http.MethodPost, "/api/v1/decks/"+deckCreated.ID+"/flashcards", cookie, flashcardRequest{
		Question: "How to say 'resgatar'?",
		Answer:   "rescue",
		Hint:     "Starts with R",
		ExtendedExample: &extendedExampleRequest{
			Text:        "I rescued a cat.",
			Translation: "Eu resgatei um gato.",
		},
	})
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d, body=%s", createRec.Code, http.StatusCreated, createRec.Body.String())
	}
	var created flashcardResponse
	mustDecode(t, createRec, &created)
	if created.Scheduling.State != "new" {
		t.Errorf("Scheduling.State = %q, want %q", created.Scheduling.State, "new")
	}
	if created.Hint != "Starts with R" {
		t.Errorf("Hint = %q, want %q", created.Hint, "Starts with R")
	}

	listRec := doJSON(t, router, http.MethodGet, "/api/v1/decks/"+deckCreated.ID+"/flashcards", cookie, nil)
	var list flashcardListResponse
	mustDecode(t, listRec, &list)
	if len(list.Items) != 1 || list.Total != 1 {
		t.Fatalf("list = %+v, want 1 flashcard", list)
	}

	updateRec := doJSON(t, router, http.MethodPatch, "/api/v1/flashcards/"+created.ID, cookie, flashcardRequest{
		Question: "Updated question",
		Answer:   "Updated answer",
	})
	var updated flashcardResponse
	mustDecode(t, updateRec, &updated)
	if updated.Question != "Updated question" || updated.ExtendedExample != nil {
		t.Errorf("updated = %+v", updated)
	}
	if updated.Hint != "" {
		t.Errorf("updated.Hint = %q, want it cleared when the update omits the hint", updated.Hint)
	}

	archiveRec := doJSON(t, router, http.MethodDelete, "/api/v1/flashcards/"+created.ID, cookie, nil)
	if archiveRec.Code != http.StatusNoContent {
		t.Fatalf("archive status = %d, want %d", archiveRec.Code, http.StatusNoContent)
	}

	listAfterArchiveRec := doJSON(t, router, http.MethodGet, "/api/v1/decks/"+deckCreated.ID+"/flashcards", cookie, nil)
	var listAfterArchive flashcardListResponse
	mustDecode(t, listAfterArchiveRec, &listAfterArchive)
	if len(listAfterArchive.Items) != 0 || listAfterArchive.Total != 0 {
		t.Errorf("list after archive = %+v, want empty", listAfterArchive)
	}
}

func TestFlashcardCreate_RejectsWhenDeckNotOwned(t *testing.T) {
	router, ownerCookie := mongoTestRouter(t)
	_, otherCookie := mongoTestRouter(t)

	deckRec := doJSON(t, router, http.MethodPost, "/api/v1/decks", ownerCookie, deckRequest{Name: "English"})
	var deckCreated deckResponse
	mustDecode(t, deckRec, &deckCreated)

	rec := doJSON(t, router, http.MethodPost, "/api/v1/decks/"+deckCreated.ID+"/flashcards", otherCookie, flashcardRequest{
		Question: "Q", Answer: "A",
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestFlashcardCreate_RejectsEmptyQuestion(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	deckRec := doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "English"})
	var deckCreated deckResponse
	mustDecode(t, deckRec, &deckCreated)

	rec := doJSON(t, router, http.MethodPost, "/api/v1/decks/"+deckCreated.ID+"/flashcards", cookie, flashcardRequest{
		Question: "", Answer: "A",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func mustDecode(t *testing.T, rec *httptest.ResponseRecorder, dest any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), dest); err != nil {
		t.Fatalf("decoding response %s: %v", rec.Body.String(), err)
	}
}

func TestDeckAndFlashcard_ArchiveListRestore(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	var deckCreated deckResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "English"}), &deckCreated)
	var cardCreated flashcardResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks/"+deckCreated.ID+"/flashcards", cookie,
		flashcardRequest{Question: "Q", Answer: "A"}), &cardCreated)

	// Archive the card, see it under ?archived=true, restore it.
	doJSON(t, router, http.MethodDelete, "/api/v1/flashcards/"+cardCreated.ID, cookie, nil)

	var archivedCards flashcardListResponse
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/decks/"+deckCreated.ID+"/flashcards?archived=true", cookie, nil), &archivedCards)
	if len(archivedCards.Items) != 1 || archivedCards.Items[0].ID != cardCreated.ID {
		t.Fatalf("archived cards = %+v, want the archived card", archivedCards)
	}

	if rec := doJSON(t, router, http.MethodPost, "/api/v1/flashcards/"+cardCreated.ID+"/restore", cookie, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("restore card status = %d, want %d, body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	var activeCards flashcardListResponse
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/decks/"+deckCreated.ID+"/flashcards", cookie, nil), &activeCards)
	if len(activeCards.Items) != 1 {
		t.Fatalf("active cards after restore = %+v, want 1", activeCards)
	}

	// Archive the deck, see it under ?archived=true, restore it.
	doJSON(t, router, http.MethodDelete, "/api/v1/decks/"+deckCreated.ID, cookie, nil)

	var archivedDecks []deckResponse
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/decks?archived=true", cookie, nil), &archivedDecks)
	if len(archivedDecks) != 1 || archivedDecks[0].ID != deckCreated.ID || archivedDecks[0].TotalCards != 1 {
		t.Fatalf("archived decks = %+v, want the archived deck with 1 card", archivedDecks)
	}

	if rec := doJSON(t, router, http.MethodPost, "/api/v1/decks/"+deckCreated.ID+"/restore", cookie, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("restore deck status = %d, want %d, body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	var activeDecks []deckResponse
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/decks", cookie, nil), &activeDecks)
	if len(activeDecks) != 1 {
		t.Fatalf("active decks after restore = %+v, want 1", activeDecks)
	}
}

func TestRestore_OwnedByAnotherUserIsNotFound(t *testing.T) {
	router, ownerCookie := mongoTestRouter(t)
	_, otherCookie := mongoTestRouter(t)

	var deckCreated deckResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks", ownerCookie, deckRequest{Name: "Private"}), &deckCreated)
	doJSON(t, router, http.MethodDelete, "/api/v1/decks/"+deckCreated.ID, ownerCookie, nil)

	rec := doJSON(t, router, http.MethodPost, "/api/v1/decks/"+deckCreated.ID+"/restore", otherCookie, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("restore by other user status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestArchivedFlashcards_AcrossDecksIncludesDeckName(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	var deckA, deckB deckResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "English"}), &deckA)
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "Français"}), &deckB)

	for _, d := range []deckResponse{deckA, deckB} {
		var card flashcardResponse
		mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks/"+d.ID+"/flashcards", cookie,
			flashcardRequest{Question: "Q " + d.Name, Answer: "A"}), &card)
		doJSON(t, router, http.MethodDelete, "/api/v1/flashcards/"+card.ID, cookie, nil)
	}

	var archived []archivedFlashcardResponse
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/archived-flashcards", cookie, nil), &archived)
	if len(archived) != 2 {
		t.Fatalf("archived = %+v, want 2 cards", archived)
	}
	byQuestion := map[string]string{}
	for _, a := range archived {
		byQuestion[a.Question] = a.DeckName
	}
	if byQuestion["Q English"] != "English" || byQuestion["Q Français"] != "Français" {
		t.Errorf("deck names = %+v", byQuestion)
	}

	if rec := doJSON(t, router, http.MethodGet, "/api/v1/archived-flashcards", nil, nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestPermanentDelete_OnlyArchivedItems(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	var d deckResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "English"}), &d)
	var card flashcardResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks/"+d.ID+"/flashcards", cookie,
		flashcardRequest{Question: "Q", Answer: "A"}), &card)

	// Active items cannot be deleted permanently.
	if rec := doJSON(t, router, http.MethodDelete, "/api/v1/archived-flashcards/"+card.ID, cookie, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("delete active card status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if rec := doJSON(t, router, http.MethodDelete, "/api/v1/archived-decks/"+d.ID, cookie, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("delete active deck status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if rec := doJSON(t, router, http.MethodGet, "/api/v1/flashcards/"+card.ID, cookie, nil); rec.Code != http.StatusOK {
		t.Fatalf("active card must survive, status = %d", rec.Code)
	}

	// Archived card: deleted for good.
	doJSON(t, router, http.MethodDelete, "/api/v1/flashcards/"+card.ID, cookie, nil)
	if rec := doJSON(t, router, http.MethodDelete, "/api/v1/archived-flashcards/"+card.ID, cookie, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete archived card status = %d, want %d, body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if rec := doJSON(t, router, http.MethodGet, "/api/v1/flashcards/"+card.ID, cookie, nil); rec.Code != http.StatusNotFound {
		t.Errorf("deleted card status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	// Archived deck: deleted for good, together with its cards.
	var second flashcardResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks/"+d.ID+"/flashcards", cookie,
		flashcardRequest{Question: "Q2", Answer: "A2"}), &second)
	doJSON(t, router, http.MethodDelete, "/api/v1/decks/"+d.ID, cookie, nil)
	if rec := doJSON(t, router, http.MethodDelete, "/api/v1/archived-decks/"+d.ID, cookie, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete archived deck status = %d, want %d, body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if rec := doJSON(t, router, http.MethodGet, "/api/v1/decks/"+d.ID, cookie, nil); rec.Code != http.StatusNotFound {
		t.Errorf("deleted deck status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if rec := doJSON(t, router, http.MethodGet, "/api/v1/flashcards/"+second.ID, cookie, nil); rec.Code != http.StatusNotFound {
		t.Errorf("card of a deleted deck status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestPermanentDelete_OwnedByAnotherUserIsNotFound(t *testing.T) {
	router, ownerCookie := mongoTestRouter(t)
	_, otherCookie := mongoTestRouter(t)

	var d deckResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks", ownerCookie, deckRequest{Name: "Private"}), &d)
	doJSON(t, router, http.MethodDelete, "/api/v1/decks/"+d.ID, ownerCookie, nil)

	if rec := doJSON(t, router, http.MethodDelete, "/api/v1/archived-decks/"+d.ID, otherCookie, nil); rec.Code != http.StatusNotFound {
		t.Errorf("delete by other user status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if rec := doJSON(t, router, http.MethodGet, "/api/v1/decks/"+d.ID, ownerCookie, nil); rec.Code != http.StatusOK {
		t.Errorf("owner's archived deck must survive, status = %d", rec.Code)
	}
}

func TestFlashcardList_SearchAndPagination(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	var d deckResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "Français"}), &d)
	for _, q := range []string{"un", "deux", "trois", "quatre", "cinq", "Français"} {
		doJSON(t, router, http.MethodPost, "/api/v1/decks/"+d.ID+"/flashcards", cookie, flashcardRequest{Question: q, Answer: "a"})
	}
	base := "/api/v1/decks/" + d.ID + "/flashcards"

	var page1, page2 flashcardListResponse
	mustDecode(t, doJSON(t, router, http.MethodGet, base+"?limit=4", cookie, nil), &page1)
	mustDecode(t, doJSON(t, router, http.MethodGet, base+"?limit=4&offset=4", cookie, nil), &page2)
	if page1.Total != 6 || len(page1.Items) != 4 || len(page2.Items) != 2 {
		t.Fatalf("pages = %d + %d items (total %d), want 4 + 2 of 6", len(page1.Items), len(page2.Items), page1.Total)
	}
	if page1.Items[0].Question != "Français" || page2.Items[1].Question != "un" {
		t.Errorf("order: first %q, last %q; want newest first ('Français') and oldest last ('un')", page1.Items[0].Question, page2.Items[1].Question)
	}

	var found flashcardListResponse
	mustDecode(t, doJSON(t, router, http.MethodGet, base+"?q=francais", cookie, nil), &found)
	if found.Total != 1 || found.Items[0].Question != "Français" {
		t.Errorf("search 'francais' = %+v, want the 'Français' card (accent-insensitive)", found)
	}

	for _, bad := range []string{"?limit=0", "?limit=abc", "?offset=-1", "?offset=x"} {
		if rec := doJSON(t, router, http.MethodGet, base+bad, cookie, nil); rec.Code != http.StatusBadRequest {
			t.Errorf("GET %s status = %d, want %d", bad, rec.Code, http.StatusBadRequest)
		}
	}
}

func TestDeckGet_ReportsTheNextReviewDateOnlyOnTheSingleDeck(t *testing.T) {
	router, cookie := mongoTestRouter(t)

	var d deckResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks", cookie, deckRequest{Name: "English"}), &d)
	var card flashcardResponse
	mustDecode(t, doJSON(t, router, http.MethodPost, "/api/v1/decks/"+d.ID+"/flashcards", cookie, flashcardRequest{Question: "Q", Answer: "A"}), &card)

	// A brand-new card is due right now, so nothing is scheduled ahead yet.
	var before deckResponse
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/decks/"+d.ID, cookie, nil), &before)
	if before.NextDueAt != "" {
		t.Fatalf("nextDueAt = %q with only a due card, want it absent", before.NextDueAt)
	}

	// Reviewing it as "good" schedules it 3 days out (the test clock is fixed at 2026-01-01 12:00 UTC).
	doJSON(t, router, http.MethodPost, "/api/v1/flashcards/"+card.ID+"/reviews", cookie, reviewRequest{Rating: "good"})
	var after deckResponse
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/decks/"+d.ID, cookie, nil), &after)
	if after.NextDueAt != "2026-01-04T12:00:00Z" {
		t.Errorf("nextDueAt = %q, want 2026-01-04T12:00:00Z", after.NextDueAt)
	}

	// The dashboard list does not carry it.
	var list []deckResponse
	mustDecode(t, doJSON(t, router, http.MethodGet, "/api/v1/decks", cookie, nil), &list)
	if len(list) != 1 || list[0].NextDueAt != "" {
		t.Errorf("deck list = %+v, want no nextDueAt in the list", list)
	}
}

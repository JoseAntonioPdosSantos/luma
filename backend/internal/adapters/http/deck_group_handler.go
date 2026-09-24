package httpapi

import (
	"encoding/json"
	"net/http"

	"flashcard-backend/internal/apperror"
	"flashcard-backend/internal/application/deckgroupservice"
	"flashcard-backend/internal/domain/deckgroup"
)

const maxDeckGroupBodyBytes = 4 * 1024

type deckGroupRequest struct {
	Name string `json:"name"`
}

type deckGroupResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toDeckGroupResponse(g deckgroup.Group) deckGroupResponse {
	return deckGroupResponse{ID: g.ID, Name: g.Name}
}

// DeckGroupHandlers implements the deck-group endpoints: named folders a
// user files decks under to organize related collections together.
type DeckGroupHandlers struct {
	groups deckgroupservice.Service
}

func NewDeckGroupHandlers(groups deckgroupservice.Service) DeckGroupHandlers {
	return DeckGroupHandlers{groups: groups}
}

func decodeDeckGroupRequest(w http.ResponseWriter, r *http.Request) (deckGroupRequest, bool) {
	var req deckGroupRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxDeckGroupBodyBytes)).Decode(&req); err != nil {
		writeError(w, r, apperror.Validation("request.invalidJson", "request body must be valid JSON"))
		return req, false
	}
	return req, true
}

// List handles GET /api/v1/deck-groups.
func (h DeckGroupHandlers) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	groups, err := h.groups.List(r.Context(), userID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	response := make([]deckGroupResponse, 0, len(groups))
	for _, g := range groups {
		response = append(response, toDeckGroupResponse(g))
	}
	writeJSON(w, http.StatusOK, response)
}

// Create handles POST /api/v1/deck-groups.
func (h DeckGroupHandlers) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	req, ok := decodeDeckGroupRequest(w, r)
	if !ok {
		return
	}

	created, err := h.groups.Create(r.Context(), userID, req.Name)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toDeckGroupResponse(created))
}

// Update handles PUT /api/v1/deck-groups/{groupID}: rename a group.
func (h DeckGroupHandlers) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	req, ok := decodeDeckGroupRequest(w, r)
	if !ok {
		return
	}

	updated, err := h.groups.Rename(r.Context(), userID, r.PathValue("groupID"), req.Name)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toDeckGroupResponse(updated))
}

// Delete handles DELETE /api/v1/deck-groups/{groupID}. The decks that were
// filed under it are not deleted; they lose their group.
func (h DeckGroupHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	if err := h.groups.Delete(r.Context(), userID, r.PathValue("groupID")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SetDeckGroup handles PUT /api/v1/decks/{deckID}/group: file a deck under a
// group, or remove it from its group with an empty/null groupId.
func (h DeckGroupHandlers) SetDeckGroup(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	var req struct {
		GroupID *string `json:"groupId"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxDeckGroupBodyBytes)).Decode(&req); err != nil {
		writeError(w, r, apperror.Validation("request.invalidJson.groupId", "request body must be valid JSON with a \"groupId\" field"))
		return
	}
	groupID := ""
	if req.GroupID != nil {
		groupID = *req.GroupID
	}
	if err := h.groups.SetDeckGroup(r.Context(), userID, r.PathValue("deckID"), groupID); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

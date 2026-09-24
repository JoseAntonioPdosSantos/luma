package httpapi

import (
	"encoding/json"
	"net/http"

	"flashcard-backend/internal/apperror"
	"flashcard-backend/internal/application/profileservice"
	"flashcard-backend/internal/domain/study"
	"flashcard-backend/internal/domain/studyprofile"
)

const maxProfileBodyBytes = 4 * 1024

type ratingRuleBody struct {
	FirstIntervalDays int     `json:"firstIntervalDays"`
	Multiplier        float64 `json:"multiplier"`
}

type rulesBody struct {
	AgainDelayMinutes int            `json:"againDelayMinutes"`
	Hard              ratingRuleBody `json:"hard"`
	Good              ratingRuleBody `json:"good"`
	Easy              ratingRuleBody `json:"easy"`
}

func (r rulesBody) toDomain() study.Rules {
	return study.Rules{
		AgainDelayMinutes: r.AgainDelayMinutes,
		Hard:              study.RatingRule(r.Hard),
		Good:              study.RatingRule(r.Good),
		Easy:              study.RatingRule(r.Easy),
	}
}

func rulesToBody(r study.Rules) rulesBody {
	return rulesBody{
		AgainDelayMinutes: r.AgainDelayMinutes,
		Hard:              ratingRuleBody(r.Hard),
		Good:              ratingRuleBody(r.Good),
		Easy:              ratingRuleBody(r.Easy),
	}
}

// studyProfileRequest is the editable part of a configuration.
type studyProfileRequest struct {
	Name string `json:"name"`
	// DailyCardLimit null (or absent) means no daily goal.
	DailyCardLimit *int      `json:"dailyCardLimit"`
	Rules          rulesBody `json:"rules"`
}

type studyProfileResponse struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	IsDefault      bool      `json:"isDefault"`
	DailyCardLimit *int      `json:"dailyCardLimit"`
	Rules          rulesBody `json:"rules"`
}

type studyProfilesResponse struct {
	ActiveProfileID string                 `json:"activeProfileId"`
	Profiles        []studyProfileResponse `json:"profiles"`
}

func toStudyProfileResponse(p studyprofile.Profile) studyProfileResponse {
	return studyProfileResponse{
		ID:             p.ID,
		Name:           p.Name,
		IsDefault:      p.IsDefault(),
		DailyCardLimit: p.DailyCardLimit,
		Rules:          rulesToBody(p.Rules),
	}
}

func (r studyProfileRequest) toInput() profileservice.Input {
	return profileservice.Input{Name: r.Name, DailyCardLimit: r.DailyCardLimit, Rules: r.Rules.toDomain()}
}

// StudyProfileHandlers implements the study configuration endpoints.
type StudyProfileHandlers struct {
	profiles profileservice.Service
}

func NewStudyProfileHandlers(profiles profileservice.Service) StudyProfileHandlers {
	return StudyProfileHandlers{profiles: profiles}
}

func decodeProfileRequest(w http.ResponseWriter, r *http.Request) (studyProfileRequest, bool) {
	var req studyProfileRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxProfileBodyBytes)).Decode(&req); err != nil {
		writeError(w, r, apperror.Validation("request.invalidJson", "request body must be valid JSON"))
		return req, false
	}
	return req, true
}

// List handles GET /api/v1/study-profiles: the built-in default first, then
// the user's own configurations, and which one is active.
func (h StudyProfileHandlers) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	overview, err := h.profiles.List(r.Context(), userID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	response := studyProfilesResponse{ActiveProfileID: overview.ActiveProfileID, Profiles: make([]studyProfileResponse, 0, len(overview.Profiles))}
	for _, p := range overview.Profiles {
		response.Profiles = append(response.Profiles, toStudyProfileResponse(p))
	}
	writeJSON(w, http.StatusOK, response)
}

// Create handles POST /api/v1/study-profiles.
func (h StudyProfileHandlers) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	req, ok := decodeProfileRequest(w, r)
	if !ok {
		return
	}

	created, err := h.profiles.Create(r.Context(), userID, req.toInput())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toStudyProfileResponse(created))
}

// Update handles PUT /api/v1/study-profiles/{profileID}. The built-in
// default cannot be changed.
func (h StudyProfileHandlers) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	req, ok := decodeProfileRequest(w, r)
	if !ok {
		return
	}

	updated, err := h.profiles.Update(r.Context(), userID, r.PathValue("profileID"), req.toInput())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toStudyProfileResponse(updated))
}

// Delete handles DELETE /api/v1/study-profiles/{profileID}. If it was the
// active configuration the user goes back to the default.
func (h StudyProfileHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	if err := h.profiles.Delete(r.Context(), userID, r.PathValue("profileID")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type activeProfileRequest struct {
	ProfileID string `json:"profileId"`
}

// SetActive handles PUT /api/v1/me/active-study-profile: choose which
// configuration to study with ("default" for the built-in one).
func (h StudyProfileHandlers) SetActive(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	var req activeProfileRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxProfileBodyBytes)).Decode(&req); err != nil {
		writeError(w, r, apperror.Validation("request.invalidJson.profileId", "request body must be valid JSON with a \"profileId\" field"))
		return
	}
	if err := h.profiles.SetActive(r.Context(), userID, req.ProfileID); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SetDeckProfile handles PUT /api/v1/decks/{deckID}/study-profile: choose the
// study configuration one deck uses instead of the general one. The body is
// {"profileId": "<id>"}, {"profileId": "default"} for the built-in one, or an
// empty/null profileId to follow the general configuration again.
func (h StudyProfileHandlers) SetDeckProfile(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	var req struct {
		ProfileID *string `json:"profileId"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxProfileBodyBytes)).Decode(&req); err != nil {
		writeError(w, r, apperror.Validation("request.invalidJson.profileId", "request body must be valid JSON with a \"profileId\" field"))
		return
	}
	profileID := ""
	if req.ProfileID != nil {
		profileID = *req.ProfileID
	}
	if err := h.profiles.SetDeckProfile(r.Context(), userID, r.PathValue("deckID"), profileID); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

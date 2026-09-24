package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"flashcard-backend/internal/apperror"
)

// errorEnvelope is the API error format from spec section 20.
type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code string `json:"code"`
	// Key is a stable, machine-readable identifier for exactly which
	// validation or lookup failed (e.g. "deck.name.tooLong"), meant for a
	// client to translate; Message is the English fallback.
	Key       string `json:"key"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

var codeToStatus = map[apperror.Code]int{
	apperror.CodeValidationError:      http.StatusBadRequest,
	apperror.CodeUnauthorized:         http.StatusUnauthorized,
	apperror.CodeForbidden:            http.StatusForbidden,
	apperror.CodeNotFound:             http.StatusNotFound,
	apperror.CodeConflict:             http.StatusConflict,
	apperror.CodePayloadTooLarge:      http.StatusRequestEntityTooLarge,
	apperror.CodeUnsupportedMediaType: http.StatusUnsupportedMediaType,
	apperror.CodeRateLimited:          http.StatusTooManyRequests,
	apperror.CodeInternalError:        http.StatusInternalServerError,
}

// writeError translates err into the standard JSON error envelope. If err
// is not an *apperror.Error, it is treated as an unexpected internal
// error: the original error is logged server-side (never sent to the
// client) and a generic INTERNAL_ERROR is returned.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		appErr = apperror.Internal(err)
	}

	if appErr.Code == apperror.CodeInternalError {
		slog.ErrorContext(r.Context(), "internal error",
			"error", appErr.Err,
			"requestId", requestIDFromContext(r.Context()),
		)
	}

	if fields := logFieldsFromContext(r.Context()); fields != nil {
		fields.errorCode = string(appErr.Code)
	}

	status, ok := codeToStatus[appErr.Code]
	if !ok {
		status = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorEnvelope{Error: errorBody{
		Code:      string(appErr.Code),
		Key:       appErr.Key,
		Message:   appErr.Message,
		RequestID: requestIDFromContext(r.Context()),
	}})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

package httpapi

import (
	"net/http"
	"time"

	"flashcard-backend/internal/apperror"
)

// locationFromQuery reads the optional tz query parameter (an IANA time
// zone name such as "America/Sao_Paulo"). Endpoints that depend on what
// "today" is use it so days follow the user's own clock; without it days
// are UTC.
func locationFromQuery(r *http.Request) (*time.Location, error) {
	tz := r.URL.Query().Get("tz")
	if tz == "" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, apperror.Validation("timezone.invalid", "tz must be a valid IANA time zone name, e.g. America/Sao_Paulo")
	}
	return loc, nil
}

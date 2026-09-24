// Package httpapi contains the HTTP adapter: routing and handlers that
// translate HTTP requests into application-service calls.
package httpapi

import (
	"context"
	"net/http"
	"time"
)

// Pinger checks connectivity to a dependency (MongoDB in practice). It
// lives here, rather than importing a MongoDB type directly, so this
// package has no compile-time dependency on the database driver.
type Pinger interface {
	Ping(ctx context.Context) error
}

// healthResponse is the payload returned by the health endpoint.
type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database,omitempty"`
}

// HealthHandlers serves GET /health.
type HealthHandlers struct {
	// db is nil when no database connectivity check is configured (e.g.
	// in HTTP-layer unit tests), in which case only liveness is reported.
	db Pinger
}

func NewHealthHandlers(db Pinger) HealthHandlers {
	return HealthHandlers{db: db}
}

// Handle reports application liveness and, if a Pinger was configured,
// database connectivity (spec section 21).
func (h HealthHandlers) Handle(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{Status: "ok"}
	status := http.StatusOK

	if h.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := h.db.Ping(ctx); err != nil {
			resp.Status = "degraded"
			resp.Database = "unavailable"
			status = http.StatusServiceUnavailable
		} else {
			resp.Database = "ok"
		}
	}

	writeJSON(w, status, resp)
}

package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

type contextKey int

const (
	requestIDContextKey contextKey = iota
	userIDContextKey
	logFieldsContextKey
)

// logFields is a mutable holder injected into the request context so a
// handler deep in the call stack (writeError) can attach fields — like
// the error code — that the single access-log line at the end of the
// request should include (spec section 21: "error code" and "relevant
// resource IDs").
type logFields struct {
	errorCode string
}

func logFieldsFromContext(ctx context.Context) *logFields {
	f, _ := ctx.Value(logFieldsContextKey).(*logFields)
	return f
}

// withRequestID assigns a request ID (generated, or forwarded from
// X-Request-ID) to every request, storing it in the context so handlers
// and error responses can include it (spec section 21).
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDContextKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newRequestID() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func requestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDContextKey).(string)
	return id
}

// withAccessLog logs one structured line per request: method, route
// pattern, status, duration, request ID, the error code if the request
// failed, and any deck/flashcard ID in the path. It never logs request
// bodies, headers, or query strings, which may carry the session cookie
// or an email address (spec section 21).
func withAccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		fields := &logFields{}
		ctx := context.WithValue(r.Context(), logFieldsContextKey, fields)
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r.WithContext(ctx))

		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"durationMs", time.Since(start).Milliseconds(),
			"requestId", requestIDFromContext(r.Context()),
		}
		if fields.errorCode != "" {
			attrs = append(attrs, "errorCode", fields.errorCode)
		}
		if deckID := r.PathValue("deckID"); deckID != "" {
			attrs = append(attrs, "deckId", deckID)
		}
		if flashcardID := r.PathValue("flashcardID"); flashcardID != "" {
			attrs = append(attrs, "flashcardId", flashcardID)
		}
		slog.InfoContext(r.Context(), "request", attrs...)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func userIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDContextKey).(string)
	return id, ok
}

func contextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey, userID)
}

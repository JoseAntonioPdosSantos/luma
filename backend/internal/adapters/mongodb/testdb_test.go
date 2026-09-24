package mongodb

import (
	"context"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// testDatabase connects to a MongoDB instance for integration tests.
//
// It reads MONGODB_TEST_URI (default: mongodb://localhost:27018) and skips
// the test if that instance is unreachable, so `go test ./...` still
// passes in environments without MongoDB (e.g. plain `go build` CI steps),
// while `docker compose up -d mongo` unlocks these tests locally.
func testDatabase(t *testing.T) *mongo.Database {
	t.Helper()

	uri := os.Getenv("MONGODB_TEST_URI")
	if uri == "" {
		uri = "mongodb://localhost:27018"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	db, err := Connect(ctx, uri, "flashcard_test")
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

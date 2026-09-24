package gridfs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"

	mongoadapter "flashcard-backend/internal/adapters/mongodb"
	"flashcard-backend/internal/ports/audiostore"
)

func testDatabase(t *testing.T) *mongo.Database {
	t.Helper()

	uri := os.Getenv("MONGODB_TEST_URI")
	if uri == "" {
		uri = "mongodb://localhost:27018"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	db, err := mongoadapter.Connect(ctx, uri, "flashcard_gridfs_test")
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

func TestStore_SaveAndOpen(t *testing.T) {
	db := testDatabase(t)
	store, err := New(db)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	ctx := context.Background()

	fileID, size, err := store.Save(ctx, bytes.NewReader([]byte("fake audio bytes")), 1024)
	if err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	if fileID == "" {
		t.Fatal("Save() should return a non-empty file ID")
	}
	if size != int64(len("fake audio bytes")) {
		t.Errorf("size = %d, want %d", size, len("fake audio bytes"))
	}

	stream, err := store.Open(ctx, fileID)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("reading stream: %v", err)
	}
	if string(data) != "fake audio bytes" {
		t.Errorf("data = %q, want %q", data, "fake audio bytes")
	}
}

func TestStore_Save_RejectsOversizedFileWithoutPersisting(t *testing.T) {
	db := testDatabase(t)
	store, err := New(db)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	ctx := context.Background()

	oversized := bytes.Repeat([]byte{0}, 100)
	_, _, err = store.Save(ctx, bytes.NewReader(oversized), 50)
	if !errors.Is(err, audiostore.ErrTooLarge) {
		t.Fatalf("error = %v, want ErrTooLarge", err)
	}

	// No file should have been left behind.
	cursor, err := store.bucket.Find(map[string]any{})
	if err != nil {
		t.Fatalf("Find() error: %v", err)
	}
	defer cursor.Close(ctx)
	if cursor.Next(ctx) {
		t.Error("an oversized upload should not persist a file")
	}
}

func TestStore_Open_NotFound(t *testing.T) {
	db := testDatabase(t)
	store, err := New(db)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if _, err := store.Open(context.Background(), "000000000000000000000000"); !errors.Is(err, audiostore.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestStore_Open_MalformedIDIsNotFound(t *testing.T) {
	db := testDatabase(t)
	store, err := New(db)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if _, err := store.Open(context.Background(), "not-a-valid-object-id"); !errors.Is(err, audiostore.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestStore_Delete(t *testing.T) {
	db := testDatabase(t)
	store, err := New(db)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	ctx := context.Background()

	fileID, _, err := store.Save(ctx, bytes.NewReader([]byte("data")), 1024)
	if err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if err := store.Delete(ctx, fileID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	if _, err := store.Open(ctx, fileID); !errors.Is(err, audiostore.ErrNotFound) {
		t.Errorf("Open() after Delete() error = %v, want ErrNotFound", err)
	}
}

func TestStore_Delete_NonexistentIsNotAnError(t *testing.T) {
	db := testDatabase(t)
	store, err := New(db)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if err := store.Delete(context.Background(), "000000000000000000000000"); err != nil {
		t.Errorf("Delete() of a nonexistent file returned an error: %v", err)
	}
	if err := store.Delete(context.Background(), "not-a-valid-object-id"); err != nil {
		t.Errorf("Delete() of a malformed id returned an error: %v", err)
	}
}

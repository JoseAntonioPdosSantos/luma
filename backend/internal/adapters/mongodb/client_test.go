package mongodb

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// listIndexKeyPatterns renders each index's "key" document as a single
// comparable string, e.g. "userId:1,archivedAt:1", so tests can assert on
// shape without depending on driver-generated index names.
func listIndexKeyPatterns(t *testing.T, ctx context.Context, db *mongo.Database, collection string) []string {
	t.Helper()

	cursor, err := db.Collection(collection).Indexes().List(ctx)
	if err != nil {
		t.Fatalf("Indexes().List(%s) error: %v", collection, err)
	}
	defer cursor.Close(ctx)

	var patterns []string
	for cursor.Next(ctx) {
		var doc struct {
			Key bson.D `bson:"key"`
		}
		if err := cursor.Decode(&doc); err != nil {
			t.Fatalf("decoding index document: %v", err)
		}

		parts := make([]string, 0, len(doc.Key))
		for _, e := range doc.Key {
			parts = append(parts, fmt.Sprintf("%s:%v", e.Key, e.Value))
		}
		patterns = append(patterns, strings.Join(parts, ","))
	}
	return patterns
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

func TestEnsureIndexes_CreatesExpectedIndexes(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()

	if err := EnsureIndexes(ctx, db); err != nil {
		t.Fatalf("EnsureIndexes() error: %v", err)
	}

	tests := []struct {
		collection string
		want       []string
	}{
		{
			collection: CollectionUsers,
			want:       []string{"email:1"},
		},
		{
			collection: CollectionDecks,
			want:       []string{"userId:1,archivedAt:1"},
		},
		{
			collection: CollectionFlashcards,
			want:       []string{"userId:1,deckId:1,archivedAt:1,scheduling.dueAt:1"},
		},
		{
			collection: CollectionReviewEvents,
			want:       []string{"userId:1,deckId:1,reviewedAt:1", "flashcardId:1,reviewedAt:1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.collection, func(t *testing.T) {
			got := listIndexKeyPatterns(t, ctx, db, tt.collection)
			for _, want := range tt.want {
				if !contains(got, want) {
					t.Errorf("collection %s: indexes = %v, want one matching %q", tt.collection, got, want)
				}
			}
		})
	}
}

func TestEnsureIndexes_UniqueEmailIndexIsUnique(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()

	if err := EnsureIndexes(ctx, db); err != nil {
		t.Fatalf("EnsureIndexes() error: %v", err)
	}

	cursor, err := db.Collection(CollectionUsers).Indexes().List(ctx)
	if err != nil {
		t.Fatalf("Indexes().List() error: %v", err)
	}
	defer cursor.Close(ctx)

	found := false
	for cursor.Next(ctx) {
		var doc struct {
			Key    bson.D `bson:"key"`
			Unique bool   `bson:"unique"`
		}
		if err := cursor.Decode(&doc); err != nil {
			t.Fatalf("decoding index document: %v", err)
		}
		if len(doc.Key) == 1 && doc.Key[0].Key == "email" {
			found = true
			if !doc.Unique {
				t.Error("the email index should be unique")
			}
		}
	}
	if !found {
		t.Fatal("no index on email found")
	}
}

func TestEnsureIndexes_IsIdempotent(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()

	if err := EnsureIndexes(ctx, db); err != nil {
		t.Fatalf("first EnsureIndexes() error: %v", err)
	}
	if err := EnsureIndexes(ctx, db); err != nil {
		t.Fatalf("second EnsureIndexes() error: %v", err)
	}
}

// Package mongodb is the MongoDB adapter: it implements the repository
// ports using the official MongoDB Go driver and owns all
// collection/index/BSON-mapping details, keeping them out of the domain
// and application layers.
package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	CollectionUsers         = "users"
	CollectionDecks         = "decks"
	CollectionFlashcards    = "flashcards"
	CollectionReviewEvents  = "review_events"
	CollectionStudyProfiles = "study_profiles"
	CollectionDeckGroups    = "deck_groups"
)

// Connect dials MongoDB and pings it, retrying for a while so the API
// survives starting before the database is ready. It returns the named
// database handle.
func Connect(ctx context.Context, uri, database string) (*mongo.Database, error) {
	const (
		attempts = 10
		delay    = 3 * time.Second
	)

	var lastErr error
	for i := 1; i <= attempts; i++ {
		db, err := connectOnce(ctx, uri, database)
		if err == nil {
			return db, nil
		}
		lastErr = err
		slog.Warn("mongodb not ready, retrying", "attempt", i, "of", attempts, "error", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}
	return nil, lastErr
}

func connectOnce(ctx context.Context, uri, database string) (*mongo.Database, error) {
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(connectCtx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connecting to mongodb: %w", err)
	}

	pingCtx, cancelPing := context.WithTimeout(ctx, 5*time.Second)
	defer cancelPing()
	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("pinging mongodb: %w", err)
	}

	return client.Database(database), nil
}

// Pinger implements httpapi.Pinger for the health endpoint, without that
// package needing a compile-time dependency on the MongoDB driver.
type Pinger struct {
	db *mongo.Database
}

func NewPinger(db *mongo.Database) Pinger {
	return Pinger{db: db}
}

func (p Pinger) Ping(ctx context.Context) error {
	return p.db.Client().Ping(ctx, nil)
}

// EnsureIndexes creates the indexes this application actually queries by.
// It is safe to call on every startup: creating an index that already
// exists with the same options is a no-op.
//
// This intentionally deviates from the example index list in spec
// section 18 in two places, per that same section's instruction to
// "review the indexes using realistic query patterns" rather than create
// one without a backing query:
//
//   - decks: the example suggests userId+name, to support rejecting
//     duplicate active deck names. That rule is explicitly optional in
//     section 6 ("if this rule is selected during implementation") and
//     was not implemented, so no query uses this shape; the index is
//     omitted. Add it back if duplicate-name rejection is implemented.
//   - flashcards: the example suggests two separate two-field indexes
//     (userId+dueAt, deckId+dueAt), but every real query
//     (ListByDeck, ListDue, CountByDeck) filters by userId AND deckId AND
//     archivedAt together, with ListDue/CountByDeck additionally
//     filtering and sorting by scheduling.dueAt. A single four-field
//     compound index serves all three methods (MongoDB can use a prefix
//     of a compound index), following the Equality-Sort-Range field
//     ordering convention, so the narrower indexes were replaced by it.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	models := map[string][]mongo.IndexModel{
		CollectionUsers: {
			{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		CollectionStudyProfiles: {
			// A user's configuration names are unique (ignoring case);
			// this also serves ListByUser as a prefix.
			{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "nameKey", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		CollectionDeckGroups: {
			// A user's group names are unique (ignoring case); this also
			// serves ListByUser as a prefix.
			{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "nameKey", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		CollectionDecks: {
			// ListActive(userId) filters archivedAt:nil.
			{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "archivedAt", Value: 1}}},
		},
		CollectionFlashcards: {
			// Serves ListByDeck (userId, deckId, archivedAt) as a prefix,
			// and ListDue/CountByDeck (all four fields; dueAt is both
			// range-filtered and the sort key, so it goes last).
			{Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "deckId", Value: 1},
				{Key: "archivedAt", Value: 1},
				{Key: "scheduling.dueAt", Value: 1},
			}},
		},
		CollectionReviewEvents: {
			// Serves DailyCounts (userId, deckId, reviewedAt range) for the
			// deck statistics screen.
			{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "deckId", Value: 1}, {Key: "reviewedAt", Value: 1}}},
			// Kept from the spec's example index list ahead of any query
			// using it (spec section 6: future debugging/auditability by
			// flashcard), since this collection is insert-only and only
			// grows — adding it later means building it against a much
			// larger collection than adding it now.
			{Keys: bson.D{{Key: "flashcardId", Value: 1}, {Key: "reviewedAt", Value: 1}}},
		},
	}

	for collection, indexes := range models {
		if _, err := db.Collection(collection).Indexes().CreateMany(ctx, indexes); err != nil {
			return fmt.Errorf("creating indexes for %s: %w", collection, err)
		}
	}
	return nil
}

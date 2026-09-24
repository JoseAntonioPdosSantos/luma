package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"flashcard-backend/internal/domain/review"
	"flashcard-backend/internal/domain/study"
	"flashcard-backend/internal/ports/repositories"
)

type reviewEventDocument struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	UserID         primitive.ObjectID `bson:"userId"`
	FlashcardID    primitive.ObjectID `bson:"flashcardId"`
	DeckID         primitive.ObjectID `bson:"deckId"`
	Rating         study.Rating       `bson:"rating"`
	PreviousState  study.CardState    `bson:"previousState"`
	NextState      study.CardState    `bson:"nextState"`
	ReviewedAt     time.Time          `bson:"reviewedAt"`
	ResponseTimeMs int64              `bson:"responseTimeMs"`
	HintUsed       bool               `bson:"hintUsed"`
}

func (d reviewEventDocument) toDomain() review.Event {
	return review.Event{
		ID:             d.ID.Hex(),
		UserID:         d.UserID.Hex(),
		FlashcardID:    d.FlashcardID.Hex(),
		DeckID:         d.DeckID.Hex(),
		Rating:         d.Rating,
		PreviousState:  d.PreviousState,
		NextState:      d.NextState,
		ReviewedAt:     d.ReviewedAt,
		ResponseTimeMs: d.ResponseTimeMs,
		HintUsed:       d.HintUsed,
	}
}

// ReviewEventRepository implements repositories.ReviewEventRepository
// against MongoDB.
type ReviewEventRepository struct {
	collection *mongo.Collection
}

func NewReviewEventRepository(db *mongo.Database) *ReviewEventRepository {
	return &ReviewEventRepository{collection: db.Collection(CollectionReviewEvents)}
}

func (r *ReviewEventRepository) DeleteByDeck(ctx context.Context, userID, deckID string) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}
	deckObjectID, err := primitive.ObjectIDFromHex(deckID)
	if err != nil {
		return err
	}
	_, err = r.collection.DeleteMany(ctx, bson.M{"userId": userObjectID, "deckId": deckObjectID})
	return err
}

func (r *ReviewEventRepository) Create(ctx context.Context, e review.Event) (review.Event, error) {
	userObjectID, err := primitive.ObjectIDFromHex(e.UserID)
	if err != nil {
		return review.Event{}, err
	}
	flashcardObjectID, err := primitive.ObjectIDFromHex(e.FlashcardID)
	if err != nil {
		return review.Event{}, err
	}
	deckObjectID, err := primitive.ObjectIDFromHex(e.DeckID)
	if err != nil {
		return review.Event{}, err
	}

	doc := reviewEventDocument{
		ID:             primitive.NewObjectID(),
		UserID:         userObjectID,
		FlashcardID:    flashcardObjectID,
		DeckID:         deckObjectID,
		Rating:         e.Rating,
		PreviousState:  e.PreviousState,
		NextState:      e.NextState,
		ReviewedAt:     e.ReviewedAt,
		ResponseTimeMs: e.ResponseTimeMs,
		HintUsed:       e.HintUsed,
	}
	if _, err := r.collection.InsertOne(ctx, doc); err != nil {
		return review.Event{}, err
	}
	return doc.toDomain(), nil
}

const dayFormat = "2006-01-02"

func (r *ReviewEventRepository) CountStudiedCards(ctx context.Context, userID, deckID string, since time.Time) (int, error) {
	filter, err := deckScopeFilter(userID, deckID)
	if err != nil {
		return 0, err
	}
	if !since.IsZero() {
		filter["reviewedAt"] = bson.M{"$gte": since}
	}

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: filter}},
		bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$flashcardId"}}}},
		bson.D{{Key: "$count", Value: "studied"}},
	}
	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	// $count emits no document at all when nothing matched.
	if !cursor.Next(ctx) {
		return 0, cursor.Err()
	}
	var row struct {
		Studied int `bson:"studied"`
	}
	if err := cursor.Decode(&row); err != nil {
		return 0, err
	}
	return row.Studied, nil
}

func (r *ReviewEventRepository) DailyCounts(ctx context.Context, userID, deckID string, since time.Time, loc *time.Location) ([]repositories.DailyReviewCount, error) {
	if loc == nil {
		loc = time.UTC
	}
	filter, err := deckScopeFilter(userID, deckID)
	if err != nil {
		return nil, err
	}
	filter["reviewedAt"] = bson.M{"$gte": since}

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: filter}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "$dateToString", Value: bson.D{
				{Key: "format", Value: "%Y-%m-%d"},
				{Key: "date", Value: "$reviewedAt"},
				{Key: "timezone", Value: loc.String()},
			}}}},
			{Key: "reviewCount", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "correctCount", Value: bson.D{{Key: "$sum", Value: bson.D{
				{Key: "$cond", Value: bson.A{
					bson.D{{Key: "$eq", Value: bson.A{"$rating", study.RatingAgain}}},
					0, 1,
				}},
			}}}},
			// Events recorded before hints existed have no hintUsed field,
			// which "$eq: true" treats as not used.
			{Key: "hintCount", Value: bson.D{{Key: "$sum", Value: bson.D{
				{Key: "$cond", Value: bson.A{
					bson.D{{Key: "$eq", Value: bson.A{"$hintUsed", true}}},
					1, 0,
				}},
			}}}},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []repositories.DailyReviewCount
	for cursor.Next(ctx) {
		var row struct {
			Day          string `bson:"_id"`
			ReviewCount  int    `bson:"reviewCount"`
			CorrectCount int    `bson:"correctCount"`
			HintCount    int    `bson:"hintCount"`
		}
		if err := cursor.Decode(&row); err != nil {
			return nil, err
		}
		date, err := time.Parse(dayFormat, row.Day)
		if err != nil {
			return nil, fmt.Errorf("parsing aggregated date %q: %w", row.Day, err)
		}
		results = append(results, repositories.DailyReviewCount{
			Date:         date,
			ReviewCount:  row.ReviewCount,
			CorrectCount: row.CorrectCount,
			HintCount:    row.HintCount,
		})
	}
	return results, cursor.Err()
}

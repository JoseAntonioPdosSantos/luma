package mongodb

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"flashcard-backend/internal/domain/study"
	"flashcard-backend/internal/domain/studyprofile"
	"flashcard-backend/internal/ports/repositories"
)

type ratingRuleDocument struct {
	FirstIntervalDays int     `bson:"firstIntervalDays"`
	Multiplier        float64 `bson:"multiplier"`
}

type rulesDocument struct {
	AgainDelayMinutes int                `bson:"againDelayMinutes"`
	Hard              ratingRuleDocument `bson:"hard"`
	Good              ratingRuleDocument `bson:"good"`
	Easy              ratingRuleDocument `bson:"easy"`
}

func rulesToDocument(r study.Rules) rulesDocument {
	return rulesDocument{
		AgainDelayMinutes: r.AgainDelayMinutes,
		Hard:              ratingRuleDocument(r.Hard),
		Good:              ratingRuleDocument(r.Good),
		Easy:              ratingRuleDocument(r.Easy),
	}
}

func (d rulesDocument) toDomain() study.Rules {
	return study.Rules{
		AgainDelayMinutes: d.AgainDelayMinutes,
		Hard:              study.RatingRule(d.Hard),
		Good:              study.RatingRule(d.Good),
		Easy:              study.RatingRule(d.Easy),
	}
}

type studyProfileDocument struct {
	ID     primitive.ObjectID `bson:"_id,omitempty"`
	UserID primitive.ObjectID `bson:"userId"`
	Name   string             `bson:"name"`
	// NameKey is the lower-cased name; a unique index on (userId, nameKey)
	// keeps a user's configuration names distinct.
	NameKey        string        `bson:"nameKey"`
	DailyCardLimit *int          `bson:"dailyCardLimit"`
	Rules          rulesDocument `bson:"rules"`
	CreatedAt      time.Time     `bson:"createdAt"`
	UpdatedAt      time.Time     `bson:"updatedAt"`
}

func (d studyProfileDocument) toDomain() studyprofile.Profile {
	return studyprofile.Profile{
		ID:             d.ID.Hex(),
		UserID:         d.UserID.Hex(),
		Name:           d.Name,
		DailyCardLimit: d.DailyCardLimit,
		Rules:          d.Rules.toDomain(),
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}
}

// StudyProfileRepository implements repositories.StudyProfileRepository
// against MongoDB.
type StudyProfileRepository struct {
	collection *mongo.Collection
}

func NewStudyProfileRepository(db *mongo.Database) *StudyProfileRepository {
	return &StudyProfileRepository{collection: db.Collection(CollectionStudyProfiles)}
}

func (r *StudyProfileRepository) Create(ctx context.Context, p studyprofile.Profile) (studyprofile.Profile, error) {
	userObjectID, err := primitive.ObjectIDFromHex(p.UserID)
	if err != nil {
		return studyprofile.Profile{}, err
	}
	doc := studyProfileDocument{
		ID:             primitive.NewObjectID(),
		UserID:         userObjectID,
		Name:           p.Name,
		NameKey:        studyprofile.NameKey(p.Name),
		DailyCardLimit: p.DailyCardLimit,
		Rules:          rulesToDocument(p.Rules),
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
	if _, err := r.collection.InsertOne(ctx, doc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return studyprofile.Profile{}, repositories.ErrDuplicate
		}
		return studyprofile.Profile{}, err
	}
	return doc.toDomain(), nil
}

func (r *StudyProfileRepository) FindByID(ctx context.Context, userID, id string) (studyprofile.Profile, error) {
	filter, err := ownedFilter(userID, id)
	if err != nil {
		return studyprofile.Profile{}, repositories.ErrNotFound
	}
	var doc studyProfileDocument
	err = r.collection.FindOne(ctx, filter).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return studyprofile.Profile{}, repositories.ErrNotFound
	}
	if err != nil {
		return studyprofile.Profile{}, err
	}
	return doc.toDomain(), nil
}

func (r *StudyProfileRepository) ListByUser(ctx context.Context, userID string) ([]studyprofile.Profile, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}
	cursor, err := r.collection.Find(ctx, bson.M{"userId": userObjectID},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}, {Key: "_id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	profiles := []studyprofile.Profile{}
	for cursor.Next(ctx) {
		var doc studyProfileDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		profiles = append(profiles, doc.toDomain())
	}
	return profiles, cursor.Err()
}

func (r *StudyProfileRepository) Update(ctx context.Context, p studyprofile.Profile) (studyprofile.Profile, error) {
	filter, err := ownedFilter(p.UserID, p.ID)
	if err != nil {
		return studyprofile.Profile{}, repositories.ErrNotFound
	}
	result := r.collection.FindOneAndUpdate(ctx, filter,
		bson.M{"$set": bson.M{
			"name":           p.Name,
			"nameKey":        studyprofile.NameKey(p.Name),
			"dailyCardLimit": p.DailyCardLimit,
			"rules":          rulesToDocument(p.Rules),
			"updatedAt":      p.UpdatedAt,
		}},
		mongoReturnAfter(),
	)
	var doc studyProfileDocument
	if err := result.Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return studyprofile.Profile{}, repositories.ErrNotFound
		}
		if mongo.IsDuplicateKeyError(err) {
			return studyprofile.Profile{}, repositories.ErrDuplicate
		}
		return studyprofile.Profile{}, err
	}
	return doc.toDomain(), nil
}

func (r *StudyProfileRepository) Delete(ctx context.Context, userID, id string) error {
	filter, err := ownedFilter(userID, id)
	if err != nil {
		return repositories.ErrNotFound
	}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return repositories.ErrNotFound
	}
	return nil
}

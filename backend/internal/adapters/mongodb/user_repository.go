package mongodb

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"flashcard-backend/internal/domain/user"
	"flashcard-backend/internal/ports/repositories"
)

type userDocument struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Email        string             `bson:"email"`
	CreatedAt    time.Time          `bson:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt"`
	LastAccessAt time.Time          `bson:"lastAccessAt"`
	// The study configuration the user selected ("" = the built-in default).
	ActiveStudyProfileID string `bson:"activeStudyProfileId,omitempty"`
	// The daily goal saved before study configurations existed; only read to
	// migrate it.
	Settings legacySettingsDocument `bson:"settings"`
	// The local day (YYYY-MM-DD) the user chose to study past the goal.
	ContinuePastGoalOn string `bson:"continuePastGoalOn,omitempty"`
	// The UI language the user chose ("" = none, client detects the browser's).
	PreferredLanguage string `bson:"preferredLanguage,omitempty"`
}

type legacySettingsDocument struct {
	DailyCardLimit *int `bson:"dailyCardLimit"`
}

func (d userDocument) toDomain() user.User {
	return user.User{
		ID:                   d.ID.Hex(),
		Email:                d.Email,
		CreatedAt:            d.CreatedAt,
		UpdatedAt:            d.UpdatedAt,
		LastAccessAt:         d.LastAccessAt,
		ActiveStudyProfileID: d.ActiveStudyProfileID,
		LegacyDailyCardLimit: d.Settings.DailyCardLimit,
		ContinuePastGoalOn:   d.ContinuePastGoalOn,
		PreferredLanguage:    d.PreferredLanguage,
	}
}

// UserRepository implements repositories.UserRepository against MongoDB.
type UserRepository struct {
	collection *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{collection: db.Collection(CollectionUsers)}
}

func (r *UserRepository) FindByEmail(ctx context.Context, normalizedEmail string) (user.User, error) {
	var doc userDocument
	err := r.collection.FindOne(ctx, bson.M{"email": normalizedEmail}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return user.User{}, repositories.ErrNotFound
	}
	if err != nil {
		return user.User{}, err
	}
	return doc.toDomain(), nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (user.User, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return user.User{}, repositories.ErrNotFound
	}

	var doc userDocument
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return user.User{}, repositories.ErrNotFound
	}
	if err != nil {
		return user.User{}, err
	}
	return doc.toDomain(), nil
}

func (r *UserRepository) Create(ctx context.Context, u user.User) (user.User, error) {
	doc := userDocument{
		ID:           primitive.NewObjectID(),
		Email:        u.Email,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
		LastAccessAt: u.LastAccessAt,
	}
	if _, err := r.collection.InsertOne(ctx, doc); err != nil {
		return user.User{}, err
	}
	return doc.toDomain(), nil
}

func (r *UserRepository) TouchLastAccess(ctx context.Context, id string, now time.Time) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return repositories.ErrNotFound
	}

	_, err = r.collection.UpdateOne(ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": bson.M{"lastAccessAt": now}},
	)
	return err
}

func (r *UserRepository) SetActiveStudyProfile(ctx context.Context, id, profileID string, now time.Time) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return repositories.ErrNotFound
	}

	result, err := r.collection.UpdateOne(ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": bson.M{"activeStudyProfileId": profileID, "updatedAt": now}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return repositories.ErrNotFound
	}
	return nil
}

func (r *UserRepository) ClearLegacyDailyCardLimit(ctx context.Context, id string, now time.Time) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return repositories.ErrNotFound
	}
	_, err = r.collection.UpdateOne(ctx,
		bson.M{"_id": objectID},
		bson.M{"$unset": bson.M{"settings": ""}, "$set": bson.M{"updatedAt": now}},
	)
	return err
}

func (r *UserRepository) SetContinuePastGoalOn(ctx context.Context, id, day string, now time.Time) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return repositories.ErrNotFound
	}

	result, err := r.collection.UpdateOne(ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": bson.M{"continuePastGoalOn": day, "updatedAt": now}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return repositories.ErrNotFound
	}
	return nil
}

func (r *UserRepository) SetPreferredLanguage(ctx context.Context, id, language string, now time.Time) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return repositories.ErrNotFound
	}

	result, err := r.collection.UpdateOne(ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": bson.M{"preferredLanguage": language, "updatedAt": now}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return repositories.ErrNotFound
	}
	return nil
}

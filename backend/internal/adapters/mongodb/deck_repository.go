package mongodb

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"flashcard-backend/internal/domain/deck"
	"flashcard-backend/internal/ports/repositories"
)

type deckDocument struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	UserID      primitive.ObjectID `bson:"userId"`
	Name        string             `bson:"name"`
	Description string             `bson:"description"`
	// The study configuration this deck uses instead of the general one
	// ("" = the general one) and the day the user chose to study it past
	// its own daily goal.
	StudyProfileID     string `bson:"studyProfileId,omitempty"`
	ContinuePastGoalOn string `bson:"continuePastGoalOn,omitempty"`
	// The deckgroup.Group this deck is filed under ("" = none).
	GroupID    string     `bson:"groupId,omitempty"`
	ArchivedAt *time.Time `bson:"archivedAt"`
	CreatedAt  time.Time  `bson:"createdAt"`
	UpdatedAt  time.Time  `bson:"updatedAt"`
}

func (d deckDocument) toDomain() deck.Deck {
	return deck.Deck{
		ID:                 d.ID.Hex(),
		UserID:             d.UserID.Hex(),
		Name:               d.Name,
		Description:        d.Description,
		StudyProfileID:     d.StudyProfileID,
		ContinuePastGoalOn: d.ContinuePastGoalOn,
		GroupID:            d.GroupID,
		ArchivedAt:         d.ArchivedAt,
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
	}
}

// DeckRepository implements repositories.DeckRepository against MongoDB.
type DeckRepository struct {
	collection *mongo.Collection
}

func NewDeckRepository(db *mongo.Database) *DeckRepository {
	return &DeckRepository{collection: db.Collection(CollectionDecks)}
}

func (r *DeckRepository) Create(ctx context.Context, d deck.Deck) (deck.Deck, error) {
	userObjectID, err := primitive.ObjectIDFromHex(d.UserID)
	if err != nil {
		return deck.Deck{}, err
	}

	doc := deckDocument{
		ID:          primitive.NewObjectID(),
		UserID:      userObjectID,
		Name:        d.Name,
		Description: d.Description,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
	if _, err := r.collection.InsertOne(ctx, doc); err != nil {
		return deck.Deck{}, err
	}
	return doc.toDomain(), nil
}

func (r *DeckRepository) FindByID(ctx context.Context, userID, id string) (deck.Deck, error) {
	filter, err := ownedFilter(userID, id)
	if err != nil {
		return deck.Deck{}, repositories.ErrNotFound
	}

	var doc deckDocument
	err = r.collection.FindOne(ctx, filter).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return deck.Deck{}, repositories.ErrNotFound
	}
	if err != nil {
		return deck.Deck{}, err
	}
	return doc.toDomain(), nil
}

func (r *DeckRepository) ListActive(ctx context.Context, userID string) ([]deck.Deck, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	cursor, err := r.collection.Find(ctx, bson.M{
		"userId":     userObjectID,
		"archivedAt": nil,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	decks := []deck.Deck{}
	for cursor.Next(ctx) {
		var doc deckDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		decks = append(decks, doc.toDomain())
	}
	return decks, cursor.Err()
}

func (r *DeckRepository) ListArchived(ctx context.Context, userID string) ([]deck.Deck, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	cursor, err := r.collection.Find(ctx,
		bson.M{"userId": userObjectID, "archivedAt": bson.M{"$ne": nil}},
		options.Find().SetSort(bson.D{{Key: "archivedAt", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	decks := []deck.Deck{}
	for cursor.Next(ctx) {
		var doc deckDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		decks = append(decks, doc.toDomain())
	}
	return decks, cursor.Err()
}

func (r *DeckRepository) Update(ctx context.Context, d deck.Deck) (deck.Deck, error) {
	filter, err := ownedFilter(d.UserID, d.ID)
	if err != nil {
		return deck.Deck{}, repositories.ErrNotFound
	}

	result := r.collection.FindOneAndUpdate(ctx, filter,
		bson.M{"$set": bson.M{
			"name":        d.Name,
			"description": d.Description,
			"updatedAt":   d.UpdatedAt,
		}},
		mongoReturnAfter(),
	)

	var doc deckDocument
	if err := result.Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return deck.Deck{}, repositories.ErrNotFound
		}
		return deck.Deck{}, err
	}
	return doc.toDomain(), nil
}

func (r *DeckRepository) Archive(ctx context.Context, userID, id string, now time.Time) error {
	filter, err := ownedFilter(userID, id)
	if err != nil {
		return repositories.ErrNotFound
	}

	result, err := r.collection.UpdateOne(ctx, filter,
		bson.M{"$set": bson.M{"archivedAt": now, "updatedAt": now}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return repositories.ErrNotFound
	}
	return nil
}

func (r *DeckRepository) Restore(ctx context.Context, userID, id string, now time.Time) error {
	filter, err := ownedFilter(userID, id)
	if err != nil {
		return repositories.ErrNotFound
	}

	result, err := r.collection.UpdateOne(ctx, filter,
		bson.M{"$set": bson.M{"archivedAt": nil, "updatedAt": now}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return repositories.ErrNotFound
	}
	return nil
}

func (r *DeckRepository) DeleteArchived(ctx context.Context, userID, id string) error {
	filter, err := ownedFilter(userID, id)
	if err != nil {
		return repositories.ErrNotFound
	}
	filter["archivedAt"] = bson.M{"$ne": nil}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return repositories.ErrNotFound
	}
	return nil
}

func ownedFilter(userID, id string) (bson.M, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	return bson.M{"_id": objectID, "userId": userObjectID}, nil
}

func (r *DeckRepository) SetStudyProfile(ctx context.Context, userID, id, profileID string, now time.Time) error {
	filter, err := ownedFilter(userID, id)
	if err != nil {
		return repositories.ErrNotFound
	}

	// Changing configuration starts over the "keep studying past the goal"
	// choice: it was made against another goal.
	update := bson.M{"$set": bson.M{"updatedAt": now}}
	if profileID == "" {
		update["$unset"] = bson.M{"studyProfileId": "", "continuePastGoalOn": ""}
	} else {
		update["$set"].(bson.M)["studyProfileId"] = profileID
		update["$unset"] = bson.M{"continuePastGoalOn": ""}
	}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return repositories.ErrNotFound
	}
	return nil
}

func (r *DeckRepository) SetContinuePastGoalOn(ctx context.Context, userID, id, day string, now time.Time) error {
	filter, err := ownedFilter(userID, id)
	if err != nil {
		return repositories.ErrNotFound
	}
	result, err := r.collection.UpdateOne(ctx, filter,
		bson.M{"$set": bson.M{"continuePastGoalOn": day, "updatedAt": now}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return repositories.ErrNotFound
	}
	return nil
}

func (r *DeckRepository) ClearStudyProfile(ctx context.Context, userID, profileID string) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}
	_, err = r.collection.UpdateMany(ctx,
		bson.M{"userId": userObjectID, "studyProfileId": profileID},
		bson.M{"$unset": bson.M{"studyProfileId": "", "continuePastGoalOn": ""}},
	)
	return err
}

func (r *DeckRepository) SetGroup(ctx context.Context, userID, id, groupID string, now time.Time) error {
	filter, err := ownedFilter(userID, id)
	if err != nil {
		return repositories.ErrNotFound
	}

	update := bson.M{"$set": bson.M{"updatedAt": now}}
	if groupID == "" {
		update["$unset"] = bson.M{"groupId": ""}
	} else {
		update["$set"].(bson.M)["groupId"] = groupID
	}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return repositories.ErrNotFound
	}
	return nil
}

func (r *DeckRepository) ClearGroup(ctx context.Context, userID, groupID string) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}
	_, err = r.collection.UpdateMany(ctx,
		bson.M{"userId": userObjectID, "groupId": groupID},
		bson.M{"$unset": bson.M{"groupId": ""}},
	)
	return err
}

func (r *DeckRepository) ListWithStudyProfile(ctx context.Context, userID string) ([]deck.Deck, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}
	cursor, err := r.collection.Find(ctx, bson.M{"userId": userObjectID, "studyProfileId": bson.M{"$exists": true, "$ne": ""}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	decks := []deck.Deck{}
	for cursor.Next(ctx) {
		var doc deckDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		decks = append(decks, doc.toDomain())
	}
	return decks, cursor.Err()
}

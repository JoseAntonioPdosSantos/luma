package mongodb

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"flashcard-backend/internal/domain/deckgroup"
	"flashcard-backend/internal/ports/repositories"
)

type deckGroupDocument struct {
	ID     primitive.ObjectID `bson:"_id,omitempty"`
	UserID primitive.ObjectID `bson:"userId"`
	Name   string             `bson:"name"`
	// NameKey is the lower-cased name; a unique index on (userId, nameKey)
	// keeps a user's group names distinct.
	NameKey   string    `bson:"nameKey"`
	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

func (d deckGroupDocument) toDomain() deckgroup.Group {
	return deckgroup.Group{
		ID:        d.ID.Hex(),
		UserID:    d.UserID.Hex(),
		Name:      d.Name,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

// DeckGroupRepository implements repositories.DeckGroupRepository against
// MongoDB.
type DeckGroupRepository struct {
	collection *mongo.Collection
}

func NewDeckGroupRepository(db *mongo.Database) *DeckGroupRepository {
	return &DeckGroupRepository{collection: db.Collection(CollectionDeckGroups)}
}

func (r *DeckGroupRepository) Create(ctx context.Context, g deckgroup.Group) (deckgroup.Group, error) {
	userObjectID, err := primitive.ObjectIDFromHex(g.UserID)
	if err != nil {
		return deckgroup.Group{}, err
	}
	doc := deckGroupDocument{
		ID:        primitive.NewObjectID(),
		UserID:    userObjectID,
		Name:      g.Name,
		NameKey:   deckgroup.NameKey(g.Name),
		CreatedAt: g.CreatedAt,
		UpdatedAt: g.UpdatedAt,
	}
	if _, err := r.collection.InsertOne(ctx, doc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return deckgroup.Group{}, repositories.ErrDuplicate
		}
		return deckgroup.Group{}, err
	}
	return doc.toDomain(), nil
}

func (r *DeckGroupRepository) FindByID(ctx context.Context, userID, id string) (deckgroup.Group, error) {
	filter, err := ownedFilter(userID, id)
	if err != nil {
		return deckgroup.Group{}, repositories.ErrNotFound
	}
	var doc deckGroupDocument
	err = r.collection.FindOne(ctx, filter).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return deckgroup.Group{}, repositories.ErrNotFound
	}
	if err != nil {
		return deckgroup.Group{}, err
	}
	return doc.toDomain(), nil
}

func (r *DeckGroupRepository) ListByUser(ctx context.Context, userID string) ([]deckgroup.Group, error) {
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

	groups := []deckgroup.Group{}
	for cursor.Next(ctx) {
		var doc deckGroupDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		groups = append(groups, doc.toDomain())
	}
	return groups, cursor.Err()
}

func (r *DeckGroupRepository) Update(ctx context.Context, g deckgroup.Group) (deckgroup.Group, error) {
	filter, err := ownedFilter(g.UserID, g.ID)
	if err != nil {
		return deckgroup.Group{}, repositories.ErrNotFound
	}
	result := r.collection.FindOneAndUpdate(ctx, filter,
		bson.M{"$set": bson.M{
			"name":      g.Name,
			"nameKey":   deckgroup.NameKey(g.Name),
			"updatedAt": g.UpdatedAt,
		}},
		mongoReturnAfter(),
	)
	var doc deckGroupDocument
	if err := result.Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return deckgroup.Group{}, repositories.ErrNotFound
		}
		if mongo.IsDuplicateKeyError(err) {
			return deckgroup.Group{}, repositories.ErrDuplicate
		}
		return deckgroup.Group{}, err
	}
	return doc.toDomain(), nil
}

func (r *DeckGroupRepository) Delete(ctx context.Context, userID, id string) error {
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

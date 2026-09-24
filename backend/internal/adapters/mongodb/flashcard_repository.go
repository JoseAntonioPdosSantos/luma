package mongodb

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"flashcard-backend/internal/domain/flashcard"
	"flashcard-backend/internal/domain/study"
	"flashcard-backend/internal/ports/repositories"
)

type extendedExampleDocument struct {
	Text        string `bson:"text"`
	Translation string `bson:"translation"`
}

type audioDocument struct {
	GridFSFileID primitive.ObjectID `bson:"gridFsFileId"`
	ContentType  string             `bson:"contentType"`
	DurationMs   int64              `bson:"durationMs"`
}

type schedulingDocument struct {
	State          study.CardState `bson:"state"`
	DueAt          time.Time       `bson:"dueAt"`
	LastReviewedAt *time.Time      `bson:"lastReviewedAt"`
	Repetitions    int             `bson:"repetitions"`
	Lapses         int             `bson:"lapses"`
	IntervalDays   int             `bson:"intervalDays"`
	EaseFactor     float64         `bson:"easeFactor"`
}

func (s schedulingDocument) toDomain() study.SchedulingState {
	return study.SchedulingState{
		State:          s.State,
		DueAt:          s.DueAt,
		LastReviewedAt: s.LastReviewedAt,
		Repetitions:    s.Repetitions,
		Lapses:         s.Lapses,
		IntervalDays:   s.IntervalDays,
		EaseFactor:     s.EaseFactor,
	}
}

func schedulingFromDomain(s study.SchedulingState) schedulingDocument {
	return schedulingDocument{
		State:          s.State,
		DueAt:          s.DueAt,
		LastReviewedAt: s.LastReviewedAt,
		Repetitions:    s.Repetitions,
		Lapses:         s.Lapses,
		IntervalDays:   s.IntervalDays,
		EaseFactor:     s.EaseFactor,
	}
}

type flashcardDocument struct {
	ID              primitive.ObjectID       `bson:"_id,omitempty"`
	UserID          primitive.ObjectID       `bson:"userId"`
	DeckID          primitive.ObjectID       `bson:"deckId"`
	Question        string                   `bson:"question"`
	Answer          string                   `bson:"answer"`
	Hint            string                   `bson:"hint"`
	ExtendedExample *extendedExampleDocument `bson:"extendedExample"`
	Audio           *audioDocument           `bson:"audio"`
	Scheduling      schedulingDocument       `bson:"scheduling"`
	ArchivedAt      *time.Time               `bson:"archivedAt"`
	CreatedAt       time.Time                `bson:"createdAt"`
	UpdatedAt       time.Time                `bson:"updatedAt"`
}

func (d flashcardDocument) toDomain() flashcard.Flashcard {
	var example *flashcard.ExtendedExample
	if d.ExtendedExample != nil {
		example = &flashcard.ExtendedExample{
			Text:        d.ExtendedExample.Text,
			Translation: d.ExtendedExample.Translation,
		}
	}

	var audio *flashcard.Audio
	if d.Audio != nil {
		audio = &flashcard.Audio{
			GridFSFileID: d.Audio.GridFSFileID.Hex(),
			ContentType:  d.Audio.ContentType,
			DurationMs:   d.Audio.DurationMs,
		}
	}

	return flashcard.Flashcard{
		ID:              d.ID.Hex(),
		UserID:          d.UserID.Hex(),
		DeckID:          d.DeckID.Hex(),
		Question:        d.Question,
		Answer:          d.Answer,
		Hint:            d.Hint,
		ExtendedExample: example,
		Audio:           audio,
		Scheduling:      d.Scheduling.toDomain(),
		ArchivedAt:      d.ArchivedAt,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}
}

// FlashcardRepository implements repositories.FlashcardRepository against
// MongoDB.
type FlashcardRepository struct {
	collection *mongo.Collection
}

func NewFlashcardRepository(db *mongo.Database) *FlashcardRepository {
	return &FlashcardRepository{collection: db.Collection(CollectionFlashcards)}
}

func (r *FlashcardRepository) Create(ctx context.Context, f flashcard.Flashcard) (flashcard.Flashcard, error) {
	userObjectID, err := primitive.ObjectIDFromHex(f.UserID)
	if err != nil {
		return flashcard.Flashcard{}, err
	}
	deckObjectID, err := primitive.ObjectIDFromHex(f.DeckID)
	if err != nil {
		return flashcard.Flashcard{}, err
	}

	doc := flashcardDocument{
		ID:         primitive.NewObjectID(),
		UserID:     userObjectID,
		DeckID:     deckObjectID,
		Question:   f.Question,
		Answer:     f.Answer,
		Hint:       f.Hint,
		Scheduling: schedulingFromDomain(f.Scheduling),
		CreatedAt:  f.CreatedAt,
		UpdatedAt:  f.UpdatedAt,
	}
	if f.ExtendedExample != nil {
		doc.ExtendedExample = &extendedExampleDocument{
			Text:        f.ExtendedExample.Text,
			Translation: f.ExtendedExample.Translation,
		}
	}

	if _, err := r.collection.InsertOne(ctx, doc); err != nil {
		return flashcard.Flashcard{}, err
	}
	return doc.toDomain(), nil
}

func (r *FlashcardRepository) FindByID(ctx context.Context, userID, id string) (flashcard.Flashcard, error) {
	filter, err := ownedFilter(userID, id)
	if err != nil {
		return flashcard.Flashcard{}, repositories.ErrNotFound
	}

	var doc flashcardDocument
	err = r.collection.FindOne(ctx, filter).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return flashcard.Flashcard{}, repositories.ErrNotFound
	}
	if err != nil {
		return flashcard.Flashcard{}, err
	}
	return doc.toDomain(), nil
}

func (r *FlashcardRepository) ListByDeck(ctx context.Context, userID, deckID string) ([]flashcard.Flashcard, error) {
	filter, err := deckScopeFilter(userID, deckID)
	if err != nil {
		return nil, err
	}
	filter["archivedAt"] = nil

	return r.find(ctx, filter, nil)
}

func (r *FlashcardRepository) SearchByDeck(ctx context.Context, userID, deckID string, opts repositories.FlashcardListOptions) ([]flashcard.Flashcard, int, error) {
	filter, err := deckScopeFilter(userID, deckID)
	if err != nil {
		return nil, 0, err
	}
	sortKey := "createdAt"
	if opts.Archived {
		filter["archivedAt"] = bson.M{"$ne": nil}
		sortKey = "archivedAt"
	} else {
		filter["archivedAt"] = nil
	}
	if opts.Query != "" {
		pattern := primitive.Regex{Pattern: searchPattern(opts.Query), Options: "i"}
		filter["$or"] = bson.A{
			bson.M{"question": pattern},
			bson.M{"answer": pattern},
			bson.M{"hint": pattern},
		}
	}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// _id breaks ties so pages never overlap or skip cards created in the
	// same instant.
	findOpts := options.Find().
		SetSort(bson.D{{Key: sortKey, Value: -1}, {Key: "_id", Value: -1}}).
		SetSkip(int64(opts.Offset)).
		SetLimit(int64(opts.Limit))
	cards, err := r.findWith(ctx, filter, findOpts)
	if err != nil {
		return nil, 0, err
	}
	return cards, int(total), nil
}

func (r *FlashcardRepository) ListArchivedByDeck(ctx context.Context, userID, deckID string) ([]flashcard.Flashcard, error) {
	filter, err := deckScopeFilter(userID, deckID)
	if err != nil {
		return nil, err
	}
	filter["archivedAt"] = bson.M{"$ne": nil}

	return r.find(ctx, filter, &findOptions{sortByArchivedAtDesc: true})
}

func (r *FlashcardRepository) ListArchivedByUser(ctx context.Context, userID string) ([]flashcard.Flashcard, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}
	filter := bson.M{"userId": userObjectID, "archivedAt": bson.M{"$ne": nil}}

	return r.find(ctx, filter, &findOptions{sortByArchivedAtDesc: true})
}

func (r *FlashcardRepository) NextDueAt(ctx context.Context, userID, deckID string, now time.Time) (*time.Time, error) {
	filter, err := deckScopeFilter(userID, deckID)
	if err != nil {
		return nil, err
	}
	filter["archivedAt"] = nil
	filter["scheduling.dueAt"] = bson.M{"$gt": now}

	var doc flashcardDocument
	err = r.collection.FindOne(ctx, filter,
		options.FindOne().SetSort(bson.D{{Key: "scheduling.dueAt", Value: 1}}),
	).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	next := doc.Scheduling.DueAt
	return &next, nil
}

func (r *FlashcardRepository) CountReviewedSince(ctx context.Context, userID string, since time.Time, scope repositories.ReviewScope) (int, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return 0, err
	}
	filter := bson.M{
		"userId":                    userObjectID,
		"scheduling.lastReviewedAt": bson.M{"$gte": since},
	}
	if scope.OnlyDeckID != "" {
		deckObjectID, err := primitive.ObjectIDFromHex(scope.OnlyDeckID)
		if err != nil {
			return 0, err
		}
		filter["deckId"] = deckObjectID
	} else if len(scope.ExcludeDeckIDs) > 0 {
		excluded := make(bson.A, 0, len(scope.ExcludeDeckIDs))
		for _, id := range scope.ExcludeDeckIDs {
			objectID, err := primitive.ObjectIDFromHex(id)
			if err != nil {
				return 0, err
			}
			excluded = append(excluded, objectID)
		}
		filter["deckId"] = bson.M{"$nin": excluded}
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	return int(count), err
}

func (r *FlashcardRepository) ListDueByStudiedToday(ctx context.Context, userID, deckID string, now, since time.Time, reviewedSince bool, limit int) ([]flashcard.Flashcard, error) {
	filter, err := deckScopeFilter(userID, deckID)
	if err != nil {
		return nil, err
	}
	filter["archivedAt"] = nil
	filter["scheduling.dueAt"] = bson.M{"$lte": now}
	if reviewedSince {
		filter["scheduling.lastReviewedAt"] = bson.M{"$gte": since}
	} else {
		// "$not $gte" also matches cards that were never reviewed (null).
		filter["scheduling.lastReviewedAt"] = bson.M{"$not": bson.M{"$gte": since}}
	}

	return r.find(ctx, filter, &findOptions{sortByDueAtAsc: true, limit: limit})
}

func (r *FlashcardRepository) ListDue(ctx context.Context, userID, deckID string, now time.Time, limit int) ([]flashcard.Flashcard, error) {
	filter, err := deckScopeFilter(userID, deckID)
	if err != nil {
		return nil, err
	}
	filter["archivedAt"] = nil
	filter["scheduling.dueAt"] = bson.M{"$lte": now}

	opts := &findOptions{sortByDueAtAsc: true, limit: limit}
	return r.find(ctx, filter, opts)
}

type findOptions struct {
	sortByDueAtAsc       bool
	sortByArchivedAtDesc bool
	limit                int
}

func (r *FlashcardRepository) find(ctx context.Context, filter bson.M, opts *findOptions) ([]flashcard.Flashcard, error) {
	findOpts := options.Find()
	if opts != nil {
		if opts.sortByDueAtAsc {
			findOpts.SetSort(bson.D{{Key: "scheduling.dueAt", Value: 1}})
		}
		if opts.sortByArchivedAtDesc {
			findOpts.SetSort(bson.D{{Key: "archivedAt", Value: -1}})
		}
		if opts.limit > 0 {
			findOpts.SetLimit(int64(opts.limit))
		}
	}

	return r.findWith(ctx, filter, findOpts)
}

func (r *FlashcardRepository) findWith(ctx context.Context, filter bson.M, findOpts *options.FindOptions) ([]flashcard.Flashcard, error) {
	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	cards := []flashcard.Flashcard{}
	for cursor.Next(ctx) {
		var doc flashcardDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		cards = append(cards, doc.toDomain())
	}
	return cards, cursor.Err()
}

func (r *FlashcardRepository) Update(ctx context.Context, f flashcard.Flashcard) (flashcard.Flashcard, error) {
	filter, err := ownedFilter(f.UserID, f.ID)
	if err != nil {
		return flashcard.Flashcard{}, repositories.ErrNotFound
	}

	set := bson.M{
		"question":   f.Question,
		"answer":     f.Answer,
		"hint":       f.Hint,
		"scheduling": schedulingFromDomain(f.Scheduling),
		"updatedAt":  f.UpdatedAt,
	}
	if f.ExtendedExample != nil {
		set["extendedExample"] = extendedExampleDocument{
			Text:        f.ExtendedExample.Text,
			Translation: f.ExtendedExample.Translation,
		}
	} else {
		set["extendedExample"] = nil
	}
	if f.Audio != nil {
		gridFSObjectID, err := primitive.ObjectIDFromHex(f.Audio.GridFSFileID)
		if err != nil {
			return flashcard.Flashcard{}, err
		}
		set["audio"] = audioDocument{
			GridFSFileID: gridFSObjectID,
			ContentType:  f.Audio.ContentType,
			DurationMs:   f.Audio.DurationMs,
		}
	} else {
		set["audio"] = nil
	}

	result := r.collection.FindOneAndUpdate(ctx, filter, bson.M{"$set": set}, mongoReturnAfter())

	var doc flashcardDocument
	if err := result.Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return flashcard.Flashcard{}, repositories.ErrNotFound
		}
		return flashcard.Flashcard{}, err
	}
	return doc.toDomain(), nil
}

func (r *FlashcardRepository) Archive(ctx context.Context, userID, id string, now time.Time) error {
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

func (r *FlashcardRepository) Restore(ctx context.Context, userID, id string, now time.Time) error {
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

func (r *FlashcardRepository) DeleteArchived(ctx context.Context, userID, id string) (flashcard.Flashcard, error) {
	filter, err := ownedFilter(userID, id)
	if err != nil {
		return flashcard.Flashcard{}, repositories.ErrNotFound
	}
	filter["archivedAt"] = bson.M{"$ne": nil}

	var doc flashcardDocument
	err = r.collection.FindOneAndDelete(ctx, filter).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return flashcard.Flashcard{}, repositories.ErrNotFound
	}
	if err != nil {
		return flashcard.Flashcard{}, err
	}
	return doc.toDomain(), nil
}

func (r *FlashcardRepository) DeleteAllByDeck(ctx context.Context, userID, deckID string) error {
	filter, err := deckScopeFilter(userID, deckID)
	if err != nil {
		return err
	}
	_, err = r.collection.DeleteMany(ctx, filter)
	return err
}

func (r *FlashcardRepository) CountByDeck(ctx context.Context, userID, deckID string, now time.Time) (total int, due int, err error) {
	filter, err := deckScopeFilter(userID, deckID)
	if err != nil {
		return 0, 0, err
	}
	filter["archivedAt"] = nil

	totalCount, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, 0, err
	}

	dueFilter := bson.M{}
	for k, v := range filter {
		dueFilter[k] = v
	}
	dueFilter["scheduling.dueAt"] = bson.M{"$lte": now}

	dueCount, err := r.collection.CountDocuments(ctx, dueFilter)
	if err != nil {
		return 0, 0, err
	}

	return int(totalCount), int(dueCount), nil
}

func (r *FlashcardRepository) CountsByState(ctx context.Context, userID, deckID string) (map[study.CardState]int, error) {
	filter, err := deckScopeFilter(userID, deckID)
	if err != nil {
		return nil, err
	}
	filter["archivedAt"] = nil

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: filter}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$scheduling.state"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	counts := make(map[study.CardState]int)
	for cursor.Next(ctx) {
		var row struct {
			State study.CardState `bson:"_id"`
			Count int             `bson:"count"`
		}
		if err := cursor.Decode(&row); err != nil {
			return nil, err
		}
		counts[row.State] = row.Count
	}
	return counts, cursor.Err()
}

func deckScopeFilter(userID, deckID string) (bson.M, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}
	deckObjectID, err := primitive.ObjectIDFromHex(deckID)
	if err != nil {
		return nil, err
	}
	return bson.M{"userId": userObjectID, "deckId": deckObjectID}, nil
}

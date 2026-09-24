// Package gridfs implements the audiostore.Store port using MongoDB
// GridFS, so flashcard audio binaries are streamed in and out rather than
// loaded fully into memory (spec section 7).
package gridfs

import (
	"context"
	"errors"
	"fmt"
	"io"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"

	"flashcard-backend/internal/ports/audiostore"
)

// audioFilename is a fixed, non-user-controlled name for every stored
// file: the client-supplied filename is never trusted or even accepted
// (spec section 7), and GridFS's own generated file ID is the real
// reference used everywhere else.
const audioFilename = "flashcard-audio"

// Store implements audiostore.Store against a MongoDB GridFS bucket.
type Store struct {
	bucket *gridfs.Bucket
}

// New builds a Store backed by db's default GridFS bucket ("fs").
func New(db *mongo.Database) (*Store, error) {
	bucket, err := gridfs.NewBucket(db)
	if err != nil {
		return nil, fmt.Errorf("creating gridfs bucket: %w", err)
	}
	return &Store{bucket: bucket}, nil
}

func (s *Store) Save(_ context.Context, r io.Reader, maxSizeBytes int64) (string, int64, error) {
	stream, err := s.bucket.OpenUploadStream(audioFilename)
	if err != nil {
		return "", 0, fmt.Errorf("opening upload stream: %w", err)
	}

	// Capped at maxSizeBytes+1 so we can detect "too large" without
	// reading an unbounded amount of attacker-controlled data.
	limited := io.LimitReader(r, maxSizeBytes+1)
	n, copyErr := io.Copy(stream, limited)
	if copyErr != nil {
		_ = stream.Abort()
		return "", 0, fmt.Errorf("writing audio: %w", copyErr)
	}
	if n > maxSizeBytes {
		_ = stream.Abort()
		return "", 0, audiostore.ErrTooLarge
	}
	if err := stream.Close(); err != nil {
		return "", 0, fmt.Errorf("finalizing upload: %w", err)
	}

	fileID, ok := stream.FileID.(primitive.ObjectID)
	if !ok {
		return "", 0, fmt.Errorf("unexpected gridfs file id type %T", stream.FileID)
	}
	return fileID.Hex(), n, nil
}

func (s *Store) Open(_ context.Context, fileID string) (io.ReadCloser, error) {
	objectID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		return nil, audiostore.ErrNotFound
	}

	stream, err := s.bucket.OpenDownloadStream(objectID)
	if err != nil {
		if errors.Is(err, gridfs.ErrFileNotFound) {
			return nil, audiostore.ErrNotFound
		}
		return nil, err
	}
	return stream, nil
}

func (s *Store) Delete(ctx context.Context, fileID string) error {
	objectID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		return nil
	}

	if err := s.bucket.DeleteContext(ctx, objectID); err != nil {
		if errors.Is(err, gridfs.ErrFileNotFound) {
			return nil
		}
		return err
	}
	return nil
}

// Package audiostore defines the port for storing and retrieving audio
// binaries. The flashcard document itself never stores audio bytes; it
// only ever stores a Store-issued file ID (spec section 7).
package audiostore

import (
	"context"
	"errors"
	"io"
)

// ErrTooLarge is returned by Save when r contains more than maxSizeBytes.
// No file is persisted in that case.
var ErrTooLarge = errors.New("audio exceeds maximum allowed size")

// ErrNotFound is returned by Open when fileID does not refer to an
// existing file (including a malformed ID).
var ErrNotFound = errors.New("audio file not found")

// Store persists and retrieves audio binaries by an opaque file ID.
type Store interface {
	// Save streams r into storage, capped at maxSizeBytes so an
	// oversized upload is rejected without ever persisting a partial or
	// oversized file. It returns the new file ID and the number of bytes
	// written.
	Save(ctx context.Context, r io.Reader, maxSizeBytes int64) (fileID string, size int64, err error)
	// Open returns a stream of the file's bytes. The caller must Close it.
	Open(ctx context.Context, fileID string) (io.ReadCloser, error)
	// Delete removes a file. Deleting a file that does not exist, or an
	// ID that is not well-formed, is not an error, so callers can use it
	// for best-effort cleanup.
	Delete(ctx context.Context, fileID string) error
}

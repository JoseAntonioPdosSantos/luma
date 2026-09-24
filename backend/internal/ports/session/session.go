// Package session defines the port for issuing and verifying session
// tokens. Email-only access (spec section 8) is intentionally not
// password-based authentication; keeping it behind this interface lets it
// be replaced later (e.g. by magic links or OAuth) without touching the
// HTTP layer.
package session

import "time"

// Manager issues and verifies opaque session tokens for a user ID.
type Manager interface {
	// Issue returns a new token for userID, valid until the returned
	// expiry.
	Issue(userID string, now time.Time) (token string, expiresAt time.Time, err error)
	// Verify returns the user ID encoded in token, or an error if the
	// token is malformed, tampered with, or expired as of now.
	Verify(token string, now time.Time) (userID string, err error)
}

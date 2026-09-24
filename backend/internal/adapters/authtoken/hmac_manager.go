// Package authtoken implements the session.Manager port as a signed,
// short-lived, stateless token: no server-side session store is needed,
// which keeps the MVP simple (spec section 8 explicitly allows this for a
// local-only prototype, as long as authentication stays behind an
// interface).
//
// Token format: "<base64url(userID|expiryUnix)>.<base64url(HMAC-SHA256)>".
// It is not a JWT; there is no need for header negotiation or multiple
// algorithms here, so a minimal custom format keeps the implementation
// small and auditable.
package authtoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"flashcard-backend/internal/apperror"
)

const defaultTTL = 30 * 24 * time.Hour

// HMACManager implements session.Manager using HMAC-SHA256 over a secret
// key.
type HMACManager struct {
	secret []byte
	ttl    time.Duration
}

// NewHMACManager builds an HMACManager. ttl <= 0 defaults to 30 days,
// a reasonable session lifetime for a low-risk, single-user MVP (spec
// section 16 documents this as an explicit limitation, not strong
// identity verification).
func NewHMACManager(secret string, ttl time.Duration) HMACManager {
	if ttl <= 0 {
		ttl = defaultTTL
	}
	return HMACManager{secret: []byte(secret), ttl: ttl}
}

func (m HMACManager) Issue(userID string, now time.Time) (string, time.Time, error) {
	expiresAt := now.Add(m.ttl)
	payload := fmt.Sprintf("%s|%d", userID, expiresAt.Unix())
	encodedPayload := base64.RawURLEncoding.EncodeToString([]byte(payload))
	signature := m.sign(encodedPayload)
	return encodedPayload + "." + signature, expiresAt, nil
}

func (m HMACManager) Verify(token string, now time.Time) (string, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return "", apperror.Unauthorized("auth.invalidSession", "invalid session token")
	}
	encodedPayload, signature := parts[0], parts[1]

	expectedSignature := m.sign(encodedPayload)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return "", apperror.Unauthorized("auth.invalidSession", "invalid session token")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return "", apperror.Unauthorized("auth.invalidSession", "invalid session token")
	}

	userID, expiresAtRaw, ok := strings.Cut(string(payloadBytes), "|")
	if !ok {
		return "", apperror.Unauthorized("auth.invalidSession", "invalid session token")
	}

	expiresAtUnix, err := strconv.ParseInt(expiresAtRaw, 10, 64)
	if err != nil {
		return "", apperror.Unauthorized("auth.invalidSession", "invalid session token")
	}
	if now.After(time.Unix(expiresAtUnix, 0)) {
		return "", apperror.Unauthorized("auth.sessionExpired", "session token expired")
	}

	return userID, nil
}

func (m HMACManager) sign(encodedPayload string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(encodedPayload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

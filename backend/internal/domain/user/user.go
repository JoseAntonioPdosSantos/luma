// Package user defines the User entity: an identity created or looked up
// by email only. See the project spec's section 8 for why this is
// intentionally not password-based authentication and section 16 for its
// security limitations.
package user

import (
	"net/mail"
	"strings"
	"time"

	"flashcard-backend/internal/apperror"
)

const MaxEmailLength = 254

// Language is a UI language the user may choose. The zero value ("") means
// no preference was saved yet, and the client falls back to detecting the
// browser's language.
const (
	LanguagePortuguese = "pt"
	LanguageEnglish    = "en"
	LanguageSpanish    = "es"
	LanguageFrench     = "fr"
)

// ValidLanguages lists every language the app supports.
var ValidLanguages = map[string]bool{
	LanguagePortuguese: true,
	LanguageEnglish:    true,
	LanguageSpanish:    true,
	LanguageFrench:     true,
}

// User is a person identified solely by a normalized email address.
type User struct {
	ID           string
	Email        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastAccessAt time.Time
	// ActiveStudyProfileID is the study configuration the user selected; ""
	// means the built-in default.
	ActiveStudyProfileID string
	// LegacyDailyCardLimit is the daily goal saved before study
	// configurations existed. It is only read to migrate it into a
	// configuration and is then cleared.
	LegacyDailyCardLimit *int
	// ContinuePastGoalOn is the local day (YYYY-MM-DD) on which the user
	// chose to keep studying past their daily goal, or "" if they never did.
	// The choice lasts until that day ends, on every device.
	ContinuePastGoalOn string
	// PreferredLanguage is the UI language the user chose (one of the
	// Language* constants), or "" if they never chose one, in which case the
	// client falls back to the browser's language.
	PreferredLanguage string
}

// ValidateLanguage checks that a language code is one this app supports.
func ValidateLanguage(raw string) (string, error) {
	if !ValidLanguages[raw] {
		return "", apperror.Validation("user.language.invalid", "language must be one of: pt, en, es, fr")
	}
	return raw, nil
}

// NormalizeEmail trims whitespace and lowercases an email address so that
// lookups and the unique index are case- and whitespace-insensitive.
func NormalizeEmail(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

// ValidateEmail normalizes and validates an email address, returning the
// normalized form or a VALIDATION_ERROR describing why it was rejected.
func ValidateEmail(raw string) (string, error) {
	email := NormalizeEmail(raw)
	if email == "" {
		return "", apperror.Validation("user.email.required", "email is required")
	}
	if len(email) > MaxEmailLength {
		return "", apperror.Validation("user.email.tooLong", "email must be at most 254 characters")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", apperror.Validation("user.email.invalid", "email is not a valid email address")
	}
	return email, nil
}

// New creates a User for the given normalized email at the given instant.
// The caller (an application service) is responsible for persistence and
// for assigning an ID.
func New(email string, now time.Time) User {
	return User{
		Email:        email,
		CreatedAt:    now,
		UpdatedAt:    now,
		LastAccessAt: now,
	}
}

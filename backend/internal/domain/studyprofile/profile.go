// Package studyprofile defines a study configuration: a named set of choices
// about how the system should react to each rating (see study.Rules) plus an
// optional daily card goal. Every user always has the built-in default
// configuration and may create their own and pick which one is active.
package studyprofile

import (
	"strings"
	"time"
	"unicode/utf8"

	"flashcard-backend/internal/apperror"
	"flashcard-backend/internal/domain/study"
)

const (
	// DefaultID identifies the built-in configuration. It is not stored: it
	// exists for every user, cannot be edited or deleted, and is what a user
	// who never chose anything uses.
	DefaultID   = "default"
	DefaultName = "Padrão"

	MaxNameLength      = 60
	MaxDailyCardLimit  = 200
	MaxProfilesPerUser = 20
)

// Profile is one study configuration.
type Profile struct {
	ID string
	// UserID is empty for the built-in default.
	UserID string
	Name   string
	// DailyCardLimit is how many different cards to study per day, or nil for
	// no daily goal.
	DailyCardLimit *int
	Rules          study.Rules
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// IsDefault reports whether this is the built-in configuration.
func (p Profile) IsDefault() bool { return p.ID == DefaultID }

// Default returns the built-in configuration: the behavior the app always
// had, with no daily goal.
func Default() Profile {
	return Profile{ID: DefaultID, Name: DefaultName, Rules: study.DefaultRules()}
}

// New creates a user's own configuration. The caller assigns the ID.
func New(userID, name string, dailyCardLimit *int, rules study.Rules, now time.Time) Profile {
	return Profile{
		UserID:         userID,
		Name:           name,
		DailyCardLimit: dailyCardLimit,
		Rules:          rules,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// ValidateName trims and validates a configuration name.
func ValidateName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", apperror.Validation("studyProfile.name.required", "name is required")
	}
	if utf8.RuneCountInString(name) > MaxNameLength {
		return "", apperror.Validation("studyProfile.name.tooLong", "name must be at most 60 characters")
	}
	return name, nil
}

// NameKey is the form of a name used to keep names unique per user without
// regard to case or surrounding spaces ("Prova" and "prova " collide).
func NameKey(name string) string { return strings.ToLower(strings.TrimSpace(name)) }

// ValidateDailyCardLimit checks the optional daily goal (nil = no goal).
func ValidateDailyCardLimit(limit *int) error {
	if limit != nil && (*limit < 1 || *limit > MaxDailyCardLimit) {
		return apperror.Validation("studyProfile.dailyCardLimit.range", "dailyCardLimit must be between 1 and 200, or null for no daily goal")
	}
	return nil
}

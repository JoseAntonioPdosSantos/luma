// Package deck defines the Deck entity: a study subject owned by exactly
// one user (e.g. English, Algorithms, System Design).
package deck

import (
	"strings"
	"time"

	"flashcard-backend/internal/apperror"
)

const (
	MaxNameLength        = 100
	MaxDescriptionLength = 1000
)

// Deck represents a study subject belonging to one user.
type Deck struct {
	ID          string
	UserID      string
	Name        string
	Description string
	// StudyProfileID is the study configuration this deck uses instead of the
	// user's general (active) one: "" means it follows the general one,
	// studyprofile.DefaultID the built-in "Padrão", anything else one of the
	// user's own configurations.
	StudyProfileID string
	// ContinuePastGoalOn is the local day (YYYY-MM-DD) on which the user
	// chose to keep studying this deck past its own daily goal; only used
	// while the deck has a configuration of its own.
	ContinuePastGoalOn string
	// GroupID is the deckgroup.Group this deck is filed under; "" means it
	// is not in any group and appears on its own on the dashboard.
	GroupID    string
	ArchivedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// IsArchived reports whether the deck has been archived.
func (d Deck) IsArchived() bool { return d.ArchivedAt != nil }

// ValidateName trims and validates a deck name.
func ValidateName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", apperror.Validation("deck.name.required", "deck name is required")
	}
	if len(name) > MaxNameLength {
		return "", apperror.Validation("deck.name.tooLong", "deck name must be at most 100 characters")
	}
	return name, nil
}

// ValidateDescription trims and validates an optional deck description.
func ValidateDescription(raw string) (string, error) {
	description := strings.TrimSpace(raw)
	if len(description) > MaxDescriptionLength {
		return "", apperror.Validation("deck.description.tooLong", "deck description must be at most 1,000 characters")
	}
	return description, nil
}

// New creates a Deck for the given owner at the given instant. The caller
// is responsible for persistence and for assigning an ID.
func New(userID, name, description string, now time.Time) Deck {
	return Deck{
		UserID:      userID,
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

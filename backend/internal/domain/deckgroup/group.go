// Package deckgroup defines the Group entity: a named folder a user can put
// decks into, to organize related collections together (e.g. an "Inglês"
// group holding "Phrasal verbs", "Verbos", "Gramática" decks). Grouping is
// purely organizational and single-level: a group holds decks, never other
// groups, and a deck belongs to at most one group at a time.
package deckgroup

import (
	"strings"
	"time"
	"unicode/utf8"

	"flashcard-backend/internal/apperror"
)

const (
	MaxNameLength    = 60
	MaxGroupsPerUser = 50
)

// Group is a named folder of decks belonging to one user.
type Group struct {
	ID        string
	UserID    string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// New creates a Group for the given owner at the given instant. The caller
// (an application service) is responsible for persistence and for
// assigning an ID.
func New(userID, name string, now time.Time) Group {
	return Group{UserID: userID, Name: name, CreatedAt: now, UpdatedAt: now}
}

// ValidateName trims and validates a group name.
func ValidateName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", apperror.Validation("deckGroup.name.required", "group name is required")
	}
	if utf8.RuneCountInString(name) > MaxNameLength {
		return "", apperror.Validation("deckGroup.name.tooLong", "group name must be at most 60 characters")
	}
	return name, nil
}

// NameKey is the form of a name used to keep names unique per user without
// regard to case or surrounding spaces ("Inglês" and "inglês " collide).
func NameKey(name string) string { return strings.ToLower(strings.TrimSpace(name)) }

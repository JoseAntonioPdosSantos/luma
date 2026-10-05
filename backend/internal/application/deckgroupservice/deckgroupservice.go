// Package deckgroupservice implements deck-group business rules: creating,
// renaming and deleting the folders a user organizes decks into (alone or
// together with their decks), and filing a deck under one (or removing it
// from its group).
package deckgroupservice

import (
	"context"
	"errors"
	"fmt"

	"flashcard-backend/internal/apperror"
	"flashcard-backend/internal/domain/deckgroup"
	"flashcard-backend/internal/ports/clock"
	"flashcard-backend/internal/ports/repositories"
)

// DeckDeleter permanently deletes one deck with its flashcards, audio and
// review history (deckservice.Service does).
type DeckDeleter interface {
	Delete(ctx context.Context, userID, deckID string) error
}

type Service struct {
	groups  repositories.DeckGroupRepository
	decks   repositories.DeckRepository
	deleter DeckDeleter
	clock   clock.Clock
}

func New(groups repositories.DeckGroupRepository, decks repositories.DeckRepository, deleter DeckDeleter, c clock.Clock) Service {
	return Service{groups: groups, decks: decks, deleter: deleter, clock: c}
}

// List returns the user's groups, oldest first.
func (s Service) List(ctx context.Context, userID string) ([]deckgroup.Group, error) {
	groups, err := s.groups.ListByUser(ctx, userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return groups, nil
}

// Create validates and saves a new group for the user.
func (s Service) Create(ctx context.Context, userID, rawName string) (deckgroup.Group, error) {
	name, err := deckgroup.ValidateName(rawName)
	if err != nil {
		return deckgroup.Group{}, err
	}

	existing, err := s.groups.ListByUser(ctx, userID)
	if err != nil {
		return deckgroup.Group{}, apperror.Internal(err)
	}
	if len(existing) >= deckgroup.MaxGroupsPerUser {
		return deckgroup.Group{}, apperror.Validation("deckGroup.limitReached", fmt.Sprintf("you can have at most %d groups", deckgroup.MaxGroupsPerUser))
	}
	if nameTaken(existing, name, "") {
		return deckgroup.Group{}, errNameTaken
	}

	// The unique index still guards against two requests racing with the
	// same name (mapWriteError turns that into the same conflict).
	created, err := s.groups.Create(ctx, deckgroup.New(userID, name, s.clock.Now()))
	return s.mapWriteError(created, err)
}

// Rename changes one of the user's groups.
func (s Service) Rename(ctx context.Context, userID, id, rawName string) (deckgroup.Group, error) {
	name, err := deckgroup.ValidateName(rawName)
	if err != nil {
		return deckgroup.Group{}, err
	}

	existing, err := s.groups.FindByID(ctx, userID, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return deckgroup.Group{}, apperror.NotFound("deckGroup.notFound", "group not found")
	}
	if err != nil {
		return deckgroup.Group{}, apperror.Internal(err)
	}

	others, err := s.groups.ListByUser(ctx, userID)
	if err != nil {
		return deckgroup.Group{}, apperror.Internal(err)
	}
	if nameTaken(others, name, id) {
		return deckgroup.Group{}, errNameTaken
	}

	existing.Name = name
	existing.UpdatedAt = s.clock.Now()

	updated, err := s.groups.Update(ctx, existing)
	return s.mapWriteError(updated, err)
}

// Delete removes a group. The decks that were in it are not deleted or
// archived: they simply lose their group and appear on their own on the
// dashboard again.
func (s Service) Delete(ctx context.Context, userID, id string) error {
	err := s.groups.Delete(ctx, userID, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return apperror.NotFound("deckGroup.notFound", "group not found")
	}
	if err != nil {
		return apperror.Internal(err)
	}
	if err := s.decks.ClearGroup(ctx, userID, id); err != nil {
		return apperror.Internal(err)
	}
	return nil
}

// DeleteWithDecks removes a group and permanently deletes every active
// deck filed under it, with their flashcards, audio and review history.
// It cannot be undone. Archived decks of the group are left in the
// archive, without a group.
//
// The decks go first and the group last, so if a deck fails to delete the
// group is still there with what is left and the deletion can be retried.
func (s Service) DeleteWithDecks(ctx context.Context, userID, id string) error {
	if _, err := s.groups.FindByID(ctx, userID, id); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return apperror.NotFound("deckGroup.notFound", "group not found")
		}
		return apperror.Internal(err)
	}

	decks, err := s.decks.ListActive(ctx, userID)
	if err != nil {
		return apperror.Internal(err)
	}
	for _, d := range decks {
		if d.GroupID != id {
			continue
		}
		if err := s.deleter.Delete(ctx, userID, d.ID); err != nil {
			return err
		}
	}
	return s.Delete(ctx, userID, id)
}

// SetDeckGroup files a deck under groupID, or removes it from its group
// when groupID is "".
func (s Service) SetDeckGroup(ctx context.Context, userID, deckID, groupID string) error {
	if _, err := s.decks.FindByID(ctx, userID, deckID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return apperror.NotFound("deck.notFound", "deck not found")
		}
		return apperror.Internal(err)
	}
	if groupID != "" {
		if _, err := s.groups.FindByID(ctx, userID, groupID); err != nil {
			if errors.Is(err, repositories.ErrNotFound) {
				return apperror.NotFound("deckGroup.notFound", "group not found")
			}
			return apperror.Internal(err)
		}
	}

	err := s.decks.SetGroup(ctx, userID, deckID, groupID, s.clock.Now())
	if errors.Is(err, repositories.ErrNotFound) {
		return apperror.NotFound("deck.notFound", "deck not found")
	}
	if err != nil {
		return apperror.Internal(err)
	}
	return nil
}

var errNameTaken = apperror.Conflict("deckGroup.duplicateName", "you already have a group with this name")

// nameTaken reports whether another group (not exceptID) already has this
// name, ignoring case and surrounding spaces.
func nameTaken(groups []deckgroup.Group, name, exceptID string) bool {
	key := deckgroup.NameKey(name)
	for _, g := range groups {
		if g.ID != exceptID && deckgroup.NameKey(g.Name) == key {
			return true
		}
	}
	return false
}

func (s Service) mapWriteError(g deckgroup.Group, err error) (deckgroup.Group, error) {
	switch {
	case err == nil:
		return g, nil
	case errors.Is(err, repositories.ErrDuplicate):
		return deckgroup.Group{}, errNameTaken
	case errors.Is(err, repositories.ErrNotFound):
		return deckgroup.Group{}, apperror.NotFound("deckGroup.notFound", "group not found")
	default:
		return deckgroup.Group{}, apperror.Internal(err)
	}
}

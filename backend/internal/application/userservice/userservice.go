// Package userservice implements the email-only access flow: validating
// an email, finding or creating the corresponding user, and touching
// their last-access timestamp. See spec section 8.
package userservice

import (
	"context"
	"errors"

	"flashcard-backend/internal/apperror"
	"flashcard-backend/internal/domain/user"
	"flashcard-backend/internal/ports/clock"
	"flashcard-backend/internal/ports/repositories"
)

// Service implements the access-flow business rules, kept independent of
// HTTP and MongoDB.
type Service struct {
	users repositories.UserRepository
	clock clock.Clock
}

func New(users repositories.UserRepository, c clock.Clock) Service {
	return Service{users: users, clock: c}
}

// GetOrCreateByEmail validates and normalizes rawEmail, then returns the
// existing user with that email or creates a new one. It also updates
// LastAccessAt.
func (s Service) GetOrCreateByEmail(ctx context.Context, rawEmail string) (user.User, error) {
	email, err := user.ValidateEmail(rawEmail)
	if err != nil {
		return user.User{}, err
	}

	now := s.clock.Now()

	existing, err := s.users.FindByEmail(ctx, email)
	switch {
	case err == nil:
		if touchErr := s.users.TouchLastAccess(ctx, existing.ID, now); touchErr != nil {
			return user.User{}, apperror.Internal(touchErr)
		}
		existing.LastAccessAt = now
		return existing, nil
	case errors.Is(err, repositories.ErrNotFound):
		created, createErr := s.users.Create(ctx, user.New(email, now))
		if createErr != nil {
			return user.User{}, apperror.Internal(createErr)
		}
		return created, nil
	default:
		return user.User{}, apperror.Internal(err)
	}
}

// GetByID returns the user with the given ID, translating a not-found
// result into apperror.NotFound.
func (s Service) GetByID(ctx context.Context, id string) (user.User, error) {
	u, err := s.users.FindByID(ctx, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return user.User{}, apperror.NotFound("user.notFound", "user not found")
	}
	if err != nil {
		return user.User{}, apperror.Internal(err)
	}
	return u, nil
}

// SetPreferredLanguage validates rawLanguage and saves it as the user's UI
// language, so it follows them to any device or browser.
func (s Service) SetPreferredLanguage(ctx context.Context, id, rawLanguage string) error {
	language, err := user.ValidateLanguage(rawLanguage)
	if err != nil {
		return err
	}
	err = s.users.SetPreferredLanguage(ctx, id, language, s.clock.Now())
	if errors.Is(err, repositories.ErrNotFound) {
		return apperror.NotFound("user.notFound", "user not found")
	}
	if err != nil {
		return apperror.Internal(err)
	}
	return nil
}

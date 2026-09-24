// Package profileservice implements study configurations: the built-in
// default every user has, the ones a user creates (name, per-rating review
// behavior and an optional daily card goal) and which one is active.
package profileservice

import (
	"context"
	"errors"
	"fmt"

	"flashcard-backend/internal/apperror"
	"flashcard-backend/internal/domain/study"
	"flashcard-backend/internal/domain/studyprofile"
	"flashcard-backend/internal/domain/user"
	"flashcard-backend/internal/ports/clock"
	"flashcard-backend/internal/ports/repositories"
)

// migratedProfileName is the name given to the configuration created from a
// daily goal saved before configurations existed.
const migratedProfileName = "Minha meta diária"

type Service struct {
	profiles repositories.StudyProfileRepository
	users    repositories.UserRepository
	decks    repositories.DeckRepository
	clock    clock.Clock
}

func New(
	profiles repositories.StudyProfileRepository,
	users repositories.UserRepository,
	decks repositories.DeckRepository,
	c clock.Clock,
) Service {
	return Service{profiles: profiles, users: users, decks: decks, clock: c}
}

// Overview is everything the settings screen needs: the configurations
// (the built-in default first, then the user's own, oldest first) and which
// one is active.
type Overview struct {
	ActiveProfileID string
	Profiles        []studyprofile.Profile
}

// Input is the editable part of a configuration.
type Input struct {
	Name           string
	DailyCardLimit *int
	Rules          study.Rules
}

func (s Service) getUser(ctx context.Context, userID string) (user.User, error) {
	u, err := s.users.FindByID(ctx, userID)
	if errors.Is(err, repositories.ErrNotFound) {
		return user.User{}, apperror.NotFound("user.notFound", "user not found")
	}
	if err != nil {
		return user.User{}, apperror.Internal(err)
	}
	return u, nil
}

// List returns the user's configurations and the active one. A daily goal
// saved before configurations existed is first turned into a configuration
// of its own (once), so nobody loses it.
func (s Service) List(ctx context.Context, userID string) (Overview, error) {
	u, err := s.getUser(ctx, userID)
	if err != nil {
		return Overview{}, err
	}
	u, err = s.migrateLegacyGoal(ctx, u)
	if err != nil {
		return Overview{}, err
	}
	custom, err := s.profiles.ListByUser(ctx, userID)
	if err != nil {
		return Overview{}, apperror.Internal(err)
	}
	return Overview{
		ActiveProfileID: resolveActiveID(u.ActiveStudyProfileID, custom),
		Profiles:        append([]studyprofile.Profile{studyprofile.Default()}, custom...),
	}, nil
}

// Active returns the configuration the user is studying with: the one they
// selected, or the built-in default if they selected none (or the one they
// selected no longer exists).
func (s Service) Active(ctx context.Context, userID string) (studyprofile.Profile, error) {
	overview, err := s.List(ctx, userID)
	if err != nil {
		return studyprofile.Profile{}, err
	}
	for _, p := range overview.Profiles {
		if p.ID == overview.ActiveProfileID {
			return p, nil
		}
	}
	return studyprofile.Default(), nil
}

// ForDeck returns the configuration a deck is studied with and whether it is
// the deck's own: a deck can pick one of the user's configurations (or the
// built-in one); when it has not, or the one it picked no longer exists, it
// follows the user's general (active) configuration.
func (s Service) ForDeck(ctx context.Context, userID, deckID string) (studyprofile.Profile, bool, error) {
	d, err := s.decks.FindByID(ctx, userID, deckID)
	if errors.Is(err, repositories.ErrNotFound) {
		return studyprofile.Profile{}, false, apperror.NotFound("deck.notFound", "deck not found")
	}
	if err != nil {
		return studyprofile.Profile{}, false, apperror.Internal(err)
	}

	if d.StudyProfileID != "" {
		own, found, err := s.findOwn(ctx, userID, d.StudyProfileID)
		if err != nil {
			return studyprofile.Profile{}, false, err
		}
		if found {
			return own, true, nil
		}
	}

	general, err := s.Active(ctx, userID)
	return general, false, err
}

// findOwn looks up a configuration a deck refers to.
func (s Service) findOwn(ctx context.Context, userID, id string) (studyprofile.Profile, bool, error) {
	if id == studyprofile.DefaultID {
		return studyprofile.Default(), true, nil
	}
	p, err := s.profiles.FindByID(ctx, userID, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return studyprofile.Profile{}, false, nil
	}
	if err != nil {
		return studyprofile.Profile{}, false, apperror.Internal(err)
	}
	return p, true, nil
}

// OwnDeckIDs returns the ids of the user's decks that are studied with a
// configuration of their own. Each of them keeps its own daily count, and the
// general configuration's count leaves them out.
func (s Service) OwnDeckIDs(ctx context.Context, userID string) ([]string, error) {
	decks, err := s.decks.ListWithStudyProfile(ctx, userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if len(decks) == 0 {
		return nil, nil
	}
	custom, err := s.profiles.ListByUser(ctx, userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	exists := map[string]bool{studyprofile.DefaultID: true}
	for _, p := range custom {
		exists[p.ID] = true
	}

	var ids []string
	for _, d := range decks {
		if exists[d.StudyProfileID] {
			ids = append(ids, d.ID)
		}
	}
	return ids, nil
}

// SetDeckProfile chooses the configuration a deck is studied with: one of the
// user's own, studyprofile.DefaultID for the built-in one, or "" to follow the
// general configuration again.
func (s Service) SetDeckProfile(ctx context.Context, userID, deckID, profileID string) error {
	if _, err := s.decks.FindByID(ctx, userID, deckID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return apperror.NotFound("deck.notFound", "deck not found")
		}
		return apperror.Internal(err)
	}
	if profileID != "" && profileID != studyprofile.DefaultID {
		if _, err := s.profiles.FindByID(ctx, userID, profileID); err != nil {
			if errors.Is(err, repositories.ErrNotFound) {
				return apperror.NotFound("studyProfile.notFound", "configuration not found")
			}
			return apperror.Internal(err)
		}
	}

	err := s.decks.SetStudyProfile(ctx, userID, deckID, profileID, s.clock.Now())
	if errors.Is(err, repositories.ErrNotFound) {
		return apperror.NotFound("deck.notFound", "deck not found")
	}
	if err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func resolveActiveID(selected string, custom []studyprofile.Profile) string {
	if selected == "" || selected == studyprofile.DefaultID {
		return studyprofile.DefaultID
	}
	for _, p := range custom {
		if p.ID == selected {
			return selected
		}
	}
	return studyprofile.DefaultID
}

// Create validates and saves a new configuration for the user.
func (s Service) Create(ctx context.Context, userID string, in Input) (studyprofile.Profile, error) {
	name, err := validate(in)
	if err != nil {
		return studyprofile.Profile{}, err
	}
	if _, err := s.getUser(ctx, userID); err != nil {
		return studyprofile.Profile{}, err
	}

	existing, err := s.profiles.ListByUser(ctx, userID)
	if err != nil {
		return studyprofile.Profile{}, apperror.Internal(err)
	}
	if len(existing) >= studyprofile.MaxProfilesPerUser {
		return studyprofile.Profile{}, apperror.Validation("studyProfile.limitReached", fmt.Sprintf("you can have at most %d configurations", studyprofile.MaxProfilesPerUser))
	}

	if nameTaken(existing, name, "") {
		return studyprofile.Profile{}, errNameTaken
	}

	// The unique index still guards against two requests racing with the
	// same name (mapWriteError turns that into the same conflict).
	created, err := s.profiles.Create(ctx, studyprofile.New(userID, name, in.DailyCardLimit, in.Rules, s.clock.Now()))
	return s.mapWriteError(created, err)
}

// Update changes one of the user's own configurations. The built-in default
// cannot be changed.
func (s Service) Update(ctx context.Context, userID, id string, in Input) (studyprofile.Profile, error) {
	if id == studyprofile.DefaultID {
		return studyprofile.Profile{}, apperror.Forbidden("studyProfile.defaultImmutable.update", "the default configuration cannot be changed")
	}
	name, err := validate(in)
	if err != nil {
		return studyprofile.Profile{}, err
	}

	existing, err := s.profiles.FindByID(ctx, userID, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return studyprofile.Profile{}, apperror.NotFound("studyProfile.notFound", "configuration not found")
	}
	if err != nil {
		return studyprofile.Profile{}, apperror.Internal(err)
	}

	others, err := s.profiles.ListByUser(ctx, userID)
	if err != nil {
		return studyprofile.Profile{}, apperror.Internal(err)
	}
	if nameTaken(others, name, id) {
		return studyprofile.Profile{}, errNameTaken
	}

	existing.Name = name
	existing.DailyCardLimit = in.DailyCardLimit
	existing.Rules = in.Rules
	existing.UpdatedAt = s.clock.Now()

	updated, err := s.profiles.Update(ctx, existing)
	return s.mapWriteError(updated, err)
}

// Delete removes one of the user's own configurations. If it was the active
// one the user goes back to the built-in default. Cards keep their schedule:
// only how later answers are handled changes.
func (s Service) Delete(ctx context.Context, userID, id string) error {
	if id == studyprofile.DefaultID {
		return apperror.Forbidden("studyProfile.defaultImmutable.delete", "the default configuration cannot be deleted")
	}
	u, err := s.getUser(ctx, userID)
	if err != nil {
		return err
	}

	err = s.profiles.Delete(ctx, userID, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return apperror.NotFound("studyProfile.notFound", "configuration not found")
	}
	if err != nil {
		return apperror.Internal(err)
	}

	if u.ActiveStudyProfileID == id {
		if err := s.users.SetActiveStudyProfile(ctx, userID, "", s.clock.Now()); err != nil {
			return apperror.Internal(err)
		}
	}
	// Decks that used it follow the general configuration again.
	if err := s.decks.ClearStudyProfile(ctx, userID, id); err != nil {
		return apperror.Internal(err)
	}
	return nil
}

// SetActive selects the configuration the user studies with. The built-in
// default is selected with studyprofile.DefaultID.
func (s Service) SetActive(ctx context.Context, userID, id string) error {
	stored := ""
	if id != "" && id != studyprofile.DefaultID {
		if _, err := s.profiles.FindByID(ctx, userID, id); err != nil {
			if errors.Is(err, repositories.ErrNotFound) {
				return apperror.NotFound("studyProfile.notFound", "configuration not found")
			}
			return apperror.Internal(err)
		}
		stored = id
	}

	err := s.users.SetActiveStudyProfile(ctx, userID, stored, s.clock.Now())
	if errors.Is(err, repositories.ErrNotFound) {
		return apperror.NotFound("user.notFound", "user not found")
	}
	if err != nil {
		return apperror.Internal(err)
	}
	return nil
}

var errNameTaken = apperror.Conflict("studyProfile.duplicateName", "you already have a configuration with this name")

// nameTaken reports whether another configuration (not exceptID) already has
// this name, ignoring case and surrounding spaces.
func nameTaken(profiles []studyprofile.Profile, name, exceptID string) bool {
	key := studyprofile.NameKey(name)
	for _, p := range profiles {
		if p.ID != exceptID && studyprofile.NameKey(p.Name) == key {
			return true
		}
	}
	return false
}

func validate(in Input) (string, error) {
	name, err := studyprofile.ValidateName(in.Name)
	if err != nil {
		return "", err
	}
	if err := studyprofile.ValidateDailyCardLimit(in.DailyCardLimit); err != nil {
		return "", err
	}
	if err := in.Rules.Validate(); err != nil {
		return "", err
	}
	return name, nil
}

func (s Service) mapWriteError(p studyprofile.Profile, err error) (studyprofile.Profile, error) {
	switch {
	case err == nil:
		return p, nil
	case errors.Is(err, repositories.ErrDuplicate):
		return studyprofile.Profile{}, errNameTaken
	case errors.Is(err, repositories.ErrNotFound):
		return studyprofile.Profile{}, apperror.NotFound("studyProfile.notFound", "configuration not found")
	default:
		return studyprofile.Profile{}, apperror.Internal(err)
	}
}

// migrateLegacyGoal turns a daily goal saved before configurations existed
// into a configuration of its own (default review behavior plus that goal)
// and selects it, so the user's study does not change. It then clears the old
// value, which makes the migration a one-time step.
func (s Service) migrateLegacyGoal(ctx context.Context, u user.User) (user.User, error) {
	if u.LegacyDailyCardLimit == nil {
		return u, nil
	}
	now := s.clock.Now()

	if studyprofile.ValidateDailyCardLimit(u.LegacyDailyCardLimit) == nil {
		created, err := s.profiles.Create(ctx, studyprofile.New(u.ID, migratedProfileName, u.LegacyDailyCardLimit, study.DefaultRules(), now))
		switch {
		case err == nil:
			if u.ActiveStudyProfileID == "" || u.ActiveStudyProfileID == studyprofile.DefaultID {
				if err := s.users.SetActiveStudyProfile(ctx, u.ID, created.ID, now); err != nil {
					return u, apperror.Internal(err)
				}
				u.ActiveStudyProfileID = created.ID
			}
		case errors.Is(err, repositories.ErrDuplicate):
			// A previous attempt got as far as creating it; just finish.
		default:
			return u, apperror.Internal(err)
		}
	}

	if err := s.users.ClearLegacyDailyCardLimit(ctx, u.ID, now); err != nil {
		return u, apperror.Internal(err)
	}
	u.LegacyDailyCardLimit = nil
	return u, nil
}

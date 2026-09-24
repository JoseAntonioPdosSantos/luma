package profileservice

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"flashcard-backend/internal/apperror"
	"flashcard-backend/internal/domain/deck"
	"flashcard-backend/internal/domain/study"
	"flashcard-backend/internal/domain/studyprofile"
	"flashcard-backend/internal/domain/user"
	"flashcard-backend/internal/ports/clock"
	"flashcard-backend/internal/ports/repositories"
)

var fixedNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

// ---------- fakes ----------

type fakeProfileRepository struct {
	items  []studyprofile.Profile
	nextID int
}

func (f *fakeProfileRepository) nameTaken(userID, name, exceptID string) bool {
	for _, p := range f.items {
		if p.UserID == userID && p.ID != exceptID && studyprofile.NameKey(p.Name) == studyprofile.NameKey(name) {
			return true
		}
	}
	return false
}

func (f *fakeProfileRepository) Create(_ context.Context, p studyprofile.Profile) (studyprofile.Profile, error) {
	if f.nameTaken(p.UserID, p.Name, "") {
		return studyprofile.Profile{}, repositories.ErrDuplicate
	}
	f.nextID++
	p.ID = fmt.Sprintf("profile-%d", f.nextID)
	f.items = append(f.items, p)
	return p, nil
}

func (f *fakeProfileRepository) FindByID(_ context.Context, userID, id string) (studyprofile.Profile, error) {
	for _, p := range f.items {
		if p.ID == id && p.UserID == userID {
			return p, nil
		}
	}
	return studyprofile.Profile{}, repositories.ErrNotFound
}

func (f *fakeProfileRepository) ListByUser(_ context.Context, userID string) ([]studyprofile.Profile, error) {
	out := []studyprofile.Profile{}
	for _, p := range f.items {
		if p.UserID == userID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakeProfileRepository) Update(_ context.Context, p studyprofile.Profile) (studyprofile.Profile, error) {
	for i, existing := range f.items {
		if existing.ID == p.ID && existing.UserID == p.UserID {
			if f.nameTaken(p.UserID, p.Name, p.ID) {
				return studyprofile.Profile{}, repositories.ErrDuplicate
			}
			f.items[i] = p
			return p, nil
		}
	}
	return studyprofile.Profile{}, repositories.ErrNotFound
}

func (f *fakeProfileRepository) Delete(_ context.Context, userID, id string) error {
	for i, p := range f.items {
		if p.ID == id && p.UserID == userID {
			f.items = append(f.items[:i], f.items[i+1:]...)
			return nil
		}
	}
	return repositories.ErrNotFound
}

type fakeUserRepository struct{ users map[string]user.User }

func newFakeUserRepository(ids ...string) *fakeUserRepository {
	f := &fakeUserRepository{users: map[string]user.User{}}
	for _, id := range ids {
		f.users[id] = user.User{ID: id, Email: id + "@example.com"}
	}
	return f
}

func (f *fakeUserRepository) FindByID(_ context.Context, id string) (user.User, error) {
	u, ok := f.users[id]
	if !ok {
		return user.User{}, repositories.ErrNotFound
	}
	return u, nil
}
func (f *fakeUserRepository) SetActiveStudyProfile(_ context.Context, id, profileID string, _ time.Time) error {
	u, ok := f.users[id]
	if !ok {
		return repositories.ErrNotFound
	}
	u.ActiveStudyProfileID = profileID
	f.users[id] = u
	return nil
}
func (f *fakeUserRepository) ClearLegacyDailyCardLimit(_ context.Context, id string, _ time.Time) error {
	u := f.users[id]
	u.LegacyDailyCardLimit = nil
	f.users[id] = u
	return nil
}
func (f *fakeUserRepository) FindByEmail(context.Context, string) (user.User, error) {
	panic("not implemented")
}
func (f *fakeUserRepository) Create(context.Context, user.User) (user.User, error) {
	panic("not implemented")
}
func (f *fakeUserRepository) TouchLastAccess(context.Context, string, time.Time) error {
	panic("not implemented")
}
func (f *fakeUserRepository) SetContinuePastGoalOn(context.Context, string, string, time.Time) error {
	panic("not implemented")
}
func (f *fakeUserRepository) SetPreferredLanguage(context.Context, string, string, time.Time) error {
	panic("not implemented")
}

// fakeDeckRepository implements only what the configuration service uses.
type fakeDeckRepository struct{ decks map[string]deck.Deck }

func (f *fakeDeckRepository) addDeck(userID, id string) {
	f.decks[id] = deck.Deck{ID: id, UserID: userID}
}
func (f *fakeDeckRepository) FindByID(_ context.Context, userID, id string) (deck.Deck, error) {
	d, ok := f.decks[id]
	if !ok || d.UserID != userID {
		return deck.Deck{}, repositories.ErrNotFound
	}
	return d, nil
}
func (f *fakeDeckRepository) SetStudyProfile(_ context.Context, userID, id, profileID string, _ time.Time) error {
	d, ok := f.decks[id]
	if !ok || d.UserID != userID {
		return repositories.ErrNotFound
	}
	d.StudyProfileID = profileID
	f.decks[id] = d
	return nil
}
func (f *fakeDeckRepository) ClearStudyProfile(_ context.Context, userID, profileID string) error {
	for id, d := range f.decks {
		if d.UserID == userID && d.StudyProfileID == profileID {
			d.StudyProfileID = ""
			f.decks[id] = d
		}
	}
	return nil
}
func (f *fakeDeckRepository) SetGroup(_ context.Context, userID, id, groupID string, _ time.Time) error {
	d, ok := f.decks[id]
	if !ok || d.UserID != userID {
		return repositories.ErrNotFound
	}
	d.GroupID = groupID
	f.decks[id] = d
	return nil
}
func (f *fakeDeckRepository) ClearGroup(_ context.Context, userID, groupID string) error {
	for id, d := range f.decks {
		if d.UserID == userID && d.GroupID == groupID {
			d.GroupID = ""
			f.decks[id] = d
		}
	}
	return nil
}
func (f *fakeDeckRepository) ListWithStudyProfile(_ context.Context, userID string) ([]deck.Deck, error) {
	var out []deck.Deck
	for _, d := range f.decks {
		if d.UserID == userID && d.StudyProfileID != "" {
			out = append(out, d)
		}
	}
	return out, nil
}
func (f *fakeDeckRepository) Create(context.Context, deck.Deck) (deck.Deck, error) {
	panic("not implemented")
}
func (f *fakeDeckRepository) ListActive(context.Context, string) ([]deck.Deck, error) {
	panic("not implemented")
}
func (f *fakeDeckRepository) ListArchived(context.Context, string) ([]deck.Deck, error) {
	panic("not implemented")
}
func (f *fakeDeckRepository) Update(context.Context, deck.Deck) (deck.Deck, error) {
	panic("not implemented")
}
func (f *fakeDeckRepository) Archive(context.Context, string, string, time.Time) error {
	panic("not implemented")
}
func (f *fakeDeckRepository) Restore(context.Context, string, string, time.Time) error {
	panic("not implemented")
}
func (f *fakeDeckRepository) SetContinuePastGoalOn(context.Context, string, string, string, time.Time) error {
	panic("not implemented")
}
func (f *fakeDeckRepository) DeleteArchived(context.Context, string, string) error {
	panic("not implemented")
}

func newTestService(userIDs ...string) (Service, *fakeProfileRepository, *fakeUserRepository) {
	svc, profiles, users, _ := newTestServiceWithDecks(userIDs...)
	return svc, profiles, users
}

func newTestServiceWithDecks(userIDs ...string) (Service, *fakeProfileRepository, *fakeUserRepository, *fakeDeckRepository) {
	if len(userIDs) == 0 {
		userIDs = []string{"user-1", "user-2"}
	}
	profiles := &fakeProfileRepository{}
	users := newFakeUserRepository(userIDs...)
	decks := &fakeDeckRepository{decks: map[string]deck.Deck{}}
	return New(profiles, users, decks, clock.Fixed{Time: fixedNow}), profiles, users, decks
}

func input(name string) Input {
	return Input{Name: name, Rules: study.DefaultRules()}
}

func intPtr(n int) *int { return &n }

func mustCreate(t *testing.T, svc Service, userID, name string) studyprofile.Profile {
	t.Helper()
	p, err := svc.Create(context.Background(), userID, input(name))
	if err != nil {
		t.Fatalf("Create(%q) error: %v", name, err)
	}
	return p
}

func code(err error) apperror.Code {
	if e, ok := err.(*apperror.Error); ok {
		return e.Code
	}
	return ""
}

// ---------- tests ----------

func TestList_EveryUserHasTheBuiltInDefaultSelected(t *testing.T) {
	svc, _, _ := newTestService()

	o, err := svc.List(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if o.ActiveProfileID != studyprofile.DefaultID || len(o.Profiles) != 1 || !o.Profiles[0].IsDefault() {
		t.Fatalf("List() = %+v, want only the default, selected", o)
	}
	if o.Profiles[0].Name != studyprofile.DefaultName || o.Profiles[0].DailyCardLimit != nil {
		t.Errorf("default = %+v, want the named built-in without a daily goal", o.Profiles[0])
	}
}

func TestCreate_AddsAConfigurationAfterTheDefault(t *testing.T) {
	svc, _, _ := newTestService()

	created, err := svc.Create(context.Background(), "user-1", Input{
		Name:           "  Prova de inglês ",
		DailyCardLimit: intPtr(40),
		Rules: study.Rules{
			AgainDelayMinutes: 5,
			Hard:              study.RatingRule{FirstIntervalDays: 1, Multiplier: 1.1},
			Good:              study.RatingRule{FirstIntervalDays: 2, Multiplier: 1.8},
			Easy:              study.RatingRule{FirstIntervalDays: 5, Multiplier: 2.5},
		},
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if created.Name != "Prova de inglês" || created.UserID != "user-1" || created.IsDefault() {
		t.Errorf("Create() = %+v, want the trimmed name owned by user-1", created)
	}

	o, _ := svc.List(context.Background(), "user-1")
	if len(o.Profiles) != 2 || !o.Profiles[0].IsDefault() || o.Profiles[1].ID != created.ID {
		t.Fatalf("List() = %+v, want [default, created]", o.Profiles)
	}
	if o.ActiveProfileID != studyprofile.DefaultID {
		t.Errorf("creating a configuration must not switch to it; active = %q", o.ActiveProfileID)
	}
	if o.Profiles[1].DailyCardLimit == nil || *o.Profiles[1].DailyCardLimit != 40 || o.Profiles[1].Rules.AgainDelayMinutes != 5 {
		t.Errorf("stored configuration = %+v, want the values sent", o.Profiles[1])
	}

	// Another user sees none of it.
	other, _ := svc.List(context.Background(), "user-2")
	if len(other.Profiles) != 1 {
		t.Errorf("another user's list = %d configurations, want only the default", len(other.Profiles))
	}
}

func TestCreate_ValidatesEverything(t *testing.T) {
	svc, _, _ := newTestService()
	bad := study.DefaultRules()
	bad.Good.FirstIntervalDays = 99 // easy is 7

	cases := map[string]Input{
		"empty name":         {Name: "  ", Rules: study.DefaultRules()},
		"name too long":      {Name: strings.Repeat("x", studyprofile.MaxNameLength+1), Rules: study.DefaultRules()},
		"goal of zero":       {Name: "a", DailyCardLimit: intPtr(0), Rules: study.DefaultRules()},
		"goal too large":     {Name: "b", DailyCardLimit: intPtr(studyprofile.MaxDailyCardLimit + 1), Rules: study.DefaultRules()},
		"rules out of order": {Name: "c", Rules: bad},
		"rules left empty":   {Name: "d"},
	}
	for name, in := range cases {
		if _, err := svc.Create(context.Background(), "user-1", in); code(err) != apperror.CodeValidationError {
			t.Errorf("%s: error = %v, want a validation error", name, err)
		}
	}
	if o, _ := svc.List(context.Background(), "user-1"); len(o.Profiles) != 1 {
		t.Errorf("nothing invalid may be saved; got %d configurations", len(o.Profiles))
	}
}

func TestCreate_NamesAreUniquePerUserIgnoringCase(t *testing.T) {
	svc, _, _ := newTestService()
	mustCreate(t, svc, "user-1", "Prova")

	if _, err := svc.Create(context.Background(), "user-1", input(" prova ")); code(err) != apperror.CodeConflict {
		t.Errorf("a repeated name (different case): error = %v, want a conflict", err)
	}
	// Another user may use the same name.
	if _, err := svc.Create(context.Background(), "user-2", input("Prova")); err != nil {
		t.Errorf("another user's same name error: %v", err)
	}
}

func TestCreate_LimitsHowManyConfigurationsAUserCanHave(t *testing.T) {
	svc, _, _ := newTestService()
	for i := 0; i < studyprofile.MaxProfilesPerUser; i++ {
		mustCreate(t, svc, "user-1", fmt.Sprintf("Config %d", i))
	}
	if _, err := svc.Create(context.Background(), "user-1", input("one too many")); code(err) != apperror.CodeValidationError {
		t.Errorf("over the limit: error = %v, want a validation error", err)
	}
}

func TestCreate_UnknownUserIsNotFound(t *testing.T) {
	svc, _, _ := newTestService()
	if _, err := svc.Create(context.Background(), "nobody", input("x")); code(err) != apperror.CodeNotFound {
		t.Errorf("error = %v, want not found", err)
	}
}

func TestUpdate_ChangesOwnConfigurationOnly(t *testing.T) {
	svc, _, _ := newTestService()
	p := mustCreate(t, svc, "user-1", "Prova")

	updated, err := svc.Update(context.Background(), "user-1", p.ID, Input{Name: "Prova final", DailyCardLimit: intPtr(15), Rules: study.DefaultRules()})
	if err != nil || updated.Name != "Prova final" || updated.DailyCardLimit == nil || *updated.DailyCardLimit != 15 {
		t.Fatalf("Update() = %+v, %v", updated, err)
	}

	// The daily goal can be removed again.
	cleared, _ := svc.Update(context.Background(), "user-1", p.ID, input("Prova final"))
	if cleared.DailyCardLimit != nil {
		t.Errorf("Update() without a goal = %v, want the goal removed", *cleared.DailyCardLimit)
	}

	if _, err := svc.Update(context.Background(), "user-2", p.ID, input("Roubada")); code(err) != apperror.CodeNotFound {
		t.Errorf("updating someone else's configuration: error = %v, want not found", err)
	}
	if _, err := svc.Update(context.Background(), "user-1", "missing", input("x")); code(err) != apperror.CodeNotFound {
		t.Errorf("updating a missing configuration: error = %v, want not found", err)
	}
}

func TestUpdate_TheDefaultCannotBeChangedAndNamesStayUnique(t *testing.T) {
	svc, _, _ := newTestService()
	if _, err := svc.Update(context.Background(), "user-1", studyprofile.DefaultID, input("Meu padrão")); code(err) != apperror.CodeForbidden {
		t.Errorf("changing the default: error = %v, want forbidden", err)
	}

	mustCreate(t, svc, "user-1", "A")
	b := mustCreate(t, svc, "user-1", "B")
	if _, err := svc.Update(context.Background(), "user-1", b.ID, input("a")); code(err) != apperror.CodeConflict {
		t.Errorf("renaming to an existing name: error = %v, want a conflict", err)
	}
	// Keeping its own name is fine.
	if _, err := svc.Update(context.Background(), "user-1", b.ID, input("B")); err != nil {
		t.Errorf("updating without renaming error: %v", err)
	}
}

func TestSetActive_SelectsACustomConfigurationOrGoesBackToTheDefault(t *testing.T) {
	svc, _, _ := newTestService()
	p := mustCreate(t, svc, "user-1", "Prova")

	if err := svc.SetActive(context.Background(), "user-1", p.ID); err != nil {
		t.Fatalf("SetActive() error: %v", err)
	}
	o, _ := svc.List(context.Background(), "user-1")
	if o.ActiveProfileID != p.ID {
		t.Fatalf("active = %q, want %q", o.ActiveProfileID, p.ID)
	}
	active, _ := svc.Active(context.Background(), "user-1")
	if active.ID != p.ID {
		t.Errorf("Active() = %+v, want the selected configuration", active)
	}

	if err := svc.SetActive(context.Background(), "user-1", studyprofile.DefaultID); err != nil {
		t.Fatalf("SetActive(default) error: %v", err)
	}
	if o, _ := svc.List(context.Background(), "user-1"); o.ActiveProfileID != studyprofile.DefaultID {
		t.Errorf("active = %q, want the default", o.ActiveProfileID)
	}
}

func TestSetActive_RejectsUnknownAndForeignConfigurations(t *testing.T) {
	svc, _, _ := newTestService()
	mine := mustCreate(t, svc, "user-1", "Prova")

	if err := svc.SetActive(context.Background(), "user-1", "missing"); code(err) != apperror.CodeNotFound {
		t.Errorf("selecting a missing configuration: error = %v, want not found", err)
	}
	if err := svc.SetActive(context.Background(), "user-2", mine.ID); code(err) != apperror.CodeNotFound {
		t.Errorf("selecting someone else's configuration: error = %v, want not found", err)
	}
	if o, _ := svc.List(context.Background(), "user-2"); o.ActiveProfileID != studyprofile.DefaultID {
		t.Errorf("a failed selection must leave the default active; got %q", o.ActiveProfileID)
	}
}

func TestDelete_RemovesItAndFallsBackToTheDefaultIfItWasActive(t *testing.T) {
	svc, _, _ := newTestService()
	a := mustCreate(t, svc, "user-1", "A")
	b := mustCreate(t, svc, "user-1", "B")
	_ = svc.SetActive(context.Background(), "user-1", a.ID)

	// Deleting an inactive one leaves the selection alone.
	if err := svc.Delete(context.Background(), "user-1", b.ID); err != nil {
		t.Fatalf("Delete(b) error: %v", err)
	}
	if o, _ := svc.List(context.Background(), "user-1"); o.ActiveProfileID != a.ID || len(o.Profiles) != 2 {
		t.Fatalf("after deleting b: %+v, want a still active", o)
	}

	if err := svc.Delete(context.Background(), "user-1", a.ID); err != nil {
		t.Fatalf("Delete(a) error: %v", err)
	}
	o, _ := svc.List(context.Background(), "user-1")
	if o.ActiveProfileID != studyprofile.DefaultID || len(o.Profiles) != 1 {
		t.Errorf("after deleting the active one: %+v, want back on the default", o)
	}
}

func TestDelete_RefusesTheDefaultAndOtherPeoplesConfigurations(t *testing.T) {
	svc, _, _ := newTestService()
	mine := mustCreate(t, svc, "user-1", "Prova")

	if err := svc.Delete(context.Background(), "user-1", studyprofile.DefaultID); code(err) != apperror.CodeForbidden {
		t.Errorf("deleting the default: error = %v, want forbidden", err)
	}
	if err := svc.Delete(context.Background(), "user-2", mine.ID); code(err) != apperror.CodeNotFound {
		t.Errorf("deleting someone else's: error = %v, want not found", err)
	}
	if o, _ := svc.List(context.Background(), "user-1"); len(o.Profiles) != 2 {
		t.Errorf("nothing may have been deleted; got %d configurations", len(o.Profiles))
	}
}

func TestActive_FallsBackToTheDefaultWhenTheSelectedOneIsGone(t *testing.T) {
	svc, _, users := newTestService()
	// The user's record still points at a configuration that no longer exists.
	u := users.users["user-1"]
	u.ActiveStudyProfileID = "vanished"
	users.users["user-1"] = u

	active, err := svc.Active(context.Background(), "user-1")
	if err != nil || !active.IsDefault() {
		t.Errorf("Active() = %+v, %v; want the default", active, err)
	}
}

func TestList_MigratesAGoalSavedBeforeConfigurationsExisted(t *testing.T) {
	svc, profiles, users := newTestService()
	u := users.users["user-1"]
	u.LegacyDailyCardLimit = intPtr(7)
	users.users["user-1"] = u

	o, err := svc.List(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(o.Profiles) != 2 || o.Profiles[1].Name != migratedProfileName {
		t.Fatalf("List() = %+v, want the default plus %q", o.Profiles, migratedProfileName)
	}
	migrated := o.Profiles[1]
	if migrated.DailyCardLimit == nil || *migrated.DailyCardLimit != 7 || migrated.Rules != study.DefaultRules() {
		t.Errorf("migrated = %+v, want the old goal (7) with the default review rules", migrated)
	}
	if o.ActiveProfileID != migrated.ID {
		t.Errorf("active = %q, want the migrated configuration so their study does not change", o.ActiveProfileID)
	}
	if users.users["user-1"].LegacyDailyCardLimit != nil {
		t.Error("the old value must be cleared after migrating")
	}

	// Once only.
	if again, _ := svc.List(context.Background(), "user-1"); len(again.Profiles) != 2 || len(profiles.items) != 1 {
		t.Errorf("a second List() created another configuration: %d profiles stored", len(profiles.items))
	}
}

func TestList_MigrationKeepsAConfigurationTheUserAlreadySelected(t *testing.T) {
	svc, _, users := newTestService()
	mine := mustCreate(t, svc, "user-1", "Prova")
	_ = svc.SetActive(context.Background(), "user-1", mine.ID)
	u := users.users["user-1"]
	u.LegacyDailyCardLimit = intPtr(9)
	users.users["user-1"] = u

	o, _ := svc.List(context.Background(), "user-1")
	if o.ActiveProfileID != mine.ID || len(o.Profiles) != 3 {
		t.Errorf("List() = active %q with %d configurations; want their own kept active plus the migrated one", o.ActiveProfileID, len(o.Profiles))
	}
}

func TestList_AnInvalidOldGoalIsJustDropped(t *testing.T) {
	svc, profiles, users := newTestService()
	u := users.users["user-1"]
	u.LegacyDailyCardLimit = intPtr(5000)
	users.users["user-1"] = u

	o, err := svc.List(context.Background(), "user-1")
	if err != nil || len(o.Profiles) != 1 || len(profiles.items) != 0 {
		t.Errorf("List() = %+v, %v; want only the default and nothing created", o.Profiles, err)
	}
	if users.users["user-1"].LegacyDailyCardLimit != nil {
		t.Error("the invalid old value must still be cleared")
	}
}

// ---------- per-deck configuration ----------

func TestForDeck_FollowsTheGeneralConfigurationUnlessTheDeckHasItsOwn(t *testing.T) {
	svc, _, _, decks := newTestServiceWithDecks()
	decks.addDeck("user-1", "deck-1")
	general := mustCreate(t, svc, "user-1", "Geral")
	_ = svc.SetActive(context.Background(), "user-1", general.ID)

	p, own, err := svc.ForDeck(context.Background(), "user-1", "deck-1")
	if err != nil || own || p.ID != general.ID {
		t.Fatalf("ForDeck() = %+v, own=%v, %v; want the general configuration, not the deck's own", p, own, err)
	}

	special := mustCreate(t, svc, "user-1", "Só deste baralho")
	if err := svc.SetDeckProfile(context.Background(), "user-1", "deck-1", special.ID); err != nil {
		t.Fatalf("SetDeckProfile() error: %v", err)
	}
	p, own, err = svc.ForDeck(context.Background(), "user-1", "deck-1")
	if err != nil || !own || p.ID != special.ID {
		t.Fatalf("ForDeck() = %+v, own=%v, %v; want the deck's own configuration to win", p, own, err)
	}

	// Back to following the general one.
	if err := svc.SetDeckProfile(context.Background(), "user-1", "deck-1", ""); err != nil {
		t.Fatalf("SetDeckProfile(\"\") error: %v", err)
	}
	if p, own, _ := svc.ForDeck(context.Background(), "user-1", "deck-1"); own || p.ID != general.ID {
		t.Errorf("after clearing: %+v, own=%v; want the general configuration again", p, own)
	}
}

func TestForDeck_ADeckCanUseTheBuiltInDefaultWhileTheGeneralOneIsCustom(t *testing.T) {
	svc, _, _, decks := newTestServiceWithDecks()
	decks.addDeck("user-1", "deck-1")
	general := mustCreate(t, svc, "user-1", "Geral")
	_ = svc.SetActive(context.Background(), "user-1", general.ID)

	if err := svc.SetDeckProfile(context.Background(), "user-1", "deck-1", studyprofile.DefaultID); err != nil {
		t.Fatalf("SetDeckProfile(default) error: %v", err)
	}
	p, own, err := svc.ForDeck(context.Background(), "user-1", "deck-1")
	if err != nil || !own || !p.IsDefault() {
		t.Errorf("ForDeck() = %+v, own=%v, %v; want the built-in default as the deck's own", p, own, err)
	}
}

func TestForDeck_AStaleConfigurationFallsBackToTheGeneralOne(t *testing.T) {
	svc, _, _, decks := newTestServiceWithDecks()
	decks.addDeck("user-1", "deck-1")
	d := decks.decks["deck-1"]
	d.StudyProfileID = "vanished"
	decks.decks["deck-1"] = d

	p, own, err := svc.ForDeck(context.Background(), "user-1", "deck-1")
	if err != nil || own || !p.IsDefault() {
		t.Errorf("ForDeck() = %+v, own=%v, %v; want the general (default) configuration, not own", p, own, err)
	}
	if ids, _ := svc.OwnDeckIDs(context.Background(), "user-1"); len(ids) != 0 {
		t.Errorf("OwnDeckIDs() = %v, want none: a deck pointing at a vanished configuration is not 'own'", ids)
	}
}

func TestForDeck_RejectsAnotherUsersDeck(t *testing.T) {
	svc, _, _, decks := newTestServiceWithDecks()
	decks.addDeck("user-1", "deck-1")

	if _, _, err := svc.ForDeck(context.Background(), "user-2", "deck-1"); code(err) != apperror.CodeNotFound {
		t.Errorf("error = %v, want not found", err)
	}
}

func TestOwnDeckIDs_ListsOnlyDecksWithAValidOwnConfiguration(t *testing.T) {
	svc, _, _, decks := newTestServiceWithDecks()
	decks.addDeck("user-1", "own-custom")
	decks.addDeck("user-1", "own-default")
	decks.addDeck("user-1", "follows-general")
	decks.addDeck("user-2", "someone-elses")
	p := mustCreate(t, svc, "user-1", "Especial")
	_ = svc.SetDeckProfile(context.Background(), "user-1", "own-custom", p.ID)
	_ = svc.SetDeckProfile(context.Background(), "user-1", "own-default", studyprofile.DefaultID)

	ids, err := svc.OwnDeckIDs(context.Background(), "user-1")
	if err != nil || len(ids) != 2 {
		t.Fatalf("OwnDeckIDs() = %v, %v; want the two decks with their own configuration", ids, err)
	}
	got := map[string]bool{ids[0]: true, ids[1]: true}
	if !got["own-custom"] || !got["own-default"] {
		t.Errorf("OwnDeckIDs() = %v, want own-custom and own-default", ids)
	}
}

func TestSetDeckProfile_ValidatesTheDeckAndTheConfiguration(t *testing.T) {
	svc, _, _, decks := newTestServiceWithDecks()
	decks.addDeck("user-1", "deck-1")
	mine := mustCreate(t, svc, "user-1", "Minha")

	if err := svc.SetDeckProfile(context.Background(), "user-1", "missing", ""); code(err) != apperror.CodeNotFound {
		t.Errorf("a missing deck: error = %v, want not found", err)
	}
	if err := svc.SetDeckProfile(context.Background(), "user-2", "deck-1", ""); code(err) != apperror.CodeNotFound {
		t.Errorf("someone else's deck: error = %v, want not found", err)
	}
	if err := svc.SetDeckProfile(context.Background(), "user-1", "deck-1", "missing"); code(err) != apperror.CodeNotFound {
		t.Errorf("a missing configuration: error = %v, want not found", err)
	}
	other := mustCreate(t, svc, "user-2", "De outra pessoa")
	if err := svc.SetDeckProfile(context.Background(), "user-1", "deck-1", other.ID); code(err) != apperror.CodeNotFound {
		t.Errorf("someone else's configuration: error = %v, want not found", err)
	}
	if got := decks.decks["deck-1"].StudyProfileID; got != "" {
		t.Errorf("a rejected choice must change nothing; deck has %q", got)
	}
	if err := svc.SetDeckProfile(context.Background(), "user-1", "deck-1", mine.ID); err != nil {
		t.Errorf("a valid choice error: %v", err)
	}
}

func TestDelete_DecksThatUsedTheConfigurationFollowTheGeneralOneAgain(t *testing.T) {
	svc, _, _, decks := newTestServiceWithDecks()
	decks.addDeck("user-1", "deck-1")
	decks.addDeck("user-1", "deck-2")
	a := mustCreate(t, svc, "user-1", "A")
	b := mustCreate(t, svc, "user-1", "B")
	_ = svc.SetDeckProfile(context.Background(), "user-1", "deck-1", a.ID)
	_ = svc.SetDeckProfile(context.Background(), "user-1", "deck-2", b.ID)

	if err := svc.Delete(context.Background(), "user-1", a.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
	if got := decks.decks["deck-1"].StudyProfileID; got != "" {
		t.Errorf("deck-1 still points at %q after its configuration was deleted", got)
	}
	if got := decks.decks["deck-2"].StudyProfileID; got != b.ID {
		t.Errorf("deck-2 = %q, want it to keep using its own configuration %q", got, b.ID)
	}
}

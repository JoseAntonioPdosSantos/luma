// Package flashcardservice implements flashcard CRUD business rules,
// including verifying that the target deck belongs to the acting user
// before any flashcard is created, read, updated, or archived (spec
// section 6: "The API must verify that the deck belongs to the current
// user").
package flashcardservice

import (
	"context"
	"errors"
	"io"
	"strings"

	"flashcard-backend/internal/apperror"
	"flashcard-backend/internal/domain/flashcard"
	"flashcard-backend/internal/ports/audiostore"
	"flashcard-backend/internal/ports/clock"
	"flashcard-backend/internal/ports/repositories"
)

type Service struct {
	flashcards        repositories.FlashcardRepository
	decks             repositories.DeckRepository
	audio             audiostore.Store
	maxAudioSizeBytes int64
	clock             clock.Clock
}

func New(
	flashcards repositories.FlashcardRepository,
	decks repositories.DeckRepository,
	audio audiostore.Store,
	maxAudioSizeBytes int64,
	c clock.Clock,
) Service {
	return Service{
		flashcards:        flashcards,
		decks:             decks,
		audio:             audio,
		maxAudioSizeBytes: maxAudioSizeBytes,
		clock:             c,
	}
}

// Draft is the validated input for creating or updating a flashcard.
type Draft struct {
	Question            string
	Answer              string
	Hint                string
	ExtendedExampleText string
	ExtendedTranslation string
}

func (s Service) Create(ctx context.Context, userID, deckID string, draft Draft) (flashcard.Flashcard, error) {
	if _, err := s.requireOwnedDeck(ctx, userID, deckID); err != nil {
		return flashcard.Flashcard{}, err
	}

	question, answer, hint, example, err := validateDraft(draft)
	if err != nil {
		return flashcard.Flashcard{}, err
	}

	now := s.clock.Now()
	card := flashcard.New(userID, deckID, question, answer, example, now)
	card.Hint = hint
	created, err := s.flashcards.Create(ctx, card)
	if err != nil {
		return flashcard.Flashcard{}, apperror.Internal(err)
	}
	return created, nil
}

func (s Service) Get(ctx context.Context, userID, id string) (flashcard.Flashcard, error) {
	card, err := s.flashcards.FindByID(ctx, userID, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return flashcard.Flashcard{}, apperror.NotFound("flashcard.notFound", "flashcard not found")
	}
	if err != nil {
		return flashcard.Flashcard{}, apperror.Internal(err)
	}
	return card, nil
}

const (
	DefaultPageSize = 30
	MaxPageSize     = 100
	MaxQueryLength  = 200
)

// Page is one page of a deck's flashcards plus the total number of cards
// that match the request (across all pages).
type Page struct {
	Cards []flashcard.Flashcard
	Total int
}

// ListPage returns one page of the deck's active flashcards, or of its
// archived ones when archived is true, optionally filtered by a search
// query. limit <= 0 uses DefaultPageSize and is capped at MaxPageSize.
func (s Service) ListPage(ctx context.Context, userID, deckID string, archived bool, query string, limit, offset int) (Page, error) {
	if _, err := s.requireOwnedDeck(ctx, userID, deckID); err != nil {
		return Page{}, err
	}

	query = strings.TrimSpace(query)
	if len(query) > MaxQueryLength {
		return Page{}, apperror.Validation("flashcard.list.query.tooLong", "search text must be at most 200 characters")
	}
	if offset < 0 {
		return Page{}, apperror.Validation("flashcard.list.offset.negative", "offset must not be negative")
	}
	if limit <= 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}

	cards, total, err := s.flashcards.SearchByDeck(ctx, userID, deckID, repositories.FlashcardListOptions{
		Archived: archived,
		Query:    query,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return Page{}, apperror.Internal(err)
	}
	return Page{Cards: cards, Total: total}, nil
}

// ArchivedFlashcard is an archived card plus the name of the deck it
// belongs to, so a cross-deck "archived" screen can say where it came from.
type ArchivedFlashcard struct {
	Card     flashcard.Flashcard
	DeckName string
}

// ListArchivedAcrossDecks returns the user's archived flashcards from all
// of their active decks, most recently archived first. Cards that belong to
// an archived deck are left out: they come back together with the deck
// when it is restored.
func (s Service) ListArchivedAcrossDecks(ctx context.Context, userID string) ([]ArchivedFlashcard, error) {
	cards, err := s.flashcards.ListArchivedByUser(ctx, userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if len(cards) == 0 {
		return []ArchivedFlashcard{}, nil
	}

	activeDecks, err := s.decks.ListActive(ctx, userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	deckNames := make(map[string]string, len(activeDecks))
	for _, d := range activeDecks {
		deckNames[d.ID] = d.Name
	}

	result := make([]ArchivedFlashcard, 0, len(cards))
	for _, c := range cards {
		name, ok := deckNames[c.DeckID]
		if !ok {
			continue
		}
		result = append(result, ArchivedFlashcard{Card: c, DeckName: name})
	}
	return result, nil
}

// DeleteArchived permanently deletes an archived flashcard and its audio
// file. It cannot be undone, so only an already-archived card is accepted;
// anything else is reported as not found.
func (s Service) DeleteArchived(ctx context.Context, userID, id string) error {
	card, err := s.flashcards.DeleteArchived(ctx, userID, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return apperror.NotFound("flashcard.archivedNotFound", "archived flashcard not found")
	}
	if err != nil {
		return apperror.Internal(err)
	}

	if card.Audio != nil {
		_ = s.audio.Delete(ctx, card.Audio.GridFSFileID)
	}
	return nil
}

// Restore un-archives a flashcard, keeping its scheduling state and
// review history.
func (s Service) Restore(ctx context.Context, userID, id string) error {
	err := s.flashcards.Restore(ctx, userID, id, s.clock.Now())
	if errors.Is(err, repositories.ErrNotFound) {
		return apperror.NotFound("flashcard.notFound", "flashcard not found")
	}
	if err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func (s Service) Update(ctx context.Context, userID, id string, draft Draft) (flashcard.Flashcard, error) {
	existing, err := s.Get(ctx, userID, id)
	if err != nil {
		return flashcard.Flashcard{}, err
	}

	question, answer, hint, example, err := validateDraft(draft)
	if err != nil {
		return flashcard.Flashcard{}, err
	}

	existing.Question = question
	existing.Answer = answer
	existing.Hint = hint
	existing.ExtendedExample = example
	existing.UpdatedAt = s.clock.Now()

	updated, err := s.flashcards.Update(ctx, existing)
	if errors.Is(err, repositories.ErrNotFound) {
		return flashcard.Flashcard{}, apperror.NotFound("flashcard.notFound", "flashcard not found")
	}
	if err != nil {
		return flashcard.Flashcard{}, apperror.Internal(err)
	}
	return updated, nil
}

func (s Service) Archive(ctx context.Context, userID, id string) error {
	err := s.flashcards.Archive(ctx, userID, id, s.clock.Now())
	if errors.Is(err, repositories.ErrNotFound) {
		return apperror.NotFound("flashcard.notFound", "flashcard not found")
	}
	if err != nil {
		return apperror.Internal(err)
	}
	return nil
}

// UploadAudio replaces (or adds) the audio attached to a flashcard.
// contentType must be one of flashcard.AllowedAudioContentTypes. r is
// streamed directly into storage, capped at s.maxAudioSizeBytes — an
// oversized upload is rejected before anything is persisted. If the card
// already had audio, the old file is deleted after the new one is
// successfully attached (spec section 7: "delete or clean up orphaned
// audio files ... when its audio is replaced").
func (s Service) UploadAudio(ctx context.Context, userID, id, contentType string, r io.Reader) (flashcard.Flashcard, error) {
	if err := flashcard.ValidateAudioContentType(contentType); err != nil {
		return flashcard.Flashcard{}, err
	}

	existing, err := s.Get(ctx, userID, id)
	if err != nil {
		return flashcard.Flashcard{}, err
	}

	fileID, _, err := s.audio.Save(ctx, r, s.maxAudioSizeBytes)
	if errors.Is(err, audiostore.ErrTooLarge) {
		return flashcard.Flashcard{}, apperror.PayloadTooLarge("audio.tooLarge", "audio file exceeds the maximum allowed size")
	}
	if err != nil {
		return flashcard.Flashcard{}, apperror.Internal(err)
	}

	previousAudio := existing.Audio
	existing.Audio = &flashcard.Audio{GridFSFileID: fileID, ContentType: contentType}
	existing.UpdatedAt = s.clock.Now()

	updated, err := s.flashcards.Update(ctx, existing)
	if err != nil {
		// The flashcard was not updated to reference the new file, so
		// clean it up rather than leaving it orphaned.
		_ = s.audio.Delete(ctx, fileID)
		return flashcard.Flashcard{}, apperror.Internal(err)
	}

	if previousAudio != nil {
		_ = s.audio.Delete(ctx, previousAudio.GridFSFileID)
	}

	return updated, nil
}

// GetAudio returns a flashcard's audio metadata and a stream of its
// bytes. The caller must Close the stream.
func (s Service) GetAudio(ctx context.Context, userID, id string) (flashcard.Audio, io.ReadCloser, error) {
	card, err := s.Get(ctx, userID, id)
	if err != nil {
		return flashcard.Audio{}, nil, err
	}
	if card.Audio == nil {
		return flashcard.Audio{}, nil, apperror.NotFound("flashcard.noAudio", "this flashcard has no audio")
	}

	stream, err := s.audio.Open(ctx, card.Audio.GridFSFileID)
	if errors.Is(err, audiostore.ErrNotFound) {
		return flashcard.Audio{}, nil, apperror.NotFound("flashcard.noAudio", "this flashcard has no audio")
	}
	if err != nil {
		return flashcard.Audio{}, nil, apperror.Internal(err)
	}
	return *card.Audio, stream, nil
}

// DeleteAudio removes a flashcard's audio, if any. It is idempotent:
// calling it on a flashcard with no audio succeeds without error.
func (s Service) DeleteAudio(ctx context.Context, userID, id string) error {
	existing, err := s.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	if existing.Audio == nil {
		return nil
	}

	oldFileID := existing.Audio.GridFSFileID
	existing.Audio = nil
	existing.UpdatedAt = s.clock.Now()

	if _, err := s.flashcards.Update(ctx, existing); err != nil {
		return apperror.Internal(err)
	}

	_ = s.audio.Delete(ctx, oldFileID)
	return nil
}

func (s Service) requireOwnedDeck(ctx context.Context, userID, deckID string) (string, error) {
	_, err := s.decks.FindByID(ctx, userID, deckID)
	if errors.Is(err, repositories.ErrNotFound) {
		return "", apperror.NotFound("deck.notFound", "deck not found")
	}
	if err != nil {
		return "", apperror.Internal(err)
	}
	return deckID, nil
}

func validateDraft(draft Draft) (question, answer, hint string, example *flashcard.ExtendedExample, err error) {
	question, err = flashcard.ValidateQuestion(draft.Question)
	if err != nil {
		return "", "", "", nil, err
	}
	answer, err = flashcard.ValidateAnswer(draft.Answer)
	if err != nil {
		return "", "", "", nil, err
	}
	hint, err = flashcard.ValidateHint(draft.Hint)
	if err != nil {
		return "", "", "", nil, err
	}

	exampleText, err := flashcard.ValidateExtendedExampleText(draft.ExtendedExampleText)
	if err != nil {
		return "", "", "", nil, err
	}
	// The translation shares the extended example's length limit: it is
	// part of the same optional field and deserves the same payload-size
	// protection (spec section 17 does not list it separately).
	exampleTranslation, err := flashcard.ValidateExtendedExampleText(draft.ExtendedTranslation)
	if err != nil {
		return "", "", "", nil, err
	}

	if exampleText != "" {
		example = &flashcard.ExtendedExample{Text: exampleText, Translation: exampleTranslation}
	}

	return question, answer, hint, example, nil
}

// Command api runs the flashcard learning system HTTP API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	// Embeds the IANA time zone database so time.LoadLocation works in the
	// minimal container image, which ships no zoneinfo.
	_ "time/tzdata"

	"flashcard-backend/internal/adapters/authtoken"
	gridfsadapter "flashcard-backend/internal/adapters/gridfs"
	httpapi "flashcard-backend/internal/adapters/http"
	mongoadapter "flashcard-backend/internal/adapters/mongodb"
	"flashcard-backend/internal/application/deckgroupservice"
	"flashcard-backend/internal/application/deckservice"
	"flashcard-backend/internal/application/flashcardservice"
	"flashcard-backend/internal/application/profileservice"
	"flashcard-backend/internal/application/studyservice"
	"flashcard-backend/internal/application/userservice"
	"flashcard-backend/internal/config"
	"flashcard-backend/internal/ports/clock"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := mongoadapter.Connect(ctx, cfg.MongoDBURI, cfg.MongoDBDatabase)
	if err != nil {
		return err
	}
	if err := mongoadapter.EnsureIndexes(ctx, db); err != nil {
		return err
	}

	audioStore, err := gridfsadapter.New(db)
	if err != nil {
		return err
	}

	realClock := clock.Real{}
	userRepo := mongoadapter.NewUserRepository(db)
	deckRepo := mongoadapter.NewDeckRepository(db)
	flashcardRepo := mongoadapter.NewFlashcardRepository(db)
	reviewRepo := mongoadapter.NewReviewEventRepository(db)
	studyProfileRepo := mongoadapter.NewStudyProfileRepository(db)
	deckGroupRepo := mongoadapter.NewDeckGroupRepository(db)
	profileSvc := profileservice.New(studyProfileRepo, userRepo, deckRepo, realClock)
	deps := httpapi.Dependencies{
		Users:      userservice.New(userRepo, realClock),
		Profiles:   profileSvc,
		Decks:      deckservice.New(deckRepo, flashcardRepo, reviewRepo, audioStore, realClock),
		DeckGroups: deckgroupservice.New(deckGroupRepo, deckRepo, realClock),
		Flashcards: flashcardservice.New(flashcardRepo, deckRepo, audioStore, cfg.MaxAudioSizeBytes, realClock),
		Study:      studyservice.New(flashcardRepo, userRepo, profileSvc, deckRepo, reviewRepo, realClock),
		Sessions:   authtoken.NewHMACManager(cfg.SessionSecret, 0),
		Clock:      realClock,
		DB:         mongoadapter.NewPinger(db),
	}

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           httpapi.NewRouter(cfg, deps),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("starting server", "port", cfg.HTTPPort, "env", cfg.AppEnv)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		slog.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

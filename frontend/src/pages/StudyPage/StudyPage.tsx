import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { apiUrl } from "../../services/api";
import { archiveFlashcard } from "../../services/flashcardService";
import { BulbIcon } from "../../components/Icons";
import { generateAutoHint } from "../../utils/autoHint";
import {
  continueStudyingPastGoal,
  getDueFlashcards,
  getStudyProgress,
  submitReview,
  type Rating,
} from "../../services/studyService";
import { getDeck } from "../../services/deckService";
import { describeNextReview } from "../../utils/nextReview";
import type { Deck, Flashcard, StudyProgress } from "../../types";

type LoadState = "loading" | "loaded" | "error";

interface Counters {
  studied: number;
  again: number;
  hard: number;
  good: number;
  easy: number;
}

const emptyCounters: Counters = { studied: 0, again: 0, hard: 0, good: 0, easy: 0 };

// Getting a card right with the hint's help is not mastering it, so once
// the hint was looked at the top rating is off the table for that card.
const RATING_BLOCKED_AFTER_HINT: Rating = "easy";

const RATING_KEYS: { value: Rating; labelKey: string }[] = [
  { value: "again", labelKey: "study.ratingAgain" },
  { value: "hard", labelKey: "study.ratingHard" },
  { value: "good", labelKey: "study.ratingGood" },
  { value: "easy", labelKey: "study.ratingEasy" },
];

export function StudyPage() {
  const { deckId } = useParams<{ deckId: string }>();
  const { t } = useTranslation();
  const ratings = RATING_KEYS.map((r) => ({ value: r.value, label: t(r.labelKey) }));

  const [loadState, setLoadState] = useState<LoadState>("loading");
  const [cards, setCards] = useState<Flashcard[]>([]);
  const [currentIndex, setCurrentIndex] = useState(0);
  const [revealed, setRevealed] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [counters, setCounters] = useState<Counters>(emptyCounters);
  const [finished, setFinished] = useState(false);
  // Today's standing against the user's daily goal (if they have one).
  const [progress, setProgress] = useState<StudyProgress | null>(null);
  // Card count and next review date of the deck, for the "everything is up
  // to date" screen. Purely informative: if it cannot be loaded that screen
  // simply falls back to the plain empty message.
  const [deckSummary, setDeckSummary] = useState<Deck | null>(null);
  // Id of the card whose hint is currently shown; tying it to the card id
  // means the hint hides itself when the next card comes up.
  const [hintShownFor, setHintShownFor] = useState<string | null>(null);
  const [confirmingArchive, setConfirmingArchive] = useState(false);
  const [archiveError, setArchiveError] = useState(false);
  const questionShownAt = useRef(Date.now());
  // Ids of the cards whose hint was looked at during this session, sent
  // along with the rating so the statistics can show how often help was
  // needed. It is state (not a ref) because the rating buttons depend on it.
  const [hintUsedIds, setHintUsedIds] = useState<Set<string>>(new Set());

  // The server decides how many cards to serve: it applies the daily goal
  // unless the user already chose to continue past it today.
  const loadDueCards = useCallback(
    () => {
      if (!deckId) return;
      setLoadState("loading");
      setFinished(false);
      Promise.all([
        getDueFlashcards(deckId),
        getStudyProgress(deckId).catch(() => null),
        getDeck(deckId).catch(() => null),
      ])
        .then(([due, studyProgress, deck]) => {
          setProgress(studyProgress);
          setDeckSummary(deck);
          setCards(due);
          setCurrentIndex(0);
          setRevealed(false);
          setCounters(emptyCounters);
          setHintUsedIds(new Set());
          questionShownAt.current = Date.now();
          setLoadState("loaded");
        })
        .catch(() => setLoadState("error"));
    },
    [deckId],
  );

  useEffect(() => {
    loadDueCards();
  }, [loadDueCards]);

  // "Continuar estudando": go on past the daily goal. The choice is saved on
  // the account, so it holds for the rest of the day on every device.
  async function continuePastGoal() {
    try {
      await continueStudyingPastGoal(deckId);
    } catch {
      setLoadState("error");
      return;
    }
    loadDueCards();
  }

  const currentCard = cards[currentIndex];

  // A card's own hint wins; cards without one get an automatic hint
  // (first letter of each word of the answer, the rest as underscores).
  const customHint = currentCard?.hint ?? "";
  const hintText = customHint || (currentCard ? generateAutoHint(currentCard.answer) : "");
  const hintIsAutomatic = customHint === "" && hintText !== "";
  const hintVisible = currentCard !== undefined && hintShownFor === currentCard.id;
  const hintUsed = currentCard !== undefined && hintUsedIds.has(currentCard.id);

  function toggleHint() {
    if (!currentCard || hintText === "") return;
    if (hintShownFor === currentCard.id) {
      setHintShownFor(null);
    } else {
      setHintUsedIds((used) => new Set(used).add(currentCard.id));
      setHintShownFor(currentCard.id);
    }
  }

  async function handleRate(rating: Rating) {
    if (!currentCard || submitting) return;
    if (hintUsed && rating === RATING_BLOCKED_AFTER_HINT) return;
    setSubmitting(true);
    const responseTimeMs = Date.now() - questionShownAt.current;
    try {
      await submitReview(currentCard.id, rating, responseTimeMs, hintUsed);
      setCounters((c) => ({ ...c, studied: c.studied + 1, [rating]: c[rating] + 1 }));

      const lastCard = currentIndex + 1 >= cards.length;
      if (lastCard) {
        // The summary depends on whether this answer used up the daily
        // goal, so wait for the fresh numbers before showing it.
        const fresh = await getStudyProgress(deckId).catch(() => null);
        if (fresh) setProgress(fresh);
        setFinished(true);
      } else {
        // Keep the goal counter current without holding up the next card.
        getStudyProgress(deckId)
          .then(setProgress)
          .catch(() => undefined);
        setCurrentIndex((i) => i + 1);
        setRevealed(false);
        setConfirmingArchive(false);
        questionShownAt.current = Date.now();
      }
    } finally {
      setSubmitting(false);
    }
  }

  // Archiving is a soft delete (the card leaves every study session but its
  // review history is kept). It is not a review, so counters are untouched.
  async function handleArchiveCurrent() {
    if (!currentCard || submitting) return;
    setSubmitting(true);
    setArchiveError(false);
    try {
      await archiveFlashcard(currentCard.id);
      const remaining = cards.filter((c) => c.id !== currentCard.id);
      setCards(remaining);
      setConfirmingArchive(false);
      setRevealed(false);
      questionShownAt.current = Date.now();
      if (currentIndex >= remaining.length) {
        setFinished(true);
      }
    } catch {
      setArchiveError(true);
    } finally {
      setSubmitting(false);
    }
  }

  // A window-level listener (rather than an onKeyDown prop on the page)
  // keeps shortcuts working regardless of which element has focus — in
  // particular, revealing the answer removes the "Show answer" button
  // from the DOM, which would otherwise drop focus to <body> and stop
  // keydown events from ever reaching a handler on this page.
  useEffect(() => {
    if (finished || loadState !== "loaded") return;

    function onKeyDown(e: globalThis.KeyboardEvent) {
      if (e.ctrlKey || e.metaKey || e.altKey) return;
      if (!revealed && (e.key === "h" || e.key === "H")) {
        e.preventDefault();
        toggleHint();
        return;
      }
      if (!revealed && (e.key === " " || e.key === "Enter")) {
        e.preventDefault();
        setRevealed(true);
        return;
      }
      if (revealed && !submitting) {
        const byKey: Record<string, Rating> = { "1": "again", "2": "hard", "3": "good", "4": "easy" };
        const rating = byKey[e.key];
        if (rating) {
          e.preventDefault();
          handleRate(rating);
        }
      }
    }

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  });

  if (!deckId) return null;

  if (loadState === "loading") {
    return (
      <main className="app-shell">
        <p className="loading">{t("study.loading")}</p>
      </main>
    );
  }

  if (loadState === "error") {
    return (
      <main className="app-shell">
        <p className="error">{t("study.loadError")}</p>
        <Link to={`/decks/${deckId}`}>{t("study.backToDeck")}</Link>
      </main>
    );
  }

  const hasGoal = progress !== null && progress.dailyCardLimit !== null;
  const goalReached = hasGoal && progress.remaining === 0;
  // Once the user chose to continue today, the goal no longer stops the session.
  const stoppedByGoal = goalReached && !progress.continuingPastGoal;

  // Opening a study session after the goal was already reached: same layout
  // as the session summary, with a clear way to carry on.
  if (cards.length === 0 && stoppedByGoal) {
    return (
      <main className="app-shell">
        <div className="session-summary">
          <div className="session-summary-icon">✓</div>
          <h1>{t("study.goalCompleteTitle")}</h1>
          <p className="hint">{t("study.goalCompleteBody", { count: progress.studiedToday })}</p>
          <div className="session-stats session-stats-two">
            <div className="session-stat">
              <span className="session-stat-value">{progress.dailyCardLimit}</span>
              <span className="session-stat-label">{t("study.goalLabel")}</span>
            </div>
            <div className="session-stat">
              <span className="session-stat-value">{progress.studiedToday}</span>
              <span className="session-stat-label">{t("study.studiedTodayLabel")}</span>
            </div>
          </div>
          <div className="button-row">
            <Link to={`/decks/${deckId}`}>
              <button className="secondary">{t("study.backToDeck")}</button>
            </Link>
            <button onClick={continuePastGoal}>{t("study.continueStudying")}</button>
          </div>
        </div>
      </main>
    );
  }

  // Nothing due, but the deck has cards: they are all scheduled for later.
  if (cards.length === 0 && deckSummary && typeof deckSummary.totalCards === "number" && deckSummary.totalCards > 0) {
    const nextDue = deckSummary.nextDueAt ? new Date(deckSummary.nextDueAt) : null;
    const next = nextDue && !Number.isNaN(nextDue.getTime()) ? describeNextReview(nextDue) : null;

    return (
      <main className="app-shell">
        <div className="session-summary">
          <div className="session-summary-icon">✓</div>
          <h1>{t("study.allCaughtUpTitle")}</h1>
          <p className="hint">
            {next ? t("study.nextReviewBody", { relative: next.relative, when: next.when }) : t("study.nothingToReview")}
          </p>
          <div className="session-stats session-stats-three">
            <div className="session-stat">
              <span className="session-stat-value">0</span>
              <span className="session-stat-label">{t("study.toReviewNowLabel")}</span>
            </div>
            <div className="session-stat">
              <span className="session-stat-value">{next ? next.relative : "—"}</span>
              <span className="session-stat-label">{t("study.nextReviewLabel")}</span>
            </div>
            <div className="session-stat">
              <span className="session-stat-value">{deckSummary.totalCards}</span>
              <span className="session-stat-label">{t("study.cardsInDeckLabel")}</span>
            </div>
          </div>
          <div className="button-row">
            <Link to={`/decks/${deckId}`}>
              <button className="secondary">{t("study.backToDeck")}</button>
            </Link>
            <Link to={`/decks/${deckId}/flashcards/new`}>
              <button>{t("study.addFlashcard")}</button>
            </Link>
          </div>
        </div>
      </main>
    );
  }

  if (cards.length === 0) {
    return (
      <main className="app-shell">
        <Link to={`/decks/${deckId}`} className="back-link">
          &larr; {t("study.backToDeck")}
        </Link>
        <p className="empty-state">{t("study.emptyState")}</p>
      </main>
    );
  }

  if (finished) {
    const errors = counters.again;
    const correct = counters.studied - errors;
    const accuracy = counters.studied > 0 ? Math.round((correct / counters.studied) * 100) : 0;

    return (
      <main className="app-shell">
        <div className="session-summary">
          <div className="session-summary-icon">✓</div>
          {stoppedByGoal ? (
            <>
              <h1>{t("study.goalCompleteTitle")}</h1>
              <p className="hint">{t("study.sessionCompleteGoalBody", { count: progress.studiedToday })}</p>
            </>
          ) : (
            <>
              <h1>{t("study.sessionCompleteTitle")}</h1>
              <p className="hint">{t("study.greatJob")}</p>
            </>
          )}
          <div className="session-stats">
            <div className="session-stat">
              <span className="session-stat-value">{counters.studied}</span>
              <span className="session-stat-label">{t("study.studiedLabel")}</span>
            </div>
            <div className="session-stat">
              <span className="session-stat-value" style={{ color: "var(--color-success)" }}>
                {correct}
              </span>
              <span className="session-stat-label">{t("study.correctLabel")}</span>
            </div>
            <div className="session-stat">
              <span className="session-stat-value" style={{ color: "var(--color-error)" }}>
                {errors}
              </span>
              <span className="session-stat-label">{t("study.errorsLabel")}</span>
            </div>
            <div className="session-stat">
              <span className="session-stat-value">{accuracy}%</span>
              <span className="session-stat-label">{t("study.accuracyLabel")}</span>
            </div>
          </div>
          <div className="button-row">
            <Link to={`/decks/${deckId}`}>
              <button className="secondary">{t("study.backToDeck")}</button>
            </Link>
            {stoppedByGoal ? (
              <button onClick={continuePastGoal}>{t("study.continueStudying")}</button>
            ) : (
              <button onClick={() => loadDueCards()}>{t("study.studyMore")}</button>
            )}
          </div>
        </div>
      </main>
    );
  }

  return (
    <main className="app-shell study-shell">
      <Link to={`/decks/${deckId}`} className="back-link">
        &larr; {t("study.backToDeck")}
      </Link>
      <p className="hint">{t("study.cardCounter", { current: currentIndex + 1, total: cards.length })}</p>
      {hasGoal && (
        <div className="goal-progress">
          <p className="hint">
            {t("study.dailyGoalProgress", { studied: progress.studiedToday, limit: progress.dailyCardLimit })}
          </p>
          <div
            className="goal-progress-track"
            role="progressbar"
            aria-label={t("study.dailyGoalAria")}
            aria-valuemin={0}
            aria-valuemax={progress.dailyCardLimit ?? 0}
            aria-valuenow={Math.min(progress.studiedToday, progress.dailyCardLimit ?? 0)}
          >
            <div
              className="goal-progress-fill"
              style={{ width: `${Math.min(100, (progress.studiedToday * 100) / (progress.dailyCardLimit ?? 1))}%` }}
            />
          </div>
        </div>
      )}

      <section className="study-card">
        <p className="study-question">{currentCard.question}</p>

        {!revealed && hintText !== "" && (
          <div className="study-hint-area">
            <button className="link-button hint-toggle" onClick={toggleHint} aria-expanded={hintVisible}>
              <BulbIcon />
              {hintVisible ? t("study.hideHint") : t("study.showHint")}
              <kbd className="key-hint">H</kbd>
            </button>
            {hintVisible && (
              <>
                <p className={hintIsAutomatic ? "study-hint study-hint-auto" : "study-hint"}>{hintText}</p>
                {hintIsAutomatic && (
                  <p className="hint study-hint-caption">{t("study.autoHintCaption")}</p>
                )}
              </>
            )}
          </div>
        )}

        {!revealed && (
          <button autoFocus onClick={() => setRevealed(true)} aria-keyshortcuts="Space Enter">
            {t("study.showAnswer")}
            {/* Decorative: the button's accessible name stays the "show answer" label. */}
            <kbd className="key-hint key-hint-on-primary" aria-hidden="true">
              {t("study.spaceKey")}
            </kbd>
          </button>
        )}

        {revealed && (
          <>
            <p className="study-answer">{currentCard.answer}</p>
            {currentCard.extendedExample && (
              <div className="study-example">
                <p>{currentCard.extendedExample.text}</p>
                {currentCard.extendedExample.translation && (
                  <p className="hint">{currentCard.extendedExample.translation}</p>
                )}
              </div>
            )}
            {currentCard.audio && (
              <audio controls src={apiUrl(currentCard.audio.url)} className="study-audio">
                {t("study.audioUnsupported")}
              </audio>
            )}

            <p className="hint rating-prompt">{t("study.howWasIt")}</p>
            <div className="rating-buttons">
              {ratings.map((r) => (
                <button
                  key={r.value}
                  className={`rating-btn rating-${r.value}`}
                  disabled={submitting || (hintUsed && r.value === RATING_BLOCKED_AFTER_HINT)}
                  onClick={() => handleRate(r.value)}
                >
                  {r.label}
                </button>
              ))}
            </div>
            {hintUsed && (
              <p className="hint rating-note">{t("study.hintUsedNote")}</p>
            )}
          </>
        )}
      </section>

      <div className="study-archive">
        {archiveError && <p className="error" role="alert">{t("study.archiveError")}</p>}
        {!confirmingArchive ? (
          <button className="link-button" onClick={() => setConfirmingArchive(true)} disabled={submitting}>
            {t("study.dontWantToSee")}
          </button>
        ) : (
          <div className="study-archive-confirm">
            <span>{t("study.archiveConfirm")}</span>
            <button className="secondary" onClick={() => setConfirmingArchive(false)} disabled={submitting}>
              {t("study.cancel")}
            </button>
            <button className="danger" onClick={handleArchiveCurrent} disabled={submitting}>
              {t("study.archive")}
            </button>
          </div>
        )}
      </div>
    </main>
  );
}

import { useEffect, useState, type FormEvent } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { archiveDeck, getDeck, updateDeck } from "../../services/deckService";
import { archiveFlashcard, deleteArchivedFlashcard, restoreFlashcard } from "../../services/flashcardService";
import { useDebouncedValue } from "../../hooks/useDebouncedValue";
import { usePagedFlashcards } from "../../hooks/usePagedFlashcards";
import { useApiErrorMessage } from "../../i18n/useApiErrorMessage";
import { ArchivedActions } from "../../components/ArchivedActions";
import { ArchiveIcon, ChevronIcon, EyeIcon, EyeOffIcon, PencilIcon } from "../../components/Icons";
import { getDeckStats } from "../../services/statsService";
import { DECK_BADGE_STYLE, initialsFor } from "../../utils/deckBadge";
import type { Deck, DeckStats } from "../../types";
import { DeckGroupPicker } from "./DeckGroupPicker";
import { DeckStudyProfile } from "./DeckStudyProfile";

type LoadState = "loading" | "loaded" | "error";

export function DeckPage() {
  const { deckId } = useParams<{ deckId: string }>();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const apiErrorMessage = useApiErrorMessage();

  const [deck, setDeck] = useState<Deck | null>(null);
  const [loadState, setLoadState] = useState<LoadState>("loading");

  const [stats, setStats] = useState<DeckStats | null>(null);
  const [showSettings, setShowSettings] = useState(false);

  const [editing, setEditing] = useState(false);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // Question and answer of the cards in the list are hidden until the user
  // asks for them, so the next card to study is not spoiled by browsing the
  // list. This holds the ids of the cards currently shown; it starts empty.
  const [revealedIds, setRevealedIds] = useState<Set<string>>(new Set());

  const [showArchived, setShowArchived] = useState(false);
  const [archivedError, setArchivedError] = useState(false);

  // The search box updates immediately; the request only goes out once the
  // user pauses typing.
  const [search, setSearch] = useState("");
  const query = useDebouncedValue(search.trim(), 300);
  const activeCards = usePagedFlashcards({ deckId, query });
  const archivedCards = usePagedFlashcards({ deckId, archived: true, enabled: showArchived });

  useEffect(() => {
    refresh();
    // A different deck starts with everything hidden again.
    setRevealedIds(new Set());
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [deckId]);

  // The stats tiles are decoration on top of the deck: if they fail to
  // load, the rest of the page still works.
  function refreshStats() {
    if (!deckId) return;
    getDeckStats(deckId)
      .then(setStats)
      .catch(() => setStats(null));
  }

  function refresh() {
    if (!deckId) return;
    refreshStats();
    setLoadState("loading");
    getDeck(deckId)
      .then((d) => {
        setDeck(d);
        setName(d.name);
        setDescription(d.description);
        setLoadState("loaded");
      })
      .catch(() => setLoadState("error"));
  }

  // Refreshes the deck's card counters without blanking the page.
  function refreshDeckCounters() {
    if (!deckId) return;
    getDeck(deckId)
      .then(setDeck)
      .catch(() => undefined);
    refreshStats();
  }

  async function handleUpdate(e: FormEvent) {
    e.preventDefault();
    if (!deckId) return;
    setFormError(null);
    setSubmitting(true);
    try {
      const updated = await updateDeck(deckId, name, description);
      setDeck(updated);
      setEditing(false);
    } catch (err) {
      setFormError(apiErrorMessage(err, "deck.updateError"));
    } finally {
      setSubmitting(false);
    }
  }

  async function handleArchiveDeck() {
    if (!deckId) return;
    if (!confirm(t("deck.archiveDeckConfirm"))) return;
    await archiveDeck(deckId);
    navigate("/dashboard");
  }

  async function handleArchiveCard(id: string) {
    if (!confirm(t("deck.archiveCardConfirm"))) return;
    await archiveFlashcard(id);
    activeCards.removeLocally(id);
    if (showArchived) archivedCards.reload();
    refreshDeckCounters();
  }

  function toggleRevealed(id: string) {
    setRevealedIds((current) => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  // "Mostrar todos" reveals every card currently in the list; once they are
  // all shown the same button hides them all again.
  function toggleAllRevealed() {
    setRevealedIds((current) => {
      const everyone = activeCards.items.every((c) => current.has(c.id));
      return everyone ? new Set() : new Set(activeCards.items.map((c) => c.id));
    });
  }

  function toggleArchived() {
    setArchivedError(false);
    setShowArchived((v) => !v);
  }

  async function handleDeleteArchivedCard(id: string) {
    setArchivedError(false);
    try {
      await deleteArchivedFlashcard(id);
      archivedCards.removeLocally(id);
    } catch {
      setArchivedError(true);
    }
  }

  async function handleRestoreCard(id: string) {
    setArchivedError(false);
    try {
      await restoreFlashcard(id);
      archivedCards.removeLocally(id);
      activeCards.reload();
      refreshDeckCounters();
    } catch {
      setArchivedError(true);
    }
  }

  const allRevealed = activeCards.items.length > 0 && activeCards.items.every((c) => revealedIds.has(c.id));

  if (loadState === "loading") {
    return (
      <main className="app-shell">
        <p className="loading">{t("deck.loading")}</p>
      </main>
    );
  }

  if (loadState === "error" || !deck) {
    return (
      <main className="app-shell">
        <p className="error">{t("deck.loadError")}</p>
        <Link to="/dashboard">{t("deck.backToDashboard")}</Link>
      </main>
    );
  }

  return (
    <main className="app-shell">
      <Link to="/dashboard" className="back-link">
        &larr; {t("deck.backToDashboard")}
      </Link>

      {!editing && (
        <>
          <div className="deck-header">
            <span className="deck-badge deck-badge-large" style={DECK_BADGE_STYLE}>
              {initialsFor(deck.name)}
            </span>
            <h1>{deck.name}</h1>
            {deck.description && <p className="deck-description">{deck.description}</p>}
            <p className="hint">{t("dashboard.cardsCount", { count: deck.totalCards })}</p>
          </div>

          <div className="deck-mastery">
            <div
              className="deck-mastery-track"
              role="progressbar"
              aria-label={t("deck.masteryAria")}
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={stats?.masteryPercent ?? 0}
            >
              <div className="deck-mastery-fill" style={{ width: `${stats?.masteryPercent ?? 0}%` }} />
            </div>
            <p className="hint">{t("deck.masteryPercent", { percent: stats?.masteryPercent ?? 0 })}</p>
          </div>

          <div className="deck-tiles">
            <div className="deck-tile">
              <span className="deck-tile-label">{t("deck.tileToReview")}</span>
              <span className="deck-tile-value">{stats?.learningCards ?? 0}</span>
            </div>
            <div className="deck-tile">
              <span className="deck-tile-label">{t("deck.tileMastered")}</span>
              <span className="deck-tile-value">{stats?.masteredCards ?? 0}</span>
            </div>
            <div className="deck-tile">
              <span className="deck-tile-label">{t("deck.tileNew")}</span>
              <span className="deck-tile-value">{stats?.newCards ?? 0}</span>
            </div>
          </div>

          <div className="deck-actions">
            <Link to={`/decks/${deck.id}/study`} className="deck-action-primary">
              <button
                className="primary-large"
                disabled={deck.totalCards === 0}
                title={deck.totalCards === 0 ? t("deck.createCardFirst") : undefined}
              >
                {t("deck.studyNow")}
              </button>
            </Link>
            <Link to={`/decks/${deck.id}/stats`}>
              <button className="secondary">{t("deck.stats")}</button>
            </Link>
            <button
              className="secondary settings-row"
              onClick={() => setShowSettings((v) => !v)}
              aria-expanded={showSettings}
            >
              <span>{t("deck.settingsRow")}</span>
              <span className={showSettings ? "settings-chevron open" : "settings-chevron"}>
                <ChevronIcon />
              </span>
            </button>
            {showSettings && (
              <>
                <DeckStudyProfile deck={deck} onChange={(profileId) => setDeck({ ...deck, studyProfileId: profileId })} />
                <DeckGroupPicker deck={deck} onChange={(groupId) => setDeck({ ...deck, groupId })} />
                <div className="deck-actions-minor">
                  <button className="secondary action-edit" onClick={() => setEditing(true)}>
                    <PencilIcon />
                    {t("deck.editDeck")}
                  </button>
                  <button className="secondary action-archive" onClick={handleArchiveDeck}>
                    <ArchiveIcon />
                    {t("deck.archiveDeck")}
                  </button>
                </div>
              </>
            )}
          </div>
        </>
      )}

      {editing && (
        <form onSubmit={handleUpdate} className="form">
          <label htmlFor="deck-name">{t("deck.nameLabel")}</label>
          <input
            id="deck-name"
            required
            maxLength={100}
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={submitting}
          />
          <label htmlFor="deck-description">{t("deck.descriptionLabel")}</label>
          <textarea
            id="deck-description"
            maxLength={1000}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            disabled={submitting}
          />
          {formError && <p className="error" role="alert">{formError}</p>}
          <div className="button-row">
            <button type="submit" disabled={submitting || name.trim() === ""}>
              {submitting ? t("common.saving") : t("common.save")}
            </button>
            <button type="button" className="secondary" onClick={() => setEditing(false)}>
              {t("deck.cancel")}
            </button>
          </div>
        </form>
      )}

      <div className="dashboard-toolbar">
        <h2>{t("deck.flashcardsHeading")}</h2>
        <Link to={`/decks/${deck.id}/flashcards/new`}>
          <button>{t("deck.createFlashcard")}</button>
        </Link>
      </div>

      {deck.totalCards > 0 && (
        <input
          type="search"
          className="search-input"
          aria-label={t("deck.searchAria")}
          placeholder={t("deck.searchPlaceholder")}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          maxLength={200}
        />
      )}

      {activeCards.status === "loading" && activeCards.items.length === 0 && (
        <p className="loading">{t("deck.loadingCards")}</p>
      )}
      {activeCards.status === "error" && <p className="error" role="alert">{t("deck.loadCardsError")}</p>}

      {activeCards.status === "loaded" && activeCards.total === 0 && (
        <p className="empty-state">
          {query ? t("deck.noResultsFor", { query }) : t("deck.emptyCards")}
        </p>
      )}

      {activeCards.items.length > 0 && (
        <>
          <div className="list-header">
            <p className="hint list-summary">
              {query
                ? t("deck.resultsFor", { count: activeCards.total, query })
                : t("deck.showingOf", { shown: activeCards.items.length, total: activeCards.total })}
            </p>
            <button className="link-button reveal-all" onClick={toggleAllRevealed} aria-pressed={allRevealed}>
              {allRevealed ? <EyeOffIcon /> : <EyeIcon />}
              {allRevealed ? t("deck.hideAll") : t("deck.showAll")}
            </button>
          </div>
          <ul className="flashcard-list" aria-busy={activeCards.status === "loading"}>
            {activeCards.items.map((card, index) => {
              const revealed = revealedIds.has(card.id);
              return (
                <li key={card.id} className="flashcard-row" data-revealed={revealed}>
                  <div className={revealed ? "flashcard-text" : "flashcard-text is-hidden"}>
                    <strong aria-hidden={!revealed}>{card.question}</strong>
                    <p className="hint" aria-hidden={!revealed}>
                      {card.answer}
                    </p>
                    {!revealed && <span className="sr-only">{t("deck.hiddenContent")}</span>}
                  </div>
                  <div className="button-row">
                    <button
                      className="secondary icon-button reveal-toggle"
                      onClick={() => toggleRevealed(card.id)}
                      aria-pressed={revealed}
                      aria-label={revealed ? t("deck.hideCardAria", { index: index + 1 }) : t("deck.showCardAria", { index: index + 1 })}
                      title={revealed ? t("deck.hideContentTitle") : t("deck.showContentTitle")}
                    >
                      {revealed ? <EyeOffIcon /> : <EyeIcon />}
                    </button>
                    <Link to={`/flashcards/${card.id}/edit`}>
                      <button className="secondary action-edit">
                        <PencilIcon />
                        {t("deck.editCard")}
                      </button>
                    </Link>
                    <button className="secondary action-archive" onClick={() => handleArchiveCard(card.id)}>
                      <ArchiveIcon />
                      {t("deck.archiveCard")}
                    </button>
                  </div>
                </li>
              );
            })}
          </ul>
          {activeCards.items.length < activeCards.total && (
            <div className="load-more">
              <button className="secondary" onClick={activeCards.loadMore} disabled={activeCards.loadingMore}>
                {activeCards.loadingMore
                  ? t("deck.loadingMore")
                  : t("deck.loadMore", { count: activeCards.total - activeCards.items.length })}
              </button>
              {activeCards.loadMoreFailed && (
                <p className="error" role="alert">{t("deck.loadMoreError")}</p>
              )}
            </div>
          )}
        </>
      )}

      <section className="archived-section">
        <button className="link-button archived-toggle" onClick={toggleArchived} aria-expanded={showArchived}>
          {showArchived ? t("deck.hideArchivedCards") : t("deck.showArchivedCards")}
        </button>
        {showArchived && (
          <>
            {(archivedError || archivedCards.status === "error") && (
              <p className="error" role="alert">{t("deck.archivedCardsError")}</p>
            )}
            {archivedCards.status === "loaded" && archivedCards.total === 0 && (
              <p className="empty-state">{t("deck.archivedCardsEmpty")}</p>
            )}
            <ul className="archived-list">
              {archivedCards.items.map((card) => (
                <li key={card.id} className="archived-row">
                  <div>
                    <strong>{card.question}</strong>
                    <p className="hint">{card.answer}</p>
                  </div>
                  <ArchivedActions
                    restoreLabel={t("deck.restoreCard")}
                    deleteLabel={t("deck.deleteCard")}
                    confirmMessage={t("deck.deleteCardConfirm")}
                    onRestore={() => handleRestoreCard(card.id)}
                    onDelete={() => handleDeleteArchivedCard(card.id)}
                  />
                </li>
              ))}
            </ul>
            {archivedCards.items.length < archivedCards.total && (
              <div className="load-more">
                <button className="secondary" onClick={archivedCards.loadMore} disabled={archivedCards.loadingMore}>
                  {archivedCards.loadingMore
                    ? t("deck.loadingMore")
                    : t("deck.loadMore", { count: archivedCards.total - archivedCards.items.length })}
                </button>
              </div>
            )}
          </>
        )}
      </section>
    </main>
  );
}

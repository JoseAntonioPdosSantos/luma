import { useEffect, useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useAuth } from "../../hooks/useAuth";
import { createDeck, deleteArchivedDeck, listArchivedDecks, listDecks, restoreDeck, setDeckGroup } from "../../services/deckService";
import { createDeckGroup, listDeckGroups } from "../../services/deckGroupService";
import { ArchivedActions } from "../../components/ArchivedActions";
import { FolderIcon, GearIcon } from "../../components/Icons";
import { DECK_BADGE_STYLE, GROUP_BADGE_STYLE, initialsFor } from "../../utils/deckBadge";
import { useApiErrorMessage } from "../../i18n/useApiErrorMessage";
import { deleteArchivedFlashcard, listAllArchivedFlashcards, restoreFlashcard } from "../../services/flashcardService";
import type { ArchivedFlashcard, Deck, DeckGroup } from "../../types";

type LoadState = "loading" | "loaded" | "error";
// "view": nothing open. "createDeck"/"createGroup": that form is open.
type Mode = "view" | "createDeck" | "createGroup";

export function DashboardPage() {
  const { user, logout } = useAuth();
  const { t } = useTranslation();
  const apiErrorMessage = useApiErrorMessage();
  const [decks, setDecks] = useState<Deck[]>([]);
  const [groups, setGroups] = useState<DeckGroup[]>([]);
  const [loadState, setLoadState] = useState<LoadState>("loading");
  const [mode, setMode] = useState<Mode>("view");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [groupId, setGroupId] = useState("");
  const [formError, setFormError] = useState<string | null>(null);
  const [groupName, setGroupName] = useState("");
  const [groupFormError, setGroupFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [showArchived, setShowArchived] = useState(false);
  const [archivedDecks, setArchivedDecks] = useState<Deck[]>([]);
  const [archivedCards, setArchivedCards] = useState<ArchivedFlashcard[]>([]);
  const [archivedError, setArchivedError] = useState(false);

  useEffect(() => {
    refreshAll();
  }, []);

  function refreshAll() {
    setLoadState("loading");
    Promise.all([listDecks(), listDeckGroups()])
      .then(([d, g]) => {
        setDecks(d);
        setGroups(g);
        setLoadState("loaded");
      })
      .catch(() => setLoadState("error"));
  }

  function loadArchived() {
    setArchivedError(false);
    Promise.all([listArchivedDecks(), listAllArchivedFlashcards()])
      .then(([decks, cards]) => {
        setArchivedDecks(decks);
        setArchivedCards(cards);
      })
      .catch(() => setArchivedError(true));
  }

  function toggleArchived() {
    const next = !showArchived;
    setShowArchived(next);
    if (next) loadArchived();
  }

  async function handleRestore(id: string) {
    setArchivedError(false);
    try {
      await restoreDeck(id);
      setArchivedDecks((list) => list.filter((d) => d.id !== id));
      refreshAll();
    } catch {
      setArchivedError(true);
    }
  }

  async function handleDeleteDeck(id: string) {
    setArchivedError(false);
    try {
      await deleteArchivedDeck(id);
      setArchivedDecks((list) => list.filter((d) => d.id !== id));
      // The deck's cards went with it.
      loadArchived();
    } catch {
      setArchivedError(true);
    }
  }

  async function handleDeleteCard(id: string) {
    setArchivedError(false);
    try {
      await deleteArchivedFlashcard(id);
      setArchivedCards((list) => list.filter((c) => c.id !== id));
    } catch {
      setArchivedError(true);
    }
  }

  async function handleRestoreCard(id: string) {
    setArchivedError(false);
    try {
      await restoreFlashcard(id);
      setArchivedCards((list) => list.filter((c) => c.id !== id));
      refreshAll();
    } catch {
      setArchivedError(true);
    }
  }

  function startCreateDeck() {
    setFormError(null);
    setName("");
    setDescription("");
    setGroupId("");
    setMode("createDeck");
  }

  function startCreateGroup() {
    setGroupFormError(null);
    setGroupName("");
    setMode("createGroup");
  }

  async function handleCreate(e: FormEvent) {
    e.preventDefault();
    setFormError(null);
    setSubmitting(true);
    try {
      const created = await createDeck(name, description);
      if (groupId) await setDeckGroup(created.id, groupId);
      setMode("view");
      refreshAll();
    } catch (err) {
      setFormError(apiErrorMessage(err, "dashboard.createError"));
    } finally {
      setSubmitting(false);
    }
  }

  async function handleCreateGroup(e: FormEvent) {
    e.preventDefault();
    setGroupFormError(null);
    setSubmitting(true);
    try {
      await createDeckGroup(groupName);
      setMode("view");
      refreshAll();
    } catch (err) {
      setGroupFormError(apiErrorMessage(err, "dashboard.createGroupError"));
    } finally {
      setSubmitting(false);
    }
  }

  const decksByGroup = new Map<string, Deck[]>();
  for (const g of groups) decksByGroup.set(g.id, []);
  const ungroupedDecks: Deck[] = [];
  for (const d of decks) {
    const bucket = d.groupId ? decksByGroup.get(d.groupId) : undefined;
    if (bucket) bucket.push(d);
    else ungroupedDecks.push(d);
  }

  return (
    <main className="app-shell">
      <div className="topbar">
        <button className="secondary" onClick={() => logout()}>
          {t("dashboard.logout")}
        </button>
        <Link to="/settings">
          <button className="secondary icon-button" aria-label={t("dashboard.settingsLabel")} title={t("dashboard.settingsLabel")}>
            <GearIcon />
          </button>
        </Link>
      </div>

      <header className="dashboard-header">
        <div className="dashboard-header-content">
          <img src="/logo-light-compact.png" alt="Luma" className="brand-logo brand-logo-light brand-logo-compact" />
          <img src="/logo-dark-compact.png" alt="Luma" className="brand-logo brand-logo-dark brand-logo-compact" />
          <p className="hint">{user?.email}</p>
        </div>
      </header>

      <div className="dashboard-toolbar">
        <h2>{t("dashboard.myDecks")}</h2>
        {mode === "view" ? (
          <div className="button-row">
            <button className="secondary" onClick={startCreateGroup}>
              {t("dashboard.createGroup")}
            </button>
            <button onClick={startCreateDeck}>{t("dashboard.createDeck")}</button>
          </div>
        ) : (
          <button className="secondary" onClick={() => setMode("view")}>
            {t("dashboard.cancel")}
          </button>
        )}
      </div>

      {mode === "createGroup" && (
        <form onSubmit={handleCreateGroup} className="form">
          <label htmlFor="group-name">{t("dashboard.groupNameLabel")}</label>
          <input
            id="group-name"
            required
            maxLength={60}
            value={groupName}
            onChange={(e) => setGroupName(e.target.value)}
            disabled={submitting}
            placeholder={t("dashboard.groupNamePlaceholder")}
          />
          {groupFormError && <p className="error" role="alert">{groupFormError}</p>}
          <button type="submit" disabled={submitting || groupName.trim() === ""}>
            {submitting ? t("dashboard.creating") : t("dashboard.saveGroup")}
          </button>
        </form>
      )}

      {mode === "createDeck" && (
        <form onSubmit={handleCreate} className="form">
          <label htmlFor="deck-name">{t("dashboard.nameLabel")}</label>
          <input
            id="deck-name"
            required
            maxLength={100}
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={submitting}
            placeholder={t("dashboard.namePlaceholder")}
          />
          <label htmlFor="deck-description">{t("dashboard.descriptionLabel")}</label>
          <textarea
            id="deck-description"
            maxLength={1000}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            disabled={submitting}
          />
          {groups.length > 0 && (
            <>
              <label htmlFor="deck-group">{t("dashboard.deckGroupLabel")}</label>
              <select id="deck-group" value={groupId} onChange={(e) => setGroupId(e.target.value)} disabled={submitting} className="profile-select">
                <option value="">{t("dashboard.noGroup")}</option>
                {groups.map((g) => (
                  <option key={g.id} value={g.id}>
                    {g.name}
                  </option>
                ))}
              </select>
            </>
          )}
          {formError && <p className="error" role="alert">{formError}</p>}
          <button type="submit" disabled={submitting || name.trim() === ""}>
            {submitting ? t("dashboard.creating") : t("dashboard.saveDeck")}
          </button>
        </form>
      )}

      {loadState === "loading" && <p className="loading">{t("dashboard.loadingDecks")}</p>}
      {loadState === "error" && <p className="error">{t("dashboard.loadError")}</p>}

      {loadState === "loaded" && decks.length === 0 && groups.length === 0 && (
        <p className="empty-state">{t("dashboard.emptyState")}</p>
      )}

      {loadState === "loaded" && (decks.length > 0 || groups.length > 0) && (
        <ul className="deck-list">
          {groups.map((g) => {
            const groupDecks = decksByGroup.get(g.id) ?? [];
            return (
              <li key={g.id} className="deck-card">
                <Link to={`/groups/${g.id}`} className="deck-card-link">
                  <span className="deck-badge" style={GROUP_BADGE_STYLE}>
                    <FolderIcon />
                  </span>
                  <span className="deck-card-body">
                    <strong>{g.name}</strong>
                    <p className="hint">{t("dashboard.groupStats", { count: groupDecks.length })}</p>
                  </span>
                </Link>
              </li>
            );
          })}
          {ungroupedDecks.map((deck) => (
            <li key={deck.id} className="deck-card">
              <Link to={`/decks/${deck.id}`} className="deck-card-link">
                <span className="deck-badge" style={DECK_BADGE_STYLE}>
                  {initialsFor(deck.name)}
                </span>
                <span className="deck-card-body">
                  <strong>{deck.name}</strong>
                  {deck.description && <p>{deck.description}</p>}
                  <p className="hint">
                    {t("dashboard.cardsCount", { count: deck.totalCards })}
                    <span className={`deck-status-pill ${deck.dueCards > 0 ? "deck-status-pill--due" : "deck-status-pill--done"}`}>
                      {deck.dueCards > 0 ? t("dashboard.dueCount", { count: deck.dueCards }) : t("dashboard.upToDate")}
                    </span>
                  </p>
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}

      <section className="archived-section">
        <button className="link-button archived-toggle" onClick={toggleArchived} aria-expanded={showArchived}>
          {showArchived ? t("dashboard.hideArchived") : t("dashboard.showArchived")}
        </button>
        {showArchived && (
          <>
            {archivedError && <p className="error" role="alert">{t("dashboard.archivedError")}</p>}
            {!archivedError && archivedDecks.length === 0 && archivedCards.length === 0 && (
              <p className="empty-state">{t("dashboard.archivedEmpty")}</p>
            )}

            {archivedDecks.length > 0 && (
              <>
                <h3 className="archived-heading">{t("dashboard.archivedDecksHeading", { count: archivedDecks.length })}</h3>
                <ul className="archived-list">
                  {archivedDecks.map((deck) => (
                    <li key={deck.id} className="archived-row">
                      <div>
                        <strong>{deck.name}</strong>
                        <p className="hint">{t("dashboard.cardsCount", { count: deck.totalCards })}</p>
                      </div>
                      <ArchivedActions
                        restoreLabel={t("dashboard.restoreDeck")}
                        deleteLabel={t("dashboard.deleteDeck")}
                        confirmMessage={t("dashboard.deleteDeckConfirm")}
                        onRestore={() => handleRestore(deck.id)}
                        onDelete={() => handleDeleteDeck(deck.id)}
                      />
                    </li>
                  ))}
                </ul>
              </>
            )}

            {archivedCards.length > 0 && (
              <>
                <h3 className="archived-heading">{t("dashboard.archivedCardsHeading", { count: archivedCards.length })}</h3>
                <ul className="archived-list">
                  {archivedCards.map((card) => (
                    <li key={card.id} className="archived-row">
                      <div>
                        <strong>{card.question}</strong>
                        <p className="hint">{card.answer}</p>
                        <p className="hint">{t("dashboard.deckOf", { name: card.deckName })}</p>
                      </div>
                      <ArchivedActions
                        restoreLabel={t("dashboard.restoreCard")}
                        deleteLabel={t("dashboard.deleteCard")}
                        confirmMessage={t("dashboard.deleteCardConfirm")}
                        onRestore={() => handleRestoreCard(card.id)}
                        onDelete={() => handleDeleteCard(card.id)}
                      />
                    </li>
                  ))}
                </ul>
              </>
            )}
          </>
        )}
      </section>
    </main>
  );
}

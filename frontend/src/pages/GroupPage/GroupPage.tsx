import { useEffect, useState, type FormEvent } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { createDeck, listDecks, setDeckGroup } from "../../services/deckService";
import { deleteDeckGroup, listDeckGroups, renameDeckGroup } from "../../services/deckGroupService";
import { useApiErrorMessage } from "../../i18n/useApiErrorMessage";
import { DECK_BADGE_STYLE, initialsFor } from "../../utils/deckBadge";
import type { Deck, DeckGroup } from "../../types";

type LoadState = "loading" | "loaded" | "error";

export function GroupPage() {
  const { groupId } = useParams<{ groupId: string }>();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const apiErrorMessage = useApiErrorMessage();

  const [group, setGroup] = useState<DeckGroup | null>(null);
  const [decks, setDecks] = useState<Deck[]>([]);
  const [loadState, setLoadState] = useState<LoadState>("loading");

  const [editing, setEditing] = useState(false);
  const [name, setName] = useState("");
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const [showCreateForm, setShowCreateForm] = useState(false);
  const [deckName, setDeckName] = useState("");
  const [deckDescription, setDeckDescription] = useState("");
  const [createError, setCreateError] = useState<string | null>(null);

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [groupId]);

  function refresh() {
    if (!groupId) return;
    setLoadState("loading");
    Promise.all([listDeckGroups(), listDecks()])
      .then(([groups, allDecks]) => {
        const found = groups.find((g) => g.id === groupId);
        if (!found) {
          setLoadState("error");
          return;
        }
        setGroup(found);
        setName(found.name);
        setDecks(allDecks.filter((d) => d.groupId === groupId));
        setLoadState("loaded");
      })
      .catch(() => setLoadState("error"));
  }

  async function handleRename(e: FormEvent) {
    e.preventDefault();
    if (!groupId) return;
    setFormError(null);
    setSubmitting(true);
    try {
      const updated = await renameDeckGroup(groupId, name);
      setGroup(updated);
      setEditing(false);
    } catch (err) {
      setFormError(apiErrorMessage(err, "group.renameError"));
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete() {
    if (!groupId) return;
    setDeleteError(null);
    setSubmitting(true);
    try {
      await deleteDeckGroup(groupId);
      navigate("/dashboard");
    } catch (err) {
      setDeleteError(apiErrorMessage(err, "group.deleteError"));
      setSubmitting(false);
    }
  }

  async function handleCreateDeck(e: FormEvent) {
    e.preventDefault();
    if (!groupId) return;
    setCreateError(null);
    setSubmitting(true);
    try {
      const created = await createDeck(deckName, deckDescription);
      await setDeckGroup(created.id, groupId);
      setDeckName("");
      setDeckDescription("");
      setShowCreateForm(false);
      refresh();
    } catch (err) {
      setCreateError(apiErrorMessage(err, "dashboard.createError"));
    } finally {
      setSubmitting(false);
    }
  }

  if (loadState === "loading") {
    return (
      <main className="app-shell">
        <p className="loading">{t("group.loading")}</p>
      </main>
    );
  }

  if (loadState === "error" || !group) {
    return (
      <main className="app-shell">
        <p className="error">{t("group.loadError")}</p>
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
        <div className="deck-header">
          <h1>{group.name}</h1>
          <p className="hint">{t("dashboard.groupStats", { count: decks.length })}</p>
          <div className="button-row">
            <button className="secondary action-edit" onClick={() => setEditing(true)}>
              {t("group.rename")}
            </button>
            {!confirmingDelete ? (
              <button className="secondary action-archive" onClick={() => setConfirmingDelete(true)}>
                {t("group.delete")}
              </button>
            ) : (
              <div className="archived-confirm" role="group" aria-label={t("group.deleteConfirmAria")}>
                <p className="archived-confirm-message">{t("group.deleteConfirmMessage", { name: group.name })}</p>
                <div className="archived-actions-buttons">
                  <button className="secondary" onClick={() => setConfirmingDelete(false)} disabled={submitting}>
                    {t("common.cancel")}
                  </button>
                  <button className="danger" onClick={handleDelete} disabled={submitting}>
                    {t("group.deletePermanently")}
                  </button>
                </div>
              </div>
            )}
          </div>
          {deleteError && <p className="error" role="alert">{deleteError}</p>}
        </div>
      )}

      {editing && (
        <form onSubmit={handleRename} className="form">
          <label htmlFor="group-name">{t("dashboard.groupNameLabel")}</label>
          <input
            id="group-name"
            required
            maxLength={60}
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={submitting}
          />
          {formError && <p className="error" role="alert">{formError}</p>}
          <div className="button-row">
            <button type="submit" disabled={submitting || name.trim() === ""}>
              {submitting ? t("common.saving") : t("common.save")}
            </button>
            <button type="button" className="secondary" onClick={() => setEditing(false)}>
              {t("common.cancel")}
            </button>
          </div>
        </form>
      )}

      <div className="dashboard-toolbar">
        <h2>{t("group.decksHeading")}</h2>
        <button onClick={() => setShowCreateForm((v) => !v)}>
          {showCreateForm ? t("dashboard.cancel") : t("dashboard.createDeck")}
        </button>
      </div>

      {showCreateForm && (
        <form onSubmit={handleCreateDeck} className="form">
          <label htmlFor="deck-name">{t("dashboard.nameLabel")}</label>
          <input
            id="deck-name"
            required
            maxLength={100}
            value={deckName}
            onChange={(e) => setDeckName(e.target.value)}
            disabled={submitting}
            placeholder={t("dashboard.namePlaceholder")}
          />
          <label htmlFor="deck-description">{t("dashboard.descriptionLabel")}</label>
          <textarea
            id="deck-description"
            maxLength={1000}
            value={deckDescription}
            onChange={(e) => setDeckDescription(e.target.value)}
            disabled={submitting}
          />
          {createError && <p className="error" role="alert">{createError}</p>}
          <button type="submit" disabled={submitting || deckName.trim() === ""}>
            {submitting ? t("dashboard.creating") : t("dashboard.saveDeck")}
          </button>
        </form>
      )}

      {decks.length === 0 ? (
        <p className="empty-state">{t("group.emptyState")}</p>
      ) : (
        <ul className="deck-list">
          {decks.map((deck) => (
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
    </main>
  );
}

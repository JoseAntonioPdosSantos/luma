import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useApiErrorMessage } from "../../i18n/useApiErrorMessage";
import { setDeckGroup } from "../../services/deckService";
import { listDeckGroups } from "../../services/deckGroupService";
import type { Deck, DeckGroup } from "../../types";

interface Props {
  deck: Deck;
  // Called after the choice was saved; undefined means "no group".
  onChange: (groupId: string | undefined) => void;
}

// Lets one deck be filed under a group (or removed from its group), from
// the deck's own settings — an alternative to doing it from the dashboard
// or from the group's own page.
export function DeckGroupPicker({ deck, onChange }: Props) {
  const { t } = useTranslation();
  const apiErrorMessage = useApiErrorMessage();
  const [groups, setGroups] = useState<DeckGroup[] | null>(null);
  const [failed, setFailed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listDeckGroups()
      .then((list) => {
        if (!Array.isArray(list)) throw new Error("unexpected response");
        setGroups(list);
      })
      .catch(() => setFailed(true));
  }, []);

  if (failed) {
    return <p className="error">{t("deckGroupPicker.loadError")}</p>;
  }
  if (!groups) {
    return <p className="loading">{t("deckGroupPicker.loading")}</p>;
  }
  // No groups exist yet: nothing to pick from, and creating one lives on
  // the dashboard, so this section stays out of the way.
  if (groups.length === 0 && !deck.groupId) {
    return null;
  }

  async function handleChange(value: string) {
    setError(null);
    setBusy(true);
    try {
      await setDeckGroup(deck.id, value === "" ? null : value);
      onChange(value === "" ? undefined : value);
    } catch (err) {
      setError(apiErrorMessage(err, "deckGroupPicker.changeError"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="settings-section deck-study-profile">
      <h2>{t("deckGroupPicker.heading")}</h2>
      <label htmlFor="deck-group-select" className="settings-label">
        {t("deckGroupPicker.selectLabel")}
      </label>
      <select
        id="deck-group-select"
        className="profile-select"
        value={deck.groupId ?? ""}
        onChange={(e) => handleChange(e.target.value)}
        disabled={busy}
      >
        <option value="">{t("deckGroupPicker.noGroup")}</option>
        {groups.map((g) => (
          <option key={g.id} value={g.id}>
            {g.name}
          </option>
        ))}
      </select>

      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
    </section>
  );
}

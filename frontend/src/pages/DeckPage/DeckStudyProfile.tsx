import { useEffect, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useApiErrorMessage } from "../../i18n/useApiErrorMessage";
import { setDeckStudyProfile } from "../../services/deckService";
import { listStudyProfiles } from "../../services/studyProfileService";
import type { Deck, StudyProfiles } from "../../types";
import { ProfileSummary } from "../SettingsPage/ProfileSummary";

interface Props {
  deck: Deck;
  // Called after the choice was saved; undefined means "follow the general configuration".
  onChange: (profileId: string | undefined) => void;
}

// Lets one deck use a study configuration of its own instead of the general
// one. The deck's own configuration wins for its cards; with none chosen the
// general (active) configuration applies.
export function DeckStudyProfile({ deck, onChange }: Props) {
  // So the settings page can bring the user back to this deck.
  const here = useLocation().pathname;
  const { t } = useTranslation();
  const apiErrorMessage = useApiErrorMessage();
  const [data, setData] = useState<StudyProfiles | null>(null);
  const [failed, setFailed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listStudyProfiles()
      .then((profiles) => {
        if (!Array.isArray(profiles?.profiles) || profiles.profiles.length === 0) throw new Error("unexpected response");
        setData(profiles);
      })
      .catch(() => setFailed(true));
  }, []);

  if (failed) {
    return <p className="error">{t("deckStudyProfile.loadError")}</p>;
  }
  if (!data) {
    return <p className="loading">{t("deckStudyProfile.loading")}</p>;
  }

  const general = data.profiles.find((p) => p.id === data.activeProfileId) ?? data.profiles[0];
  // A choice that points at a configuration that no longer exists counts as none.
  const own = deck.studyProfileId ? data.profiles.find((p) => p.id === deck.studyProfileId) : undefined;
  const effective = own ?? general;

  async function handleChange(value: string) {
    setError(null);
    setBusy(true);
    try {
      await setDeckStudyProfile(deck.id, value === "" ? null : value);
      onChange(value === "" ? undefined : value);
    } catch (err) {
      setError(apiErrorMessage(err, "deckStudyProfile.changeError"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="settings-section deck-study-profile">
      <h2>{t("deckStudyProfile.heading")}</h2>
      <label htmlFor="deck-profile-select" className="settings-label">
        {t("deckStudyProfile.selectLabel")}
      </label>
      <select
        id="deck-profile-select"
        className="profile-select"
        value={own ? own.id : ""}
        onChange={(e) => handleChange(e.target.value)}
        disabled={busy}
      >
        <option value="">{t("deckStudyProfile.useGeneral", { name: general.name })}</option>
        {data.profiles.map((p) => (
          <option key={p.id} value={p.id}>
            {p.name}
          </option>
        ))}
      </select>
      <p className="hint">
        {own
          ? t("deckStudyProfile.usingOwn", { name: own.name })
          : t("deckStudyProfile.usingGeneral", { name: general.name })}
      </p>

      <ProfileSummary profile={effective} />

      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      <Link to="/settings" state={{ from: here }} className="hint">
        {t("deckStudyProfile.manageLink")}
      </Link>
    </section>
  );
}

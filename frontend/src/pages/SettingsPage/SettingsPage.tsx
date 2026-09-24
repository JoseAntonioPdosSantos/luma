import { useCallback, useEffect, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ApiError } from "../../services/api";
import {
  createStudyProfile,
  deleteStudyProfile,
  listStudyProfiles,
  setActiveStudyProfile,
  updateStudyProfile,
} from "../../services/studyProfileService";
import { setLanguage as saveLanguage } from "../../services/sessionService";
import { getStudyProgress } from "../../services/studyService";
import { SUPPORTED_LANGUAGES, type SupportedLanguage } from "../../i18n";
import { useApiErrorMessage } from "../../i18n/useApiErrorMessage";
import type { StudyProfile, StudyProfiles, StudyProgress } from "../../types";
import { backTarget } from "../../utils/returnTo";
import { emptyForm, formFromProfile, validateForm, type ProfileForm } from "../../utils/studyRules";
import { ProfileFormFields } from "./ProfileFormFields";
import { ProfileSummary } from "./ProfileSummary";

type LoadState = "loading" | "loaded" | "error";
type Mode = { kind: "view" } | { kind: "create" } | { kind: "edit"; id: string };

export function SettingsPage() {
  const { t, i18n } = useTranslation();
  const apiErrorMessage = useApiErrorMessage();
  // "Voltar" returns to where the user opened the settings from (a deck, for
  // instance), not always to the dashboard.
  const back = backTarget(useLocation().state);
  const [loadState, setLoadState] = useState<LoadState>("loading");
  const [data, setData] = useState<StudyProfiles | null>(null);
  const [progress, setProgress] = useState<StudyProgress | null>(null);
  const [mode, setMode] = useState<Mode>({ kind: "view" });
  const [form, setForm] = useState<ProfileForm>(emptyForm());
  const [showErrors, setShowErrors] = useState(false);
  const [serverErrors, setServerErrors] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [languageSaving, setLanguageSaving] = useState(false);
  const [languageError, setLanguageError] = useState<string | null>(null);

  const refreshProgress = useCallback(() => {
    getStudyProgress()
      .then(setProgress)
      .catch(() => undefined);
  }, []);

  useEffect(() => {
    listStudyProfiles()
      .then((profiles) => {
        setData(profiles);
        setLoadState("loaded");
      })
      .catch(() => setLoadState("error"));
    refreshProgress();
  }, [refreshProgress]);

  if (loadState === "loading") {
    return (
      <main className="app-shell">
        <p className="loading">{t("settings.loading")}</p>
      </main>
    );
  }

  if (loadState === "error" || !data) {
    return (
      <main className="app-shell">
        <p className="error">{t("settings.loadError")}</p>
        <Link to={back.path}>{back.label}</Link>
      </main>
    );
  }

  const defaultProfile = data.profiles.find((p) => p.isDefault) ?? data.profiles[0];
  const active: StudyProfile = data.profiles.find((p) => p.id === data.activeProfileId) ?? defaultProfile;
  const { errors: clientErrors, input } = validateForm(form);
  const errors = { ...clientErrors, ...serverErrors };

  function resetMessages() {
    setNotice(null);
    setError(null);
  }

  async function reload() {
    const fresh = await listStudyProfiles();
    setData(fresh);
    refreshProgress();
    return fresh;
  }

  async function handleSelect(id: string) {
    if (id === data?.activeProfileId) return;
    resetMessages();
    setBusy(true);
    setMode({ kind: "view" });
    setConfirmingDelete(false);
    try {
      await setActiveStudyProfile(id);
      const fresh = await reload();
      const chosen = fresh.profiles.find((p) => p.id === id);
      setNotice(t("settings.switchNotice", { name: chosen?.name ?? "" }));
    } catch (err) {
      setError(apiErrorMessage(err, "settings.switchError"));
    } finally {
      setBusy(false);
    }
  }

  function startCreate() {
    resetMessages();
    setConfirmingDelete(false);
    setForm(emptyForm(defaultProfile.rules));
    setShowErrors(false);
    setServerErrors({});
    setMode({ kind: "create" });
  }

  function startEdit() {
    resetMessages();
    setConfirmingDelete(false);
    setForm(formFromProfile(active));
    setShowErrors(false);
    setServerErrors({});
    setMode({ kind: "edit", id: active.id });
  }

  function cancelForm() {
    setMode({ kind: "view" });
    setShowErrors(false);
    setServerErrors({});
  }

  async function handleSubmit() {
    setShowErrors(true);
    if (!input || mode.kind === "view") return;
    resetMessages();
    setServerErrors({});
    setBusy(true);
    try {
      if (mode.kind === "create") {
        const created = await createStudyProfile(input);
        await reload();
        setNotice(t("settings.createdNotice", { name: created.name }));
      } else {
        await updateStudyProfile(mode.id, input);
        await reload();
        setNotice(t("settings.savedNotice"));
      }
      setMode({ kind: "view" });
      setShowErrors(false);
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        setServerErrors({ name: t("settings.duplicateNameError") });
      } else {
        setError(apiErrorMessage(err, "settings.saveError"));
      }
    } finally {
      setBusy(false);
    }
  }

  async function handleDelete() {
    resetMessages();
    setBusy(true);
    try {
      await deleteStudyProfile(active.id);
      await reload();
      setConfirmingDelete(false);
      setNotice(t("settings.deletedNotice", { deleted: active.name, fallback: defaultProfile.name }));
    } catch (err) {
      setError(apiErrorMessage(err, "settings.deleteError"));
    } finally {
      setBusy(false);
    }
  }

  async function handleLanguageChange(language: SupportedLanguage) {
    setLanguageError(null);
    setLanguageSaving(true);
    const previous = i18n.resolvedLanguage ?? i18n.language;
    try {
      // Switch immediately so the page feels responsive, and roll back if
      // the save fails.
      await i18n.changeLanguage(language);
      await saveLanguage(language);
    } catch (err) {
      await i18n.changeLanguage(previous);
      setLanguageError(apiErrorMessage(err, "settings.languageSaveError"));
    } finally {
      setLanguageSaving(false);
    }
  }

  return (
    <main className="app-shell">
      <Link to={back.path} className="back-link">
        &larr; {back.label}
      </Link>
      <h1>{t("settings.title")}</h1>

      <section className="settings-section">
        <h2>{t("settings.languageHeading")}</h2>
        <p className="hint">{t("settings.languageHint")}</p>
        <label htmlFor="language-select" className="settings-label">
          {t("language.label")}
        </label>
        <select
          id="language-select"
          className="profile-select"
          value={i18n.resolvedLanguage ?? i18n.language}
          onChange={(e) => handleLanguageChange(e.target.value as SupportedLanguage)}
          disabled={languageSaving}
        >
          {SUPPORTED_LANGUAGES.map((lang) => (
            <option key={lang} value={lang}>
              {t(`language.${lang}`)}
            </option>
          ))}
        </select>
        {languageError && (
          <p className="error" role="alert">
            {languageError}
          </p>
        )}
      </section>

      <section className="settings-section">
        <h2>{t("settings.studyConfigHeading")}</h2>
        <p className="hint">{t("settings.studyConfigHint")}</p>

        <label htmlFor="profile-select" className="settings-label">
          {t("settings.activeConfigLabel")}
        </label>
        <select
          id="profile-select"
          className="profile-select"
          value={active.id}
          onChange={(e) => handleSelect(e.target.value)}
          disabled={busy}
        >
          {data.profiles.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </select>
        <p className="hint">{t("settings.switchHint")}</p>
        <p className="hint">{t("settings.perDeckHint")}</p>

        <ProfileSummary profile={active} />

        {progress && (
          <p className="hint">
            {progress.dailyCardLimit !== null
              ? t("settings.progressWithGoal", { count: progress.studiedToday, limit: progress.dailyCardLimit })
              : t("settings.progressNoGoal", { count: progress.studiedToday })}
          </p>
        )}

        {mode.kind === "view" && (
          <div className="button-row">
            {active.isDefault ? (
              <p className="hint">{t("settings.editDisabledHint", { name: active.name })}</p>
            ) : (
              <>
                <button className="secondary action-edit" onClick={startEdit} disabled={busy}>
                  {t("settings.edit")}
                </button>
                {!confirmingDelete ? (
                  <button className="secondary action-archive" onClick={() => setConfirmingDelete(true)} disabled={busy}>
                    {t("settings.delete")}
                  </button>
                ) : (
                  <div className="archived-confirm" role="group" aria-label={t("settings.deleteConfirmAria")}>
                    <p className="archived-confirm-message">
                      {t("settings.deleteConfirmMessage", { active: active.name, fallback: defaultProfile.name })}
                    </p>
                    <div className="archived-actions-buttons">
                      <button className="secondary" onClick={() => setConfirmingDelete(false)} disabled={busy}>
                        {t("settings.cancel")}
                      </button>
                      <button className="danger" onClick={handleDelete} disabled={busy}>
                        {t("settings.deletePermanently")}
                      </button>
                    </div>
                  </div>
                )}
              </>
            )}
            <button onClick={startCreate} disabled={busy}>
              {t("settings.newConfig")}
            </button>
          </div>
        )}

        {notice && (
          <p className="success" role="status">
            {notice}
          </p>
        )}
        {error && (
          <p className="error" role="alert">
            {error}
          </p>
        )}
      </section>

      {mode.kind !== "view" && (
        <ProfileFormFields
          title={mode.kind === "create" ? t("settings.newConfigTitle") : t("settings.editConfigTitle", { name: active.name })}
          form={form}
          onChange={(next) => {
            setForm(next);
            setServerErrors({});
          }}
          errors={errors}
          showErrors={showErrors}
          saving={busy}
          onSubmit={handleSubmit}
          onCancel={cancelForm}
          onRestoreDefaults={() => setForm({ ...emptyForm(defaultProfile.rules), name: form.name, useDailyLimit: form.useDailyLimit, dailyLimit: form.dailyLimit })}
        />
      )}
    </main>
  );
}

import type { FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { RestoreIcon } from "../../components/Icons";
import { LIMITS, type FormErrors, type ProfileForm, type RatingKey } from "../../utils/studyRules";

interface Props {
  title: string;
  form: ProfileForm;
  onChange: (form: ProfileForm) => void;
  // Validation errors, shown only after the user tried to save.
  errors: FormErrors;
  showErrors: boolean;
  saving: boolean;
  onSubmit: () => void;
  onCancel: () => void;
  onRestoreDefaults: () => void;
}

// Create/edit form of a study configuration: a name, an optional daily goal
// and, for each rating, how the system should react.
export function ProfileFormFields({
  title,
  form,
  onChange,
  errors,
  showErrors,
  saving,
  onSubmit,
  onCancel,
  onRestoreDefaults,
}: Props) {
  const { t } = useTranslation();
  const RATINGS: { key: RatingKey; label: string; help: string }[] = [
    { key: "hard", label: t("profileForm.ratingHardLabel"), help: t("profileForm.ratingHardHelp") },
    { key: "good", label: t("profileForm.ratingGoodLabel"), help: t("profileForm.ratingGoodHelp") },
    { key: "easy", label: t("profileForm.ratingEasyLabel"), help: t("profileForm.ratingEasyHelp") },
  ];
  const err = (key: string) => (showErrors ? errors[key] : undefined);

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    onSubmit();
  }

  return (
    <form className="form settings-section profile-form" onSubmit={handleSubmit} aria-label={title} noValidate>
      <h2>{title}</h2>

      <label htmlFor="profile-name">{t("profileForm.nameLabel")}</label>
      <input
        id="profile-name"
        value={form.name}
        maxLength={LIMITS.nameLength * 2}
        onChange={(e) => onChange({ ...form, name: e.target.value })}
        disabled={saving}
        placeholder={t("profileForm.namePlaceholder")}
        aria-invalid={Boolean(err("name"))}
      />
      {err("name") && (
        <p className="error" role="alert">
          {err("name")}
        </p>
      )}

      <div className="profile-daily">
        <label className="checkbox-label" htmlFor="profile-daily-toggle">
          <input
            id="profile-daily-toggle"
            type="checkbox"
            checked={form.useDailyLimit}
            onChange={(e) => onChange({ ...form, useDailyLimit: e.target.checked })}
            disabled={saving}
          />
          {t("profileForm.setDailyGoal")}
        </label>
        {form.useDailyLimit && (
          <>
            <label htmlFor="profile-daily-limit">{t("profileForm.cardsPerDay")}</label>
            <input
              id="profile-daily-limit"
              type="number"
              inputMode="numeric"
              min={LIMITS.dailyCards.min}
              max={LIMITS.dailyCards.max}
              value={form.dailyLimit}
              onChange={(e) => onChange({ ...form, dailyLimit: e.target.value })}
              disabled={saving}
              aria-invalid={Boolean(err("dailyLimit"))}
            />
            {err("dailyLimit") && (
              <p className="error" role="alert">
                {err("dailyLimit")}
              </p>
            )}
          </>
        )}
      </div>

      <div className="profile-values-header">
        <p className="hint">
          {t("profileForm.valuesIntroBefore")}
          <strong>{t("profileForm.valuesIntroStrong")}</strong>
          {t("profileForm.valuesIntroAfter")}
        </p>
        {/* Next to the values it resets, and apart from the save/cancel actions. */}
        <button type="button" className="link-button restore-defaults" onClick={onRestoreDefaults} disabled={saving}>
          <RestoreIcon />
          {t("profileForm.restoreDefaults")}
        </button>
      </div>

      <fieldset className="rating-fieldset">
        <legend>{t("profileForm.againLegend")}</legend>
        <p className="hint">{t("profileForm.againHelp")}</p>
        <label htmlFor="again-minutes">{t("profileForm.againMinutesLabel")}</label>
        <input
          id="again-minutes"
          type="number"
          inputMode="numeric"
          min={LIMITS.againMinutes.min}
          max={LIMITS.againMinutes.max}
          value={form.againDelayMinutes}
          onChange={(e) => onChange({ ...form, againDelayMinutes: e.target.value })}
          disabled={saving}
          aria-invalid={Boolean(err("againDelayMinutes"))}
        />
        {err("againDelayMinutes") && (
          <p className="error" role="alert">
            {err("againDelayMinutes")}
          </p>
        )}
      </fieldset>

      {RATINGS.map(({ key, label, help }) => (
        <fieldset className="rating-fieldset" key={key}>
          <legend>{label}</legend>
          <p className="hint">{help}</p>
          <label htmlFor={`${key}-days`}>{t("profileForm.daysLabel")}</label>
          <input
            id={`${key}-days`}
            type="number"
            inputMode="numeric"
            min={LIMITS.firstIntervalDays.min}
            max={LIMITS.firstIntervalDays.max}
            value={form[key].firstIntervalDays}
            onChange={(e) => onChange({ ...form, [key]: { ...form[key], firstIntervalDays: e.target.value } })}
            disabled={saving}
            aria-invalid={Boolean(err(`${key}.firstIntervalDays`))}
          />
          {err(`${key}.firstIntervalDays`) && (
            <p className="error" role="alert">
              {err(`${key}.firstIntervalDays`)}
            </p>
          )}
          <label htmlFor={`${key}-multiplier`}>{t("profileForm.multiplierLabel")}</label>
          <input
            id={`${key}-multiplier`}
            type="text"
            inputMode="decimal"
            value={form[key].multiplier}
            onChange={(e) => onChange({ ...form, [key]: { ...form[key], multiplier: e.target.value } })}
            disabled={saving}
            aria-invalid={Boolean(err(`${key}.multiplier`))}
          />
          {err(`${key}.multiplier`) && (
            <p className="error" role="alert">
              {err(`${key}.multiplier`)}
            </p>
          )}
        </fieldset>
      ))}

      {showErrors && errors.orderDays && (
        <p className="error" role="alert">
          {errors.orderDays}
        </p>
      )}
      {showErrors && errors.orderMultiplier && (
        <p className="error" role="alert">
          {errors.orderMultiplier}
        </p>
      )}

      <div className="profile-form-actions">
        <button type="button" className="secondary" onClick={onCancel} disabled={saving}>
          {t("profileForm.cancel")}
        </button>
        <button type="submit" disabled={saving}>
          {saving ? t("profileForm.saving") : t("profileForm.save")}
        </button>
      </div>
    </form>
  );
}

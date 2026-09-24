import i18n from "../i18n";
import type { RatingRule, StudyProfile, StudyProfileInput, StudyRules } from "../types";

// The same limits the server enforces (the server has the last word).
export const LIMITS = {
  nameLength: 60,
  dailyCards: { min: 1, max: 200 },
  againMinutes: { min: 1, max: 1440 },
  firstIntervalDays: { min: 1, max: 365 },
  multiplier: { min: 1, max: 5 },
} as const;

export type RatingKey = "hard" | "good" | "easy";

// Fields are kept as text so an input can be emptied and retyped.
export interface RatingRuleForm {
  firstIntervalDays: string;
  multiplier: string;
}

export interface ProfileForm {
  name: string;
  useDailyLimit: boolean;
  dailyLimit: string;
  againDelayMinutes: string;
  hard: RatingRuleForm;
  good: RatingRuleForm;
  easy: RatingRuleForm;
}

// Values of the built-in configuration, used to prefill a new one.
export const FALLBACK_RULES: StudyRules = {
  againDelayMinutes: 10,
  hard: { firstIntervalDays: 1, multiplier: 1.2 },
  good: { firstIntervalDays: 3, multiplier: 2 },
  easy: { firstIntervalDays: 7, multiplier: 3 },
};

// 1.2 -> "1,2", 2 -> "2" (Brazilian decimal comma).
export function formatMultiplier(value: number): string {
  return String(value).replace(".", ",");
}

const ruleToForm = (r: RatingRule): RatingRuleForm => ({
  firstIntervalDays: String(r.firstIntervalDays),
  multiplier: formatMultiplier(r.multiplier),
});

export function emptyForm(rules: StudyRules = FALLBACK_RULES): ProfileForm {
  return {
    name: "",
    useDailyLimit: false,
    dailyLimit: "20",
    againDelayMinutes: String(rules.againDelayMinutes),
    hard: ruleToForm(rules.hard),
    good: ruleToForm(rules.good),
    easy: ruleToForm(rules.easy),
  };
}

export function formFromProfile(p: StudyProfile): ProfileForm {
  return {
    ...emptyForm(p.rules),
    name: p.name,
    useDailyLimit: p.dailyCardLimit !== null,
    dailyLimit: p.dailyCardLimit !== null ? String(p.dailyCardLimit) : "20",
  };
}

const isInt = (text: string) => /^\d+$/.test(text.trim());

// Accepts "1,2" and "1.2".
function parseDecimal(text: string): number | null {
  const normalized = text.trim().replace(",", ".");
  if (!/^\d+(\.\d+)?$/.test(normalized)) return null;
  return Number(normalized);
}

export type FormErrors = Record<string, string>;

// Checks a form and, when it is valid, returns what to send to the server.
export function validateForm(form: ProfileForm): { errors: FormErrors; input: StudyProfileInput | null } {
  const errors: FormErrors = {};

  const name = form.name.trim();
  if (name === "") errors.name = i18n.t("studyRules.errors.nameRequired");
  else if (Array.from(name).length > LIMITS.nameLength)
    errors.name = i18n.t("studyRules.errors.nameTooLong", { max: LIMITS.nameLength });

  let dailyCardLimit: number | null = null;
  if (form.useDailyLimit) {
    const { min, max } = LIMITS.dailyCards;
    if (!isInt(form.dailyLimit) || Number(form.dailyLimit) < min || Number(form.dailyLimit) > max) {
      errors.dailyLimit = i18n.t("studyRules.errors.dailyLimitRange", { min, max });
    } else {
      dailyCardLimit = Number(form.dailyLimit);
    }
  }

  const again = LIMITS.againMinutes;
  const againDelayMinutes = Number(form.againDelayMinutes);
  if (!isInt(form.againDelayMinutes) || againDelayMinutes < again.min || againDelayMinutes > again.max) {
    errors.againDelayMinutes = i18n.t("studyRules.errors.againRange", { min: again.min, max: again.max });
  }

  // Days and multipliers are collected separately so that the "a better answer
  // must not come back sooner" checks still run when some other field is wrong.
  const days: Partial<Record<RatingKey, number>> = {};
  const multipliers: Partial<Record<RatingKey, number>> = {};
  for (const key of ["hard", "good", "easy"] as const) {
    const f = form[key];
    const d = LIMITS.firstIntervalDays;
    const m = LIMITS.multiplier;
    const dayValue = Number(f.firstIntervalDays);
    if (!isInt(f.firstIntervalDays) || dayValue < d.min || dayValue > d.max) {
      errors[`${key}.firstIntervalDays`] = i18n.t("studyRules.errors.daysRange", { min: d.min, max: d.max });
    } else {
      days[key] = dayValue;
    }
    const multiplier = parseDecimal(f.multiplier);
    if (multiplier === null || multiplier < m.min || multiplier > m.max) {
      errors[`${key}.multiplier`] = i18n.t("studyRules.errors.multiplierRange", { min: m.min, max: m.max });
    } else {
      multipliers[key] = multiplier;
    }
  }

  if (days.hard !== undefined && days.good !== undefined && days.easy !== undefined) {
    if (days.hard > days.good || days.good > days.easy) {
      errors.orderDays = i18n.t("studyRules.errors.orderDays");
    }
  }
  if (multipliers.hard !== undefined && multipliers.good !== undefined && multipliers.easy !== undefined) {
    if (multipliers.hard > multipliers.good || multipliers.good > multipliers.easy) {
      errors.orderMultiplier = i18n.t("studyRules.errors.orderMultiplier");
    }
  }

  const rule = (key: RatingKey): RatingRule | null =>
    days[key] !== undefined && multipliers[key] !== undefined
      ? { firstIntervalDays: days[key] as number, multiplier: multipliers[key] as number }
      : null;
  const hard = rule("hard");
  const good = rule("good");
  const easy = rule("easy");

  if (Object.keys(errors).length > 0 || !hard || !good || !easy) return { errors, input: null };
  return { errors, input: { name, dailyCardLimit, rules: { againDelayMinutes, hard, good, easy } } };
}

// "10 minutos", "1 hora", "1 hora e 30 minutos".
export function describeMinutes(minutes: number): string {
  if (minutes < 60) return i18n.t("studyRules.minutes", { count: minutes });
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  return rest === 0
    ? i18n.t("studyRules.hours", { count: hours })
    : i18n.t("studyRules.hoursAndMinutes", {
        hours: i18n.t("studyRules.hours", { count: hours }),
        minutes: i18n.t("studyRules.minutes", { count: rest }),
      });
}

// "1 dia; depois × 1,2".
export function describeRating(rule: RatingRule): string {
  return i18n.t("studyRules.daysForNewCard", {
    count: rule.firstIntervalDays,
    multiplier: formatMultiplier(rule.multiplier),
  });
}

export function describeDailyLimit(limit: number | null): string {
  return limit === null ? i18n.t("studyRules.noDailyGoal") : i18n.t("studyRules.cardsPerDay", { count: limit });
}

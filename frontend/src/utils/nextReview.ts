import i18n from "../i18n";

export interface NextReview {
  // Short and friendly: "em 2 dias", "amanhã", "em 3 horas", "em 20 minutos".
  relative: string;
  // The exact moment in the user's local time: "23/09, às 21:13".
  when: string;
}

const pad2 = (n: number) => String(n).padStart(2, "0");
const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();

// Describes when the next review comes up, relative to now and in the user's
// local time, for the "everything is up to date" screen.
export function describeNextReview(next: Date, now: Date = new Date()): NextReview {
  const when = i18n.t("nextReview.when", {
    day: pad2(next.getDate()),
    month: pad2(next.getMonth() + 1),
    hour: pad2(next.getHours()),
    minute: pad2(next.getMinutes()),
  });

  const diffMs = next.getTime() - now.getTime();
  if (diffMs <= 0) return { relative: i18n.t("nextReview.now"), when };

  const minutes = Math.max(1, Math.ceil(diffMs / 60_000));
  if (minutes < 60) return { relative: i18n.t("nextReview.inMinutes", { count: minutes }), when };

  // Whole calendar days between the two dates (rounded, so a daylight-saving
  // change in between does not skew it).
  const days = Math.round((startOfDay(next) - startOfDay(now)) / 86_400_000);
  if (days <= 0) {
    const hours = Math.max(1, Math.round(diffMs / 3_600_000));
    return { relative: i18n.t("nextReview.inHours", { count: hours }), when };
  }
  if (days === 1) return { relative: i18n.t("nextReview.tomorrow"), when };
  return { relative: i18n.t("nextReview.inDays", { count: days }), when };
}

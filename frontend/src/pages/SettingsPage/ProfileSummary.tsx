import { useTranslation } from "react-i18next";
import type { StudyProfile } from "../../types";
import { describeDailyLimit, describeMinutes, describeRating } from "../../utils/studyRules";

// What a study configuration does, in plain words, one line per rating.
export function ProfileSummary({ profile }: { profile: StudyProfile }) {
  const { rules } = profile;
  const { t } = useTranslation();
  return (
    <dl className="settings-summary" aria-label={t("profileSummary.aria", { name: profile.name })}>
      <div>
        <dt>{t("profileSummary.dailyGoal")}</dt>
        <dd>{describeDailyLimit(profile.dailyCardLimit)}</dd>
      </div>
      <div>
        <dt>{t("profileSummary.again")}</dt>
        <dd>{t("profileSummary.againValue", { time: describeMinutes(rules.againDelayMinutes) })}</dd>
      </div>
      <div>
        <dt>{t("profileSummary.hard")}</dt>
        <dd>{describeRating(rules.hard)}</dd>
      </div>
      <div>
        <dt>{t("profileSummary.good")}</dt>
        <dd>{describeRating(rules.good)}</dd>
      </div>
      <div>
        <dt>{t("profileSummary.easy")}</dt>
        <dd>{describeRating(rules.easy)}</dd>
      </div>
    </dl>
  );
}

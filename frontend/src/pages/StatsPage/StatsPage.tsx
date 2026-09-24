import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { getDeckStats } from "../../services/statsService";
import type { DeckStats } from "../../types";

type LoadState = "loading" | "loaded" | "error";

const RING_RADIUS = 60;
const RING_CIRCUMFERENCE = 2 * Math.PI * RING_RADIUS;

// YYYY-MM-DD of a date in the browser's own time zone (toISOString would
// give the UTC day, which is a different day for part of every evening).
function localISODate(d: Date): string {
  const month = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${d.getFullYear()}-${month}-${day}`;
}

function lastNDays(n: number): string[] {
  const days: string[] = [];
  const today = new Date();
  for (let i = n - 1; i >= 0; i--) {
    const d = new Date(today);
    d.setDate(d.getDate() - i);
    days.push(localISODate(d));
  }
  return days;
}

function formatDayLabel(isoDate: string): string {
  const [, month, day] = isoDate.split("-");
  return `${day}/${month}`;
}

export function StatsPage() {
  const { deckId } = useParams<{ deckId: string }>();
  const { t } = useTranslation();
  const [stats, setStats] = useState<DeckStats | null>(null);
  const [loadState, setLoadState] = useState<LoadState>("loading");

  useEffect(() => {
    if (!deckId) return;
    getDeckStats(deckId)
      .then((s) => {
        setStats(s);
        setLoadState("loaded");
      })
      .catch(() => setLoadState("error"));
  }, [deckId]);

  if (loadState === "loading") {
    return (
      <main className="app-shell">
        <p className="loading">{t("stats.loading")}</p>
      </main>
    );
  }

  if (loadState === "error" || !stats) {
    return (
      <main className="app-shell">
        <p className="error">{t("stats.loadError")}</p>
        <Link to={`/decks/${deckId}`}>{t("stats.backToDeck")}</Link>
      </main>
    );
  }

  const days = lastNDays(7);
  const byDate = new Map(stats.dailyPerformance.map((p) => [p.date, p]));
  const maxReviews = Math.max(1, ...days.map((d) => byDate.get(d)?.reviewCount ?? 0));

  const windowReviews = stats.dailyPerformance.reduce((sum, p) => sum + p.reviewCount, 0);
  const windowHintReviews = stats.dailyPerformance.reduce((sum, p) => sum + (p.hintCount ?? 0), 0);

  const dashOffset = RING_CIRCUMFERENCE * (1 - stats.masteryPercent / 100);

  return (
    <main className="app-shell">
      <Link to={`/decks/${deckId}`} className="back-link">
        &larr; {t("stats.backToDeck")}
      </Link>
      <h1>{t("stats.title", { name: stats.name })}</h1>

      <div className="stats-mastery">
        <svg width="140" height="140" viewBox="0 0 140 140" role="img" aria-label={t("stats.masteryAria", { percent: stats.masteryPercent })}>
          <circle cx="70" cy="70" r={RING_RADIUS} className="stats-ring-track" strokeWidth="12" fill="none" />
          <circle
            cx="70"
            cy="70"
            r={RING_RADIUS}
            className="stats-ring-progress"
            strokeWidth="12"
            fill="none"
            strokeLinecap="round"
            strokeDasharray={RING_CIRCUMFERENCE}
            strokeDashoffset={dashOffset}
            transform="rotate(-90 70 70)"
          />
          <text x="70" y="66" textAnchor="middle" className="stats-ring-percent">
            {stats.masteryPercent}%
          </text>
          <text x="70" y="86" textAnchor="middle" className="stats-ring-label">
            {t("stats.masteryLabel")}
          </text>
        </svg>

        <div className="stats-breakdown">
          <div className="stats-stat" title={t("stats.masteredTitle")}>
            <span className="stats-stat-value" style={{ color: "var(--color-success)" }}>
              {stats.masteredCards}
            </span>
            <span className="stats-stat-label">{t("stats.mastered")}</span>
          </div>
          <div className="stats-stat" title={t("stats.learningTitle")}>
            <span className="stats-stat-value" style={{ color: "var(--color-warning)" }}>
              {stats.learningCards}
            </span>
            <span className="stats-stat-label">{t("stats.learning")}</span>
          </div>
          <div className="stats-stat">
            <span className="stats-stat-value" style={{ color: "var(--color-primary)" }}>
              {stats.newCards}
            </span>
            <span className="stats-stat-label">{t("stats.new")}</span>
          </div>
        </div>
      </div>

      <h2>{t("stats.studiedCardsHeading")}</h2>
      <div className="deck-tiles">
        <div className="deck-tile">
          <span className="deck-tile-label">{t("stats.today")}</span>
          <span className="deck-tile-value">{stats.studiedToday ?? 0}</span>
        </div>
        <div className="deck-tile">
          <span className="deck-tile-label">{t("stats.last7Days")}</span>
          <span className="deck-tile-value">{stats.studiedLast7Days ?? 0}</span>
        </div>
        <div className="deck-tile">
          <span className="deck-tile-label">{t("stats.total")}</span>
          <span className="deck-tile-value">{stats.studiedTotal ?? 0}</span>
        </div>
      </div>
      <p className="hint">{t("stats.perCardOnceHint")}</p>

      <h2>{t("stats.performanceHeading")}</h2>
      {windowReviews > 0 && (
        <p className="hint stats-hint-usage">
          {windowHintReviews > 0
            ? t("stats.hintUsageWithPercent", {
                used: windowHintReviews,
                total: windowReviews,
                percent: Math.round((windowHintReviews * 100) / windowReviews),
              })
            : t("stats.hintUsage", { used: windowHintReviews, total: windowReviews })}
        </p>
      )}
      {stats.dailyPerformance.length === 0 ? (
        <p className="empty-state">{t("stats.noReviews")}</p>
      ) : (
        <div className="stats-chart">
          {days.map((day) => {
            const entry = byDate.get(day);
            const reviewCount = entry?.reviewCount ?? 0;
            const heightPercent = (reviewCount / maxReviews) * 100;
            return (
              <div className="stats-bar-column" key={day}>
                <div className="stats-bar-track">
                  <div
                    className="stats-bar-fill"
                    style={{ height: `${heightPercent}%` }}
                    title={
                      entry
                        ? t("stats.barTitleWithData", { count: entry.reviewCount, accuracy: entry.accuracyPercent })
                        : t("stats.barTitleEmpty")
                    }
                  />
                </div>
                <span className="stats-bar-label">{formatDayLabel(day)}</span>
              </div>
            );
          })}
        </div>
      )}
    </main>
  );
}

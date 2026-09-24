export interface User {
  id: string;
  email: string;
  // One of the SUPPORTED_LANGUAGES codes, or absent if the user never chose
  // one (the app then falls back to detecting the browser's language).
  language?: string;
}

export interface Deck {
  id: string;
  name: string;
  description: string;
  totalCards: number;
  dueCards: number;
  // When the next not-yet-due card comes due (ISO 8601). Only present when
  // a single deck is fetched, and only if a card is scheduled ahead.
  nextDueAt?: string;
  // The study configuration this deck uses instead of the general one;
  // absent when the deck follows the general configuration.
  studyProfileId?: string;
  // The deck group this deck is filed under; absent when it is not in any
  // group and appears on its own on the dashboard.
  groupId?: string;
}

// A named folder of decks (single-level: a group holds decks, never other
// groups), used to organize related collections together.
export interface DeckGroup {
  id: string;
  name: string;
}

export interface ExtendedExample {
  text: string;
  translation: string;
}

export interface Audio {
  contentType: string;
  durationMs: number;
  url: string;
}

export interface Scheduling {
  state: "new" | "learning" | "review" | "mature";
  dueAt: string;
  intervalDays: number;
  repetitions: number;
  lapses: number;
}

export interface Flashcard {
  id: string;
  deckId: string;
  question: string;
  answer: string;
  hint?: string;
  extendedExample?: ExtendedExample;
  audio?: Audio;
  scheduling: Scheduling;
}

export interface DailyPerformance {
  date: string;
  reviewCount: number;
  accuracyPercent: number;
  hintCount: number;
}

export interface DeckStats {
  deckId: string;
  name: string;
  totalCards: number;
  newCards: number;
  learningCards: number;
  masteredCards: number;
  masteryPercent: number;
  // Distinct cards reviewed at least once (each card counts once, however
  // many times it was reviewed). "Today" and the 7 days follow the browser's
  // time zone.
  studiedToday: number;
  studiedLast7Days: number;
  studiedTotal: number;
  dailyPerformance: DailyPerformance[];
}

export interface ArchivedFlashcard extends Flashcard {
  deckName: string;
}

// How one of the three "success" ratings reschedules a card.
export interface RatingRule {
  // Days until a card that is new (or being learned again) comes back.
  firstIntervalDays: number;
  // Factor applied to the previous interval when a learned card is reviewed again.
  multiplier: number;
}

// How the system reacts to each rating. In the app the buttons are
// "Não lembrei" (again), "Lembrei com esforço" (hard), "Lembrei" (good),
// "Lembrei facilmente" (easy).
export interface StudyRules {
  againDelayMinutes: number;
  hard: RatingRule;
  good: RatingRule;
  easy: RatingRule;
}

// A named study configuration: review behavior per rating plus an optional
// daily card goal (null = no goal).
export interface StudyProfile {
  id: string;
  name: string;
  isDefault: boolean;
  dailyCardLimit: number | null;
  rules: StudyRules;
}

export interface StudyProfiles {
  activeProfileId: string;
  profiles: StudyProfile[];
}

export interface StudyProfileInput {
  name: string;
  dailyCardLimit: number | null;
  rules: StudyRules;
}

export interface StudyProgress {
  dailyCardLimit: number | null;
  // Different cards studied today, across all decks.
  studiedToday: number;
  // Cards that still fit in today's goal; null when there is no goal.
  remaining: number | null;
  // True once the user chose, today, to keep studying past the goal. It is
  // saved on the account, so it holds on every device until the day ends.
  continuingPastGoal: boolean;
}

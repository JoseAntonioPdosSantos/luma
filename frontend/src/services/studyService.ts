import { api } from "./api";
import { timeZoneParam } from "./timezone";
import type { Flashcard, StudyProgress } from "../types";

export type Rating = "again" | "hard" | "good" | "easy";

export function getDueFlashcards(deckId: string): Promise<Flashcard[]> {
  const tz = timeZoneParam();
  return api.get<Flashcard[]>(`/api/v1/decks/${deckId}/due-flashcards${tz ? `?${tz}` : ""}`);
}

function query(deckId: string | undefined): string {
  const params = [timeZoneParam(), deckId ? `deckId=${encodeURIComponent(deckId)}` : ""].filter(Boolean);
  return params.length > 0 ? `?${params.join("&")}` : "";
}

// The daily goal that applies and how many different cards were studied today
// toward it. For a deck with a configuration of its own it is that deck's goal
// and count; otherwise (or without a deckId) the general goal.
export async function getStudyProgress(deckId?: string): Promise<StudyProgress> {
  const raw = await api.get<Partial<StudyProgress> | undefined>(`/api/v1/study/progress${query(deckId)}`);
  return {
    dailyCardLimit: raw?.dailyCardLimit ?? null,
    studiedToday: raw?.studiedToday ?? 0,
    remaining: raw?.remaining ?? null,
    continuingPastGoal: raw?.continuingPastGoal ?? false,
  };
}

// The user chose to keep studying past the daily goal. The server remembers
// it for the rest of the day (the user's own day) on every device. For a deck
// with a configuration of its own the choice belongs to that deck alone.
export function continueStudyingPastGoal(deckId?: string): Promise<void> {
  return api.post<void>(`/api/v1/study/continue-past-goal${query(deckId)}`);
}

export function submitReview(
  flashcardId: string,
  rating: Rating,
  responseTimeMs: number,
  // Whether the learner looked at the hint before answering (statistics only).
  hintUsed = false,
): Promise<Flashcard> {
  return api.post<Flashcard>(`/api/v1/flashcards/${flashcardId}/reviews`, {
    rating,
    responseTimeMs,
    hintUsed,
  });
}

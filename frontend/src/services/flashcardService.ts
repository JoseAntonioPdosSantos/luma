import { api } from "./api";
import type { ArchivedFlashcard, Flashcard } from "../types";

export interface FlashcardInput {
  question: string;
  answer: string;
  // Empty string (or omitted) means no hint; on update it clears the hint.
  hint?: string;
  extendedExample?: { text: string; translation: string };
}

export interface FlashcardPage {
  items: Flashcard[];
  // Number of cards matching the request across all pages.
  total: number;
}

export interface ListFlashcardsOptions {
  archived?: boolean;
  // Search text (question, answer or hint; ignores case and accents).
  query?: string;
  limit?: number;
  offset?: number;
}

// One page of a deck's flashcards, newest first (archived ones when
// options.archived is set).
export function listFlashcards(deckId: string, options: ListFlashcardsOptions = {}): Promise<FlashcardPage> {
  const params = new URLSearchParams();
  if (options.archived) params.set("archived", "true");
  if (options.query) params.set("q", options.query);
  if (options.limit !== undefined) params.set("limit", String(options.limit));
  if (options.offset) params.set("offset", String(options.offset));
  const qs = params.toString();
  return api.get<FlashcardPage>(`/api/v1/decks/${deckId}/flashcards${qs ? `?${qs}` : ""}`);
}

// Archived flashcards across all of the user's active decks, each with the
// name of the deck it came from.
export function listAllArchivedFlashcards(): Promise<ArchivedFlashcard[]> {
  return api.get<ArchivedFlashcard[]>("/api/v1/archived-flashcards");
}

export function getFlashcard(id: string): Promise<Flashcard> {
  return api.get<Flashcard>(`/api/v1/flashcards/${id}`);
}

export function createFlashcard(deckId: string, input: FlashcardInput): Promise<Flashcard> {
  return api.post<Flashcard>(`/api/v1/decks/${deckId}/flashcards`, input);
}

export function updateFlashcard(id: string, input: FlashcardInput): Promise<Flashcard> {
  return api.patch<Flashcard>(`/api/v1/flashcards/${id}`, input);
}

export function archiveFlashcard(id: string): Promise<void> {
  return api.delete<void>(`/api/v1/flashcards/${id}`);
}

// Permanently deletes an already-archived flashcard. Cannot be undone.
export function deleteArchivedFlashcard(id: string): Promise<void> {
  return api.delete<void>(`/api/v1/archived-flashcards/${id}`);
}

export function restoreFlashcard(id: string): Promise<void> {
  return api.post<void>(`/api/v1/flashcards/${id}/restore`);
}

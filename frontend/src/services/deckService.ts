import { api } from "./api";
import type { Deck } from "../types";

export function listDecks(): Promise<Deck[]> {
  return api.get<Deck[]>("/api/v1/decks");
}

export function listArchivedDecks(): Promise<Deck[]> {
  return api.get<Deck[]>("/api/v1/decks?archived=true");
}

export function getDeck(id: string): Promise<Deck> {
  return api.get<Deck>(`/api/v1/decks/${id}`);
}

export function createDeck(name: string, description: string): Promise<Deck> {
  return api.post<Deck>("/api/v1/decks", { name, description });
}

export function updateDeck(id: string, name: string, description: string): Promise<Deck> {
  return api.patch<Deck>(`/api/v1/decks/${id}`, { name, description });
}

// Chooses the study configuration one deck uses; null makes it follow the
// general configuration again.
export function setDeckStudyProfile(deckId: string, profileId: string | null): Promise<void> {
  return api.put<void>(`/api/v1/decks/${deckId}/study-profile`, { profileId });
}

// Files a deck under a group; null removes it from its group.
export function setDeckGroup(deckId: string, groupId: string | null): Promise<void> {
  return api.put<void>(`/api/v1/decks/${deckId}/group`, { groupId });
}

export function archiveDeck(id: string): Promise<void> {
  return api.delete<void>(`/api/v1/decks/${id}`);
}

// Permanently deletes an already-archived deck with its flashcards. Cannot
// be undone.
export function deleteArchivedDeck(id: string): Promise<void> {
  return api.delete<void>(`/api/v1/archived-decks/${id}`);
}

export function restoreDeck(id: string): Promise<void> {
  return api.post<void>(`/api/v1/decks/${id}/restore`);
}

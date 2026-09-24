import { api } from "./api";
import type { Flashcard } from "../types";

export function uploadAudio(flashcardId: string, file: File): Promise<Flashcard> {
  return api.postFile<Flashcard>(`/api/v1/flashcards/${flashcardId}/audio`, file.type, file);
}

export function deleteAudio(flashcardId: string): Promise<void> {
  return api.delete<void>(`/api/v1/flashcards/${flashcardId}/audio`);
}

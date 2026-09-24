import { api } from "./api";
import { timeZoneParam } from "./timezone";
import type { DeckStats } from "../types";

export function getDeckStats(deckId: string): Promise<DeckStats> {
  const tz = timeZoneParam();
  return api.get<DeckStats>(`/api/v1/decks/${deckId}/stats${tz ? `?${tz}` : ""}`);
}

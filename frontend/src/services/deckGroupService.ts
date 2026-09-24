import { api } from "./api";
import type { DeckGroup } from "../types";

export function listDeckGroups(): Promise<DeckGroup[]> {
  return api.get<DeckGroup[]>("/api/v1/deck-groups");
}

export function createDeckGroup(name: string): Promise<DeckGroup> {
  return api.post<DeckGroup>("/api/v1/deck-groups", { name });
}

export function renameDeckGroup(id: string, name: string): Promise<DeckGroup> {
  return api.put<DeckGroup>(`/api/v1/deck-groups/${id}`, { name });
}

export function deleteDeckGroup(id: string): Promise<void> {
  return api.delete<void>(`/api/v1/deck-groups/${id}`);
}

import { api } from "./api";
import type { User } from "../types";

export function login(email: string): Promise<{ user: User }> {
  return api.post<{ user: User }>("/api/v1/session", { email });
}

export function logout(): Promise<void> {
  return api.delete<void>("/api/v1/session");
}

export function me(): Promise<User> {
  return api.get<User>("/api/v1/me");
}

export function setLanguage(language: string): Promise<void> {
  return api.put<void>("/api/v1/me/language", { language });
}

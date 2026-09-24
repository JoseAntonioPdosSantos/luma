import { api } from "./api";
import type { StudyProfile, StudyProfileInput, StudyProfiles } from "../types";

// The built-in "Padrão" first, then the user's own configurations, and which
// one is active.
export function listStudyProfiles(): Promise<StudyProfiles> {
  return api.get<StudyProfiles>("/api/v1/study-profiles");
}

export function createStudyProfile(input: StudyProfileInput): Promise<StudyProfile> {
  return api.post<StudyProfile>("/api/v1/study-profiles", input);
}

export function updateStudyProfile(id: string, input: StudyProfileInput): Promise<StudyProfile> {
  return api.put<StudyProfile>(`/api/v1/study-profiles/${id}`, input);
}

export function deleteStudyProfile(id: string): Promise<void> {
  return api.delete<void>(`/api/v1/study-profiles/${id}`);
}

// Chooses the configuration used from now on ("default" is the built-in one).
export function setActiveStudyProfile(profileId: string): Promise<void> {
  return api.put<void>("/api/v1/me/active-study-profile", { profileId });
}

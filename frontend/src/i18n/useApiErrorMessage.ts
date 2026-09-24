import { useTranslation } from "react-i18next";
import { ApiError } from "../services/api";

// The "errors" resource is a flat map keyed by the backend's own dotted
// error key (e.g. "deck.name.tooLong"), not a nested i18next path — see
// locales/pt.json's "errors" block. Reading it as one flat object sidesteps
// the fact that some keys (e.g. "request.invalidJson" and
// "request.invalidJson.profileId") would otherwise collide as both a leaf
// string and a parent under i18next's normal dot-nesting lookup.
function errorsBundle(i18nInstance: import("i18next").i18n, language: string): Record<string, string> | undefined {
  const bundle = i18nInstance.getResourceBundle(language, "translation") as
    | { errors?: Record<string, string> }
    | undefined;
  return bundle?.errors;
}

// Returns a function that turns a caught error into a message to show the
// user: a translated version of the backend's error key when there is one
// and we have it, otherwise the given fallback translation key.
export function useApiErrorMessage() {
  const { t, i18n } = useTranslation();
  return (err: unknown, fallbackKey: string): string => {
    if (err instanceof ApiError && err.key) {
      const translated = errorsBundle(i18n, i18n.resolvedLanguage ?? i18n.language)?.[err.key];
      if (translated) return translated;
    }
    return t(fallbackKey);
  };
}

// Where the settings page should send the user back to: the page they opened
// it from (a deck, for instance) instead of always the dashboard. The origin
// travels in the router's navigation state, so it cannot be forged from a URL.

import i18n from "../i18n";

export interface ReturnState {
  from?: string;
}

export interface BackTarget {
  path: string;
  label: string;
}

const dashboard = (): BackTarget => ({ path: "/dashboard", label: i18n.t("returnTo.dashboard") });

// Only in-app paths are accepted ("/decks/abc", never "https://…" or "//host").
function isInternalPath(value: unknown): value is string {
  return typeof value === "string" && value.startsWith("/") && !value.startsWith("//");
}

// Reads the navigation state and says where "Voltar" goes and what to call it.
export function backTarget(state: unknown): BackTarget {
  const from = (state as ReturnState | null | undefined)?.from;
  if (!isInternalPath(from)) return dashboard();
  if (/^\/decks\/[^/]+$/.test(from)) return { path: from, label: i18n.t("returnTo.deck") };
  if (from === "/dashboard") return dashboard();
  return { path: from, label: i18n.t("returnTo.generic") };
}

import { describe, expect, it } from "vitest";
import { backTarget } from "./returnTo";

describe("backTarget", () => {
  it("goes back to the dashboard when nothing says otherwise", () => {
    expect(backTarget(undefined)).toEqual({ path: "/dashboard", label: "Voltar ao painel" });
    expect(backTarget(null)).toEqual({ path: "/dashboard", label: "Voltar ao painel" });
    expect(backTarget({})).toEqual({ path: "/dashboard", label: "Voltar ao painel" });
  });

  it("goes back to the deck the user came from", () => {
    expect(backTarget({ from: "/decks/6ab04ab9" })).toEqual({ path: "/decks/6ab04ab9", label: "Voltar à coleção" });
  });

  it("uses a neutral label for any other in-app page", () => {
    expect(backTarget({ from: "/decks/abc/stats" })).toEqual({ path: "/decks/abc/stats", label: "Voltar" });
  });

  it("keeps the dashboard wording when it came from the dashboard", () => {
    expect(backTarget({ from: "/dashboard" })).toEqual({ path: "/dashboard", label: "Voltar ao painel" });
  });

  it("ignores anything that is not an in-app path", () => {
    for (const from of ["https://evil.example", "//evil.example/x", "javascript:alert(1)", "decks/abc", "", 42, {}]) {
      expect(backTarget({ from })).toEqual({ path: "/dashboard", label: "Voltar ao painel" });
    }
  });
});

import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { SettingsPage } from "../SettingsPage/SettingsPage";
import { DeckPage } from "./DeckPage";

function jsonResponse(body: unknown) {
  return Promise.resolve({
    ok: true,
    status: 200,
    headers: new Headers({ "content-type": "application/json" }),
    json: async () => body,
  });
}

const scheduling = { state: "new", dueAt: "2026-01-01T00:00:00Z", intervalDays: 0, repetitions: 0, lapses: 0 };
const card = { id: "c1", deckId: "d1", question: "Archived question", answer: "Archived answer", scheduling };

describe("DeckPage", () => {
  it("lists archived flashcards on demand and restores one", async () => {
    let active: unknown[] = [];
    let archived: unknown[] = [card];
    let deck = { id: "d1", name: "English", description: "", totalCards: 0, dueCards: 0 };

    const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (url.endsWith("/api/v1/flashcards/c1/restore") && init?.method === "POST") {
        active = [card];
        archived = [];
        deck = { ...deck, totalCards: 1 };
        return Promise.resolve({ ok: true, status: 204, headers: new Headers(), json: async () => undefined });
      }
      if (url.includes("archived=true")) return jsonResponse({ items: archived, total: archived.length });
      if (url.includes("/api/v1/decks/d1/flashcards")) return jsonResponse({ items: active, total: active.length });
      if (url.endsWith("/api/v1/decks/d1")) return jsonResponse(deck);
      return jsonResponse({});
    });
    vi.stubGlobal("fetch", fetchMock);

    render(
      <MemoryRouter initialEntries={["/decks/d1"]}>
        <Routes>
          <Route path="/decks/:deckId" element={<DeckPage />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(await screen.findByText(/nenhum flashcard ainda/i)).toBeInTheDocument();
    expect(screen.queryByText("Archived question")).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: /ver flashcards arquivados/i }));
    expect(await screen.findByText("Archived question")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: /restaurar/i }));

    await waitFor(() => expect(screen.queryByText(/nenhum flashcard ainda/i)).not.toBeInTheDocument());
    expect(screen.getByText("Nenhum flashcard arquivado nesta coleção.")).toBeInTheDocument();
    expect(screen.getAllByText("Archived question")).toHaveLength(1);
  });
  it("permanently deletes an archived flashcard after confirming", async () => {
    let archived: unknown[] = [card];
    const deck = { id: "d1", name: "English", description: "", totalCards: 0, dueCards: 0 };

    const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (url.endsWith("/api/v1/archived-flashcards/c1") && init?.method === "DELETE") {
        archived = [];
        return Promise.resolve({ ok: true, status: 204, headers: new Headers(), json: async () => undefined });
      }
      if (url.includes("archived=true")) return jsonResponse({ items: archived, total: archived.length });
      if (url.includes("/api/v1/decks/d1/flashcards")) return jsonResponse({ items: [], total: 0 });
      if (url.endsWith("/api/v1/decks/d1")) return jsonResponse(deck);
      return jsonResponse({});
    });
    vi.stubGlobal("fetch", fetchMock);

    render(
      <MemoryRouter initialEntries={["/decks/d1"]}>
        <Routes>
          <Route path="/decks/:deckId" element={<DeckPage />} />
        </Routes>
      </MemoryRouter>,
    );

    await userEvent.click(await screen.findByRole("button", { name: /ver flashcards arquivados/i }));
    expect(await screen.findByText("Archived question")).toBeInTheDocument();
    // Only the main list hides card contents; archived cards are not studied.
    const archivedRow = screen.getByText("Archived question").closest("li") as HTMLElement;
    expect(archivedRow).not.toHaveAttribute("data-revealed");
    expect(archivedRow.querySelector(".is-hidden")).toBeNull();
    expect(screen.getByText("Archived question")).not.toHaveAttribute("aria-hidden");

    await userEvent.click(screen.getByRole("button", { name: "Excluir" }));
    expect(fetchMock).not.toHaveBeenCalledWith(expect.stringContaining("/archived-flashcards/"), expect.anything());
    await userEvent.click(screen.getByRole("button", { name: "Excluir definitivamente" }));

    await waitFor(() => expect(screen.queryByText("Archived question")).not.toBeInTheDocument());
    expect(screen.getByText("Nenhum flashcard arquivado nesta coleção.")).toBeInTheDocument();
  });
  it("shows the deck summary and keeps edit/archive behind the settings row", async () => {
    const deck = { id: "d1", name: "Inglês", description: "", totalCards: 233, dueCards: 5 };
    const stats = {
      deckId: "d1",
      name: "Inglês",
      totalCards: 233,
      newCards: 12,
      learningCards: 23,
      masteredCards: 198,
      masteryPercent: 85,
      dailyPerformance: [],
    };

    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation((url: string) => {
        if (url.includes("/api/v1/decks/d1/stats")) return jsonResponse(stats);
        if (url.includes("/api/v1/decks/d1/flashcards")) return jsonResponse({ items: [], total: 0 });
        if (url.endsWith("/api/v1/decks/d1")) return jsonResponse(deck);
        return jsonResponse({});
      }),
    );

    render(
      <MemoryRouter initialEntries={["/decks/d1"]}>
        <Routes>
          <Route path="/decks/:deckId" element={<DeckPage />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(await screen.findByText("233 cards")).toBeInTheDocument();
    expect(await screen.findByText("85% de domínio geral")).toBeInTheDocument();
    expect(screen.getByText("Para revisar").nextElementSibling).toHaveTextContent("23");
    expect(screen.getByText("Aprendidos").nextElementSibling).toHaveTextContent("198");
    expect(screen.getByText("Novos").nextElementSibling).toHaveTextContent("12");

    expect(screen.getByRole("button", { name: "Estudar agora" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Estatísticas" })).toBeInTheDocument();

    // Edit / archive live inside "Configurações da coleção".
    expect(screen.queryByRole("button", { name: "Editar coleção" })).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Configurações da coleção" }));
    expect(screen.getByRole("button", { name: "Editar coleção" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Arquivar coleção" })).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Configurações da coleção" }));
    expect(screen.queryByRole("button", { name: "Editar coleção" })).not.toBeInTheDocument();
  });
  describe("search and pagination", () => {
    const makeCards = (n: number) =>
      Array.from({ length: n }, (_, i) => ({
        id: `c${i + 1}`,
        deckId: "d1",
        question: `Card ${i + 1}`,
        answer: `answer ${i + 1}`,
        scheduling,
      }));

    // A fake backend for one deck: filters by q and pages with limit/offset,
    // exactly like the real endpoint.
    function pagedBackend(all: ReturnType<typeof makeCards>) {
      const deck = { id: "d1", name: "English", description: "", totalCards: all.length, dueCards: 0 };
      return vi.fn().mockImplementation((url: string) => {
        const u = new URL(url);
        if (u.pathname === "/api/v1/decks/d1/flashcards") {
          const q = (u.searchParams.get("q") ?? "").toLowerCase();
          const offset = Number(u.searchParams.get("offset") ?? 0);
          const limit = Number(u.searchParams.get("limit") ?? 30);
          const matches = all.filter((c) => c.question.toLowerCase().includes(q));
          return jsonResponse({ items: matches.slice(offset, offset + limit), total: matches.length });
        }
        if (u.pathname === "/api/v1/decks/d1") return jsonResponse(deck);
        return jsonResponse({});
      });
    }

    function renderDeck() {
      return render(
        <MemoryRouter initialEntries={["/decks/d1"]}>
          <Routes>
            <Route path="/decks/:deckId" element={<DeckPage />} />
          </Routes>
        </MemoryRouter>,
      );
    }

    it("shows the first page and loads the rest on demand", async () => {
      vi.stubGlobal("fetch", pagedBackend(makeCards(45)));

      renderDeck();

      expect(await screen.findByText("Mostrando 30 de 45 cards")).toBeInTheDocument();
      expect(screen.getAllByRole("listitem")).toHaveLength(30);
      expect(screen.queryByText("Card 31")).not.toBeInTheDocument();

      await userEvent.click(screen.getByRole("button", { name: "Carregar mais (15 restantes)" }));

      expect(await screen.findByText("Mostrando 45 de 45 cards")).toBeInTheDocument();
      expect(screen.getByText("Card 45")).toBeInTheDocument();
      expect(screen.queryByRole("button", { name: /carregar mais/i })).not.toBeInTheDocument();
    });

    it("searches (after a short pause) and reports the number of results", async () => {
      const fetchMock = pagedBackend(makeCards(45));
      vi.stubGlobal("fetch", fetchMock);

      renderDeck();
      await screen.findByText("Mostrando 30 de 45 cards");

      await userEvent.type(screen.getByRole("searchbox", { name: "Buscar flashcards" }), "card 4");

      // "Card 4", "Card 40" ... "Card 45": 7 matches.
      expect(await screen.findByText("7 resultados para “card 4”")).toBeInTheDocument();
      expect(screen.getByText("Card 45")).toBeInTheDocument();
      expect(screen.queryByText("Card 1")).not.toBeInTheDocument();

      // Typing does not fire one request per keystroke.
      const searchCalls = fetchMock.mock.calls.filter(([url]) => new URL(String(url)).searchParams.has("q"));
      expect(searchCalls).toHaveLength(1);
    });

    const rowOf = (question: string) => screen.getByText(question).closest("li") as HTMLElement;

    it("hides the question and answer of every card until the user shows them", async () => {
      vi.stubGlobal("fetch", pagedBackend(makeCards(3)));

      renderDeck();
      await screen.findByText("Mostrando 3 de 3 cards");

      for (const n of [1, 2, 3]) {
        expect(rowOf(`Card ${n}`)).toHaveAttribute("data-revealed", "false");
        expect(screen.getByRole("button", { name: `Mostrar card ${n}` })).toHaveAttribute("aria-pressed", "false");
      }
      // Hidden from screen readers too, which are told the content is hidden.
      expect(screen.getByText("Card 1")).toHaveAttribute("aria-hidden", "true");
      expect(screen.getByText("answer 1")).toHaveAttribute("aria-hidden", "true");
      expect(within(rowOf("Card 1")).getByText("Conteúdo oculto.")).toBeInTheDocument();
      expect(rowOf("Card 1").querySelector(".is-hidden")).not.toBeNull();
    });

    it("shows one card with its eye button and hides it again", async () => {
      vi.stubGlobal("fetch", pagedBackend(makeCards(3)));

      renderDeck();
      await screen.findByText("Mostrando 3 de 3 cards");

      await userEvent.click(screen.getByRole("button", { name: "Mostrar card 2" }));

      expect(rowOf("Card 2")).toHaveAttribute("data-revealed", "true");
      expect(rowOf("Card 2").querySelector(".is-hidden")).toBeNull();
      expect(screen.getByText("Card 2")).toHaveAttribute("aria-hidden", "false");
      expect(within(rowOf("Card 2")).queryByText("Conteúdo oculto.")).not.toBeInTheDocument();
      // The others stay hidden.
      expect(rowOf("Card 1")).toHaveAttribute("data-revealed", "false");
      expect(rowOf("Card 3")).toHaveAttribute("data-revealed", "false");

      await userEvent.click(screen.getByRole("button", { name: "Ocultar card 2" }));
      expect(rowOf("Card 2")).toHaveAttribute("data-revealed", "false");
      expect(screen.getByRole("button", { name: "Mostrar card 2" })).toBeInTheDocument();
    });

    it("shows and hides all the cards of the list with one button", async () => {
      vi.stubGlobal("fetch", pagedBackend(makeCards(3)));

      renderDeck();
      await screen.findByText("Mostrando 3 de 3 cards");

      await userEvent.click(screen.getByRole("button", { name: "Mostrar todos" }));
      for (const n of [1, 2, 3]) expect(rowOf(`Card ${n}`)).toHaveAttribute("data-revealed", "true");

      await userEvent.click(screen.getByRole("button", { name: "Ocultar todos" }));
      for (const n of [1, 2, 3]) expect(rowOf(`Card ${n}`)).toHaveAttribute("data-revealed", "false");
      expect(screen.getByRole("button", { name: "Mostrar todos" })).toBeInTheDocument();
    });

    it("offers to hide all only when every card is shown, and keeps cards loaded later hidden", async () => {
      vi.stubGlobal("fetch", pagedBackend(makeCards(45)));

      renderDeck();
      await screen.findByText("Mostrando 30 de 45 cards");
      await userEvent.click(screen.getByRole("button", { name: "Mostrar todos" }));
      expect(screen.getByRole("button", { name: "Ocultar todos" })).toBeInTheDocument();

      // Showing one by one until the last one also flips the button.
      await userEvent.click(screen.getByRole("button", { name: "Ocultar card 5" }));
      expect(screen.getByRole("button", { name: "Mostrar todos" })).toBeInTheDocument();
      await userEvent.click(screen.getByRole("button", { name: "Mostrar card 5" }));
      expect(screen.getByRole("button", { name: "Ocultar todos" })).toBeInTheDocument();

      // Cards that arrive with "Carregar mais" start hidden, the earlier ones stay shown.
      await userEvent.click(screen.getByRole("button", { name: "Carregar mais (15 restantes)" }));
      await screen.findByText("Mostrando 45 de 45 cards");
      expect(rowOf("Card 1")).toHaveAttribute("data-revealed", "true");
      expect(rowOf("Card 45")).toHaveAttribute("data-revealed", "false");
      expect(screen.getByRole("button", { name: "Mostrar todos" })).toBeInTheDocument();
    });

    it("keeps search results hidden too", async () => {
      vi.stubGlobal("fetch", pagedBackend(makeCards(45)));

      renderDeck();
      await screen.findByText("Mostrando 30 de 45 cards");
      await userEvent.type(screen.getByRole("searchbox", { name: "Buscar flashcards" }), "card 4");

      await screen.findByText("7 resultados para “card 4”");
      expect(rowOf("Card 45")).toHaveAttribute("data-revealed", "false");
    });

    it("says so when the search has no results", async () => {
      vi.stubGlobal("fetch", pagedBackend(makeCards(5)));

      renderDeck();
      await screen.findByText("Mostrando 5 de 5 cards");

      await userEvent.type(screen.getByRole("searchbox", { name: "Buscar flashcards" }), "zzz");

      expect(await screen.findByText("Nenhum flashcard encontrado para “zzz”.")).toBeInTheDocument();
      expect(screen.queryByText(/nenhum flashcard ainda/i)).not.toBeInTheDocument();
    });
  });
  describe("study configuration of the deck", () => {
    const rules = (again: number) => ({
      againDelayMinutes: again,
      hard: { firstIntervalDays: 1, multiplier: 1.2 },
      good: { firstIntervalDays: 3, multiplier: 2 },
      easy: { firstIntervalDays: 7, multiplier: 3 },
    });
    const padrao = { id: "default", name: "Padrão", isDefault: true, dailyCardLimit: null, rules: rules(10) };
    const prova = { id: "p1", name: "Prova", isDefault: false, dailyCardLimit: 30, rules: rules(5) };
    const geral = { id: "p2", name: "Geral", isDefault: false, dailyCardLimit: 8, rules: rules(15) };

    function deckBackend(deckExtra: object = {}, profiles = { activeProfileId: "p2", profiles: [padrao, prova, geral] }) {
      let deck: Record<string, unknown> = { id: "d1", name: "English", description: "", totalCards: 3, dueCards: 0, ...deckExtra };
      const sent: { method: string; path: string; body?: unknown }[] = [];
      const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
        const u = new URL(url);
        const method = init?.method ?? "GET";
        if (method !== "GET") sent.push({ method, path: u.pathname, body: init?.body ? JSON.parse(String(init.body)) : undefined });
        if (u.pathname === "/api/v1/study-profiles") return jsonResponse(profiles);
        if (u.pathname === "/api/v1/deck-groups") return jsonResponse([]);
        if (u.pathname === "/api/v1/decks/d1/study-profile" && method === "PUT") {
          const { profileId } = JSON.parse(String(init?.body));
          deck = { ...deck };
          if (profileId) deck.studyProfileId = profileId;
          else delete deck.studyProfileId;
          return Promise.resolve({ ok: true, status: 204, headers: new Headers(), json: async () => undefined });
        }
        if (u.pathname === "/api/v1/decks/d1/flashcards") return jsonResponse({ items: [], total: 0 });
        if (u.pathname === "/api/v1/decks/d1") return jsonResponse(deck);
        return jsonResponse({});
      });
      return { fetchMock, sent };
    }

    function renderDeck() {
      return render(
        <MemoryRouter initialEntries={["/decks/d1"]}>
          <Routes>
            <Route path="/decks/:deckId" element={<DeckPage />} />
          </Routes>
        </MemoryRouter>,
      );
    }

    async function openSettings() {
      await userEvent.click(await screen.findByRole("button", { name: "Configurações da coleção" }));
      return screen.findByRole("combobox", { name: "Configuração desta coleção" });
    }

    it("follows the general configuration by default and says which one it is", async () => {
      vi.stubGlobal("fetch", deckBackend().fetchMock);

      renderDeck();
      const picker = await openSettings();

      expect(picker).toHaveValue("");
      const options = within(picker).getAllByRole("option").map((o) => o.textContent);
      expect(options).toEqual(["Usar a configuração geral (Geral)", "Padrão", "Prova", "Geral"]);
      expect(screen.getByText(/segue a configuração geral \(“Geral”\)/i)).toBeInTheDocument();
      // The summary is the general configuration's.
      const summary = screen.getByLabelText("Resumo da configuração Geral");
      expect(within(summary).getByText("8 cards por dia")).toBeInTheDocument();
      expect(within(summary).getByText("volta em 15 minutos")).toBeInTheDocument();
    });

    it("gives the deck a configuration of its own and can go back to the general one", async () => {
      const api = deckBackend();
      vi.stubGlobal("fetch", api.fetchMock);

      renderDeck();
      const picker = await openSettings();

      await userEvent.selectOptions(picker, "Prova");
      expect(await screen.findByText(/usa a configuração “Prova”, no lugar da geral/i)).toBeInTheDocument();
      expect(screen.getByText(/conta só os cards desta coleção/i)).toBeInTheDocument();
      expect(api.sent).toContainEqual({ method: "PUT", path: "/api/v1/decks/d1/study-profile", body: { profileId: "p1" } });
      const summary = screen.getByLabelText("Resumo da configuração Prova");
      expect(within(summary).getByText("30 cards por dia")).toBeInTheDocument();
      expect(picker).toHaveValue("p1");

      await userEvent.selectOptions(picker, "");
      expect(await screen.findByText(/segue a configuração geral/i)).toBeInTheDocument();
      expect(api.sent[api.sent.length - 1]).toEqual({ method: "PUT", path: "/api/v1/decks/d1/study-profile", body: { profileId: null } });
      expect(picker).toHaveValue("");
    });

    it("shows the deck's own configuration when it already has one", async () => {
      vi.stubGlobal("fetch", deckBackend({ studyProfileId: "p1" }).fetchMock);

      renderDeck();
      const picker = await openSettings();

      expect(picker).toHaveValue("p1");
      expect(screen.getByText(/usa a configuração “Prova”/i)).toBeInTheDocument();
    });

    it("treats a configuration that no longer exists as none", async () => {
      vi.stubGlobal("fetch", deckBackend({ studyProfileId: "vanished" }).fetchMock);

      renderDeck();
      const picker = await openSettings();

      expect(picker).toHaveValue("");
      expect(screen.getByText(/segue a configuração geral/i)).toBeInTheDocument();
    });

    it("reports a failure to change the configuration and keeps the previous one", async () => {
      const api = deckBackend();
      vi.stubGlobal(
        "fetch",
        vi.fn().mockImplementation((url: string, init?: RequestInit) =>
          init?.method === "PUT" && url.includes("/study-profile")
            ? Promise.resolve({
                ok: false,
                status: 404,
                headers: new Headers({ "content-type": "application/json" }),
                json: async () => ({
                  error: { code: "NOT_FOUND", key: "studyProfile.notFound", message: "configuration not found", requestId: "x" },
                }),
              })
            : api.fetchMock(url, init),
        ),
      );

      renderDeck();
      const picker = await openSettings();
      await userEvent.selectOptions(picker, "Prova");

      // The backend's error key is translated, rather than showing its raw
      // (always-English) message.
      expect(await screen.findByRole("alert")).toHaveTextContent(/configuração não encontrada/i);
      expect(picker).toHaveValue("");
    });

    it("opens the settings from the deck and the back link returns to that same deck", async () => {
      vi.stubGlobal("fetch", deckBackend().fetchMock);

      render(
        <MemoryRouter initialEntries={["/decks/d1"]}>
          <Routes>
            <Route path="/decks/:deckId" element={<DeckPage />} />
            <Route path="/settings" element={<SettingsPage />} />
          </Routes>
        </MemoryRouter>,
      );
      await openSettings();

      await userEvent.click(screen.getByRole("link", { name: "Criar ou editar configurações" }));

      // The settings page says where it goes back to...
      const back = await screen.findByRole("link", { name: /voltar à coleção/i });
      expect(back).toHaveAttribute("href", "/decks/d1");
      expect(screen.getByRole("heading", { name: "Configurações" })).toBeInTheDocument();

      // ...and following it lands on the deck, not on the dashboard.
      await userEvent.click(back);
      expect(await screen.findByRole("heading", { name: "English" })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Estudar agora" })).toBeInTheDocument();
    });

    it("does not load the configurations until the settings are opened", async () => {
      const api = deckBackend();
      vi.stubGlobal("fetch", api.fetchMock);

      renderDeck();
      await screen.findByRole("button", { name: "Configurações da coleção" });

      expect(api.fetchMock.mock.calls.some(([url]) => String(url).includes("/study-profiles"))).toBe(false);
    });
  });
});

import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { AuthProvider } from "../../hooks/useAuth";
import { DashboardPage } from "./DashboardPage";

function jsonResponse(body: unknown, ok = true, status = 200) {
  return Promise.resolve({
    ok,
    status,
    headers: new Headers({ "content-type": "application/json" }),
    json: async () => body,
  });
}

describe("DashboardPage", () => {
  it("shows an empty state, then lists a deck after creating one", async () => {
    let decks: unknown[] = [];

    const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (url.includes("/api/v1/me")) {
        return jsonResponse({ id: "u1", email: "user@example.com" });
      }
      if (url.includes("/api/v1/deck-groups")) return jsonResponse([]);
      if (url.includes("/api/v1/decks") && init?.method === "POST") {
        const created = { id: "d1", name: "English", description: "", totalCards: 0, dueCards: 0 };
        decks = [created];
        return jsonResponse(created, true, 201);
      }
      if (url.includes("/api/v1/decks")) {
        return jsonResponse(decks);
      }
      return jsonResponse({}, false, 404);
    });
    vi.stubGlobal("fetch", fetchMock);

    render(
      <MemoryRouter>
        <AuthProvider>
          <DashboardPage />
        </AuthProvider>
      </MemoryRouter>,
    );

    expect(await screen.findByText(/ainda não tem coleções/i)).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Criar coleção" }));
    await userEvent.type(screen.getByLabelText("Nome"), "English");
    await userEvent.click(screen.getByRole("button", { name: "Salvar coleção" }));

    await waitFor(() => expect(screen.getByText("English")).toBeInTheDocument());
  });
  it("creates a group and a deck filed under it, shown separately from ungrouped decks", async () => {
    let groups: unknown[] = [];
    let decks: unknown[] = [{ id: "d0", name: "Loose deck", description: "", totalCards: 1, dueCards: 0 }];

    const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (url.includes("/api/v1/me")) return jsonResponse({ id: "u1", email: "user@example.com" });
      if (url.includes("/api/v1/deck-groups") && init?.method === "POST") {
        const { name } = JSON.parse(String(init.body));
        const created = { id: "g1", name };
        groups = [created];
        return jsonResponse(created, true, 201);
      }
      if (url.includes("/api/v1/deck-groups")) return jsonResponse(groups);
      if (url.includes("/api/v1/decks") && init?.method === "POST") {
        const { name } = JSON.parse(String(init.body));
        const created = { id: "d1", name, description: "", totalCards: 0, dueCards: 0 };
        decks = [...decks, created];
        return jsonResponse(created, true, 201);
      }
      if (url.endsWith("/api/v1/decks/d1/group") && init?.method === "PUT") {
        const { groupId } = JSON.parse(String(init.body));
        decks = decks.map((d) => (d && typeof d === "object" && (d as { id: string }).id === "d1" ? { ...d, groupId } : d));
        return Promise.resolve({ ok: true, status: 204, headers: new Headers(), json: async () => undefined });
      }
      if (url.includes("/api/v1/decks")) return jsonResponse(decks);
      return jsonResponse({}, false, 404);
    });
    vi.stubGlobal("fetch", fetchMock);

    render(
      <MemoryRouter>
        <AuthProvider>
          <DashboardPage />
        </AuthProvider>
      </MemoryRouter>,
    );

    expect(await screen.findByText("Loose deck")).toBeInTheDocument();

    // Create the group first, so it is offered when creating the deck.
    await userEvent.click(screen.getByRole("button", { name: "Criar grupo" }));
    await userEvent.type(screen.getByLabelText("Nome do grupo"), "Inglês");
    await userEvent.click(screen.getByRole("button", { name: "Salvar grupo" }));
    await waitFor(() => expect(screen.getByText("Inglês")).toBeInTheDocument());
    expect(screen.getByText("0 coleção")).toBeInTheDocument();

    // Create a deck already filed under that group.
    await userEvent.click(screen.getByRole("button", { name: "Criar coleção" }));
    await userEvent.type(screen.getByLabelText("Nome"), "Phrasal verbs");
    await userEvent.selectOptions(screen.getByLabelText("Grupo (opcional)"), "Inglês");
    await userEvent.click(screen.getByRole("button", { name: "Salvar coleção" }));

    // The grouped deck no longer appears on its own; the group's count grows.
    await waitFor(() => expect(screen.getByText("1 coleção")).toBeInTheDocument());
    expect(screen.queryByText("Phrasal verbs")).not.toBeInTheDocument();
    expect(screen.getByText("Loose deck")).toBeInTheDocument();
  });
  it("lists archived decks and cards from all decks on demand and restores them", async () => {
    let active: unknown[] = [];
    let archivedDecks: unknown[] = [{ id: "d9", name: "Old deck", description: "", totalCards: 3, dueCards: 0 }];
    let archivedCards: unknown[] = [
      {
        id: "c1",
        deckId: "d2",
        deckName: "Français",
        question: "Test card",
        answer: "Test answer",
        scheduling: { state: "new", dueAt: "2026-01-01T00:00:00Z", intervalDays: 0, repetitions: 0, lapses: 0 },
      },
    ];

    const noContent = () => Promise.resolve({ ok: true, status: 204, headers: new Headers(), json: async () => undefined });
    const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (url.includes("/api/v1/me")) {
        return jsonResponse({ id: "u1", email: "user@example.com" });
      }
      if (url.endsWith("/api/v1/decks/d9/restore") && init?.method === "POST") {
        active = archivedDecks;
        archivedDecks = [];
        return noContent();
      }
      if (url.endsWith("/api/v1/flashcards/c1/restore") && init?.method === "POST") {
        archivedCards = [];
        return noContent();
      }
      if (url.endsWith("/api/v1/archived-flashcards")) return jsonResponse(archivedCards);
      if (url.includes("/api/v1/deck-groups")) return jsonResponse([]);
      if (url.includes("/api/v1/decks?archived=true")) return jsonResponse(archivedDecks);
      if (url.includes("/api/v1/decks")) return jsonResponse(active);
      return jsonResponse({}, false, 404);
    });
    vi.stubGlobal("fetch", fetchMock);

    render(
      <MemoryRouter>
        <AuthProvider>
          <DashboardPage />
        </AuthProvider>
      </MemoryRouter>,
    );

    expect(await screen.findByText(/ainda não tem coleções/i)).toBeInTheDocument();
    expect(screen.queryByText("Old deck")).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: /ver arquivados/i }));
    expect(await screen.findByText("Old deck")).toBeInTheDocument();
    expect(screen.getByText("Test card")).toBeInTheDocument();
    expect(screen.getByText("Coleção: Français")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Restaurar card" }));
    await waitFor(() => expect(screen.queryByText("Test card")).not.toBeInTheDocument());

    await userEvent.click(screen.getByRole("button", { name: "Restaurar coleção" }));
    await waitFor(() => expect(screen.queryByText(/ainda não tem coleções/i)).not.toBeInTheDocument());
    expect(screen.getByText("Nenhum item arquivado.")).toBeInTheDocument();
    expect(screen.getAllByText("Old deck")).toHaveLength(1);
  });
  it("permanently deletes an archived card and deck only after confirming", async () => {
    let archivedDecks: unknown[] = [{ id: "d9", name: "Old deck", description: "", totalCards: 3, dueCards: 0 }];
    let archivedCards: unknown[] = [
      {
        id: "c1",
        deckId: "d2",
        deckName: "Français",
        question: "Test card",
        answer: "Test answer",
        scheduling: { state: "new", dueAt: "2026-01-01T00:00:00Z", intervalDays: 0, repetitions: 0, lapses: 0 },
      },
    ];

    const noContent = () => Promise.resolve({ ok: true, status: 204, headers: new Headers(), json: async () => undefined });
    const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (url.includes("/api/v1/me")) return jsonResponse({ id: "u1", email: "user@example.com" });
      if (url.endsWith("/api/v1/archived-flashcards/c1") && init?.method === "DELETE") {
        archivedCards = [];
        return noContent();
      }
      if (url.endsWith("/api/v1/archived-decks/d9") && init?.method === "DELETE") {
        archivedDecks = [];
        return noContent();
      }
      if (url.endsWith("/api/v1/archived-flashcards")) return jsonResponse(archivedCards);
      if (url.includes("/api/v1/deck-groups")) return jsonResponse([]);
      if (url.includes("/api/v1/decks?archived=true")) return jsonResponse(archivedDecks);
      if (url.includes("/api/v1/decks")) return jsonResponse([]);
      return jsonResponse({}, false, 404);
    });
    vi.stubGlobal("fetch", fetchMock);

    render(
      <MemoryRouter>
        <AuthProvider>
          <DashboardPage />
        </AuthProvider>
      </MemoryRouter>,
    );

    await userEvent.click(await screen.findByRole("button", { name: /ver arquivados/i }));
    expect(await screen.findByText("Test card")).toBeInTheDocument();

    // Cancelling the confirmation deletes nothing.
    await userEvent.click(screen.getByRole("button", { name: "Excluir card" }));
    expect(screen.getByText("Este card será apagado para sempre.")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Cancelar" }));
    expect(screen.getByText("Test card")).toBeInTheDocument();
    expect(fetchMock).not.toHaveBeenCalledWith(expect.stringContaining("/archived-flashcards/c1"), expect.anything());

    // Confirming deletes the card...
    await userEvent.click(screen.getByRole("button", { name: "Excluir card" }));
    await userEvent.click(screen.getByRole("button", { name: "Excluir definitivamente" }));
    await waitFor(() => expect(screen.queryByText("Test card")).not.toBeInTheDocument());
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/api/v1/archived-flashcards/c1"),
      expect.objectContaining({ method: "DELETE" }),
    );

    // ...and the deck.
    await userEvent.click(screen.getByRole("button", { name: "Excluir coleção" }));
    await userEvent.click(screen.getByRole("button", { name: "Excluir definitivamente" }));
    await waitFor(() => expect(screen.queryByText("Old deck")).not.toBeInTheDocument());
    expect(screen.getByText("Nenhum item arquivado.")).toBeInTheDocument();
  });
  it("links to the settings page", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation((url: string) =>
        url.includes("/api/v1/me") ? jsonResponse({ id: "u1", email: "user@example.com" }) : jsonResponse([]),
      ),
    );

    render(
      <MemoryRouter>
        <AuthProvider>
          <DashboardPage />
        </AuthProvider>
      </MemoryRouter>,
    );

    const settings = await screen.findByRole("button", { name: "Configurações" });
    expect(settings.closest("a")).toHaveAttribute("href", "/settings");
    // The settings button is just an icon (its name comes from aria-label).
    expect(settings).not.toHaveTextContent("Configurações");
    expect(settings.querySelector("svg")).not.toBeNull();

    // "Sair" comes first (left), settings last (right) in the top bar.
    const logout = screen.getByRole("button", { name: "Sair" });
    expect(logout.compareDocumentPosition(settings) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });
});

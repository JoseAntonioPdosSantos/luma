import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { GroupPage } from "./GroupPage";

function jsonResponse(body: unknown, ok = true, status = 200) {
  return Promise.resolve({
    ok,
    status,
    headers: new Headers({ "content-type": "application/json" }),
    json: async () => body,
  });
}

function renderGroupPage() {
  return render(
    <MemoryRouter initialEntries={["/groups/g1"]}>
      <Routes>
        <Route path="/groups/:groupId" element={<GroupPage />} />
        <Route path="/dashboard" element={<p>Dashboard</p>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("GroupPage", () => {
  it("lists the group's decks and shows an empty state when it has none", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation((url: string) => {
        if (url.includes("/api/v1/deck-groups")) return jsonResponse([{ id: "g1", name: "Inglês" }]);
        if (url.includes("/api/v1/decks")) return jsonResponse([]);
        return jsonResponse({}, false, 404);
      }),
    );

    renderGroupPage();

    expect(await screen.findByRole("heading", { name: "Inglês" })).toBeInTheDocument();
    expect(screen.getByText("0 coleção")).toBeInTheDocument();
    expect(screen.getByText("Nenhuma coleção neste grupo ainda.")).toBeInTheDocument();
  });

  it("only lists decks that belong to this group", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation((url: string) => {
        if (url.includes("/api/v1/deck-groups")) return jsonResponse([{ id: "g1", name: "Inglês" }]);
        if (url.includes("/api/v1/decks")) {
          return jsonResponse([
            { id: "d1", name: "Phrasal verbs", description: "", totalCards: 5, dueCards: 1, groupId: "g1" },
            { id: "d2", name: "Outro grupo", description: "", totalCards: 2, dueCards: 0, groupId: "g2" },
            { id: "d3", name: "Sem grupo", description: "", totalCards: 1, dueCards: 0 },
          ]);
        }
        return jsonResponse({}, false, 404);
      }),
    );

    renderGroupPage();

    expect(await screen.findByText("Phrasal verbs")).toBeInTheDocument();
    expect(screen.queryByText("Outro grupo")).not.toBeInTheDocument();
    expect(screen.queryByText("Sem grupo")).not.toBeInTheDocument();
    expect(screen.getByText("1 coleção")).toBeInTheDocument();
  });

  it("renames the group", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation((url: string, init?: RequestInit) => {
        if (url.endsWith("/api/v1/deck-groups/g1") && init?.method === "PUT") {
          const { name } = JSON.parse(String(init.body));
          return jsonResponse({ id: "g1", name });
        }
        if (url.includes("/api/v1/deck-groups")) return jsonResponse([{ id: "g1", name: "Inglês" }]);
        if (url.includes("/api/v1/decks")) return jsonResponse([]);
        return jsonResponse({}, false, 404);
      }),
    );

    renderGroupPage();
    await screen.findByRole("heading", { name: "Inglês" });

    await userEvent.click(screen.getByRole("button", { name: "Renomear" }));
    const input = screen.getByLabelText("Nome do grupo");
    await userEvent.clear(input);
    await userEvent.type(input, "Inglês avançado");
    await userEvent.click(screen.getByRole("button", { name: "Salvar" }));

    expect(await screen.findByRole("heading", { name: "Inglês avançado" })).toBeInTheDocument();
  });

  it("deletes the group after confirming and returns to the dashboard", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation((url: string, init?: RequestInit) => {
        if (url.endsWith("/api/v1/deck-groups/g1") && init?.method === "DELETE") {
          return Promise.resolve({ ok: true, status: 204, headers: new Headers(), json: async () => undefined });
        }
        if (url.includes("/api/v1/deck-groups")) return jsonResponse([{ id: "g1", name: "Inglês" }]);
        if (url.includes("/api/v1/decks")) return jsonResponse([]);
        return jsonResponse({}, false, 404);
      }),
    );

    renderGroupPage();
    await screen.findByRole("heading", { name: "Inglês" });

    await userEvent.click(screen.getByRole("button", { name: "Excluir" }));
    await userEvent.click(screen.getByRole("button", { name: "Excluir grupo" }));

    expect(await screen.findByText("Dashboard")).toBeInTheDocument();
  });

  it("creates a deck already filed under this group", async () => {
    let decks: unknown[] = [];
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation((url: string, init?: RequestInit) => {
        if (url.includes("/api/v1/deck-groups")) return jsonResponse([{ id: "g1", name: "Inglês" }]);
        if (url.includes("/api/v1/decks") && init?.method === "POST") {
          const { name } = JSON.parse(String(init.body));
          const created = { id: "d1", name, description: "", totalCards: 0, dueCards: 0 };
          decks = [...decks, created];
          return jsonResponse(created, true, 201);
        }
        if (url.endsWith("/api/v1/decks/d1/group") && init?.method === "PUT") {
          decks = decks.map((d) => (d && typeof d === "object" && (d as { id: string }).id === "d1" ? { ...d, groupId: "g1" } : d));
          return Promise.resolve({ ok: true, status: 204, headers: new Headers(), json: async () => undefined });
        }
        if (url.includes("/api/v1/decks")) return jsonResponse(decks);
        return jsonResponse({}, false, 404);
      }),
    );

    renderGroupPage();
    await screen.findByRole("heading", { name: "Inglês" });

    await userEvent.click(screen.getByRole("button", { name: "Criar coleção" }));
    await userEvent.type(screen.getByLabelText("Nome"), "Verbos");
    await userEvent.click(screen.getByRole("button", { name: "Salvar coleção" }));

    await waitFor(() => expect(screen.getByText("Verbos")).toBeInTheDocument());
  });
});

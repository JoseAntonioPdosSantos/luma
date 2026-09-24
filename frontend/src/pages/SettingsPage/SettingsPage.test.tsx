import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import type { StudyProfile, StudyProfiles } from "../../types";
import { SettingsPage } from "./SettingsPage";

const DEFAULT_PROFILE: StudyProfile = {
  id: "default",
  name: "Padrão",
  isDefault: true,
  dailyCardLimit: null,
  rules: {
    againDelayMinutes: 10,
    hard: { firstIntervalDays: 1, multiplier: 1.2 },
    good: { firstIntervalDays: 3, multiplier: 2 },
    easy: { firstIntervalDays: 7, multiplier: 3 },
  },
};

const PROVA: StudyProfile = {
  id: "p1",
  name: "Prova",
  isDefault: false,
  dailyCardLimit: 30,
  rules: {
    againDelayMinutes: 5,
    hard: { firstIntervalDays: 1, multiplier: 1.1 },
    good: { firstIntervalDays: 2, multiplier: 1.8 },
    easy: { firstIntervalDays: 5, multiplier: 2.5 },
  },
};

function json(body: unknown, status = 200) {
  return Promise.resolve({
    ok: status < 400,
    status,
    headers: new Headers({ "content-type": "application/json" }),
    json: async () => body,
  });
}
const noContent = () => Promise.resolve({ ok: true, status: 204, headers: new Headers(), json: async () => undefined });

// A tiny in-memory copy of the study-profile API.
function backend(initial: StudyProfiles, studiedToday = 0) {
  const state: StudyProfiles = structuredClone(initial);
  let nextId = 10;
  const sent: { method: string; path: string; body?: unknown }[] = [];

  const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
    const path = new URL(url).pathname;
    const method = init?.method ?? "GET";
    const body = init?.body ? JSON.parse(String(init.body)) : undefined;
    if (method !== "GET") sent.push({ method, path, body });

    if (path === "/api/v1/study-profiles" && method === "GET") return json(state);
    if (path === "/api/v1/study-profiles" && method === "POST") {
      if (state.profiles.some((p) => p.name.trim().toLowerCase() === String(body.name).trim().toLowerCase())) {
        return json({ error: { code: "CONFLICT", message: "you already have a configuration with this name", requestId: "x" } }, 409);
      }
      const created: StudyProfile = { id: `p${nextId++}`, name: String(body.name).trim(), isDefault: false, dailyCardLimit: body.dailyCardLimit, rules: body.rules };
      state.profiles.push(created);
      return json(created, 201);
    }
    const one = path.match(/^\/api\/v1\/study-profiles\/(.+)$/);
    if (one && method === "PUT") {
      const p = state.profiles.find((x) => x.id === one[1])!;
      Object.assign(p, { name: String(body.name).trim(), dailyCardLimit: body.dailyCardLimit, rules: body.rules });
      return json(p);
    }
    if (one && method === "DELETE") {
      state.profiles = state.profiles.filter((x) => x.id !== one[1]);
      if (state.activeProfileId === one[1]) state.activeProfileId = "default";
      return noContent();
    }
    if (path === "/api/v1/me/active-study-profile" && method === "PUT") {
      state.activeProfileId = body.profileId;
      return noContent();
    }
    if (path === "/api/v1/study/progress") {
      const active = state.profiles.find((p) => p.id === state.activeProfileId)!;
      const limit = active.dailyCardLimit;
      return json({ dailyCardLimit: limit, studiedToday, remaining: limit === null ? null : Math.max(0, limit - studiedToday), continuingPastGoal: false });
    }
    return json({}, 404);
  });
  return { fetchMock, state, sent };
}

const INITIAL: StudyProfiles = { activeProfileId: "default", profiles: [DEFAULT_PROFILE] };
const WITH_PROVA: StudyProfiles = { activeProfileId: "default", profiles: [DEFAULT_PROFILE, PROVA] };

function renderSettings() {
  return render(
    <MemoryRouter>
      <SettingsPage />
    </MemoryRouter>,
  );
}

const select = () => screen.findByRole("combobox", { name: "Configuração ativa" });
const group = (name: string) => within(screen.getByRole("group", { name }));

async function typeInto(el: HTMLElement, text: string) {
  await userEvent.clear(el);
  if (text !== "") await userEvent.type(el, text);
}

describe("SettingsPage", () => {
  it("starts on the named default configuration and shows what it does", async () => {
    vi.stubGlobal("fetch", backend(INITIAL, 4).fetchMock);

    renderSettings();

    const picker = await select();
    expect(picker).toHaveValue("default");
    expect(within(picker).getAllByRole("option").map((o) => o.textContent)).toEqual(["Padrão"]);

    const summary = screen.getByLabelText("Resumo da configuração Padrão");
    expect(within(summary).getByText("Sem meta diária")).toBeInTheDocument();
    expect(within(summary).getByText("volta em 10 minutos")).toBeInTheDocument();
    expect(within(summary).getByText("1 dia para um card novo; depois × 1,2")).toBeInTheDocument();
    expect(within(summary).getByText("3 dias para um card novo; depois × 2")).toBeInTheDocument();
    expect(within(summary).getByText("7 dias para um card novo; depois × 3")).toBeInTheDocument();
    expect(screen.getByText(/hoje você estudou 4 cards\./i)).toBeInTheDocument();

    // The built-in one is fixed.
    expect(screen.getByText(/é fixa e não pode ser editada/i)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Editar" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Excluir" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Nova configuração" })).toBeInTheDocument();
  });

  it("switches the active configuration from the select", async () => {
    const api = backend(WITH_PROVA, 3);
    vi.stubGlobal("fetch", api.fetchMock);

    renderSettings();
    const picker = await select();
    expect(within(picker).getAllByRole("option").map((o) => o.textContent)).toEqual(["Padrão", "Prova"]);

    await userEvent.selectOptions(picker, "Prova");

    expect(await screen.findByText("Configuração ativa: Prova.")).toBeInTheDocument();
    expect(api.sent).toContainEqual({ method: "PUT", path: "/api/v1/me/active-study-profile", body: { profileId: "p1" } });
    expect(picker).toHaveValue("p1");
    const summary = screen.getByLabelText("Resumo da configuração Prova");
    expect(within(summary).getByText("30 cards por dia")).toBeInTheDocument();
    expect(within(summary).getByText("volta em 5 minutos")).toBeInTheDocument();
    expect(screen.getByText(/de uma meta de 30/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Editar" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Excluir" })).toBeInTheDocument();

    // And back to the default.
    await userEvent.selectOptions(picker, "Padrão");
    await screen.findByText("Configuração ativa: Padrão.");
    expect(api.state.activeProfileId).toBe("default");
  });

  it("creates a named configuration, prefilled with the default values, without switching to it", async () => {
    const api = backend(INITIAL);
    vi.stubGlobal("fetch", api.fetchMock);

    renderSettings();
    await userEvent.click(await screen.findByRole("button", { name: "Nova configuração" }));

    // The form starts from the built-in values.
    expect(screen.getByLabelText("Voltar a aparecer em (minutos)")).toHaveValue(10);
    expect(group("Lembrei com esforço").getByLabelText("Dias para um card novo")).toHaveValue(1);
    expect(group("Lembrei com esforço").getByLabelText("Multiplicador nas revisões seguintes")).toHaveValue("1,2");
    expect(group("Lembrei facilmente").getByLabelText("Dias para um card novo")).toHaveValue(7);
    expect(screen.queryByLabelText("Cards por dia")).not.toBeInTheDocument();

    // Without a name it cannot be saved.
    await userEvent.click(screen.getByRole("button", { name: "Salvar configuração" }));
    expect(await screen.findByText("Dê um nome para a configuração.")).toBeInTheDocument();
    expect(api.sent).toEqual([]);

    await userEvent.type(screen.getByLabelText("Nome da configuração"), "Prova de inglês");
    await userEvent.click(screen.getByLabelText("Definir uma meta diária"));
    await typeInto(screen.getByLabelText("Cards por dia"), "30");
    await typeInto(screen.getByLabelText("Voltar a aparecer em (minutos)"), "5");
    await typeInto(group("Lembrei").getByLabelText("Dias para um card novo"), "4");
    await typeInto(group("Lembrei").getByLabelText("Multiplicador nas revisões seguintes"), "2,5");
    await typeInto(group("Lembrei facilmente").getByLabelText("Multiplicador nas revisões seguintes"), "3.5");
    await userEvent.click(screen.getByRole("button", { name: "Salvar configuração" }));

    expect(await screen.findByText(/configuração “Prova de inglês” criada/i)).toBeInTheDocument();
    expect(api.sent).toEqual([
      {
        method: "POST",
        path: "/api/v1/study-profiles",
        body: {
          name: "Prova de inglês",
          dailyCardLimit: 30,
          rules: {
            againDelayMinutes: 5,
            hard: { firstIntervalDays: 1, multiplier: 1.2 },
            good: { firstIntervalDays: 4, multiplier: 2.5 },
            easy: { firstIntervalDays: 7, multiplier: 3.5 },
          },
        },
      },
    ]);

    // It is in the list, but the default is still the selected one until the user chooses.
    const picker = screen.getByRole("combobox", { name: "Configuração ativa" });
    expect(within(picker).getAllByRole("option").map((o) => o.textContent)).toEqual(["Padrão", "Prova de inglês"]);
    expect(picker).toHaveValue("default");
    expect(screen.queryByRole("form", { name: "Nova configuração" })).not.toBeInTheDocument();
  });

  it("shows what is wrong instead of saving invalid values", async () => {
    const api = backend(INITIAL);
    vi.stubGlobal("fetch", api.fetchMock);

    renderSettings();
    await userEvent.click(await screen.findByRole("button", { name: "Nova configuração" }));
    await userEvent.type(screen.getByLabelText("Nome da configuração"), "Inválida");
    await userEvent.click(screen.getByLabelText("Definir uma meta diária"));
    await typeInto(screen.getByLabelText("Cards por dia"), "500");
    await typeInto(screen.getByLabelText("Voltar a aparecer em (minutos)"), "0");
    await typeInto(group("Lembrei").getByLabelText("Dias para um card novo"), "9"); // Lembrei facilmente is 7
    await typeInto(group("Lembrei com esforço").getByLabelText("Multiplicador nas revisões seguintes"), "9");
    await userEvent.click(screen.getByRole("button", { name: "Salvar configuração" }));

    expect(await screen.findByText(/número inteiro entre 1 e 200/i)).toBeInTheDocument();
    expect(screen.getByText(/de 1 a 1440 minutos/i)).toBeInTheDocument();
    expect(screen.getByText(/os dias devem aumentar/i)).toBeInTheDocument();
    expect(screen.getByText(/valor de 1 a 5/i)).toBeInTheDocument();
    expect(api.sent).toEqual([]);
  });

  it("tells the user when the name is already taken and keeps the form open", async () => {
    const api = backend(WITH_PROVA);
    vi.stubGlobal("fetch", api.fetchMock);

    renderSettings();
    await userEvent.click(await screen.findByRole("button", { name: "Nova configuração" }));
    await userEvent.type(screen.getByLabelText("Nome da configuração"), "prova");
    await userEvent.click(screen.getByRole("button", { name: "Salvar configuração" }));

    expect(await screen.findByText("Você já tem uma configuração com esse nome.")).toBeInTheDocument();
    expect(screen.getByRole("form", { name: "Nova configuração" })).toBeInTheDocument();
  });

  it("restores the default values in the form but keeps the name", async () => {
    vi.stubGlobal("fetch", backend(INITIAL).fetchMock);

    renderSettings();
    await userEvent.click(await screen.findByRole("button", { name: "Nova configuração" }));
    await userEvent.type(screen.getByLabelText("Nome da configuração"), "Minha");
    await typeInto(screen.getByLabelText("Voltar a aparecer em (minutos)"), "99");
    await typeInto(group("Lembrei").getByLabelText("Dias para um card novo"), "5");

    await userEvent.click(screen.getByRole("button", { name: "Restaurar valores padrão" }));

    expect(screen.getByLabelText("Voltar a aparecer em (minutos)")).toHaveValue(10);
    expect(group("Lembrei").getByLabelText("Dias para um card novo")).toHaveValue(3);
    expect(screen.getByLabelText("Nome da configuração")).toHaveValue("Minha");
  });

  it("edits the selected custom configuration", async () => {
    const api = backend({ ...WITH_PROVA, activeProfileId: "p1" });
    vi.stubGlobal("fetch", api.fetchMock);

    renderSettings();
    await userEvent.click(await screen.findByRole("button", { name: "Editar" }));

    expect(screen.getByLabelText("Nome da configuração")).toHaveValue("Prova");
    expect(screen.getByLabelText("Definir uma meta diária")).toBeChecked();
    expect(screen.getByLabelText("Cards por dia")).toHaveValue(30);
    expect(screen.getByLabelText("Voltar a aparecer em (minutos)")).toHaveValue(5);

    await typeInto(screen.getByLabelText("Nome da configuração"), "Prova final");
    await userEvent.click(screen.getByLabelText("Definir uma meta diária")); // switch the goal off
    await userEvent.click(screen.getByRole("button", { name: "Salvar configuração" }));

    expect(await screen.findByText("Alterações salvas.")).toBeInTheDocument();
    expect(api.sent).toHaveLength(1);
    expect(api.sent[0]).toMatchObject({ method: "PUT", path: "/api/v1/study-profiles/p1", body: { name: "Prova final", dailyCardLimit: null } });
    const summary = screen.getByLabelText("Resumo da configuração Prova final");
    expect(within(summary).getByText("Sem meta diária")).toBeInTheDocument();
  });

  it("deletes a configuration only after confirming, and goes back to the default", async () => {
    const api = backend({ ...WITH_PROVA, activeProfileId: "p1" });
    vi.stubGlobal("fetch", api.fetchMock);

    renderSettings();
    await userEvent.click(await screen.findByRole("button", { name: "Excluir" }));
    expect(screen.getByText(/excluir “Prova”\?/i)).toBeInTheDocument();

    // Cancelling changes nothing.
    await userEvent.click(screen.getByRole("button", { name: "Cancelar" }));
    expect(api.sent).toEqual([]);

    await userEvent.click(screen.getByRole("button", { name: "Excluir" }));
    await userEvent.click(screen.getByRole("button", { name: "Excluir definitivamente" }));

    expect(await screen.findByText(/configuração “Prova” excluída/i)).toBeInTheDocument();
    expect(api.sent).toEqual([{ method: "DELETE", path: "/api/v1/study-profiles/p1", body: undefined }]);
    const picker = screen.getByRole("combobox", { name: "Configuração ativa" });
    expect(picker).toHaveValue("default");
    expect(within(picker).getAllByRole("option")).toHaveLength(1);
  });

  it("reports a failure to switch and keeps the previous configuration", async () => {
    const api = backend(WITH_PROVA);
    const failing = vi.fn().mockImplementation((url: string, init?: RequestInit) =>
      init?.method === "PUT" && url.includes("active-study-profile")
        ? json({ error: { code: "NOT_FOUND", key: "studyProfile.notFound", message: "configuration not found", requestId: "x" } }, 404)
        : api.fetchMock(url, init),
    );
    vi.stubGlobal("fetch", failing);

    renderSettings();
    const picker = await select();
    await userEvent.selectOptions(picker, "Prova");

    // The backend's error key is translated, rather than showing its raw
    // (always-English) message.
    expect(await screen.findByRole("alert")).toHaveTextContent(/configuração não encontrada/i);
    await waitFor(() => expect(picker).toHaveValue("default"));
  });

  it("shows an error when the configurations cannot be loaded", async () => {
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => json({}, 500)));

    renderSettings();

    expect(await screen.findByText(/não foi possível carregar as configurações/i)).toBeInTheDocument();
  });
  it("groups save and cancel at the bottom of the form and keeps restore next to the values", async () => {
    vi.stubGlobal("fetch", backend(INITIAL).fetchMock);

    renderSettings();
    await userEvent.click(await screen.findByRole("button", { name: "Nova configuração" }));

    const form = screen.getByRole("form", { name: "Nova configuração" });
    const buttons = within(form).getAllByRole("button");
    const names = buttons.map((b) => b.textContent);
    // Restore comes first (above the values); cancel and save close the form, save last.
    expect(names).toEqual(["Restaurar valores padrão", "Cancelar", "Salvar configuração"]);

    const restore = within(form).getByRole("button", { name: "Restaurar valores padrão" });
    const firstRating = within(form).getByRole("group", { name: "Não lembrei" });
    const actions = within(form).getByRole("button", { name: "Salvar configuração" }).parentElement!;
    expect(restore.compareDocumentPosition(firstRating) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(actions).toContainElement(within(form).getByRole("button", { name: "Cancelar" }));
    expect(actions).not.toContainElement(restore);
  });
  describe("the back link", () => {
    function renderFrom(state?: unknown) {
      return render(
        <MemoryRouter initialEntries={[{ pathname: "/settings", state }]}>
          <SettingsPage />
        </MemoryRouter>,
      );
    }

    it("goes back to the deck the user came from", async () => {
      vi.stubGlobal("fetch", backend(INITIAL).fetchMock);

      renderFrom({ from: "/decks/6ab04ab9" });

      const link = await screen.findByRole("link", { name: /voltar à coleção/i });
      expect(link).toHaveAttribute("href", "/decks/6ab04ab9");
      expect(screen.queryByRole("link", { name: /voltar ao painel/i })).not.toBeInTheDocument();
    });

    it("goes back to the dashboard when opened from there or from nowhere in particular", async () => {
      vi.stubGlobal("fetch", backend(INITIAL).fetchMock);

      renderFrom();

      expect(await screen.findByRole("link", { name: /voltar ao painel/i })).toHaveAttribute("href", "/dashboard");
    });

    it("ignores a return address that is not an in-app path", async () => {
      vi.stubGlobal("fetch", backend(INITIAL).fetchMock);

      renderFrom({ from: "https://evil.example/decks/1" });

      expect(await screen.findByRole("link", { name: /voltar ao painel/i })).toHaveAttribute("href", "/dashboard");
    });

    it("also offers the way back when the settings cannot be loaded", async () => {
      vi.stubGlobal("fetch", vi.fn().mockImplementation(() => json({}, 500)));

      renderFrom({ from: "/decks/6ab04ab9" });

      expect(await screen.findByText(/não foi possível carregar as configurações/i)).toBeInTheDocument();
      expect(screen.getByRole("link", { name: "Voltar à coleção" })).toHaveAttribute("href", "/decks/6ab04ab9");
    });
  });
});

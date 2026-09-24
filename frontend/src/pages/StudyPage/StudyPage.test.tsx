import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { StudyPage } from "./StudyPage";

function jsonResponse(body: unknown) {
  return Promise.resolve({
    ok: true,
    status: 200,
    headers: new Headers({ "content-type": "application/json" }),
    json: async () => body,
  });
}

const dueCard = {
  id: "card-1",
  deckId: "deck-1",
  question: "How to say 'resgatar'?",
  answer: "rescue",
  scheduling: { state: "new", dueAt: "2026-01-01T00:00:00Z", intervalDays: 0, repetitions: 0, lapses: 0 },
};

function renderStudyPage() {
  return render(
    <MemoryRouter initialEntries={["/decks/deck-1/study"]}>
      <Routes>
        <Route path="/decks/:deckId/study" element={<StudyPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("StudyPage", () => {
  it("shows an empty state when there is nothing due", async () => {
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => jsonResponse([])));

    renderStudyPage();

    expect(await screen.findByText(/nenhum cartão para revisar/i)).toBeInTheDocument();
  });

  it("hides the answer and rating buttons until revealed, then submits a rating", async () => {
    const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (init?.method === "POST" && url.includes("/reviews")) {
        return jsonResponse({ ...dueCard, scheduling: { ...dueCard.scheduling, state: "review" } });
      }
      return jsonResponse([dueCard]);
    });
    vi.stubGlobal("fetch", fetchMock);

    renderStudyPage();

    expect(await screen.findByText("How to say 'resgatar'?")).toBeInTheDocument();
    expect(screen.queryByText("rescue")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Lembrei" })).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Mostrar resposta" }));

    expect(screen.getByText("rescue")).toBeInTheDocument();
    const goodButton = screen.getByRole("button", { name: "Lembrei" });

    await userEvent.click(goodButton);

    await waitFor(() => expect(screen.getByText("Sessão concluída!")).toBeInTheDocument());
    const studiedLabel = screen.getByText("Estudados");
    expect(studiedLabel.previousElementSibling).toHaveTextContent("1");
  });
  it("archives the current card after confirmation and skips to the next one", async () => {
    const secondCard = { ...dueCard, id: "card-2", question: "How to say 'sair'?", answer: "leave" };
    const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (init?.method === "DELETE" && url.endsWith("/flashcards/card-1")) {
        return Promise.resolve({ ok: true, status: 204, headers: new Headers(), json: async () => undefined });
      }
      return jsonResponse([dueCard, secondCard]);
    });
    vi.stubGlobal("fetch", fetchMock);

    renderStudyPage();

    expect(await screen.findByText("How to say 'resgatar'?")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: /não quero mais ver este cartão/i }));
    await userEvent.click(screen.getByRole("button", { name: "Arquivar" }));

    expect(await screen.findByText("How to say 'sair'?")).toBeInTheDocument();
    expect(screen.queryByText("How to say 'resgatar'?")).not.toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/flashcards/card-1"),
      expect.objectContaining({ method: "DELETE" }),
    );
  });
  it("shows the card's own hint, and hides it once the answer is shown", async () => {
    const cardWithHint = { ...dueCard, id: "card-h", hint: "Starts with R" };
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => jsonResponse([cardWithHint])));

    renderStudyPage();

    expect(await screen.findByText("How to say 'resgatar'?")).toBeInTheDocument();
    expect(screen.queryByText("Starts with R")).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: /ver dica/i }));
    expect(screen.getByText("Starts with R")).toBeInTheDocument();
    expect(screen.queryByText(/dica automática/i)).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: /ocultar dica/i }));
    expect(screen.queryByText("Starts with R")).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: /ver dica/i }));
    await userEvent.click(screen.getByRole("button", { name: "Mostrar resposta" }));
    expect(screen.queryByText("Starts with R")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /dica/i })).not.toBeInTheDocument();
  });

  it("falls back to an automatic hint for a card without one", async () => {
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => jsonResponse([dueCard])));

    renderStudyPage();

    expect(await screen.findByText("How to say 'resgatar'?")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: /ver dica/i }));

    expect(screen.getByText("r _ _ _ _ _")).toBeInTheDocument();
    expect(screen.getByText(/dica automática/i)).toBeInTheDocument();
  });

  it("toggles the hint with the H key", async () => {
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => jsonResponse([{ ...dueCard, hint: "Starts with R" }])));

    renderStudyPage();

    expect(await screen.findByText("How to say 'resgatar'?")).toBeInTheDocument();
    await userEvent.keyboard("h");
    expect(screen.getByText("Starts with R")).toBeInTheDocument();
    await userEvent.keyboard("h");
    expect(screen.queryByText("Starts with R")).not.toBeInTheDocument();
  });

  it("reports whether the hint was used when rating a card", async () => {
    const second = { ...dueCard, id: "card-2", question: "Second?", answer: "leave" };
    const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (init?.method === "POST" && url.includes("/reviews")) return jsonResponse(dueCard);
      return jsonResponse([dueCard, second]);
    });
    vi.stubGlobal("fetch", fetchMock);

    renderStudyPage();

    // First card: look at the hint, then answer.
    expect(await screen.findByText("How to say 'resgatar'?")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: /ver dica/i }));
    await userEvent.click(screen.getByRole("button", { name: "Mostrar resposta" }));
    await userEvent.click(screen.getByRole("button", { name: "Lembrei" }));

    // Second card: answer without opening the hint.
    expect(await screen.findByText("Second?")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Mostrar resposta" }));
    await userEvent.click(screen.getByRole("button", { name: "Lembrei" }));

    await waitFor(() => expect(screen.getByText("Sessão concluída!")).toBeInTheDocument());
    const bodies = fetchMock.mock.calls
      .filter(([url, init]) => String(url).includes("/reviews") && init?.method === "POST")
      .map(([, init]) => JSON.parse(init.body));
    expect(bodies.map((b) => b.hintUsed)).toEqual([true, false]);
  });
  it("blocks \"Lembrei facilmente\" only for a card whose hint was used", async () => {
    const second = { ...dueCard, id: "card-2", question: "Second?", answer: "leave" };
    const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (init?.method === "POST" && url.includes("/reviews")) return jsonResponse(dueCard);
      return jsonResponse([dueCard, second]);
    });
    vi.stubGlobal("fetch", fetchMock);

    renderStudyPage();

    // First card: open the hint, then even closing it keeps "Lembrei facilmente" blocked.
    expect(await screen.findByText("How to say 'resgatar'?")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: /ver dica/i }));
    await userEvent.click(screen.getByRole("button", { name: /ocultar dica/i }));
    await userEvent.click(screen.getByRole("button", { name: "Mostrar resposta" }));

    expect(screen.getByRole("button", { name: "Lembrei facilmente" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Lembrei" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Lembrei com esforço" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Não lembrei" })).toBeEnabled();
    expect(screen.getByText(/você usou a dica/i)).toBeInTheDocument();

    // The keyboard shortcut for "Lembrei facilmente" (4) is blocked too.
    await userEvent.keyboard("4");
    expect(fetchMock).not.toHaveBeenCalledWith(expect.stringContaining("/reviews"), expect.anything());

    await userEvent.click(screen.getByRole("button", { name: "Lembrei" }));

    // Second card: the hint was not used, so "Lembrei facilmente" is available again.
    expect(await screen.findByText("Second?")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Mostrar resposta" }));
    expect(screen.getByRole("button", { name: "Lembrei facilmente" })).toBeEnabled();
    expect(screen.queryByText(/você usou a dica/i)).not.toBeInTheDocument();
  });
  describe("daily card goal", () => {
    const progressUrl = "/api/v1/study/progress";

    // Backend stub: due cards come from `dueByCall` (one list per call to
    // due-flashcards); progress answers come from `progressSteps`, one per
    // call (the last one repeats). Once the user POSTs "continue past goal"
    // every later progress answer says so, as the real server does.
    function goalBackend(dueByCall: unknown[][], progressSteps: object[]) {
      let dueCalls = 0;
      let progressCalls = 0;
      let continued = false;
      return vi.fn().mockImplementation((url: string, init?: RequestInit) => {
        if (init?.method === "POST" && url.includes("/reviews")) return jsonResponse(dueCard);
        if (init?.method === "POST" && url.includes("/study/continue-past-goal")) {
          continued = true;
          return Promise.resolve({ ok: true, status: 204, headers: new Headers(), json: async () => undefined });
        }
        if (url.includes(progressUrl)) {
          const step = progressSteps[Math.min(progressCalls++, progressSteps.length - 1)];
          return jsonResponse({ continuingPastGoal: false, ...step, ...(continued ? { continuingPastGoal: true } : {}) });
        }
        return jsonResponse(dueByCall[Math.min(dueCalls++, dueByCall.length - 1)]);
      });
    }

    function postedUrls(fetchMock: ReturnType<typeof vi.fn>) {
      return fetchMock.mock.calls.filter(([, init]) => init?.method === "POST").map(([url]) => String(url));
    }

    it("shows nothing about a goal when the user has none (the default)", async () => {
      vi.stubGlobal("fetch", goalBackend([[dueCard]], [{ dailyCardLimit: null, studiedToday: 3, remaining: null }]));

      renderStudyPage();

      expect(await screen.findByText("How to say 'resgatar'?")).toBeInTheDocument();
      expect(screen.queryByText(/meta de hoje/i)).not.toBeInTheDocument();
    });

    it("shows the progress towards the goal while studying", async () => {
      vi.stubGlobal("fetch", goalBackend([[dueCard]], [{ dailyCardLimit: 20, studiedToday: 12, remaining: 8 }]));

      renderStudyPage();

      expect(await screen.findByText("Meta de hoje: 12 de 20 cards")).toBeInTheDocument();
      expect(screen.getByRole("progressbar", { name: "Meta de hoje" })).toHaveAttribute("aria-valuenow", "12");
    });

    it("congratulates the user when the last card reaches the goal and lets them continue studying", async () => {
      const fetchMock = goalBackend(
        [[dueCard], [dueCard]],
        [
          { dailyCardLimit: 5, studiedToday: 4, remaining: 1 }, // when the session starts
          { dailyCardLimit: 5, studiedToday: 5, remaining: 0 }, // after the last answer
          { dailyCardLimit: 5, studiedToday: 5, remaining: 0 },
        ],
      );
      vi.stubGlobal("fetch", fetchMock);

      renderStudyPage();
      await userEvent.click(await screen.findByRole("button", { name: "Mostrar resposta" }));
      await userEvent.click(screen.getByRole("button", { name: "Lembrei" }));

      expect(await screen.findByText("Meta do dia concluída!")).toBeInTheDocument();
      expect(screen.getByText(/você estudou 5 cards hoje/i)).toBeInTheDocument();

      // "Continuar estudando" saves the choice on the account and loads more cards.
      await userEvent.click(screen.getByRole("button", { name: "Continuar estudando" }));
      await screen.findByText("How to say 'resgatar'?");
      const continueCalls = postedUrls(fetchMock).filter((u) => u.includes("/study/continue-past-goal"));
      expect(continueCalls).toHaveLength(1);
      expect(continueCalls[0]).toContain(`tz=${encodeURIComponent(Intl.DateTimeFormat().resolvedOptions().timeZone)}`);
    });

    it("explains that the goal was already reached when no card is served", async () => {
      const fetchMock = goalBackend([[], [dueCard]], [{ dailyCardLimit: 10, studiedToday: 10, remaining: 0 }]);
      vi.stubGlobal("fetch", fetchMock);

      renderStudyPage();

      // Same layout as the session summary: icon, title, numbers and buttons.
      expect(await screen.findByRole("heading", { name: "Meta do dia concluída!" })).toBeInTheDocument();
      expect(screen.getByText(/você já estudou 10 cards hoje/i)).toBeInTheDocument();
      expect(screen.getByText("Meta").previousElementSibling).toHaveTextContent("10");
      expect(screen.getByText("Estudados hoje").previousElementSibling).toHaveTextContent("10");
      expect(screen.getByRole("button", { name: "Voltar à coleção" }).closest("a")).toHaveAttribute("href", "/decks/deck-1");
      expect(screen.queryByText(/nenhum cartão para revisar/i)).not.toBeInTheDocument();

      await userEvent.click(screen.getByRole("button", { name: "Continuar estudando" }));
      expect(await screen.findByText("How to say 'resgatar'?")).toBeInTheDocument();
      expect(postedUrls(fetchMock).some((u) => u.includes("/study/continue-past-goal"))).toBe(true);
    });

    it("keeps the usual empty message when there is simply nothing due", async () => {
      vi.stubGlobal("fetch", goalBackend([[]], [{ dailyCardLimit: 10, studiedToday: 3, remaining: 7 }]));

      renderStudyPage();

      expect(await screen.findByText(/nenhum cartão para revisar/i)).toBeInTheDocument();
      expect(screen.queryByText(/atingiu a meta/i)).not.toBeInTheDocument();
    });

    it("does not stop at the goal when the user already chose to continue today, on any device", async () => {
      // The server says so: the choice was saved on the account (for example
      // from the phone) and this is another browser.
      const fetchMock = goalBackend(
        [[dueCard]],
        [{ dailyCardLimit: 10, studiedToday: 25, remaining: 0, continuingPastGoal: true }],
      );
      vi.stubGlobal("fetch", fetchMock);

      renderStudyPage();

      // The study starts straight away, with no "goal reached" stop or button.
      expect(await screen.findByText("How to say 'resgatar'?")).toBeInTheDocument();
      expect(screen.queryByText(/atingiu a meta/i)).not.toBeInTheDocument();
      expect(screen.getByText("Meta de hoje: 25 de 10 cards")).toBeInTheDocument();
      expect(postedUrls(fetchMock).some((u) => u.includes("/study/continue-past-goal"))).toBe(false);

      // Finishing shows the regular summary, not "Meta do dia concluída!".
      await userEvent.click(screen.getByRole("button", { name: "Mostrar resposta" }));
      await userEvent.click(screen.getByRole("button", { name: "Lembrei" }));
      expect(await screen.findByText("Sessão concluída!")).toBeInTheDocument();
      expect(screen.queryByText("Meta do dia concluída!")).not.toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Estudar mais" })).toBeInTheDocument();
    });

    it("shows an error if the choice to continue could not be saved", async () => {
      const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
        if (init?.method === "POST") return Promise.resolve({ ok: false, status: 500, headers: new Headers(), json: async () => ({}) });
        if (url.includes(progressUrl)) return jsonResponse({ dailyCardLimit: 3, studiedToday: 3, remaining: 0, continuingPastGoal: false });
        return jsonResponse([]);
      });
      vi.stubGlobal("fetch", fetchMock);

      renderStudyPage();
      await userEvent.click(await screen.findByRole("button", { name: "Continuar estudando" }));

      expect(await screen.findByText(/não foi possível carregar os cartões/i)).toBeInTheDocument();
    });

    it("scopes the goal and the choice to continue to the deck being studied", async () => {
      const fetchMock = goalBackend(
        [[], [dueCard]],
        [{ dailyCardLimit: 3, studiedToday: 3, remaining: 0, continuingPastGoal: false }],
      );
      vi.stubGlobal("fetch", fetchMock);

      renderStudyPage();
      await userEvent.click(await screen.findByRole("button", { name: "Continuar estudando" }));
      await screen.findByText("How to say 'resgatar'?");

      const urls = fetchMock.mock.calls.map(([url]) => String(url));
      const progressUrls = urls.filter((u) => u.includes(progressUrl));
      expect(progressUrls.length).toBeGreaterThan(0);
      for (const u of progressUrls) expect(u).toContain("deckId=deck-1");
      const continueUrl = urls.find((u) => u.includes("/study/continue-past-goal"));
      expect(continueUrl).toContain("deckId=deck-1");
    });

    it("sends the browser's time zone so 'today' is the user's own day", async () => {
      const fetchMock = goalBackend([[dueCard]], [{ dailyCardLimit: null, studiedToday: 0, remaining: null }]);
      vi.stubGlobal("fetch", fetchMock);

      renderStudyPage();
      await screen.findByText("How to say 'resgatar'?");

      const zone = encodeURIComponent(Intl.DateTimeFormat().resolvedOptions().timeZone);
      const urls = fetchMock.mock.calls.map(([url]) => String(url));
      expect(urls.find((u) => u.includes("/due-flashcards"))).toContain(`tz=${zone}`);
      expect(urls.find((u) => u.includes(progressUrl))).toContain(`tz=${zone}`);
    });
  });
  it("reveals the answer with Space or Enter and shows the shortcut on the button", async () => {
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => jsonResponse([dueCard, { ...dueCard, id: "card-2", question: "Second?", answer: "leave" }])));

    renderStudyPage();

    const button = await screen.findByRole("button", { name: "Mostrar resposta" });
    // The key cap is visible text but not part of the accessible name.
    expect(button).toHaveTextContent("Espaço");
    expect(button).toHaveAttribute("aria-keyshortcuts", "Space Enter");

    await userEvent.keyboard(" ");
    expect(screen.getByText("rescue")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Lembrei" }));
    expect(await screen.findByText("Second?")).toBeInTheDocument();
    expect(screen.queryByText("leave")).not.toBeInTheDocument();

    await userEvent.keyboard("{Enter}");
    expect(screen.getByText("leave")).toBeInTheDocument();
  });
  describe("everything up to date", () => {
    // Backend stub for a deck whose cards are all scheduled for later.
    function upToDateBackend(deck: unknown, progress: object = { dailyCardLimit: null, studiedToday: 0, remaining: null }) {
      return vi.fn().mockImplementation((url: string) => {
        if (url.includes("/due-flashcards")) return jsonResponse([]);
        if (url.includes("/api/v1/study/progress")) return jsonResponse(progress);
        if (url.endsWith("/api/v1/decks/deck-1")) return jsonResponse(deck);
        return jsonResponse({});
      });
    }

    it("tells the user when the next review comes up and offers to add a card", async () => {
      // Local noon two calendar days from now: robust whatever the time of day.
      const inTwoDays = new Date();
      inTwoDays.setDate(inTwoDays.getDate() + 2);
      inTwoDays.setHours(12, 0, 0, 0);
      vi.stubGlobal(
        "fetch",
        upToDateBackend({ id: "deck-1", name: "English", description: "", totalCards: 5, dueCards: 0, nextDueAt: inTwoDays.toISOString() }),
      );

      renderStudyPage();

      expect(await screen.findByRole("heading", { name: "Tudo em dia!" })).toBeInTheDocument();
      expect(screen.getByText(/sua próxima revisão é em 2 dias \(\d\d\/\d\d, às 12:00\)/i)).toBeInTheDocument();
      expect(screen.getByText("Para revisar agora").previousElementSibling).toHaveTextContent("0");
      expect(screen.getByText("Próxima revisão").previousElementSibling).toHaveTextContent("em 2 dias");
      expect(screen.getByText("Cards na coleção").previousElementSibling).toHaveTextContent("5");
      expect(screen.getByRole("button", { name: "Voltar à coleção" }).closest("a")).toHaveAttribute("href", "/decks/deck-1");
      expect(screen.getByRole("button", { name: "Adicionar flashcard" }).closest("a")).toHaveAttribute(
        "href",
        "/decks/deck-1/flashcards/new",
      );
      expect(screen.queryByText(/nenhum cartão para revisar/i)).not.toBeInTheDocument();
    });

    it("still says everything is up to date if the next date is unknown", async () => {
      vi.stubGlobal("fetch", upToDateBackend({ id: "deck-1", name: "English", description: "", totalCards: 3, dueCards: 0 }));

      renderStudyPage();

      expect(await screen.findByRole("heading", { name: "Tudo em dia!" })).toBeInTheDocument();
      expect(screen.getByText(/não há nada para revisar agora/i)).toBeInTheDocument();
    });

    it("keeps the plain empty message for a deck without any card", async () => {
      vi.stubGlobal("fetch", upToDateBackend({ id: "deck-1", name: "English", description: "", totalCards: 0, dueCards: 0 }));

      renderStudyPage();

      expect(await screen.findByText(/nenhum cartão para revisar agora/i)).toBeInTheDocument();
      expect(screen.queryByText("Tudo em dia!")).not.toBeInTheDocument();
    });

    it("gives the daily-goal screen priority over this one", async () => {
      vi.stubGlobal(
        "fetch",
        upToDateBackend(
          { id: "deck-1", name: "English", description: "", totalCards: 5, dueCards: 3 },
          { dailyCardLimit: 2, studiedToday: 2, remaining: 0, continuingPastGoal: false },
        ),
      );

      renderStudyPage();

      expect(await screen.findByRole("heading", { name: "Meta do dia concluída!" })).toBeInTheDocument();
      expect(screen.queryByText("Tudo em dia!")).not.toBeInTheDocument();
    });
  });
});

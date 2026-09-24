import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { StatsPage } from "./StatsPage";

// The page draws the chart for the browser's local days, so the tests must
// build "today" the same way.
function localToday(): string {
  const d = new Date();
  const month = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${d.getFullYear()}-${month}-${day}`;
}

function statsResponse(dailyPerformance: unknown[], studied = { studiedToday: 0, studiedLast7Days: 0, studiedTotal: 0 }) {
  return Promise.resolve({
    ok: true,
    status: 200,
    headers: new Headers({ "content-type": "application/json" }),
    json: async () => ({
      deckId: "d1",
      name: "English",
      totalCards: 10,
      newCards: 2,
      learningCards: 3,
      masteredCards: 5,
      masteryPercent: 50,
      ...studied,
      dailyPerformance,
    }),
  });
}

function renderStats() {
  return render(
    <MemoryRouter initialEntries={["/decks/d1/stats"]}>
      <Routes>
        <Route path="/decks/:deckId/stats" element={<StatsPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("StatsPage", () => {
  it("reports how often the hint was used in the last 7 days", async () => {
    const today = localToday();
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation(() =>
        statsResponse([{ date: today, reviewCount: 8, accuracyPercent: 75, hintCount: 2 }]),
      ),
    );

    renderStats();

    expect(await screen.findByText(/dica usada em 2 de 8 revisões \(25%\)/i)).toBeInTheDocument();
  });

  it("does not mention hints when there were no reviews", async () => {
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => statsResponse([])));

    renderStats();

    expect(await screen.findByText(/nenhuma revisão nos últimos 7 dias/i)).toBeInTheDocument();
    expect(screen.queryByText(/dica usada/i)).not.toBeInTheDocument();
  });
  it("shows how many distinct cards were studied today, in the last 7 days and in total", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation(() => statsResponse([], { studiedToday: 12, studiedLast7Days: 45, studiedTotal: 210 })),
    );

    renderStats();

    expect(await screen.findByText("Cards estudados")).toBeInTheDocument();
    expect(screen.getByText("Hoje").nextElementSibling).toHaveTextContent("12");
    expect(screen.getByText("Últimos 7 dias").nextElementSibling).toHaveTextContent("45");
    expect(screen.getByText("Total").nextElementSibling).toHaveTextContent("210");
  });

  it("asks the server for the stats in the browser's time zone", async () => {
    const fetchMock = vi.fn().mockImplementation(() => statsResponse([]));
    vi.stubGlobal("fetch", fetchMock);

    renderStats();
    await screen.findByText("Cards estudados");

    const zone = Intl.DateTimeFormat().resolvedOptions().timeZone;
    expect(String(fetchMock.mock.calls[0][0])).toContain(`/api/v1/decks/d1/stats?tz=${encodeURIComponent(zone)}`);
  });
});

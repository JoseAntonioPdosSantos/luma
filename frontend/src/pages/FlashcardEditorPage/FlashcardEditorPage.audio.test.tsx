import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { FlashcardEditorPage } from "./FlashcardEditorPage";

function jsonResponse(body: unknown) {
  return Promise.resolve({
    ok: true,
    status: 200,
    headers: new Headers({ "content-type": "application/json" }),
    json: async () => body,
  });
}

const baseCard = {
  id: "card-1",
  deckId: "deck-1",
  question: "Q",
  answer: "A",
  scheduling: { state: "new", dueAt: "2026-01-01T00:00:00Z", intervalDays: 0, repetitions: 0, lapses: 0 },
};

function renderEditPage() {
  return render(
    <MemoryRouter initialEntries={["/flashcards/card-1/edit"]}>
      <Routes>
        <Route path="/flashcards/:flashcardId/edit" element={<FlashcardEditorPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("FlashcardEditorPage audio", () => {
  it("uploads audio and shows a player, then removes it", async () => {
    const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (init?.method === "POST" && url.includes("/audio")) {
        return jsonResponse({
          ...baseCard,
          audio: { contentType: "audio/mpeg", durationMs: 0, url: "/api/v1/flashcards/card-1/audio" },
        });
      }
      if (init?.method === "DELETE" && url.includes("/audio")) {
        return Promise.resolve({
          ok: true,
          status: 204,
          headers: new Headers(),
          json: async () => undefined,
        });
      }
      return jsonResponse(baseCard);
    });
    vi.stubGlobal("fetch", fetchMock);

    renderEditPage();

    const fileInput = await screen.findByLabelText("Áudio (opcional)");
    const file = new File(["fake mp3 bytes"], "clip.mp3", { type: "audio/mpeg" });

    await userEvent.upload(fileInput, file);

    await waitFor(() => expect(document.querySelector("audio")).toBeInTheDocument());
    expect(screen.getByRole("button", { name: "Remover áudio" })).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Remover áudio" }));

    await waitFor(() => expect(document.querySelector("audio")).not.toBeInTheDocument());
  });

  it("rejects an unsupported file type without calling the API", async () => {
    // A real browser's file picker would already filter by the input's
    // accept attribute (as does user-event's upload()); this simulates a
    // file that bypassed that filtering, e.g. dropped in from the OS with
    // a mismatched extension, to verify the app's own defense-in-depth
    // content-type check.
    const fetchMock = vi.fn().mockImplementation(() => jsonResponse(baseCard));
    vi.stubGlobal("fetch", fetchMock);

    renderEditPage();

    const fileInput = await screen.findByLabelText("Áudio (opcional)");
    const file = new File(["not audio"], "clip.txt", { type: "text/plain" });
    fireEvent.change(fileInput, { target: { files: [file] } });

    expect(await screen.findByRole("alert")).toHaveTextContent(/MP3, WAV ou OGG/i);
    expect(fetchMock).not.toHaveBeenCalledWith(expect.stringContaining("/audio"), expect.anything());
  });
});

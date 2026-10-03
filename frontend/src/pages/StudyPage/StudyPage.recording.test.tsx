import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { StudyPage } from "./StudyPage";

function jsonResponse(body: unknown) {
  return Promise.resolve({
    ok: true,
    status: 200,
    headers: new Headers({ "content-type": "application/json" }),
    json: async () => body,
  });
}

const scheduling = { state: "new", dueAt: "2026-01-01T00:00:00Z", intervalDays: 0, repetitions: 0, lapses: 0 };
const firstCard = {
  id: "card-1",
  deckId: "deck-1",
  question: "How to say 'resgatar'?",
  answer: "rescue",
  audio: { url: "/api/v1/flashcards/card-1/audio", contentType: "audio/mpeg" },
  scheduling,
};
const secondCard = { ...firstCard, id: "card-2", question: "How to say 'sair'?", answer: "leave", audio: undefined };

// Minimal stand-in for the browser's MediaRecorder: stop() hands back one
// chunk of audio, like a real recorder flushing its buffer.
class FakeMediaRecorder {
  state: "inactive" | "recording" = "inactive";
  mimeType = "audio/webm";
  ondataavailable: ((e: { data: Blob }) => void) | null = null;
  onstop: (() => void) | null = null;
  start() {
    this.state = "recording";
  }
  stop() {
    this.state = "inactive";
    this.ondataavailable?.({ data: new Blob(["voice"], { type: this.mimeType }) });
    this.onstop?.();
  }
}

const stopTrack = vi.fn();
const getUserMedia = vi.fn();
const revokeObjectURL = vi.fn();

function renderStudyPage() {
  return render(
    <MemoryRouter initialEntries={["/decks/deck-1/study"]}>
      <Routes>
        <Route path="/decks/:deckId/study" element={<StudyPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  stopTrack.mockReset();
  revokeObjectURL.mockReset();
  getUserMedia.mockReset().mockResolvedValue({ getTracks: () => [{ stop: stopTrack }] });
  vi.stubGlobal("MediaRecorder", FakeMediaRecorder);
  Object.defineProperty(navigator, "mediaDevices", { value: { getUserMedia }, configurable: true });
  let n = 0;
  URL.createObjectURL = vi.fn(() => `blob:take-${++n}`);
  URL.revokeObjectURL = revokeObjectURL;
  vi.stubGlobal(
    "fetch",
    vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (init?.method === "POST" && url.includes("/reviews")) return jsonResponse(firstCard);
      return jsonResponse([firstCard, secondCard]);
    }),
  );
});

afterEach(() => {
  vi.unstubAllGlobals();
  Object.defineProperty(navigator, "mediaDevices", { value: undefined, configurable: true });
});

describe("StudyPage voice recording", () => {
  it("records a take, plays it next to the original audio, and drops it on the next card", async () => {
    renderStudyPage();
    await screen.findByText("How to say 'resgatar'?");

    await userEvent.click(screen.getByRole("button", { name: /gravar minha voz/i }));
    expect(getUserMedia).toHaveBeenCalledWith({ audio: true });
    await userEvent.click(await screen.findByRole("button", { name: /parar gravação/i }));

    expect(stopTrack).toHaveBeenCalled();
    expect(screen.getByLabelText("Sua gravação")).toHaveAttribute("src", "blob:take-1");

    await userEvent.click(screen.getByRole("button", { name: "Mostrar resposta" }));
    // The take survives revealing the answer so it can be compared.
    expect(screen.getByLabelText("Áudio original")).toBeInTheDocument();
    expect(screen.getByLabelText("Sua gravação")).toHaveAttribute("src", "blob:take-1");

    await userEvent.click(screen.getByRole("button", { name: "Lembrei" }));

    expect(await screen.findByText("How to say 'sair'?")).toBeInTheDocument();
    expect(screen.queryByLabelText("Sua gravação")).not.toBeInTheDocument();
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:take-1");
    expect(screen.getByRole("button", { name: /gravar minha voz/i })).toBeInTheDocument();
  });

  it("toggles recording with the R key and lets the take be discarded", async () => {
    renderStudyPage();
    await screen.findByText("How to say 'resgatar'?");

    await userEvent.keyboard("r");
    await screen.findByRole("button", { name: /parar gravação/i });
    await userEvent.keyboard("r");

    expect(screen.getByLabelText("Sua gravação")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: /descartar gravação/i }));

    expect(screen.queryByLabelText("Sua gravação")).not.toBeInTheDocument();
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:take-1");
  });

  it("explains when microphone access is denied", async () => {
    getUserMedia.mockRejectedValue(new DOMException("denied", "NotAllowedError"));
    renderStudyPage();
    await screen.findByText("How to say 'resgatar'?");

    await act(async () => {
      await userEvent.click(screen.getByRole("button", { name: /gravar minha voz/i }));
    });

    expect(screen.getByRole("alert")).toHaveTextContent(/sem permissão para usar o microfone/i);
  });

  it("says so when the browser cannot record", async () => {
    vi.stubGlobal("MediaRecorder", undefined);
    vi.stubGlobal("isSecureContext", true);
    renderStudyPage();
    await screen.findByText("How to say 'resgatar'?");

    expect(screen.getByText(/não permite gravar áudio/i)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /gravar minha voz/i })).not.toBeInTheDocument();
  });

  it("points to localhost when the page is not a secure context", async () => {
    Object.defineProperty(navigator, "mediaDevices", { value: undefined, configurable: true });
    vi.stubGlobal("isSecureContext", false);
    renderStudyPage();
    await screen.findByText("How to say 'resgatar'?");

    expect(screen.getByText(/abra o app pelo endereço http:\/\/localhost:5173/i)).toBeInTheDocument();
  });
});

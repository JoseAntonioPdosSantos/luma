import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { FlashcardEditorPage } from "./FlashcardEditorPage";

describe("FlashcardEditorPage", () => {
  it("keeps save disabled until question and answer are filled in", async () => {
    vi.stubGlobal("fetch", vi.fn());

    render(
      <MemoryRouter initialEntries={["/decks/d1/flashcards/new"]}>
        <Routes>
          <Route path="/decks/:deckId/flashcards/new" element={<FlashcardEditorPage />} />
        </Routes>
      </MemoryRouter>,
    );

    const saveButton = screen.getByRole("button", { name: "Salvar" });
    expect(saveButton).toBeDisabled();

    await userEvent.type(screen.getByLabelText("Pergunta *"), "How to say 'rescue'?");
    expect(saveButton).toBeDisabled();

    await userEvent.type(screen.getByLabelText("Resposta *"), "resgatar");
    expect(saveButton).toBeEnabled();
  });
  it("sends the optional hint when creating a flashcard", async () => {
    const fetchMock = vi.fn().mockImplementation(() =>
      Promise.resolve({
        ok: true,
        status: 201,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({ id: "c1" }),
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    render(
      <MemoryRouter initialEntries={["/decks/d1/flashcards/new"]}>
        <Routes>
          <Route path="/decks/:deckId/flashcards/new" element={<FlashcardEditorPage />} />
          <Route path="/decks/:deckId" element={<p>deck page</p>} />
        </Routes>
      </MemoryRouter>,
    );

    await userEvent.type(screen.getByLabelText("Pergunta *"), "How to say 'rescue'?");
    await userEvent.type(screen.getByLabelText("Resposta *"), "resgatar");
    await userEvent.type(screen.getByLabelText("Dica (opcional)"), "Começa com R");
    await userEvent.click(screen.getByRole("button", { name: "Salvar" }));

    await waitFor(() => expect(screen.getByText("deck page")).toBeInTheDocument());
    const [, init] = fetchMock.mock.calls[0];
    expect(JSON.parse(init.body)).toMatchObject({ question: "How to say 'rescue'?", answer: "resgatar", hint: "Começa com R" });
  });
});

import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { AuthProvider } from "../../hooks/useAuth";
import { LoginPage } from "./LoginPage";

function renderLoginPage() {
  return render(
    <MemoryRouter>
      <AuthProvider>
        <LoginPage />
      </AuthProvider>
    </MemoryRouter>,
  );
}

describe("LoginPage", () => {
  it("disables submit until an email is entered", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({ error: { code: "UNAUTHORIZED", message: "auth required" } }),
      }),
    );

    renderLoginPage();

    const button = await screen.findByRole("button", { name: "Continuar" });
    expect(button).toBeDisabled();

    await userEvent.type(screen.getByLabelText("Email"), "user@example.com");
    expect(button).toBeEnabled();
  });

  it("shows an error message when the server rejects the request", async () => {
    // The email is syntactically valid so the browser's own type="email"
    // constraint validation lets the form submit; the mocked server then
    // rejects it, exercising our error-rendering path rather than jsdom's
    // native validation.
    const fetchMock = vi.fn().mockImplementation((url: string) => {
      if (url.includes("/api/v1/me")) {
        return Promise.resolve({
          ok: false,
          status: 401,
          headers: new Headers({ "content-type": "application/json" }),
          json: async () => ({ error: { code: "UNAUTHORIZED", message: "auth required" } }),
        });
      }
      return Promise.resolve({
        ok: false,
        status: 429,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({
          error: { code: "RATE_LIMITED", key: "rateLimit.exceeded", message: "too many requests, please try again later" },
        }),
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    renderLoginPage();

    await userEvent.type(await screen.findByLabelText("Email"), "user@example.com");
    await userEvent.click(screen.getByRole("button", { name: "Continuar" }));

    // The backend's error key is translated, rather than showing its raw
    // (always-English) message.
    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent("Muitas tentativas. Aguarde um pouco e tente de novo."),
    );
  });
});

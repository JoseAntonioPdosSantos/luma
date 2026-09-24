import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { App } from "./App";

describe("App", () => {
  it("shows the login screen when the user is not authenticated", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({ error: { code: "UNAUTHORIZED", message: "authentication required" } }),
      }),
    );

    render(<App />);

    expect(await screen.findByRole("button", { name: "Continuar" })).toBeInTheDocument();
  });
});

import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach, beforeAll } from "vitest";
import i18n from "./i18n";

// jsdom's navigator.language (and any leftover localStorage cache) would
// otherwise make the detector pick whatever the test runner's locale is.
// Existing tests assert exact Portuguese strings, so tests always run in
// Portuguese regardless of the machine running them. This must be awaited
// (changeLanguage is asynchronous) before any test body runs.
beforeAll(async () => {
  await i18n.changeLanguage("pt");
});

afterEach(() => cleanup());

import { describe, expect, it } from "vitest";
import { generateAutoHint } from "./autoHint";

describe("generateAutoHint", () => {
  it("keeps the first letter and masks the rest of a single word", () => {
    expect(generateAutoHint("rescue")).toBe("r _ _ _ _ _");
  });

  it("masks every word of a phrase and keeps punctuation", () => {
    expect(generateAutoHint("Le nouveau copain ?")).toBe("L _   n _ _ _ _ _ _   c _ _ _ _ _   ?");
  });

  it("keeps a leading punctuation mark and reveals the first letter after it", () => {
    expect(generateAutoHint("¿Qué?")).toBe("¿ Q _ _ ?");
  });

  it("handles accented and non-latin letters", () => {
    expect(generateAutoHint("ça")).toBe("ç _");
  });

  it("returns an empty hint for an empty answer", () => {
    expect(generateAutoHint("   ")).toBe("");
  });

  it("truncates very long answers", () => {
    const hint = generateAutoHint("a".repeat(500));
    expect(hint.endsWith("…")).toBe(true);
    expect(hint.length).toBeLessThan(500);
  });
});

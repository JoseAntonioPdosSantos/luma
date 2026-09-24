import { describe, expect, it } from "vitest";
import { describeNextReview } from "./nextReview";

// Local-time dates, so the tests do not depend on the machine's time zone.
const at = (y: number, m: number, d: number, h = 0, min = 0) => new Date(y, m - 1, d, h, min);

describe("describeNextReview", () => {
  const now = at(2026, 9, 21, 10, 0);

  it("counts minutes when it is less than an hour away", () => {
    expect(describeNextReview(at(2026, 9, 21, 10, 20), now).relative).toBe("em 20 minutos");
    expect(describeNextReview(new Date(now.getTime() + 30_000), now).relative).toBe("em 1 minuto");
  });

  it("counts hours when it is later the same day", () => {
    expect(describeNextReview(at(2026, 9, 21, 13, 0), now).relative).toBe("em 3 horas");
    expect(describeNextReview(at(2026, 9, 21, 11, 5), now).relative).toBe("em 1 hora");
  });

  it("says 'amanhã' for the next calendar day, even a few hours away", () => {
    expect(describeNextReview(at(2026, 9, 22, 8, 0), at(2026, 9, 21, 23, 0)).relative).toBe("amanhã");
    expect(describeNextReview(at(2026, 9, 22, 21, 13), now).relative).toBe("amanhã");
  });

  it("counts calendar days after that", () => {
    expect(describeNextReview(at(2026, 9, 23, 21, 13), now).relative).toBe("em 2 dias");
    expect(describeNextReview(at(2026, 10, 21, 9, 0), now).relative).toBe("em 30 dias");
  });

  it("gives the exact local date and time", () => {
    expect(describeNextReview(at(2026, 9, 23, 21, 13), now).when).toBe("23/09, às 21:13");
    expect(describeNextReview(at(2026, 1, 5, 7, 5), at(2025, 12, 31, 12, 0)).when).toBe("05/01, às 07:05");
  });

  it("says 'agora' if the moment already arrived", () => {
    expect(describeNextReview(at(2026, 9, 21, 9, 59), now).relative).toBe("agora");
  });
});

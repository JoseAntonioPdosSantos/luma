import { describe, expect, it } from "vitest";
import type { StudyProfile } from "../types";
import {
  FALLBACK_RULES,
  describeDailyLimit,
  describeMinutes,
  describeRating,
  emptyForm,
  formFromProfile,
  formatMultiplier,
  validateForm,
} from "./studyRules";

const validForm = () => ({ ...emptyForm(), name: "Prova" });

describe("validateForm", () => {
  it("accepts the default values with a name and returns what to send", () => {
    const { errors, input } = validateForm(validForm());
    expect(errors).toEqual({});
    expect(input).toEqual({ name: "Prova", dailyCardLimit: null, rules: FALLBACK_RULES });
  });

  it("requires a name and limits its length", () => {
    expect(validateForm({ ...validForm(), name: "   " }).errors.name).toBeDefined();
    expect(validateForm({ ...validForm(), name: "x".repeat(61) }).errors.name).toBeDefined();
    expect(validateForm({ ...validForm(), name: "é".repeat(60) }).errors.name).toBeUndefined();
    expect(validateForm({ ...validForm(), name: "  Prova  " }).input?.name).toBe("Prova");
  });

  it("checks the daily goal only when it is switched on", () => {
    expect(validateForm({ ...validForm(), useDailyLimit: false, dailyLimit: "abc" }).errors.dailyLimit).toBeUndefined();
    expect(validateForm({ ...validForm(), useDailyLimit: true, dailyLimit: "30" }).input?.dailyCardLimit).toBe(30);
    for (const bad of ["", "0", "201", "2.5", "abc"]) {
      expect(validateForm({ ...validForm(), useDailyLimit: true, dailyLimit: bad }).errors.dailyLimit).toBeDefined();
    }
  });

  it("checks the minutes for a failed card", () => {
    for (const bad of ["", "0", "1441", "1,5"]) {
      expect(validateForm({ ...validForm(), againDelayMinutes: bad }).errors.againDelayMinutes).toBeDefined();
    }
    expect(validateForm({ ...validForm(), againDelayMinutes: "1440" }).errors.againDelayMinutes).toBeUndefined();
  });

  it("checks days and multipliers, accepting a decimal comma", () => {
    const form = validForm();
    expect(validateForm({ ...form, hard: { firstIntervalDays: "0", multiplier: "1,2" } }).errors["hard.firstIntervalDays"]).toBeDefined();
    expect(validateForm({ ...form, hard: { firstIntervalDays: "1", multiplier: "0,9" } }).errors["hard.multiplier"]).toBeDefined();
    expect(validateForm({ ...form, easy: { firstIntervalDays: "7", multiplier: "5,5" } }).errors["easy.multiplier"]).toBeDefined();
    expect(validateForm({ ...form, easy: { firstIntervalDays: "7", multiplier: "abc" } }).errors["easy.multiplier"]).toBeDefined();
    expect(validateForm({ ...form, good: { firstIntervalDays: "3", multiplier: "1,5" } }).input?.rules.good.multiplier).toBe(1.5);
    expect(validateForm({ ...form, good: { firstIntervalDays: "3", multiplier: "1.5" } }).input?.rules.good.multiplier).toBe(1.5);
  });

  it("does not let a better answer come back sooner than a worse one", () => {
    const form = validForm();
    const days = validateForm({ ...form, good: { firstIntervalDays: "9", multiplier: "2" } }); // easy is 7 days
    expect(days.errors.orderDays).toBeDefined();
    expect(days.input).toBeNull();

    const mult = validateForm({ ...form, hard: { firstIntervalDays: "1", multiplier: "2,5" } }); // good is 2
    expect(mult.errors.orderMultiplier).toBeDefined();

    // Equal values are fine.
    const equal = validateForm({ ...form, hard: { firstIntervalDays: "3", multiplier: "2" } });
    expect(equal.errors).toEqual({});
  });
});

describe("forms and descriptions", () => {
  const profile: StudyProfile = {
    id: "p1",
    name: "Calma",
    isDefault: false,
    dailyCardLimit: 15,
    rules: { againDelayMinutes: 90, hard: { firstIntervalDays: 2, multiplier: 1.5 }, good: { firstIntervalDays: 4, multiplier: 2.5 }, easy: { firstIntervalDays: 9, multiplier: 4 } },
  };

  it("builds a form from a configuration, with the goal switched on only if it has one", () => {
    const form = formFromProfile(profile);
    expect(form).toMatchObject({ name: "Calma", useDailyLimit: true, dailyLimit: "15", againDelayMinutes: "90" });
    expect(form.hard).toEqual({ firstIntervalDays: "2", multiplier: "1,5" });
    expect(formFromProfile({ ...profile, dailyCardLimit: null }).useDailyLimit).toBe(false);
  });

  it("describes minutes, ratings and goals in plain words", () => {
    expect(describeMinutes(1)).toBe("1 minuto");
    expect(describeMinutes(10)).toBe("10 minutos");
    expect(describeMinutes(60)).toBe("1 hora");
    expect(describeMinutes(90)).toBe("1 hora e 30 minutos");
    expect(describeMinutes(120)).toBe("2 horas");
    expect(describeRating({ firstIntervalDays: 1, multiplier: 1.2 })).toBe("1 dia para um card novo; depois × 1,2");
    expect(describeRating({ firstIntervalDays: 3, multiplier: 2 })).toBe("3 dias para um card novo; depois × 2");
    expect(describeDailyLimit(null)).toBe("Sem meta diária");
    expect(describeDailyLimit(1)).toBe("1 card por dia");
    expect(describeDailyLimit(30)).toBe("30 cards por dia");
    expect(formatMultiplier(1.25)).toBe("1,25");
  });
});

import { describe, expect, it } from "vitest";
import type { components } from "../api/schema.d.ts";
import { computeNextOccurrence, computeStaleness, stalenessColor } from "./staleness.ts";

type Task = components["schemas"]["Task"];
type StalenessInput = Pick<
  Task,
  "recurrenceType" | "recurrenceRule" | "lastCompletedAt" | "createdAt"
>;

describe("computeNextOccurrence", () => {
  it("returns the next daily occurrence after a given date", () => {
    const anchor = new Date("2026-01-01T00:00:00Z");
    const next = computeNextOccurrence("FREQ=DAILY", anchor);
    expect(next).toEqual(new Date("2026-01-02T00:00:00Z"));
  });

  it("returns the next weekly occurrence after a given date", () => {
    const anchor = new Date("2026-01-05T00:00:00Z"); // Monday
    const next = computeNextOccurrence("FREQ=WEEKLY", anchor);
    expect(next).toEqual(new Date("2026-01-12T00:00:00Z"));
  });

  it("returns the next biweekly occurrence", () => {
    const anchor = new Date("2026-01-05T00:00:00Z"); // Monday
    const next = computeNextOccurrence("FREQ=WEEKLY;INTERVAL=2", anchor);
    expect(next).toEqual(new Date("2026-01-19T00:00:00Z"));
  });

  it("handles multi-day rules (BYDAY=MO,TU) — next from Monday is Tuesday", () => {
    const anchor = new Date("2026-01-05T00:00:00Z"); // Monday
    const next = computeNextOccurrence("FREQ=WEEKLY;BYDAY=MO,TU", anchor);
    expect(next).toEqual(new Date("2026-01-06T00:00:00Z")); // Tuesday
  });

  it("handles multi-day rules (BYDAY=MO,TU) — next from Tuesday is Monday", () => {
    const anchor = new Date("2026-01-06T00:00:00Z"); // Tuesday
    const next = computeNextOccurrence("FREQ=WEEKLY;BYDAY=MO,TU", anchor);
    expect(next).toEqual(new Date("2026-01-12T00:00:00Z")); // Next Monday
  });

  it("returns the next monthly occurrence", () => {
    const anchor = new Date("2026-01-15T00:00:00Z");
    const next = computeNextOccurrence("FREQ=MONTHLY;BYMONTHDAY=15", anchor);
    expect(next).toEqual(new Date("2026-02-15T00:00:00Z"));
  });

  it("returns the next yearly occurrence", () => {
    const anchor = new Date("2026-03-01T00:00:00Z");
    const next = computeNextOccurrence("FREQ=YEARLY", anchor);
    expect(next).toEqual(new Date("2027-03-01T00:00:00Z"));
  });

  it("handles RRULE: prefix in the string", () => {
    const anchor = new Date("2026-01-01T00:00:00Z");
    const next = computeNextOccurrence("RRULE:FREQ=DAILY", anchor);
    expect(next).toEqual(new Date("2026-01-02T00:00:00Z"));
  });

  it("returns null for malformed rrule string", () => {
    const anchor = new Date("2026-01-01T00:00:00Z");
    expect(computeNextOccurrence("INVALID", anchor)).toBeNull();
  });

  it("returns null for empty string", () => {
    const anchor = new Date("2026-01-01T00:00:00Z");
    expect(computeNextOccurrence("", anchor)).toBeNull();
  });
});

describe("computeStaleness", () => {
  const baseTask: StalenessInput = {
    recurrenceType: "completion_based",
    recurrenceRule: "FREQ=WEEKLY",
    lastCompletedAt: "2026-01-05T00:00:00Z", // Monday
    createdAt: "2026-01-01T00:00:00Z",
  };

  it("returns null for one_off tasks", () => {
    const task: StalenessInput = { ...baseTask, recurrenceType: "one_off" };
    expect(computeStaleness(task, "initial")).toBeNull();
  });

  it("returns null when recurrenceRule is null", () => {
    const task = { ...baseTask, recurrenceRule: null };
    expect(computeStaleness(task, "initial")).toBeNull();
  });

  it("returns null for completion-category status", () => {
    expect(computeStaleness(baseTask, "completion")).toBeNull();
  });

  it("computes correct ratio — halfway through weekly cycle", () => {
    // Anchor: Mon Jan 5. Next due: Mon Jan 12. Interval: 7 days.
    // Now: Thu Jan 8 12:00 = 3.5 days elapsed. Ratio = 3.5/7 = 0.5
    const now = new Date("2026-01-08T12:00:00Z");
    const ratio = computeStaleness(baseTask, "initial", now);
    expect(ratio).toBeCloseTo(0.5, 1);
  });

  it("computes ratio > 1 for overdue tasks", () => {
    // Anchor: Mon Jan 5. Next due: Mon Jan 12. Interval: 7 days.
    // Now: Thu Jan 15 = 10 days elapsed. Ratio = 10/7 ≈ 1.43
    const now = new Date("2026-01-15T00:00:00Z");
    const ratio = computeStaleness(baseTask, "initial", now);
    expect(ratio).toBeCloseTo(10 / 7, 1);
  });

  it("uses createdAt when lastCompletedAt is null", () => {
    const task = { ...baseTask, lastCompletedAt: null };
    // Anchor: Jan 1. Next due: Jan 8 (weekly from Jan 1). Interval: 7 days.
    // Now: Jan 4 12:00 = 3.5 days elapsed. Ratio = 3.5/7 = 0.5
    const now = new Date("2026-01-04T12:00:00Z");
    const ratio = computeStaleness(task, "initial", now);
    expect(ratio).toBeCloseTo(0.5, 1);
  });

  it("uses lastCompletedAt over createdAt when both present", () => {
    // Anchor should be lastCompletedAt (Jan 5), not createdAt (Jan 1)
    // Next due: Jan 12. Now: Jan 8 12:00. Ratio = 3.5/7 = 0.5
    const now = new Date("2026-01-08T12:00:00Z");
    const ratio = computeStaleness(baseTask, "initial", now);
    expect(ratio).toBeCloseTo(0.5, 1);
  });

  it("returns ratio of 0 at exact anchor time", () => {
    const now = new Date("2026-01-05T00:00:00Z");
    const ratio = computeStaleness(baseTask, "initial", now);
    expect(ratio).toBeCloseTo(0, 1);
  });

  it("handles multi-day rules with cycle-specific intervals", () => {
    // BYDAY=MO,TU: anchor Mon Jan 5 → next Tue Jan 6 (1-day interval)
    const task = { ...baseTask, recurrenceRule: "FREQ=WEEKLY;BYDAY=MO,TU" };
    // Now: Mon Jan 5 12:00 = 0.5 days elapsed. Interval: 1 day. Ratio = 0.5
    const now = new Date("2026-01-05T12:00:00Z");
    const ratio = computeStaleness(task, "initial", now);
    expect(ratio).toBeCloseTo(0.5, 1);
  });

  it("handles multi-day rules — longer gap from Tuesday anchor", () => {
    // BYDAY=MO,TU: anchor Tue Jan 6 → next Mon Jan 12 (6-day interval)
    const task = {
      ...baseTask,
      recurrenceRule: "FREQ=WEEKLY;BYDAY=MO,TU",
      lastCompletedAt: "2026-01-06T00:00:00Z", // Tuesday
    };
    // Now: Fri Jan 9 = 3 days elapsed. Interval: 6 days. Ratio = 3/6 = 0.5
    const now = new Date("2026-01-09T00:00:00Z");
    const ratio = computeStaleness(task, "initial", now);
    expect(ratio).toBeCloseTo(0.5, 1);
  });

  it("returns null for on_dependency type with null rule", () => {
    const task: StalenessInput = {
      ...baseTask,
      recurrenceType: "on_dependency",
      recurrenceRule: null,
    };
    expect(computeStaleness(task, "initial")).toBeNull();
  });

  it("works with fixed_non_accumulating recurrence type", () => {
    const task: StalenessInput = { ...baseTask, recurrenceType: "fixed_non_accumulating" };
    const now = new Date("2026-01-08T12:00:00Z");
    const ratio = computeStaleness(task, "initial", now);
    expect(ratio).toBeCloseTo(0.5, 1);
  });

  it("works with intermediate status category", () => {
    const now = new Date("2026-01-08T12:00:00Z");
    const ratio = computeStaleness(baseTask, "intermediate", now);
    expect(ratio).toBeCloseTo(0.5, 1);
  });

  it("works when statusCategory is undefined", () => {
    const now = new Date("2026-01-08T12:00:00Z");
    const ratio = computeStaleness(baseTask, undefined, now);
    expect(ratio).toBeCloseTo(0.5, 1);
  });

  it("returns a negative ratio when now is before the anchor", () => {
    const now = new Date("2026-01-04T00:00:00Z"); // 1 day before anchor (Jan 5)
    const ratio = computeStaleness(baseTask, "initial", now);
    expect(ratio).toBeCloseTo(-1 / 7, 2);
  });
});

describe("stalenessColor", () => {
  function parseRgb(color: string): { r: number; g: number; b: number } {
    const match = color.match(/rgb\((\d+), (\d+), (\d+)\)/);
    if (!match) throw new Error(`Invalid color: ${color}`);
    return { r: Number(match[1]), g: Number(match[2]), b: Number(match[3]) };
  }

  it("returns green at ratio 0", () => {
    const { r, g, b } = parseRgb(stalenessColor(0));
    expect(r).toBeLessThan(100);
    expect(g).toBeGreaterThan(150);
    expect(b).toBeLessThan(100);
  });

  it("returns yellow-ish at ratio 0.5", () => {
    const { r, g, b } = parseRgb(stalenessColor(0.5));
    expect(r).toBeGreaterThan(200);
    expect(g).toBeGreaterThan(200);
    expect(b).toBeLessThan(100);
  });

  it("returns red at ratio 1.0", () => {
    const { r, g, b } = parseRgb(stalenessColor(1.0));
    expect(r).toBeGreaterThan(200);
    expect(g).toBeLessThan(100);
    expect(b).toBeLessThan(100);
  });

  it("clamps at ratio > 1 — still red", () => {
    const at1 = stalenessColor(1.0);
    const at2 = stalenessColor(2.0);
    expect(at2).toBe(at1);
  });

  it("clamps negative ratio to 0 — returns green", () => {
    const atNeg = stalenessColor(-0.5);
    const at0 = stalenessColor(0);
    expect(atNeg).toBe(at0);
  });

  it("returns a valid rgb() string", () => {
    const color = stalenessColor(0.75);
    expect(color).toMatch(/^rgb\(\d+, \d+, \d+\)$/);
  });
});

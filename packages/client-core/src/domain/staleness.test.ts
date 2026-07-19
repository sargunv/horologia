import { describe, expect, it } from "vitest";

import type { components } from "../api/schema.d.ts";
import { computeNextOccurrence, computeStaleness } from "./staleness.ts";

type Task = components["schemas"]["Task"];
type StalenessInput = Pick<
  Task,
  "recurrenceType" | "recurrenceRule" | "lastCompletedAt" | "createdAt"
>;

describe("computeNextOccurrence", () => {
  it.each([
    ["FREQ=DAILY", "2026-01-01T00:00:00Z", "2026-01-02T00:00:00Z"],
    ["FREQ=WEEKLY;INTERVAL=2", "2026-01-05T00:00:00Z", "2026-01-19T00:00:00Z"],
    ["FREQ=WEEKLY;BYDAY=MO,TU", "2026-01-05T00:00:00Z", "2026-01-06T00:00:00Z"],
    ["FREQ=MONTHLY;BYMONTHDAY=15", "2026-01-15T00:00:00Z", "2026-02-15T00:00:00Z"],
    ["FREQ=YEARLY", "2026-03-01T00:00:00Z", "2027-03-01T00:00:00Z"],
    ["RRULE:FREQ=DAILY", "2026-01-01T00:00:00Z", "2026-01-02T00:00:00Z"],
  ])("finds the next occurrence for %s", (rule, anchor, expected) => {
    expect(computeNextOccurrence(rule, new Date(anchor))).toEqual(new Date(expected));
  });

  it("rejects malformed rules", () => {
    expect(computeNextOccurrence("INVALID", new Date("2026-01-01T00:00:00Z"))).toBeNull();
  });
});

describe("computeStaleness", () => {
  const task: StalenessInput = {
    recurrenceType: "completion_based",
    recurrenceRule: "FREQ=WEEKLY",
    lastCompletedAt: "2026-01-05T00:00:00Z",
    createdAt: "2026-01-01T00:00:00Z",
  };

  it("excludes non-recurring, unconfigured, and completed tasks", () => {
    expect(computeStaleness({ ...task, recurrenceType: "one_off" }, "initial")).toBeNull();
    expect(computeStaleness({ ...task, recurrenceRule: null }, "initial")).toBeNull();
    expect(computeStaleness(task, "completion")).toBeNull();
  });

  it.each([
    ["2026-01-08T12:00:00Z", 0.5],
    ["2026-01-15T00:00:00Z", 10 / 7],
    ["2026-01-04T00:00:00Z", -1 / 7],
  ])("measures elapsed recurrence time at %s", (now, expected) => {
    expect(computeStaleness(task, "initial", new Date(now))).toBeCloseTo(expected, 2);
  });

  it("falls back to creation time before the first completion", () => {
    expect(
      computeStaleness(
        { ...task, lastCompletedAt: null },
        "initial",
        new Date("2026-01-04T12:00:00Z"),
      ),
    ).toBeCloseTo(0.5, 2);
  });

  it("uses the current interval for uneven multi-day schedules", () => {
    expect(
      computeStaleness(
        { ...task, recurrenceRule: "FREQ=WEEKLY;BYDAY=MO,TU" },
        "initial",
        new Date("2026-01-05T12:00:00Z"),
      ),
    ).toBeCloseTo(0.5, 2);
  });
});

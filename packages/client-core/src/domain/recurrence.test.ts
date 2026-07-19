import { describe, expect, it } from "vitest";

import {
  buildRRule,
  describeRule,
  parseRecurrenceDurationInput,
  parseRRule,
} from "./recurrence.ts";

describe("portable recurrence rules", () => {
  it("round-trips an advanced fixed schedule", () => {
    const parsed = parseRRule("FREQ=MONTHLY;INTERVAL=2;BYDAY=-1FR;UNTIL=20261231T000000Z");
    expect(parsed).toMatchObject({
      freq: "MONTHLY",
      interval: 2,
      nthWeekday: [{ ordinal: -1, weekday: "FR" }],
      until: "2026-12-31",
      parseError: false,
    });
    expect(parseRRule(buildRRule(parsed))).toEqual(parsed);
    expect(describeRule(parsed)).toBe("Every 2 months on the Last Friday until 2026-12-31");
  });

  it("maps human durations to recurrence intervals", () => {
    expect(parseRecurrenceDurationInput("3 weeks")).toEqual({
      freq: "WEEKLY",
      interval: 3,
      label: "Every 3 weeks",
    });
  });

  it("preserves a safe editable default for malformed rules", () => {
    expect(parseRRule("definitely not an RRULE")).toMatchObject({
      freq: "WEEKLY",
      interval: 1,
      parseError: true,
    });
  });
});

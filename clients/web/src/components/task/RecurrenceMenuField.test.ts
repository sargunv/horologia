import { describe, expect, it } from "vitest";
import {
  buildRRule,
  describeRule,
  parseRecurrenceDurationInput as parseDurationInput,
  parseRRule,
} from "@horologia/client-core/domain/recurrence";

describe("parseRRule", () => {
  it("returns defaults for null input", () => {
    const result = parseRRule(null);
    expect(result.freq).toBe("WEEKLY");
    expect(result.interval).toBe(1);
    expect(result.byweekday).toEqual([]);
    expect(result.bymonthday).toEqual([]);
    expect(result.nthWeekday).toEqual([]);
    expect(result.bymonth).toEqual([]);
    expect(result.until).toBeNull();
    expect(result.parseError).toBe(false);
  });

  it("returns defaults for empty string", () => {
    const result = parseRRule("");
    expect(result.freq).toBe("WEEKLY");
    expect(result.parseError).toBe(false);
  });

  it("parses daily frequency", () => {
    const result = parseRRule("FREQ=DAILY;INTERVAL=3");
    expect(result.freq).toBe("DAILY");
    expect(result.interval).toBe(3);
    expect(result.parseError).toBe(false);
  });

  it("parses weekly with weekdays", () => {
    const result = parseRRule("FREQ=WEEKLY;BYDAY=MO,WE,FR");
    expect(result.freq).toBe("WEEKLY");
    expect(result.interval).toBe(1);
    expect(result.byweekday).toEqual(["MO", "WE", "FR"]);
  });

  it("parses weekly with interval", () => {
    const result = parseRRule("FREQ=WEEKLY;INTERVAL=2;BYDAY=TU,TH");
    expect(result.freq).toBe("WEEKLY");
    expect(result.interval).toBe(2);
    expect(result.byweekday).toEqual(["TU", "TH"]);
  });

  it("parses monthly by monthday", () => {
    const result = parseRRule("FREQ=MONTHLY;BYMONTHDAY=15");
    expect(result.freq).toBe("MONTHLY");
    expect(result.bymonthday).toEqual([15]);
    expect(result.nthWeekday).toEqual([]);
  });

  it("parses monthly by nth weekday", () => {
    const result = parseRRule("FREQ=MONTHLY;BYDAY=2MO");
    expect(result.freq).toBe("MONTHLY");
    expect(result.nthWeekday).toEqual([{ ordinal: 2, weekday: "MO" }]);
  });

  it("parses monthly by last weekday", () => {
    const result = parseRRule("FREQ=MONTHLY;BYDAY=-1FR");
    expect(result.freq).toBe("MONTHLY");
    expect(result.nthWeekday).toEqual([{ ordinal: -1, weekday: "FR" }]);
  });

  it("parses yearly with months", () => {
    const result = parseRRule("FREQ=YEARLY;BYMONTH=1,6,12");
    expect(result.freq).toBe("YEARLY");
    expect(result.bymonth).toEqual([1, 6, 12]);
  });

  it("parses until date", () => {
    const result = parseRRule("FREQ=WEEKLY;UNTIL=20261231T000000Z");
    expect(result.freq).toBe("WEEKLY");
    expect(result.until).toBe("2026-12-31");
  });

  it("handles RRULE: prefix", () => {
    const result = parseRRule("RRULE:FREQ=DAILY");
    expect(result.freq).toBe("DAILY");
    expect(result.parseError).toBe(false);
  });

  it("sets parseError on malformed input", () => {
    const result = parseRRule("NOT_A_VALID_RRULE");
    expect(result.parseError).toBe(true);
    expect(result.freq).toBe("WEEKLY"); // defaults
  });

  it("normalizes explicit INTERVAL=1 away", () => {
    const parsed = parseRRule("FREQ=DAILY;INTERVAL=1");
    expect(parsed.interval).toBe(1);
    const rebuilt = buildRRule(parsed);
    expect(rebuilt).not.toContain("INTERVAL");
  });

  it("preserves multiple nth weekdays", () => {
    const result = parseRRule("FREQ=MONTHLY;BYDAY=2MO,3FR");
    expect(result.nthWeekday).toEqual([
      { ordinal: 2, weekday: "MO" },
      { ordinal: 3, weekday: "FR" },
    ]);
  });
});

describe("buildRRule", () => {
  it("builds daily with interval 1", () => {
    const result = buildRRule({
      freq: "DAILY",
      interval: 1,
      byweekday: [],
      bymonthday: [],
      nthWeekday: [],
      bymonth: [],
      until: null,
      parseError: false,
    });
    expect(result).toContain("FREQ=DAILY");
    expect(result).not.toContain("INTERVAL");
  });

  it("builds daily with interval > 1", () => {
    const result = buildRRule({
      freq: "DAILY",
      interval: 3,
      byweekday: [],
      bymonthday: [],
      nthWeekday: [],
      bymonth: [],
      until: null,
      parseError: false,
    });
    expect(result).toContain("FREQ=DAILY");
    expect(result).toContain("INTERVAL=3");
  });

  it("builds weekly with weekdays", () => {
    const result = buildRRule({
      freq: "WEEKLY",
      interval: 1,
      byweekday: ["MO", "WE", "FR"],
      bymonthday: [],
      nthWeekday: [],
      bymonth: [],
      until: null,
      parseError: false,
    });
    expect(result).toContain("FREQ=WEEKLY");
    expect(result).toContain("BYDAY=MO,WE,FR");
  });

  it("builds monthly with bymonthday", () => {
    const result = buildRRule({
      freq: "MONTHLY",
      interval: 1,
      byweekday: [],
      bymonthday: [15],
      nthWeekday: [],
      bymonth: [],
      until: null,
      parseError: false,
    });
    expect(result).toContain("FREQ=MONTHLY");
    expect(result).toContain("BYMONTHDAY=15");
  });

  it("builds monthly with nth weekday", () => {
    const result = buildRRule({
      freq: "MONTHLY",
      interval: 1,
      byweekday: [],
      bymonthday: [],
      nthWeekday: [{ ordinal: 2, weekday: "MO" }],
      bymonth: [],
      until: null,
      parseError: false,
    });
    expect(result).toContain("FREQ=MONTHLY");
    expect(result).toContain("BYDAY=+2MO");
  });

  it("builds yearly with months", () => {
    const result = buildRRule({
      freq: "YEARLY",
      interval: 1,
      byweekday: [],
      bymonthday: [],
      nthWeekday: [],
      bymonth: [1, 6],
      until: null,
      parseError: false,
    });
    expect(result).toContain("FREQ=YEARLY");
    expect(result).toContain("BYMONTH=1,6");
  });

  it("includes UNTIL when set", () => {
    const result = buildRRule({
      freq: "WEEKLY",
      interval: 1,
      byweekday: [],
      bymonthday: [],
      nthWeekday: [],
      bymonth: [],
      until: "2026-12-31",
      parseError: false,
    });
    expect(result).toContain("UNTIL=");
  });
});

describe("parseRRule/buildRRule roundtrip", () => {
  const cases = [
    "FREQ=DAILY;INTERVAL=3",
    "FREQ=WEEKLY;BYDAY=MO,WE,FR",
    "FREQ=WEEKLY;INTERVAL=2",
    "FREQ=MONTHLY;BYMONTHDAY=15",
    "FREQ=MONTHLY;BYDAY=2MO",
    "FREQ=MONTHLY;BYDAY=-1FR",
    "FREQ=YEARLY;BYMONTH=1,6,12",
    "FREQ=YEARLY",
    "FREQ=WEEKLY;UNTIL=20261231T000000Z",
    "FREQ=MONTHLY;BYDAY=2MO,3FR",
  ];

  for (const rrule of cases) {
    it(`roundtrips: ${rrule}`, () => {
      const parsed = parseRRule(rrule);
      expect(parsed.parseError).toBe(false);
      const rebuilt = buildRRule(parsed);
      const reparsed = parseRRule(rebuilt);
      expect(reparsed.freq).toBe(parsed.freq);
      expect(reparsed.interval).toBe(parsed.interval);
      expect(reparsed.byweekday).toEqual(parsed.byweekday);
      expect(reparsed.bymonthday).toEqual(parsed.bymonthday);
      expect(reparsed.nthWeekday).toEqual(parsed.nthWeekday);
      expect(reparsed.bymonth).toEqual(parsed.bymonth);
    });
  }
});

describe("parseDurationInput", () => {
  it("parses '1 day'", () => {
    const result = parseDurationInput("1 day");
    expect(result).toEqual({ freq: "DAILY", interval: 1, label: "Every 1 day" });
  });

  it("parses '3 days'", () => {
    const result = parseDurationInput("3 days");
    expect(result).toEqual({ freq: "DAILY", interval: 3, label: "Every 3 days" });
  });

  it("parses '1 week'", () => {
    const result = parseDurationInput("1 week");
    expect(result).toEqual({ freq: "WEEKLY", interval: 1, label: "Every 1 week" });
  });

  it("parses '2 weeks'", () => {
    const result = parseDurationInput("2 weeks");
    expect(result).toEqual({ freq: "WEEKLY", interval: 2, label: "Every 2 weeks" });
  });

  it("parses '1 month'", () => {
    const result = parseDurationInput("1 month");
    expect(result?.freq).toBe("MONTHLY");
    expect(result?.interval).toBe(1);
  });

  it("parses '3 months'", () => {
    const result = parseDurationInput("3 months");
    expect(result?.freq).toBe("MONTHLY");
    expect(result?.interval).toBe(3);
  });

  it("parses '1 year'", () => {
    const result = parseDurationInput("1 year");
    expect(result?.freq).toBe("YEARLY");
    expect(result?.interval).toBe(1);
  });

  it("returns null for empty string", () => {
    expect(parseDurationInput("")).toBeNull();
  });

  it("returns null for garbage input", () => {
    expect(parseDurationInput("asdf")).toBeNull();
  });

  it("returns null for negative duration", () => {
    expect(parseDurationInput("-3 days")).toBeNull();
  });

  it("falls back to daily for non-standard durations", () => {
    const result = parseDurationInput("45 days");
    expect(result?.freq).toBe("DAILY");
    expect(result?.interval).toBe(45);
  });

  it("rounds 12 hours up to 1 day", () => {
    const result = parseDurationInput("12 hours");
    expect(result).toEqual({ freq: "DAILY", interval: 1, label: "Every 1 day" });
  });

  it("returns null for durations less than ~12 hours", () => {
    expect(parseDurationInput("6 hours")).toBeNull();
  });
});

describe("describeRule", () => {
  it("describes daily", () => {
    expect(
      describeRule({
        freq: "DAILY",
        interval: 1,
        byweekday: [],
        bymonthday: [],
        nthWeekday: [],
        bymonth: [],
        until: null,
        parseError: false,
      }),
    ).toBe("Every day");
  });

  it("describes daily with interval", () => {
    expect(
      describeRule({
        freq: "DAILY",
        interval: 3,
        byweekday: [],
        bymonthday: [],
        nthWeekday: [],
        bymonth: [],
        until: null,
        parseError: false,
      }),
    ).toBe("Every 3 days");
  });

  it("describes weekly with days", () => {
    expect(
      describeRule({
        freq: "WEEKLY",
        interval: 1,
        byweekday: ["MO", "FR"],
        bymonthday: [],
        nthWeekday: [],
        bymonth: [],
        until: null,
        parseError: false,
      }),
    ).toBe("Every week on Mon, Fri");
  });

  it("describes monthly with day", () => {
    expect(
      describeRule({
        freq: "MONTHLY",
        interval: 1,
        byweekday: [],
        bymonthday: [15],
        nthWeekday: [],
        bymonth: [],
        until: null,
        parseError: false,
      }),
    ).toBe("Every month on day 15");
  });

  it("describes monthly with last day", () => {
    expect(
      describeRule({
        freq: "MONTHLY",
        interval: 1,
        byweekday: [],
        bymonthday: [-1],
        nthWeekday: [],
        bymonth: [],
        until: null,
        parseError: false,
      }),
    ).toBe("Every month on day last");
  });

  it("describes monthly with nth weekday", () => {
    expect(
      describeRule({
        freq: "MONTHLY",
        interval: 1,
        byweekday: [],
        bymonthday: [],
        nthWeekday: [{ ordinal: 1, weekday: "MO" }],
        bymonth: [],
        until: null,
        parseError: false,
      }),
    ).toBe("Every month on the 1st Monday");
  });

  it("describes until date", () => {
    expect(
      describeRule({
        freq: "WEEKLY",
        interval: 1,
        byweekday: [],
        bymonthday: [],
        nthWeekday: [],
        bymonth: [],
        until: "2026-12-31",
        parseError: false,
      }),
    ).toBe("Every week until 2026-12-31");
  });
});

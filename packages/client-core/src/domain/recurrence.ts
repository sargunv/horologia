import parseDuration from "parse-duration";
import { RRule, Weekday } from "rrule";

export type RecurrenceFrequency = "DAILY" | "WEEKLY" | "MONTHLY" | "YEARLY";
export const WEEKDAY_CODES = ["MO", "TU", "WE", "TH", "FR", "SA", "SU"] as const;
export type WeekdayCode = (typeof WEEKDAY_CODES)[number];

export const WEEKDAY_LABELS: Record<WeekdayCode, string> = {
  MO: "Monday",
  TU: "Tuesday",
  WE: "Wednesday",
  TH: "Thursday",
  FR: "Friday",
  SA: "Saturday",
  SU: "Sunday",
};

const FREQUENCY_LABELS: Record<RecurrenceFrequency, { singular: string; plural: string }> = {
  DAILY: { singular: "day", plural: "days" },
  WEEKLY: { singular: "week", plural: "weeks" },
  MONTHLY: { singular: "month", plural: "months" },
  YEARLY: { singular: "year", plural: "years" },
};

const RRULE_WEEKDAY: Record<WeekdayCode, Weekday> = {
  MO: RRule.MO,
  TU: RRule.TU,
  WE: RRule.WE,
  TH: RRule.TH,
  FR: RRule.FR,
  SA: RRule.SA,
  SU: RRule.SU,
};

const RRULE_FREQ_TO_CODE: Partial<Record<number, RecurrenceFrequency>> = {
  [RRule.DAILY]: "DAILY",
  [RRule.WEEKLY]: "WEEKLY",
  [RRule.MONTHLY]: "MONTHLY",
  [RRule.YEARLY]: "YEARLY",
};

const FREQ_TO_RRULE: Record<RecurrenceFrequency, number> = {
  DAILY: RRule.DAILY,
  WEEKLY: RRule.WEEKLY,
  MONTHLY: RRule.MONTHLY,
  YEARLY: RRule.YEARLY,
};

export const RECURRENCE_ORDINAL_LABELS: Record<number, string> = {
  1: "1st",
  2: "2nd",
  3: "3rd",
  4: "4th",
  [-1]: "Last",
};

export const RECURRENCE_MONTH_LABELS = [
  "January",
  "February",
  "March",
  "April",
  "May",
  "June",
  "July",
  "August",
  "September",
  "October",
  "November",
  "December",
];

export interface ParsedRecurrenceRule {
  freq: RecurrenceFrequency;
  interval: number;
  byweekday: WeekdayCode[];
  bymonthday: number[];
  nthWeekday: { ordinal: number; weekday: WeekdayCode }[];
  bymonth: number[];
  until: string | null;
  parseError: boolean;
}

function emptyRule(): ParsedRecurrenceRule {
  return {
    freq: "WEEKLY",
    interval: 1,
    byweekday: [],
    bymonthday: [],
    nthWeekday: [],
    bymonth: [],
    until: null,
    parseError: false,
  };
}

function toArray<T>(value: T | T[]): T[] {
  return Array.isArray(value) ? value : [value];
}

export function parseRRule(rrule: string | null): ParsedRecurrenceRule {
  const defaults = emptyRule();
  if (!rrule) return defaults;

  try {
    const normalized = rrule.startsWith("RRULE:") ? rrule : `RRULE:${rrule}`;
    const options = RRule.fromString(normalized).origOptions;
    const byweekday: WeekdayCode[] = [];
    const nthWeekday: ParsedRecurrenceRule["nthWeekday"] = [];
    for (const weekday of options.byweekday ? toArray(options.byweekday) : []) {
      if (!(weekday instanceof Weekday)) continue;
      const code = WEEKDAY_CODES[weekday.weekday];
      if (!code) continue;
      if (weekday.n !== undefined && weekday.n !== 0) {
        nthWeekday.push({ ordinal: weekday.n, weekday: code });
      } else {
        byweekday.push(code);
      }
    }
    const until = options.until
      ? `${options.until.getUTCFullYear()}-${String(options.until.getUTCMonth() + 1).padStart(2, "0")}-${String(options.until.getUTCDate()).padStart(2, "0")}`
      : null;
    return {
      freq: RRULE_FREQ_TO_CODE[options.freq ?? RRule.WEEKLY] ?? "WEEKLY",
      interval: options.interval ?? 1,
      byweekday,
      bymonthday: options.bymonthday ? toArray(options.bymonthday) : [],
      nthWeekday,
      bymonth: options.bymonth ? toArray(options.bymonth) : [],
      until,
      parseError: false,
    };
  } catch {
    return { ...defaults, parseError: true };
  }
}

export function buildRRule(parsed: ParsedRecurrenceRule): string {
  const options: ConstructorParameters<typeof RRule>[0] = {
    freq: FREQ_TO_RRULE[parsed.freq],
  };
  if (parsed.interval !== 1) options.interval = parsed.interval;
  if (parsed.freq === "WEEKLY" && parsed.byweekday.length > 0) {
    options.byweekday = parsed.byweekday.map((day) => RRULE_WEEKDAY[day]);
  }
  if ((parsed.freq === "MONTHLY" || parsed.freq === "YEARLY") && parsed.nthWeekday.length > 0) {
    options.byweekday = parsed.nthWeekday.map((entry) =>
      RRULE_WEEKDAY[entry.weekday].nth(entry.ordinal),
    );
  } else if (
    (parsed.freq === "MONTHLY" || parsed.freq === "YEARLY") &&
    parsed.bymonthday.length > 0
  ) {
    options.bymonthday = parsed.bymonthday;
  }
  if (parsed.freq === "YEARLY" && parsed.bymonth.length > 0) {
    options.bymonth = parsed.bymonth;
  }
  if (parsed.until) options.until = new Date(`${parsed.until}T00:00:00Z`);
  return new RRule(options).toString();
}

export interface ParsedRecurrenceDuration {
  freq: RecurrenceFrequency;
  interval: number;
  label: string;
}

function duration(freq: RecurrenceFrequency, interval: number): ParsedRecurrenceDuration {
  const labels = FREQUENCY_LABELS[freq];
  return {
    freq,
    interval,
    label: `Every ${interval} ${interval === 1 ? labels.singular : labels.plural}`,
  };
}

export function parseRecurrenceDurationInput(input: string): ParsedRecurrenceDuration | null {
  const milliseconds = parseDuration(input);
  if (milliseconds === null || milliseconds <= 0) return null;
  const units: [RecurrenceFrequency, number][] = [
    ["YEARLY", 31_557_600_000],
    ["MONTHLY", 2_629_800_000],
    ["WEEKLY", 604_800_000],
    ["DAILY", 86_400_000],
  ];
  for (const [freq, size] of units) {
    if (milliseconds >= size && milliseconds % size === 0) {
      return duration(freq, Math.round(milliseconds / size));
    }
  }
  const days = Math.round(milliseconds / 86_400_000);
  return days > 0 ? duration("DAILY", days) : null;
}

export function describeRecurrenceMonthDays(days: number[]): string {
  const parts = days
    .filter((day) => day > 0)
    .sort((a, b) => a - b)
    .map(String);
  if (days.includes(-1)) parts.push("last");
  return parts.join(", ");
}

export function describeRule(parsed: ParsedRecurrenceRule): string {
  const labels = FREQUENCY_LABELS[parsed.freq];
  const unit = parsed.interval === 1 ? labels.singular : labels.plural;
  let description = parsed.interval === 1 ? `Every ${unit}` : `Every ${parsed.interval} ${unit}`;
  if (parsed.freq === "WEEKLY" && parsed.byweekday.length > 0) {
    description += ` on ${parsed.byweekday.map((day) => WEEKDAY_LABELS[day].slice(0, 3)).join(", ")}`;
  }
  if (parsed.freq === "MONTHLY" || parsed.freq === "YEARLY") {
    if (parsed.nthWeekday.length > 0) {
      const values = parsed.nthWeekday.map(
        (entry) =>
          `${RECURRENCE_ORDINAL_LABELS[entry.ordinal] ?? String(entry.ordinal)} ${WEEKDAY_LABELS[entry.weekday]}`,
      );
      description += ` on the ${values.join(", ")}`;
    } else if (parsed.bymonthday.length > 0) {
      description += ` on day ${describeRecurrenceMonthDays(parsed.bymonthday)}`;
    }
  }
  if (parsed.freq === "YEARLY" && parsed.bymonth.length > 0) {
    description += ` in ${parsed.bymonth.map((month) => RECURRENCE_MONTH_LABELS[month - 1]).join(", ")}`;
  }
  if (parsed.until) description += ` until ${parsed.until}`;
  return description;
}

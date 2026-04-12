import { Menu, Portal } from "@skeletonlabs/skeleton-react";
import { Calendar, Check, ChevronRight, RefreshCw, X } from "lucide-react";
import { useMemo } from "react";
import parseDuration from "parse-duration";
import { RRule, Weekday } from "rrule";
import type { components } from "../../api/schema.d.ts";
import { FieldPill } from "../FieldPill.tsx";
import { MENU_ITEM_CLASS, SearchableMenuContent } from "../SearchableMenuContent.tsx";
import { addDays, formatDateDisplay, parseDateInput, toISODate } from "../../lib/dates.ts";
import { useMenuSearch } from "../../lib/useMenuSearch.ts";
import { useTaskPatch } from "../../lib/mutations.ts";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";

type TaskRecurrenceType = components["schemas"]["TaskRecurrenceType"];

// Zag.js reads z-index from Menu.Content's computed style and propagates
// it to the positioner via --z-index. Apply z-index to Content (not
// Positioner) so Zag picks it up. Child portals render before parent
// portals in DOM, so deeper menus need higher z-index.
const Z_SUBMENU = "z-10";
const Z_DETAIL = "z-20";

type FreqCode = "DAILY" | "WEEKLY" | "MONTHLY" | "YEARLY";

const RECURRENCE_TYPE_LABELS: Record<TaskRecurrenceType, string> = {
  one_off: "One-off",
  completion_based: "On completion",
  fixed_non_accumulating: "Fixed",
  fixed_accumulating: "Fixed (accum.)",
  on_dependency: "On dependency",
};

const TYPES_WITH_RULE = new Set<TaskRecurrenceType>([
  "completion_based",
  "fixed_non_accumulating",
  "fixed_accumulating",
]);

const FREQ_LABELS: Record<FreqCode, { singular: string; plural: string }> = {
  DAILY: { singular: "day", plural: "days" },
  WEEKLY: { singular: "week", plural: "weeks" },
  MONTHLY: { singular: "month", plural: "months" },
  YEARLY: { singular: "year", plural: "years" },
};

const WEEKDAY_CODES = ["MO", "TU", "WE", "TH", "FR", "SA", "SU"] as const;
type WeekdayCode = (typeof WEEKDAY_CODES)[number];

const WEEKDAY_LABELS: Record<WeekdayCode, string> = {
  MO: "Monday",
  TU: "Tuesday",
  WE: "Wednesday",
  TH: "Thursday",
  FR: "Friday",
  SA: "Saturday",
  SU: "Sunday",
};

const WEEKDAY_SHORT_LABELS: Record<WeekdayCode, string> = {
  MO: "M",
  TU: "Tu",
  WE: "W",
  TH: "Th",
  FR: "F",
  SA: "Sa",
  SU: "Su",
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

const RRULE_FREQ_TO_CODE: Partial<Record<number, FreqCode>> = {
  [RRule.DAILY]: "DAILY",
  [RRule.WEEKLY]: "WEEKLY",
  [RRule.MONTHLY]: "MONTHLY",
  [RRule.YEARLY]: "YEARLY",
};

const FREQ_TO_RRULE: Record<FreqCode, number> = {
  DAILY: RRule.DAILY,
  WEEKLY: RRule.WEEKLY,
  MONTHLY: RRule.MONTHLY,
  YEARLY: RRule.YEARLY,
};

const ORDINAL_LABELS: Record<number, string> = {
  1: "1st",
  2: "2nd",
  3: "3rd",
  4: "4th",
  [-1]: "Last",
};

const DAY_NUMBERS = Array.from({ length: 31 }, (_, i) => i + 1);

const ORDINALS = [1, 2, 3, 4, -1] as const;

const MONTH_SHORT_LABELS = [
  "Jan",
  "Feb",
  "Mar",
  "Apr",
  "May",
  "Jun",
  "Jul",
  "Aug",
  "Sep",
  "Oct",
  "Nov",
  "Dec",
];

const MONTH_LABELS = [
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

// ─── RRULE Parsing ──────────────────────────────────────────────────────────

export interface ParsedRule {
  freq: FreqCode;
  interval: number;
  byweekday: WeekdayCode[];
  bymonthday: number[];
  nthWeekday: { ordinal: number; weekday: WeekdayCode } | null;
  bymonth: number[];
  until: string | null;
  parseError: boolean;
}

function toArray<T>(value: T | T[]): T[] {
  return Array.isArray(value) ? value : [value];
}

export function parseRRule(rruleStr: string | null): ParsedRule {
  const defaults: ParsedRule = {
    freq: "WEEKLY",
    interval: 1,
    byweekday: [],
    bymonthday: [],
    nthWeekday: null,
    bymonth: [],
    until: null,
    parseError: false,
  };

  if (!rruleStr) return defaults;

  try {
    const normalized = rruleStr.startsWith("RRULE:") ? rruleStr : `RRULE:${rruleStr}`;
    const rule = RRule.fromString(normalized);
    const opts = rule.origOptions;

    const freq = RRULE_FREQ_TO_CODE[opts.freq ?? RRule.WEEKLY] ?? "WEEKLY";
    const interval = opts.interval ?? 1;
    const byweekday: WeekdayCode[] = [];
    const bymonthday: number[] = [];
    let nthWeekday: ParsedRule["nthWeekday"] = null;

    if (opts.byweekday) {
      for (const w of toArray(opts.byweekday)) {
        if (w instanceof Weekday) {
          if (w.n !== undefined && w.n !== 0) {
            const code = WEEKDAY_CODES[w.weekday];
            if (code) nthWeekday = { ordinal: w.n, weekday: code };
          } else {
            const code = WEEKDAY_CODES[w.weekday];
            if (code) byweekday.push(code);
          }
        }
      }
    }

    if (opts.bymonthday) {
      bymonthday.push(...toArray(opts.bymonthday));
    }

    const bymonth: number[] = opts.bymonth ? toArray(opts.bymonth) : [];

    let until: string | null = null;
    if (opts.until) {
      const d = opts.until;
      until = `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, "0")}-${String(d.getUTCDate()).padStart(2, "0")}`;
    }

    return { freq, interval, byweekday, bymonthday, nthWeekday, bymonth, until, parseError: false };
  } catch {
    return { ...defaults, parseError: true };
  }
}

export function buildRRule(parsed: ParsedRule): string {
  const options: ConstructorParameters<typeof RRule>[0] = {
    freq: FREQ_TO_RRULE[parsed.freq],
  };
  if (parsed.interval !== 1) options.interval = parsed.interval;
  if (parsed.freq === "WEEKLY" && parsed.byweekday.length > 0) {
    options.byweekday = parsed.byweekday.map((d) => RRULE_WEEKDAY[d]);
  }
  if ((parsed.freq === "MONTHLY" || parsed.freq === "YEARLY") && parsed.nthWeekday) {
    options.byweekday = [RRULE_WEEKDAY[parsed.nthWeekday.weekday].nth(parsed.nthWeekday.ordinal)];
  } else if (
    (parsed.freq === "MONTHLY" || parsed.freq === "YEARLY") &&
    parsed.bymonthday.length > 0
  ) {
    options.bymonthday = parsed.bymonthday;
  }
  if (parsed.freq === "YEARLY" && parsed.bymonth.length > 0) {
    options.bymonth = parsed.bymonth;
  }
  if (parsed.until) {
    options.until = new Date(parsed.until + "T00:00:00Z");
  }
  return new RRule(options).toString();
}

// ─── Duration Parsing ───────────────────────────────────────────────────────

const MS_DAY = 86400000;
const MS_WEEK = 604800000;
const MS_MONTH = 2629800000;
const MS_YEAR = 31557600000;

export interface ParsedDuration {
  freq: FreqCode;
  interval: number;
  label: string;
}

function formatFreqLabel(freq: FreqCode, interval: number): string {
  const { singular, plural } = FREQ_LABELS[freq];
  return `Every ${interval} ${interval === 1 ? singular : plural}`;
}

export function parseDurationInput(input: string): ParsedDuration | null {
  const ms = parseDuration(input);
  if (ms == null || ms <= 0) return null;

  if (ms >= MS_YEAR && ms % MS_YEAR === 0) {
    const n = Math.round(ms / MS_YEAR);
    return { freq: "YEARLY", interval: n, label: formatFreqLabel("YEARLY", n) };
  }
  if (ms >= MS_MONTH && ms % MS_MONTH === 0) {
    const n = Math.round(ms / MS_MONTH);
    return { freq: "MONTHLY", interval: n, label: formatFreqLabel("MONTHLY", n) };
  }
  if (ms >= MS_WEEK && ms % MS_WEEK === 0) {
    const n = Math.round(ms / MS_WEEK);
    return { freq: "WEEKLY", interval: n, label: formatFreqLabel("WEEKLY", n) };
  }
  if (ms >= MS_DAY && ms % MS_DAY === 0) {
    const n = Math.round(ms / MS_DAY);
    return { freq: "DAILY", interval: n, label: formatFreqLabel("DAILY", n) };
  }

  const days = Math.round(ms / MS_DAY);
  if (days > 0) {
    return { freq: "DAILY", interval: days, label: formatFreqLabel("DAILY", days) };
  }

  return null;
}

// ─── Display Helpers ────────────────────────────────────────────────────────

function describeMonthDays(bymonthday: number[]): string {
  const hasLast = bymonthday.includes(-1);
  const positives = bymonthday.filter((d) => d > 0).sort((a, b) => a - b);
  const parts: string[] = [];
  if (positives.length > 0) parts.push(...positives.map(String));
  if (hasLast) parts.push("last");
  return parts.join(", ");
}

export function describeRule(parsed: ParsedRule): string {
  const labels = FREQ_LABELS[parsed.freq];
  const unit = parsed.interval === 1 ? labels.singular : labels.plural;
  let desc = parsed.interval === 1 ? `Every ${unit}` : `Every ${parsed.interval} ${unit}`;

  if (parsed.freq === "WEEKLY" && parsed.byweekday.length > 0) {
    desc += ` on ${parsed.byweekday.map((d) => WEEKDAY_LABELS[d].slice(0, 3)).join(", ")}`;
  }
  if (parsed.freq === "MONTHLY" || parsed.freq === "YEARLY") {
    if (parsed.nthWeekday) {
      const ordLabel =
        ORDINAL_LABELS[parsed.nthWeekday.ordinal] ?? String(parsed.nthWeekday.ordinal);
      desc += ` on the ${ordLabel} ${WEEKDAY_LABELS[parsed.nthWeekday.weekday]}`;
    } else if (parsed.bymonthday.length > 0) {
      desc += ` on day ${describeMonthDays(parsed.bymonthday)}`;
    }
  }
  if (parsed.freq === "YEARLY" && parsed.bymonth.length > 0) {
    desc += ` in ${parsed.bymonth.map((m) => MONTH_LABELS[m - 1]).join(", ")}`;
  }
  if (parsed.until) {
    desc += ` until ${parsed.until}`;
  }

  return desc;
}

function getDisplayValue(
  recurrenceType: TaskRecurrenceType,
  recurrenceRule: string | null,
): string {
  if (recurrenceType === "one_off") return "One-off";
  if (recurrenceType === "on_dependency") return "On dependency";

  const typeLabel = RECURRENCE_TYPE_LABELS[recurrenceType];
  if (!TYPES_WITH_RULE.has(recurrenceType) || !recurrenceRule) return typeLabel;

  const parsed = parseRRule(recurrenceRule);
  return `${typeLabel}: ${describeRule(parsed)}`;
}

// ─── Frequency Submenu ──────────────────────────────────────────────────────

const FREQ_SHORTCUTS = [
  { label: "Daily", freq: "DAILY" as const },
  { label: "Weekly", freq: "WEEKLY" as const },
  { label: "Monthly", freq: "MONTHLY" as const },
  { label: "Yearly", freq: "YEARLY" as const },
];

function FreqSubMenu({
  recurrenceType,
  currentRule,
  onSave,
  className,
}: {
  recurrenceType: TaskRecurrenceType;
  currentRule: ParsedRule;
  onSave: (update: { recurrenceType: TaskRecurrenceType; recurrenceRule?: string | null }) => void;
  className?: string;
}) {
  const search = useMenuSearch();
  const parsedDuration = useMemo(() => parseDurationInput(search.query), [search.query]);

  function selectFreq(freq: FreqCode, interval: number) {
    const newRule: ParsedRule = {
      ...currentRule,
      freq,
      interval,
      byweekday: freq === "WEEKLY" ? currentRule.byweekday : [],
      bymonthday: freq === "MONTHLY" || freq === "YEARLY" ? currentRule.bymonthday : [],
      nthWeekday: freq === "MONTHLY" || freq === "YEARLY" ? currentRule.nthWeekday : null,
      bymonth: freq === "YEARLY" ? currentRule.bymonth : [],
    };
    onSave({ recurrenceType, recurrenceRule: buildRRule(newRule) });
  }

  function toggleWeekday(day: WeekdayCode) {
    const newDays = currentRule.byweekday.includes(day)
      ? currentRule.byweekday.filter((d) => d !== day)
      : [...currentRule.byweekday, day];
    const newRule: ParsedRule = { ...currentRule, byweekday: newDays };
    onSave({ recurrenceType, recurrenceRule: buildRRule(newRule) });
  }

  function toggleMonthDay(day: number) {
    const newDays = currentRule.bymonthday.includes(day)
      ? currentRule.bymonthday.filter((d) => d !== day)
      : [...currentRule.bymonthday, day].sort((a, b) => a - b);
    const newRule: ParsedRule = {
      ...currentRule,
      bymonthday: newDays,
      nthWeekday: null,
      byweekday: [],
    };
    onSave({ recurrenceType, recurrenceRule: buildRRule(newRule) });
  }

  function selectNthWeekday(nthWeekday: ParsedRule["nthWeekday"]) {
    const newRule: ParsedRule = {
      ...currentRule,
      bymonthday: [],
      nthWeekday,
      byweekday: [],
    };
    onSave({ recurrenceType, recurrenceRule: buildRRule(newRule) });
  }

  function toggleYearlyMonth(monthIndex: number) {
    const month = monthIndex + 1;
    const newMonths = currentRule.bymonth.includes(month)
      ? currentRule.bymonth.filter((m) => m !== month)
      : [...currentRule.bymonth, month].sort((a, b) => a - b);
    const newRule: ParsedRule = { ...currentRule, bymonth: newMonths };
    onSave({ recurrenceType, recurrenceRule: buildRRule(newRule) });
  }

  const filteredShortcuts = useMemo(() => {
    if (!search.query) return FREQ_SHORTCUTS;
    return FREQ_SHORTCUTS.filter((s) => s.label.toLowerCase().includes(search.query.toLowerCase()));
  }, [search.query]);

  const isSearching = search.query.length > 0;

  return (
    <SearchableMenuContent
      inputProps={search.inputProps}
      placeholder='e.g. "2 weeks", "3 months"'
      className={className}
    >
      {/* Parsed duration from search input */}
      {isSearching && parsedDuration && (
        <Menu.Item
          value={`parsed-${parsedDuration.freq}-${parsedDuration.interval}`}
          className={MENU_ITEM_CLASS}
          onClick={() => selectFreq(parsedDuration.freq, parsedDuration.interval)}
        >
          <Menu.ItemText>{parsedDuration.label}</Menu.ItemText>
        </Menu.Item>
      )}

      {/* Frequency shortcuts */}
      {filteredShortcuts.map((shortcut) => {
        const isCurrent = currentRule.freq === shortcut.freq;
        return (
          <Menu.OptionItem
            key={shortcut.label}
            type="radio"
            checked={isCurrent}
            value={shortcut.label}
            onCheckedChange={(checked) => {
              if (checked)
                selectFreq(
                  shortcut.freq,
                  currentRule.freq === shortcut.freq ? currentRule.interval : 1,
                );
            }}
            className={MENU_ITEM_CLASS}
          >
            <Menu.ItemIndicator className="invisible data-[state=checked]:visible">
              <Check className="size-4" />
            </Menu.ItemIndicator>
            <Menu.ItemText>{shortcut.label}</Menu.ItemText>
          </Menu.OptionItem>
        );
      })}

      {isSearching && !parsedDuration && filteredShortcuts.length === 0 && (
        <div className="text-surface-500 px-3 py-2 text-sm">No matching options</div>
      )}

      {/* Conditional submenu entries for freq-specific options */}
      {currentRule.freq === "WEEKLY" && (
        <>
          <Menu.Separator />
          <Menu typeahead={false} closeOnSelect={false}>
            <Menu.TriggerItem value="weekdays" className="justify-start gap-2 text-sm">
              <Menu.ItemText>
                {currentRule.byweekday.length > 0
                  ? `On ${currentRule.byweekday.map((d) => WEEKDAY_SHORT_LABELS[d]).join(", ")}`
                  : "On days..."}
              </Menu.ItemText>
              <Menu.ItemIndicator className="ml-auto">
                <ChevronRight className="size-4" />
              </Menu.ItemIndicator>
            </Menu.TriggerItem>
            <Portal>
              <Menu.Positioner>
                <Menu.Content className={Z_DETAIL}>
                  <div className="flex items-center gap-1">
                    {WEEKDAY_CODES.map((day) => {
                      const active = currentRule.byweekday.includes(day);
                      return (
                        <button
                          key={day}
                          type="button"
                          onClick={() => toggleWeekday(day)}
                          aria-label={WEEKDAY_LABELS[day]}
                          aria-pressed={active}
                          className={`flex size-7 items-center justify-center rounded-full text-xs font-medium transition-colors ${
                            active
                              ? "preset-filled-primary-500"
                              : "preset-outlined-surface-200-800 hover:preset-tonal-surface"
                          }`}
                        >
                          {WEEKDAY_SHORT_LABELS[day]}
                        </button>
                      );
                    })}
                  </div>
                </Menu.Content>
              </Menu.Positioner>
            </Portal>
          </Menu>
        </>
      )}

      {currentRule.freq === "MONTHLY" && (
        <>
          <Menu.Separator />
          <Menu typeahead={false} closeOnSelect={false}>
            <Menu.TriggerItem value="monthday" className="justify-start gap-2 text-sm">
              <Menu.ItemText>
                {currentRule.nthWeekday
                  ? `On the ${ORDINAL_LABELS[currentRule.nthWeekday.ordinal] ?? ""} ${WEEKDAY_LABELS[currentRule.nthWeekday.weekday]}`
                  : currentRule.bymonthday.length > 0
                    ? `On day ${describeMonthDays(currentRule.bymonthday)}`
                    : "On day..."}
              </Menu.ItemText>
              <Menu.ItemIndicator className="ml-auto">
                <ChevronRight className="size-4" />
              </Menu.ItemIndicator>
            </Menu.TriggerItem>
            <Portal>
              <Menu.Positioner>
                <Menu.Content className={Z_DETAIL}>
                  <div className="text-surface-500 mb-1.5 text-xs">Day of month</div>
                  <div className="grid grid-cols-7 gap-1">
                    {DAY_NUMBERS.map((d) => {
                      const hasLast =
                        currentRule.bymonthday.includes(-1) && !currentRule.nthWeekday;
                      const active = currentRule.bymonthday.includes(d) && !currentRule.nthWeekday;
                      const isShortMonthDay = d >= 29;
                      const lastDayHint = hasLast && d >= 28 && !active;
                      return (
                        <button
                          key={d}
                          type="button"
                          onClick={() => toggleMonthDay(d)}
                          aria-pressed={active}
                          className={`flex size-7 items-center justify-center rounded text-xs font-medium transition-colors ${
                            active
                              ? "preset-filled-primary-500"
                              : lastDayHint
                                ? "preset-tonal-primary hover:preset-tonal-surface"
                                : isShortMonthDay
                                  ? "outline outline-1 outline-dashed outline-surface-300-700 hover:preset-tonal-surface"
                                  : "preset-outlined-surface-200-800 hover:preset-tonal-surface"
                          }`}
                        >
                          {d}
                        </button>
                      );
                    })}
                    <div className="col-span-4">
                      <button
                        type="button"
                        onClick={() => toggleMonthDay(-1)}
                        aria-pressed={
                          currentRule.bymonthday.includes(-1) && !currentRule.nthWeekday
                        }
                        className={`flex h-7 items-center rounded px-2 text-xs font-medium transition-colors ${
                          currentRule.bymonthday.includes(-1) && !currentRule.nthWeekday
                            ? "preset-filled-primary-500"
                            : "preset-outlined-surface-200-800 hover:preset-tonal-surface"
                        }`}
                      >
                        Last
                      </button>
                    </div>
                  </div>
                  <div className="text-surface-500 mt-1 text-xs">29–31 skipped in short months</div>
                  <Menu.Separator />
                  <div className="text-surface-500 mb-1.5 text-xs">Or on the Nth weekday</div>
                  <div className="mb-1 flex items-center justify-between">
                    {ORDINALS.map((ord) => {
                      const active = currentRule.nthWeekday?.ordinal === ord;
                      return (
                        <button
                          key={ord}
                          type="button"
                          onClick={() =>
                            selectNthWeekday({
                              ordinal: ord,
                              weekday: currentRule.nthWeekday?.weekday ?? "MO",
                            })
                          }
                          aria-label={`${ORDINAL_LABELS[ord]} week of month`}
                          aria-pressed={active}
                          className={`rounded px-2 py-1 text-xs font-medium transition-colors ${
                            active
                              ? "preset-filled-primary-500"
                              : "preset-outlined-surface-200-800 hover:preset-tonal-surface"
                          }`}
                        >
                          {ORDINAL_LABELS[ord]}
                        </button>
                      );
                    })}
                  </div>
                  <div className="flex items-center justify-between">
                    {WEEKDAY_CODES.map((day) => {
                      const active = currentRule.nthWeekday?.weekday === day;
                      return (
                        <button
                          key={day}
                          type="button"
                          onClick={() =>
                            selectNthWeekday({
                              ordinal: currentRule.nthWeekday?.ordinal ?? 1,
                              weekday: day,
                            })
                          }
                          aria-label={WEEKDAY_LABELS[day]}
                          aria-pressed={active}
                          className={`flex size-7 items-center justify-center rounded-full text-xs font-medium transition-colors ${
                            active
                              ? "preset-filled-primary-500"
                              : "preset-outlined-surface-200-800 hover:preset-tonal-surface"
                          }`}
                        >
                          {WEEKDAY_SHORT_LABELS[day]}
                        </button>
                      );
                    })}
                  </div>
                </Menu.Content>
              </Menu.Positioner>
            </Portal>
          </Menu>
        </>
      )}

      {currentRule.freq === "YEARLY" && (
        <>
          <Menu.Separator />
          <Menu typeahead={false} closeOnSelect={false}>
            <Menu.TriggerItem value="months" className="justify-start gap-2 text-sm">
              <Menu.ItemText>
                {currentRule.bymonth.length > 0
                  ? `In ${currentRule.bymonth.map((m) => MONTH_SHORT_LABELS[m - 1]).join(", ")}`
                  : "In months..."}
              </Menu.ItemText>
              <Menu.ItemIndicator className="ml-auto">
                <ChevronRight className="size-4" />
              </Menu.ItemIndicator>
            </Menu.TriggerItem>
            <Portal>
              <Menu.Positioner>
                <Menu.Content className={Z_DETAIL}>
                  <div className="grid grid-cols-6 gap-1">
                    {MONTH_SHORT_LABELS.map((label, index) => {
                      const month = index + 1;
                      const active = currentRule.bymonth.includes(month);
                      return (
                        <button
                          key={label}
                          type="button"
                          onClick={() => toggleYearlyMonth(index)}
                          aria-label={MONTH_LABELS[index]}
                          aria-pressed={active}
                          className={`rounded px-1.5 py-1 text-xs font-medium transition-colors ${
                            active
                              ? "preset-filled-primary-500"
                              : "preset-outlined-surface-200-800 hover:preset-tonal-surface"
                          }`}
                        >
                          {label}
                        </button>
                      );
                    })}
                  </div>
                </Menu.Content>
              </Menu.Positioner>
            </Portal>
          </Menu>
        </>
      )}

      {/* Until date */}
      <Menu typeahead={false} closeOnSelect={false}>
        <Menu.TriggerItem value="until" className="justify-start gap-2 text-sm">
          <Menu.ItemText>
            {currentRule.until ? `Until ${currentRule.until}` : "Until (no end date)"}
          </Menu.ItemText>
          <Menu.ItemIndicator className="ml-auto">
            <ChevronRight className="size-4" />
          </Menu.ItemIndicator>
        </Menu.TriggerItem>
        <Portal>
          <Menu.Positioner>
            <UntilDateSubMenu
              className={Z_DETAIL}
              currentUntil={currentRule.until}
              currentRule={currentRule}
              recurrenceType={recurrenceType}
              onSave={onSave}
            />
          </Menu.Positioner>
        </Portal>
      </Menu>
    </SearchableMenuContent>
  );
}

// ─── Until Date Submenu ─────────────────────────────────────────────────────

function UntilDateSubMenu({
  currentUntil,
  currentRule,
  recurrenceType,
  onSave,
  className,
}: {
  currentUntil: string | null;
  currentRule: ParsedRule;
  recurrenceType: TaskRecurrenceType;
  onSave: (update: { recurrenceType: TaskRecurrenceType; recurrenceRule?: string | null }) => void;
  className?: string;
}) {
  const search = useMenuSearch();
  const parsedDate = useMemo(() => parseDateInput(search.query), [search.query]);
  const today = new Date();

  function selectUntil(isoDate: string | null) {
    const newRule: ParsedRule = { ...currentRule, until: isoDate };
    onSave({ recurrenceType, recurrenceRule: buildRRule(newRule) });
  }

  const shortcuts = [
    { label: "In 3 months", offsetDays: 90 },
    { label: "In 6 months", offsetDays: 180 },
    { label: "In 1 year", offsetDays: 365 },
    { label: "In 2 years", offsetDays: 730 },
  ];

  return (
    <SearchableMenuContent
      inputProps={search.inputProps}
      placeholder='e.g. "dec 2026", "in 6 months"'
      className={className}
    >
      {currentUntil && (
        <Menu.Item
          value="clear-until"
          className={`text-error-500 ${MENU_ITEM_CLASS}`}
          onClick={() => selectUntil(null)}
        >
          <X className="size-4" aria-hidden="true" />
          <Menu.ItemText>No end date</Menu.ItemText>
        </Menu.Item>
      )}

      {search.query ? (
        parsedDate ? (
          <Menu.Item
            value={parsedDate.value}
            className={MENU_ITEM_CLASS}
            onClick={() => selectUntil(parsedDate.value)}
          >
            <Calendar className="size-4" aria-hidden="true" />
            <Menu.ItemText>{parsedDate.label}</Menu.ItemText>
            <span className="text-surface-500 ml-auto text-xs">{parsedDate.value}</span>
          </Menu.Item>
        ) : (
          <div className="text-surface-500 px-3 py-2 text-sm">No matching dates</div>
        )
      ) : (
        shortcuts.map((shortcut) => {
          const date = addDays(today, shortcut.offsetDays);
          const isoDate = toISODate(date);
          return (
            <Menu.Item
              key={shortcut.label}
              value={isoDate}
              className={MENU_ITEM_CLASS}
              onClick={() => selectUntil(isoDate)}
            >
              <Menu.ItemText>{shortcut.label}</Menu.ItemText>
              <span className="text-surface-500 ml-auto text-xs">{formatDateDisplay(date)}</span>
            </Menu.Item>
          );
        })
      )}
    </SearchableMenuContent>
  );
}

// ─── Main Component ─────────────────────────────────────────────────────────

export function RecurrenceMenuField({
  spaceSlug,
  taskId,
  recurrenceType,
  recurrenceRule,
}: {
  spaceSlug: string;
  taskId: string;
  recurrenceType: TaskRecurrenceType;
  recurrenceRule: string | null;
}) {
  const mutation = useTaskPatch(spaceSlug, taskId);
  const search = useMenuSearch();

  const currentRule = useMemo(() => parseRRule(recurrenceRule), [recurrenceRule]);
  const displayValue = useMemo(
    () => getDisplayValue(recurrenceType, recurrenceRule),
    [recurrenceType, recurrenceRule],
  );

  const handleSave = mutation.mutate;

  const typeItems = useMemo(() => {
    const items = [
      { value: "one_off" as const, label: "One-off", hasSubmenu: false },
      { value: "completion_based" as const, label: "On completion", hasSubmenu: true },
      { value: "fixed_non_accumulating" as const, label: "Fixed", hasSubmenu: true },
      { value: "fixed_accumulating" as const, label: "Fixed (accumulating)", hasSubmenu: true },
      { value: "on_dependency" as const, label: "On dependency", hasSubmenu: false },
    ];
    if (!search.query) return items;
    return items.filter((item) => item.label.toLowerCase().includes(search.query.toLowerCase()));
  }, [search.query]);

  return (
    <>
      <Menu {...search.menuProps} closeOnSelect={false}>
        <FieldPill
          icon={<RefreshCw className="size-3.5" aria-hidden="true" />}
          label="Recurrence"
          value={displayValue}
        />
        <Portal>
          <Menu.Positioner>
            <SearchableMenuContent
              inputProps={search.inputProps}
              placeholder="Search recurrence..."
            >
              {typeItems.length === 0 ? (
                <div className="text-surface-500 px-3 py-2 text-sm">No matching types</div>
              ) : (
                typeItems.map((item) =>
                  item.hasSubmenu ? (
                    <Menu key={item.value} typeahead={false} closeOnSelect={false}>
                      <Menu.TriggerItem value={item.value} className="justify-start gap-2 text-sm">
                        {recurrenceType === item.value ? (
                          <Check className="size-4" aria-hidden="true" />
                        ) : (
                          <span className="size-4" />
                        )}
                        <Menu.ItemText>{item.label}</Menu.ItemText>
                        <Menu.ItemIndicator className="ml-auto">
                          <ChevronRight className="size-4" />
                        </Menu.ItemIndicator>
                      </Menu.TriggerItem>
                      <Portal>
                        <Menu.Positioner>
                          <FreqSubMenu
                            className={Z_SUBMENU}
                            recurrenceType={item.value}
                            currentRule={currentRule}
                            onSave={handleSave}
                          />
                        </Menu.Positioner>
                      </Portal>
                    </Menu>
                  ) : (
                    <Menu.Item
                      key={item.value}
                      value={item.value}
                      className={MENU_ITEM_CLASS}
                      onClick={() => {
                        const update: Parameters<typeof handleSave>[0] = {
                          recurrenceType: item.value,
                        };
                        if (item.value === "one_off") update.recurrenceRule = null;
                        handleSave(update);
                      }}
                    >
                      {recurrenceType === item.value ? (
                        <Check className="size-4" aria-hidden="true" />
                      ) : (
                        <span className="size-4" />
                      )}
                      <Menu.ItemText>{item.label}</Menu.ItemText>
                    </Menu.Item>
                  ),
                )
              )}
            </SearchableMenuContent>
          </Menu.Positioner>
        </Portal>
      </Menu>
      {currentRule.parseError && (
        <div className="text-warning-500 text-xs">Could not parse recurrence rule</div>
      )}
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </>
  );
}

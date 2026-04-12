import { Menu, Portal } from "@skeletonlabs/skeleton-react";
import * as chrono from "chrono-node/en";
import { Calendar, Check, ChevronRight, RefreshCw, X } from "lucide-react";
import { useMemo } from "react";
import parseDuration from "parse-duration";
import { RRule, Weekday } from "rrule";
import type { components } from "../../api/schema.d.ts";
import { FieldPill } from "../FieldPill.tsx";
import { SearchableMenuContent } from "../SearchableMenuContent.tsx";
import { useMenuSearch } from "../../lib/useMenuSearch.ts";
import { useTaskPatch } from "../../lib/mutations.ts";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";

type TaskRecurrenceType = components["schemas"]["TaskRecurrenceType"];

// ─── Constants ──────────────────────────────────────────────────────────────

const ITEM_CLASS = "justify-start gap-2 text-sm";

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

const RRULE_WEEKDAY: Record<WeekdayCode, Weekday> = {
  MO: RRule.MO,
  TU: RRule.TU,
  WE: RRule.WE,
  TH: RRule.TH,
  FR: RRule.FR,
  SA: RRule.SA,
  SU: RRule.SU,
};

const WEEKDAY_INDEX_TO_CODE: WeekdayCode[] = ["MO", "TU", "WE", "TH", "FR", "SA", "SU"];

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

const DAY_NUMBERS = Array.from({ length: 28 }, (_, i) => i + 1);

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

interface ParsedRule {
  freq: FreqCode;
  interval: number;
  byweekday: WeekdayCode[];
  bymonthday: number | null;
  nthWeekday: { ordinal: number; weekday: WeekdayCode } | null;
  bymonth: number[];
  until: string | null;
}

function toArray<T>(value: T | T[]): T[] {
  return Array.isArray(value) ? value : [value];
}

function parseRRule(rruleStr: string | null): ParsedRule {
  const defaults: ParsedRule = {
    freq: "WEEKLY",
    interval: 1,
    byweekday: [],
    bymonthday: null,
    nthWeekday: null,
    bymonth: [],
    until: null,
  };

  if (!rruleStr) return defaults;

  try {
    const normalized = rruleStr.startsWith("RRULE:") ? rruleStr : `RRULE:${rruleStr}`;
    const rule = RRule.fromString(normalized);
    const opts = rule.origOptions;

    const freq = RRULE_FREQ_TO_CODE[opts.freq ?? RRule.WEEKLY] ?? "WEEKLY";
    const interval = opts.interval ?? 1;
    const byweekday: WeekdayCode[] = [];
    let bymonthday: number | null = null;
    let nthWeekday: ParsedRule["nthWeekday"] = null;

    if (opts.byweekday) {
      for (const w of toArray(opts.byweekday)) {
        if (w instanceof Weekday) {
          if (w.n !== undefined && w.n !== 0) {
            const code = WEEKDAY_INDEX_TO_CODE[w.weekday];
            if (code) nthWeekday = { ordinal: w.n, weekday: code };
          } else {
            const code = WEEKDAY_INDEX_TO_CODE[w.weekday];
            if (code) byweekday.push(code);
          }
        }
      }
    }

    if (opts.bymonthday) {
      const days = toArray(opts.bymonthday);
      if (days[0] != null) bymonthday = days[0];
    }

    const bymonth: number[] = opts.bymonth ? toArray(opts.bymonth) : [];

    let until: string | null = null;
    if (opts.until) {
      const d = opts.until;
      until = `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, "0")}-${String(d.getUTCDate()).padStart(2, "0")}`;
    }

    return { freq, interval, byweekday, bymonthday, nthWeekday, bymonth, until };
  } catch {
    return defaults;
  }
}

function buildRRule(parsed: ParsedRule): string {
  const options: ConstructorParameters<typeof RRule>[0] = {
    freq: FREQ_TO_RRULE[parsed.freq],
  };
  if (parsed.interval !== 1) options.interval = parsed.interval;
  if (parsed.freq === "WEEKLY" && parsed.byweekday.length > 0) {
    options.byweekday = parsed.byweekday.map((d) => RRULE_WEEKDAY[d]);
  }
  if ((parsed.freq === "MONTHLY" || parsed.freq === "YEARLY") && parsed.nthWeekday) {
    options.byweekday = [RRULE_WEEKDAY[parsed.nthWeekday.weekday].nth(parsed.nthWeekday.ordinal)];
  } else if ((parsed.freq === "MONTHLY" || parsed.freq === "YEARLY") && parsed.bymonthday != null) {
    options.bymonthday = [parsed.bymonthday];
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
const MS_MONTH = 2629800000; // parse-duration's month
const MS_YEAR = 31557600000; // parse-duration's year

interface ParsedDuration {
  freq: FreqCode;
  interval: number;
  label: string;
}

function parseDurationInput(input: string): ParsedDuration | null {
  const ms = parseDuration(input);
  if (ms == null || ms <= 0) return null;

  // Try to find the best-fit frequency
  if (ms >= MS_YEAR && ms % MS_YEAR === 0) {
    const n = Math.round(ms / MS_YEAR);
    return { freq: "YEARLY", interval: n, label: `Every ${n} ${n === 1 ? "year" : "years"}` };
  }
  if (ms >= MS_MONTH && ms % MS_MONTH === 0) {
    const n = Math.round(ms / MS_MONTH);
    return { freq: "MONTHLY", interval: n, label: `Every ${n} ${n === 1 ? "month" : "months"}` };
  }
  if (ms >= MS_WEEK && ms % MS_WEEK === 0) {
    const n = Math.round(ms / MS_WEEK);
    return { freq: "WEEKLY", interval: n, label: `Every ${n} ${n === 1 ? "week" : "weeks"}` };
  }
  if (ms >= MS_DAY && ms % MS_DAY === 0) {
    const n = Math.round(ms / MS_DAY);
    return { freq: "DAILY", interval: n, label: `Every ${n} ${n === 1 ? "day" : "days"}` };
  }

  // Fallback: approximate to days
  const days = Math.round(ms / MS_DAY);
  if (days > 0) {
    return { freq: "DAILY", interval: days, label: `Every ${days} ${days === 1 ? "day" : "days"}` };
  }

  return null;
}

// ─── Display Helpers ────────────────────────────────────────────────────────

function describeRule(parsed: ParsedRule): string {
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
    } else if (parsed.bymonthday != null) {
      desc += ` on day ${parsed.bymonthday}`;
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

// ─── Date helpers (reused from DueDateField) ────────────────────────────────

function toISODate(date: Date): string {
  const y = date.getFullYear().toString().padStart(4, "0");
  const m = (date.getMonth() + 1).toString().padStart(2, "0");
  const d = date.getDate().toString().padStart(2, "0");
  return `${y}-${m}-${d}`;
}

function formatDateDisplay(date: Date): string {
  return date.toLocaleDateString(undefined, {
    weekday: "short",
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function parseDateInput(input: string): { label: string; value: string } | null {
  if (!input.trim()) return null;
  const parsed = chrono.parseDate(input);
  if (!parsed) return null;
  return { label: formatDateDisplay(parsed), value: toISODate(parsed) };
}

// ─── Frequency Submenu ──────────────────────────────────────────────────────

interface FreqSubMenuProps {
  recurrenceType: TaskRecurrenceType;
  currentRule: ParsedRule;
  onSave: (update: { recurrenceType: TaskRecurrenceType; recurrenceRule?: string | null }) => void;
}

function FreqSubMenu({ recurrenceType, currentRule, onSave }: FreqSubMenuProps) {
  const search = useMenuSearch();
  const parsedDuration = useMemo(() => parseDurationInput(search.query), [search.query]);

  const isCurrentType = currentRule.freq !== undefined;

  function selectFreq(freq: FreqCode, interval: number) {
    const newRule: ParsedRule = {
      ...currentRule,
      freq,
      interval,
      // Reset freq-specific options when changing frequency
      byweekday: freq === "WEEKLY" ? currentRule.byweekday : [],
      bymonthday: freq === "MONTHLY" || freq === "YEARLY" ? currentRule.bymonthday : null,
      nthWeekday: freq === "MONTHLY" || freq === "YEARLY" ? currentRule.nthWeekday : null,
      bymonth: freq === "YEARLY" ? currentRule.bymonth : [],
    };
    onSave({ recurrenceType, recurrenceRule: buildRRule(newRule) });
  }

  const filteredShortcuts = useMemo(() => {
    const shortcuts = [
      { label: "Daily", freq: "DAILY" as const, interval: 1 },
      { label: "Weekly", freq: "WEEKLY" as const, interval: 1 },
      { label: "Every 2 weeks", freq: "WEEKLY" as const, interval: 2 },
      { label: "Monthly", freq: "MONTHLY" as const, interval: 1 },
      { label: "Yearly", freq: "YEARLY" as const, interval: 1 },
    ];
    if (!search.query) return shortcuts;
    return shortcuts.filter((s) => s.label.toLowerCase().includes(search.query.toLowerCase()));
  }, [search.query]);

  return (
    <SearchableMenuContent inputProps={search.inputProps} placeholder='e.g. "2 weeks", "3 months"'>
      {search.query ? (
        parsedDuration ? (
          <>
            <Menu.Item
              value={`parsed-${parsedDuration.freq}-${parsedDuration.interval}`}
              className={ITEM_CLASS}
              closeOnSelect
              onClick={() => selectFreq(parsedDuration.freq, parsedDuration.interval)}
            >
              <Menu.ItemText>{parsedDuration.label}</Menu.ItemText>
            </Menu.Item>
            {/* Also show filtered shortcuts */}
            {filteredShortcuts.map((shortcut) => (
              <Menu.Item
                key={shortcut.label}
                value={shortcut.label}
                className={ITEM_CLASS}
                closeOnSelect
                onClick={() => selectFreq(shortcut.freq, shortcut.interval)}
              >
                <Menu.ItemText>{shortcut.label}</Menu.ItemText>
              </Menu.Item>
            ))}
          </>
        ) : filteredShortcuts.length > 0 ? (
          filteredShortcuts.map((shortcut) => (
            <Menu.Item
              key={shortcut.label}
              value={shortcut.label}
              className={ITEM_CLASS}
              closeOnSelect
              onClick={() => selectFreq(shortcut.freq, shortcut.interval)}
            >
              <Menu.ItemText>{shortcut.label}</Menu.ItemText>
            </Menu.Item>
          ))
        ) : (
          <div className="text-surface-500 px-3 py-2 text-sm">No matching options</div>
        )
      ) : (
        <>
          {filteredShortcuts.map((shortcut) => {
            const isCurrent =
              isCurrentType &&
              currentRule.freq === shortcut.freq &&
              currentRule.interval === shortcut.interval;
            const hasSubmenu =
              shortcut.freq === "WEEKLY" ||
              shortcut.freq === "MONTHLY" ||
              shortcut.freq === "YEARLY";

            if (hasSubmenu) {
              return (
                <Menu key={shortcut.label}>
                  <Menu.TriggerItem
                    value={shortcut.label}
                    className={`justify-start gap-2 text-sm`}
                  >
                    {isCurrent ? (
                      <Check className="size-4" aria-hidden="true" />
                    ) : (
                      <span className="size-4" />
                    )}
                    <Menu.ItemText>{shortcut.label}</Menu.ItemText>
                    <Menu.ItemIndicator className="ml-auto">
                      <ChevronRight className="size-4" />
                    </Menu.ItemIndicator>
                  </Menu.TriggerItem>
                  <Portal>
                    <Menu.Positioner>
                      <FreqDetailSubMenu
                        freq={shortcut.freq}
                        interval={shortcut.interval}
                        currentRule={currentRule}
                        recurrenceType={recurrenceType}
                        onSave={onSave}
                      />
                    </Menu.Positioner>
                  </Portal>
                </Menu>
              );
            }

            return (
              <Menu.Item
                key={shortcut.label}
                value={shortcut.label}
                className={ITEM_CLASS}
                closeOnSelect
                onClick={() => selectFreq(shortcut.freq, shortcut.interval)}
              >
                {isCurrent ? (
                  <Check className="size-4" aria-hidden="true" />
                ) : (
                  <span className="size-4" />
                )}
                <Menu.ItemText>{shortcut.label}</Menu.ItemText>
              </Menu.Item>
            );
          })}
        </>
      )}
    </SearchableMenuContent>
  );
}

// ─── Frequency Detail Submenu (weekly days, monthly day, yearly months) ──────

function FreqDetailSubMenu({
  freq,
  interval,
  currentRule,
  recurrenceType,
  onSave,
}: {
  freq: FreqCode;
  interval: number;
  currentRule: ParsedRule;
  recurrenceType: TaskRecurrenceType;
  onSave: (update: { recurrenceType: TaskRecurrenceType; recurrenceRule?: string | null }) => void;
}) {
  const search = useMenuSearch();

  if (freq === "WEEKLY") {
    const isCurrentFreq = currentRule.freq === "WEEKLY" && currentRule.interval === interval;
    const currentDays = isCurrentFreq ? currentRule.byweekday : [];

    const filteredDays = useMemo(() => {
      if (!search.query) return [...WEEKDAY_CODES];
      return WEEKDAY_CODES.filter((d) =>
        WEEKDAY_LABELS[d].toLowerCase().includes(search.query.toLowerCase()),
      );
    }, [search.query]);

    function toggleDay(day: WeekdayCode) {
      const newDays = currentDays.includes(day)
        ? currentDays.filter((d) => d !== day)
        : [...currentDays, day];
      const newRule: ParsedRule = {
        ...currentRule,
        freq: "WEEKLY",
        interval,
        byweekday: newDays,
        bymonthday: null,
        nthWeekday: null,
        bymonth: [],
      };
      onSave({ recurrenceType, recurrenceRule: buildRRule(newRule) });
    }

    return (
      <SearchableMenuContent inputProps={search.inputProps} placeholder="Search days...">
        {filteredDays.length === 0 ? (
          <div className="text-surface-500 px-3 py-2 text-sm">No matching days</div>
        ) : (
          filteredDays.map((day) => (
            <Menu.OptionItem
              key={day}
              type="checkbox"
              checked={currentDays.includes(day)}
              value={day}
              onCheckedChange={() => toggleDay(day)}
              className={ITEM_CLASS}
            >
              <Menu.ItemIndicator className="invisible data-[state=checked]:visible">
                <Check className="size-4" />
              </Menu.ItemIndicator>
              <Menu.ItemText>{WEEKDAY_LABELS[day]}</Menu.ItemText>
            </Menu.OptionItem>
          ))
        )}
      </SearchableMenuContent>
    );
  }

  if (freq === "MONTHLY") {
    const items = useMemo(() => {
      const dayItems = DAY_NUMBERS.map((d) => ({
        value: `day-${d}`,
        label: `On day ${d}`,
        apply: () => {
          const newRule: ParsedRule = {
            ...currentRule,
            freq: "MONTHLY",
            interval,
            bymonthday: d,
            nthWeekday: null,
            byweekday: [],
            bymonth: [],
          };
          onSave({ recurrenceType, recurrenceRule: buildRRule(newRule) });
        },
      }));

      const nthItems: typeof dayItems = [];
      for (const ordinal of [1, 2, 3, 4, -1]) {
        for (const day of WEEKDAY_CODES) {
          const ordLabel = ORDINAL_LABELS[ordinal] ?? String(ordinal);
          nthItems.push({
            value: `nth-${ordinal}-${day}`,
            label: `${ordLabel} ${WEEKDAY_LABELS[day]}`,
            apply: () => {
              const newRule: ParsedRule = {
                ...currentRule,
                freq: "MONTHLY",
                interval,
                bymonthday: null,
                nthWeekday: { ordinal, weekday: day },
                byweekday: [],
                bymonth: [],
              };
              onSave({ recurrenceType, recurrenceRule: buildRRule(newRule) });
            },
          });
        }
      }

      return [...dayItems, ...nthItems];
    }, [currentRule, interval, recurrenceType, onSave]);

    const filtered = useMemo(() => {
      if (!search.query) return items.slice(0, 15); // Show first 15 by default
      return items.filter((item) => item.label.toLowerCase().includes(search.query.toLowerCase()));
    }, [items, search.query]);

    return (
      <SearchableMenuContent
        inputProps={search.inputProps}
        placeholder='e.g. "day 15", "1st Monday"'
      >
        {filtered.length === 0 ? (
          <div className="text-surface-500 px-3 py-2 text-sm">No matching options</div>
        ) : (
          filtered.map((item) => (
            <Menu.Item
              key={item.value}
              value={item.value}
              className={ITEM_CLASS}
              closeOnSelect
              onClick={item.apply}
            >
              <Menu.ItemText>{item.label}</Menu.ItemText>
            </Menu.Item>
          ))
        )}
      </SearchableMenuContent>
    );
  }

  if (freq === "YEARLY") {
    const isCurrentFreq = currentRule.freq === "YEARLY" && currentRule.interval === interval;
    const currentMonths = isCurrentFreq ? currentRule.bymonth : [];

    const filtered = useMemo(() => {
      if (!search.query) return MONTH_LABELS;
      return MONTH_LABELS.filter((m) => m.toLowerCase().includes(search.query.toLowerCase()));
    }, [search.query]);

    function toggleMonth(monthIndex: number) {
      const month = monthIndex + 1;
      const newMonths = currentMonths.includes(month)
        ? currentMonths.filter((m) => m !== month)
        : [...currentMonths, month].sort((a, b) => a - b);
      const newRule: ParsedRule = {
        ...currentRule,
        freq: "YEARLY",
        interval,
        bymonth: newMonths,
        byweekday: [],
      };
      onSave({ recurrenceType, recurrenceRule: buildRRule(newRule) });
    }

    return (
      <SearchableMenuContent inputProps={search.inputProps} placeholder="Search months...">
        {filtered.length === 0 ? (
          <div className="text-surface-500 px-3 py-2 text-sm">No matching months</div>
        ) : (
          filtered.map((label) => {
            const index = MONTH_LABELS.indexOf(label);
            const month = index + 1;
            return (
              <Menu.OptionItem
                key={label}
                type="checkbox"
                checked={currentMonths.includes(month)}
                value={label}
                onCheckedChange={() => toggleMonth(index)}
                className={ITEM_CLASS}
              >
                <Menu.ItemIndicator className="invisible data-[state=checked]:visible">
                  <Check className="size-4" />
                </Menu.ItemIndicator>
                <Menu.ItemText>{label}</Menu.ItemText>
              </Menu.OptionItem>
            );
          })
        )}
      </SearchableMenuContent>
    );
  }

  return <Menu.Content />;
}

// ─── Until Date Submenu ─────────────────────────────────────────────────────

function UntilDateSubMenu({
  currentUntil,
  currentRule,
  recurrenceType,
  onSave,
}: {
  currentUntil: string | null;
  currentRule: ParsedRule;
  recurrenceType: TaskRecurrenceType;
  onSave: (update: { recurrenceType: TaskRecurrenceType; recurrenceRule?: string | null }) => void;
}) {
  const search = useMenuSearch();
  const parsedDate = useMemo(() => parseDateInput(search.query), [search.query]);
  const today = useMemo(() => new Date(), []);

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
    >
      {currentUntil && (
        <Menu.Item
          value="clear-until"
          className={`text-error-500 ${ITEM_CLASS}`}
          closeOnSelect
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
            className={ITEM_CLASS}
            closeOnSelect
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
          const date = new Date(today);
          date.setDate(date.getDate() + shortcut.offsetDays);
          const isoDate = toISODate(date);
          return (
            <Menu.Item
              key={shortcut.label}
              value={isoDate}
              className={ITEM_CLASS}
              closeOnSelect
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

  const hasRule = TYPES_WITH_RULE.has(recurrenceType);

  function handleSave(update: {
    recurrenceType: TaskRecurrenceType;
    recurrenceRule?: string | null;
  }) {
    mutation.reset();
    mutation.mutate(update);
  }

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
                    <Menu key={item.value} typeahead={false}>
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
                      className={ITEM_CLASS}
                      closeOnSelect
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

              {hasRule && (
                <>
                  <Menu.Separator />
                  <Menu typeahead={false}>
                    <Menu.TriggerItem value="until" className="justify-start gap-2 text-sm">
                      <Calendar className="size-4" aria-hidden="true" />
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
                          currentUntil={currentRule.until}
                          currentRule={currentRule}
                          recurrenceType={recurrenceType}
                          onSave={handleSave}
                        />
                      </Menu.Positioner>
                    </Portal>
                  </Menu>
                </>
              )}
            </SearchableMenuContent>
          </Menu.Positioner>
        </Portal>
      </Menu>
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </>
  );
}

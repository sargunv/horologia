import { parseDate } from "@skeletonlabs/skeleton-react";
import { AlertTriangle, X } from "lucide-react";
import { useEffect, useId, useMemo, useRef, useState } from "react";
import { DateField } from "./DateField.tsx";
// TODO: Switch to rrule-temporal (https://www.npmjs.com/package/rrule-temporal)
// for proper Temporal API support and better timezone handling.
import { RRule, Weekday } from "rrule";
import type { components } from "../api/schema.d.ts";

type TaskRecurrenceType = components["schemas"]["TaskRecurrenceType"];

// ─── Types ───────────────────────────────────────────────────────────────────

type WeekdayCode = "MO" | "TU" | "WE" | "TH" | "FR" | "SA" | "SU";
type Ordinal = 1 | 2 | 3 | 4 | -1;

type MonthlyMode =
  | { type: "bymonthday"; day: number }
  | { type: "nthweekday"; ordinal: Ordinal; weekday: WeekdayCode };

type FreqCode = "DAILY" | "WEEKLY" | "MONTHLY" | "YEARLY";

interface RuleFormState {
  freq: FreqCode;
  interval: number;
  byweekday: WeekdayCode[];
  monthlyMode: MonthlyMode;
  bymonth: number[];
  until: string;
}

// ─── Constants ───────────────────────────────────────────────────────────────

const PARSE_WARNING = "This rule can't be fully edited here. Saving will replace it.";

const WEEKDAY_CODES: WeekdayCode[] = ["SU", "MO", "TU", "WE", "TH", "FR", "SA"];

const WEEKDAY_LABELS: Record<WeekdayCode, string> = {
  SU: "S",
  MO: "M",
  TU: "T",
  WE: "W",
  TH: "T",
  FR: "F",
  SA: "S",
};

const WEEKDAY_FULL_LABELS: Record<WeekdayCode, string> = {
  SU: "Sunday",
  MO: "Monday",
  TU: "Tuesday",
  WE: "Wednesday",
  TH: "Thursday",
  FR: "Friday",
  SA: "Saturday",
};

const FREQ_OPTIONS: { value: FreqCode; unit: string }[] = [
  { value: "DAILY", unit: "day" },
  { value: "WEEKLY", unit: "week" },
  { value: "MONTHLY", unit: "month" },
  { value: "YEARLY", unit: "year" },
];

const ORDINAL_OPTIONS: { value: Ordinal; label: string }[] = [
  { value: 1, label: "1st" },
  { value: 2, label: "2nd" },
  { value: 3, label: "3rd" },
  { value: 4, label: "4th" },
  { value: -1, label: "Last" },
];

const MONTH_LABELS = [
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

const RRULE_WEEKDAY: Record<WeekdayCode, Weekday> = {
  MO: RRule.MO,
  TU: RRule.TU,
  WE: RRule.WE,
  TH: RRule.TH,
  FR: RRule.FR,
  SA: RRule.SA,
  SU: RRule.SU,
};

// rrule indexes weekdays 0=MO through 6=SU
const WEEKDAY_INDEX_TO_CODE: WeekdayCode[] = ["MO", "TU", "WE", "TH", "FR", "SA", "SU"];

const FREQ_TO_RRULE: Record<FreqCode, number> = {
  DAILY: RRule.DAILY,
  WEEKLY: RRule.WEEKLY,
  MONTHLY: RRule.MONTHLY,
  YEARLY: RRule.YEARLY,
};

const RRULE_FREQ_TO_CODE: Partial<Record<number, FreqCode>> = {
  [RRule.DAILY]: "DAILY",
  [RRule.WEEKLY]: "WEEKLY",
  [RRule.MONTHLY]: "MONTHLY",
  [RRule.YEARLY]: "YEARLY",
};

const RECURRENCE_TYPE_LABELS: Record<TaskRecurrenceType, string> = {
  one_off: "One-off",
  completion_based: "Completion-based",
  fixed_non_accumulating: "Fixed (non-accumulating)",
  fixed_accumulating: "Fixed (accumulating)",
  on_dependency: "On dependency",
};

const TYPES_WITH_RULE = new Set<TaskRecurrenceType>([
  "completion_based",
  "fixed_non_accumulating",
  "fixed_accumulating",
]);

// ─── Type Guards & Helpers ───────────────────────────────────────────────────

const FREQ_CODE_SET = new Set<string>(FREQ_OPTIONS.map((f) => f.value));
function isFreqCode(value: string): value is FreqCode {
  return FREQ_CODE_SET.has(value);
}

const ORDINAL_VALUE_SET = new Set<number>(ORDINAL_OPTIONS.map((o) => o.value));
function isOrdinal(value: number): value is Ordinal {
  return ORDINAL_VALUE_SET.has(value);
}

const WEEKDAY_CODE_SET = new Set<string>(WEEKDAY_CODES);
function isWeekdayCode(value: string): value is WeekdayCode {
  return WEEKDAY_CODE_SET.has(value);
}

function isRecurrenceType(value: string): value is TaskRecurrenceType {
  return value in RECURRENCE_TYPE_LABELS;
}

function toOrdinal(n: number): Ordinal {
  return isOrdinal(n) ? n : 1;
}

function toMonthDay(n: number): number {
  return Math.max(1, Math.min(31, Math.round(n) || 1));
}

function toArray<T>(value: T | T[]): T[] {
  return Array.isArray(value) ? value : [value];
}

// ─── Default State ───────────────────────────────────────────────────────────

const DEFAULT_RULE_STATE: RuleFormState = {
  freq: "WEEKLY",
  interval: 1,
  byweekday: [],
  monthlyMode: { type: "bymonthday", day: 1 },
  bymonth: [],
  until: "",
};

// ─── Parsing ─────────────────────────────────────────────────────────────────

interface ParseResult {
  state: RuleFormState;
  warning: string | null;
}

function parseRuleToFormState(rruleStr: string | null): ParseResult {
  if (!rruleStr) {
    return { state: DEFAULT_RULE_STATE, warning: null };
  }

  try {
    const normalized = rruleStr.startsWith("RRULE:") ? rruleStr : `RRULE:${rruleStr}`;
    const rule = RRule.fromString(normalized);
    const opts = rule.origOptions;

    let warning: string | null = null;

    if (
      opts.count != null ||
      opts.bysetpos != null ||
      opts.byyearday != null ||
      opts.byweekno != null ||
      opts.byhour != null ||
      opts.byminute != null ||
      opts.bysecond != null
    ) {
      warning = PARSE_WARNING;
    }

    const freq = RRULE_FREQ_TO_CODE[opts.freq ?? RRule.WEEKLY] ?? "WEEKLY";
    const interval = opts.interval ?? 1;

    const byweekday: WeekdayCode[] = [];
    let monthlyMode: MonthlyMode = { type: "bymonthday", day: 1 };
    let monthlyModeSetByWeekday = false;

    if (opts.byweekday) {
      const weekdays = toArray(opts.byweekday);
      const nthEntries = weekdays.filter(
        (w): w is Weekday => w instanceof Weekday && w.n !== undefined && w.n !== 0,
      );
      const plainEntries = weekdays.filter(
        (w): w is Weekday => w instanceof Weekday && (w.n === undefined || w.n === 0),
      );

      const first = nthEntries[0];
      if (first && (freq === "MONTHLY" || freq === "YEARLY")) {
        const weekday = WEEKDAY_INDEX_TO_CODE[first.weekday];
        if (weekday) {
          monthlyMode = { type: "nthweekday", ordinal: toOrdinal(first.n ?? 1), weekday };
          monthlyModeSetByWeekday = true;
        }
        if (nthEntries.length > 1) warning = PARSE_WARNING;
      } else {
        for (const w of plainEntries) {
          const code = WEEKDAY_INDEX_TO_CODE[w.weekday];
          if (code) byweekday.push(code);
        }
        // Fallback: single plain weekday on MONTHLY/YEARLY → assume "1st [weekday]"
        const firstPlain = plainEntries[0];
        if (firstPlain && plainEntries.length === 1 && (freq === "MONTHLY" || freq === "YEARLY")) {
          const code = WEEKDAY_INDEX_TO_CODE[firstPlain.weekday];
          if (code) {
            monthlyMode = { type: "nthweekday", ordinal: 1, weekday: code };
            monthlyModeSetByWeekday = true;
          }
        }
      }
    }

    if (opts.bymonthday) {
      const days = toArray(opts.bymonthday);
      const firstDay = days[0];
      if (firstDay != null) {
        if (monthlyModeSetByWeekday) warning = PARSE_WARNING;
        monthlyMode = { type: "bymonthday", day: toMonthDay(firstDay) };
        if (days.length > 1) warning = PARSE_WARNING;
      }
    }

    const bymonth: number[] = [];
    if (opts.bymonth) bymonth.push(...toArray(opts.bymonth));

    let until = "";
    if (opts.until) {
      const d = opts.until;
      until = `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, "0")}-${String(d.getUTCDate()).padStart(2, "0")}`;
    }

    return { state: { freq, interval, byweekday, monthlyMode, bymonth, until }, warning };
  } catch {
    return { state: DEFAULT_RULE_STATE, warning: PARSE_WARNING };
  }
}

// ─── Serialization ───────────────────────────────────────────────────────────

function formStateToRruleString(state: RuleFormState): string {
  const options: ConstructorParameters<typeof RRule>[0] = {
    freq: FREQ_TO_RRULE[state.freq],
  };

  // Only emit INTERVAL when it differs from the default
  if (state.interval !== 1) {
    options.interval = state.interval;
  }

  if (state.freq === "WEEKLY" && state.byweekday.length > 0) {
    options.byweekday = state.byweekday.map((d) => RRULE_WEEKDAY[d]);
  }

  if (state.freq === "MONTHLY" || state.freq === "YEARLY") {
    if (state.monthlyMode.type === "bymonthday") {
      options.bymonthday = [state.monthlyMode.day];
    } else {
      options.byweekday = [RRULE_WEEKDAY[state.monthlyMode.weekday].nth(state.monthlyMode.ordinal)];
    }
  }

  if (state.freq === "YEARLY" && state.bymonth.length > 0) {
    options.bymonth = state.bymonth;
  }

  if (state.until) {
    options.until = new Date(state.until + "T00:00:00Z");
  }

  return new RRule(options).toString();
}

/** Normalize an RRULE string by round-tripping through parse+serialize. */
function normalizeRrule(rruleStr: string | null): string {
  if (!rruleStr) return "";
  return formStateToRruleString(parseRuleToFormState(rruleStr).state);
}

// ─── Component ───────────────────────────────────────────────────────────────

export function RecurrenceRuleEditor({
  recurrenceType,
  recurrenceRule,
  onSave,
  disabled,
}: {
  recurrenceType: TaskRecurrenceType;
  recurrenceRule: string | null;
  onSave: (update: { recurrenceType: TaskRecurrenceType; recurrenceRule?: string | null }) => void;
  disabled?: boolean | undefined;
}) {
  const [draftType, setDraftType] = useState(recurrenceType);
  const [{ state: initialRule, warning: initialWarning }] = useState(() =>
    parseRuleToFormState(recurrenceRule),
  );
  const [ruleState, setRuleState] = useState(initialRule);
  const [warning, setWarning] = useState(initialWarning);
  const [parseFailed, setParseFailed] = useState(initialWarning != null && !recurrenceRule);
  const [editing, setEditing] = useState(false);
  const cancellingRef = useRef(false);
  const datePickerOpenRef = useRef(false);

  // Normalize the saved value once for stable dirty comparison
  const [savedNormalized, setSavedNormalized] = useState(() => ({
    type: recurrenceType,
    rule: normalizeRrule(recurrenceRule),
  }));

  // Sync from props when not editing
  useEffect(() => {
    if (!editing) {
      setDraftType(recurrenceType);
      const parsed = parseRuleToFormState(recurrenceRule);
      setRuleState(parsed.state);
      setWarning(parsed.warning);
      setParseFailed(parsed.warning != null && !recurrenceRule);
      setSavedNormalized({ type: recurrenceType, rule: normalizeRrule(recurrenceRule) });
    }
  }, [recurrenceType, recurrenceRule, editing]);

  const showRuleEditor = TYPES_WITH_RULE.has(draftType);

  const currentNormalized = useMemo(
    () => ({
      type: draftType,
      rule: showRuleEditor ? formStateToRruleString(ruleState) : "",
    }),
    [draftType, ruleState, showRuleEditor],
  );

  const isDirty =
    currentNormalized.type !== savedNormalized.type ||
    currentNormalized.rule !== savedNormalized.rule;

  function save() {
    setEditing(false);
    if (!isDirty || parseFailed) return;

    const update: { recurrenceType: TaskRecurrenceType; recurrenceRule?: string | null } = {
      recurrenceType: draftType,
    };

    if (TYPES_WITH_RULE.has(draftType)) {
      update.recurrenceRule = formStateToRruleString(ruleState);
    } else if (TYPES_WITH_RULE.has(savedNormalized.type)) {
      // Switching away from a rule-bearing type: clear the rule
      update.recurrenceRule = null;
    }

    setSavedNormalized(currentNormalized);
    setWarning(null);
    onSave(update);
  }

  function cancel() {
    setDraftType(savedNormalized.type);
    const parsed = parseRuleToFormState(
      TYPES_WITH_RULE.has(savedNormalized.type) ? savedNormalized.rule || null : null,
    );
    setRuleState(parsed.state);
    setWarning(parsed.warning);
    setEditing(false);
  }

  function updateRule(patch: Partial<RuleFormState>) {
    setRuleState((prev) => ({ ...prev, ...patch }));
  }

  const untilDateValue = useMemo(() => {
    if (!ruleState.until) return null;
    try {
      return parseDate(ruleState.until);
    } catch {
      return null;
    }
  }, [ruleState.until]);

  return (
    <div
      className="flex flex-col gap-3"
      onFocus={() => setEditing(true)}
      onBlur={(e) => {
        if (cancellingRef.current) {
          cancellingRef.current = false;
          return;
        }
        if (!(e.relatedTarget instanceof Node) || !e.currentTarget.contains(e.relatedTarget)) {
          if (!datePickerOpenRef.current) save();
        }
      }}
    >
      {/* Recurrence type selector */}
      <select
        value={draftType}
        aria-label="Recurrence type"
        onChange={(e) => {
          if (isRecurrenceType(e.target.value)) setDraftType(e.target.value);
        }}
        disabled={disabled}
        className="select preset-outlined-surface-200-800 w-full"
      >
        {Object.entries(RECURRENCE_TYPE_LABELS).map(([value, label]) => (
          <option key={value} value={value}>
            {label}
          </option>
        ))}
      </select>

      {/* Rule editor (only for rule-bearing types) */}
      {showRuleEditor && (
        <>
          {/* Warning banner */}
          {warning && (
            <div className="flex items-center gap-2 rounded-base bg-warning-500/10 px-3 py-2 text-sm text-warning-600 dark:text-warning-400">
              <AlertTriangle className="size-4 shrink-0" aria-hidden="true" />
              <span className="flex-1">{warning}</span>
              <button
                type="button"
                onClick={() => setWarning(null)}
                className="shrink-0 rounded p-0.5 hover:bg-warning-500/10"
                aria-label="Dismiss warning"
              >
                <X className="size-3.5" />
              </button>
            </div>
          )}

          {/* Frequency + interval */}
          <div className="flex items-center gap-2">
            <span className="text-surface-600-400 text-sm">Every</span>
            <input
              type="number"
              min={1}
              value={ruleState.interval}
              onChange={(e) => updateRule({ interval: Math.max(1, parseInt(e.target.value) || 1) })}
              className="input preset-outlined-surface-200-800 w-16 text-center"
              disabled={disabled}
              aria-label="Repeat interval"
            />
            <select
              value={ruleState.freq}
              onChange={(e) => {
                if (isFreqCode(e.target.value)) updateRule({ freq: e.target.value });
              }}
              className="select preset-outlined-surface-200-800"
              disabled={disabled}
              aria-label="Frequency"
            >
              {FREQ_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {ruleState.interval === 1 ? opt.unit : opt.unit + "s"}
                </option>
              ))}
            </select>
          </div>

          {/* Weekly: day-of-week toggles */}
          {ruleState.freq === "WEEKLY" && (
            <WeekdayToggleRow
              selected={ruleState.byweekday}
              onToggle={(day) => {
                const next = ruleState.byweekday.includes(day)
                  ? ruleState.byweekday.filter((d) => d !== day)
                  : [...ruleState.byweekday, day];
                updateRule({ byweekday: next });
              }}
              disabled={disabled}
            />
          )}

          {/* Monthly: mode selector + detail */}
          {ruleState.freq === "MONTHLY" && (
            <MonthlyPanel
              mode={ruleState.monthlyMode}
              onModeChange={(monthlyMode) => updateRule({ monthlyMode })}
              disabled={disabled}
            />
          )}

          {/* Yearly: month picker + monthly sub-mode */}
          {ruleState.freq === "YEARLY" && (
            <YearlyPanel
              bymonth={ruleState.bymonth}
              monthlyMode={ruleState.monthlyMode}
              onBymonthChange={(bymonth) => updateRule({ bymonth })}
              onModeChange={(monthlyMode) => updateRule({ monthlyMode })}
              disabled={disabled}
            />
          )}

          {/* Until row */}
          <div className="flex items-center gap-2">
            <span className="text-surface-600-400 text-sm">Until</span>
            <DateField
              value={untilDateValue}
              onChange={(v) => updateRule({ until: v?.toString() ?? "" })}
              onOpenChange={(open) => {
                datePickerOpenRef.current = open;
              }}
              disabled={disabled}
              aria-label="End date"
            />
            {ruleState.until && (
              <button
                type="button"
                onClick={() => updateRule({ until: "" })}
                disabled={disabled}
                className="btn btn-sm preset-outlined-surface-200-800"
                aria-label="Clear end date"
                title="No end date"
              >
                <X className="size-3.5" aria-hidden="true" />
              </button>
            )}
            {!ruleState.until && <span className="text-surface-500 text-sm">No end date</span>}
          </div>
        </>
      )}

      {/* Action bar */}
      {isDirty && (
        <div className="flex justify-end gap-2 border-t border-surface-200-800 pt-2">
          <button
            type="button"
            onMouseDown={() => {
              cancellingRef.current = true;
            }}
            onClick={() => {
              cancellingRef.current = false;
              cancel();
            }}
            disabled={disabled}
            className="btn btn-sm preset-outlined-surface-200-800"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={save}
            disabled={disabled}
            className="btn btn-sm preset-filled-primary-500"
          >
            Save
          </button>
        </div>
      )}
    </div>
  );
}

// ─── Sub-components ──────────────────────────────────────────────────────────

function WeekdayToggleRow({
  selected,
  onToggle,
  disabled,
}: {
  selected: WeekdayCode[];
  onToggle: (day: WeekdayCode) => void;
  disabled?: boolean | undefined;
}) {
  return (
    <div className="flex items-center gap-1">
      <span className="text-surface-600-400 mr-1 text-sm">On</span>
      {WEEKDAY_CODES.map((day) => {
        const active = selected.includes(day);
        return (
          <button
            key={day}
            type="button"
            onClick={() => onToggle(day)}
            disabled={disabled}
            aria-label={WEEKDAY_FULL_LABELS[day]}
            aria-pressed={active}
            className={`flex size-8 items-center justify-center rounded-full text-xs font-medium transition-colors ${
              active
                ? "preset-filled-primary-500"
                : "preset-outlined-surface-200-800 hover:preset-tonal-surface"
            }`}
          >
            {WEEKDAY_LABELS[day]}
          </button>
        );
      })}
    </div>
  );
}

function MonthlyPanel({
  mode,
  onModeChange,
  disabled,
}: {
  mode: MonthlyMode;
  onModeChange: (mode: MonthlyMode) => void;
  disabled?: boolean | undefined;
}) {
  const radioName = useId();

  return (
    <div className="flex flex-col gap-2">
      {/* By month day */}
      <label className="flex items-center gap-2">
        <input
          type="radio"
          name={radioName}
          checked={mode.type === "bymonthday"}
          onChange={() =>
            onModeChange({ type: "bymonthday", day: mode.type === "bymonthday" ? mode.day : 1 })
          }
          disabled={disabled}
          className="accent-primary-500"
        />
        <span className="text-surface-600-400 text-sm">On day</span>
        <input
          type="number"
          min={1}
          max={31}
          value={mode.type === "bymonthday" ? mode.day : 1}
          onChange={(e) => {
            onModeChange({ type: "bymonthday", day: toMonthDay(parseInt(e.target.value) || 1) });
          }}
          onBlur={(e) => {
            if (mode.type !== "bymonthday") return;
            onModeChange({ type: "bymonthday", day: toMonthDay(parseInt(e.target.value) || 1) });
          }}
          className="input preset-outlined-surface-200-800 w-16 text-center"
          disabled={disabled || mode.type !== "bymonthday"}
          aria-label="Day of month"
        />
      </label>

      {/* Nth weekday */}
      <label className="flex items-center gap-2">
        <input
          type="radio"
          name={radioName}
          checked={mode.type === "nthweekday"}
          onChange={() =>
            onModeChange({
              type: "nthweekday",
              ordinal: mode.type === "nthweekday" ? mode.ordinal : 1,
              weekday: mode.type === "nthweekday" ? mode.weekday : "MO",
            })
          }
          disabled={disabled}
          className="accent-primary-500"
        />
        <span className="text-surface-600-400 text-sm">On the</span>
        <select
          value={mode.type === "nthweekday" ? mode.ordinal : 1}
          onChange={(e) => {
            const n = parseInt(e.target.value);
            if (!isOrdinal(n)) return;
            onModeChange({
              type: "nthweekday",
              ordinal: n,
              weekday: mode.type === "nthweekday" ? mode.weekday : "MO",
            });
          }}
          className="select preset-outlined-surface-200-800"
          disabled={disabled || mode.type !== "nthweekday"}
          aria-label="Ordinal"
        >
          {ORDINAL_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
        <select
          value={mode.type === "nthweekday" ? mode.weekday : "MO"}
          onChange={(e) => {
            if (!isWeekdayCode(e.target.value)) return;
            onModeChange({
              type: "nthweekday",
              ordinal: mode.type === "nthweekday" ? mode.ordinal : 1,
              weekday: e.target.value,
            });
          }}
          className="select preset-outlined-surface-200-800"
          disabled={disabled || mode.type !== "nthweekday"}
          aria-label="Day of week"
        >
          {WEEKDAY_CODES.map((day) => (
            <option key={day} value={day}>
              {WEEKDAY_FULL_LABELS[day]}
            </option>
          ))}
        </select>
      </label>
    </div>
  );
}

function MonthToggleRow({
  selected,
  onToggle,
  disabled,
}: {
  selected: number[];
  onToggle: (month: number) => void;
  disabled?: boolean | undefined;
}) {
  return (
    <div className="flex flex-wrap items-center gap-1">
      <span className="text-surface-600-400 mr-1 text-sm">In</span>
      {MONTH_LABELS.map((label, i) => {
        const month = i + 1;
        const active = selected.includes(month);
        return (
          <button
            key={month}
            type="button"
            onClick={() => onToggle(month)}
            disabled={disabled}
            aria-label={label}
            aria-pressed={active}
            className={`rounded-base px-2 py-1 text-xs font-medium transition-colors ${
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
  );
}

function YearlyPanel({
  bymonth,
  monthlyMode,
  onBymonthChange,
  onModeChange,
  disabled,
}: {
  bymonth: number[];
  monthlyMode: MonthlyMode;
  onBymonthChange: (months: number[]) => void;
  onModeChange: (mode: MonthlyMode) => void;
  disabled?: boolean | undefined;
}) {
  return (
    <div className="flex flex-col gap-2">
      <MonthToggleRow
        selected={bymonth}
        onToggle={(month) => {
          const next = bymonth.includes(month)
            ? bymonth.filter((m) => m !== month)
            : [...bymonth, month].sort((a, b) => a - b);
          onBymonthChange(next);
        }}
        disabled={disabled}
      />
      <MonthlyPanel mode={monthlyMode} onModeChange={onModeChange} disabled={disabled} />
    </div>
  );
}

import { Calendar, Check, ChevronRight, RefreshCw, X } from "lucide-react";
import { type ReactNode, useMemo } from "react";
import {
  buildRRule,
  describeRecurrenceMonthDays as describeMonthDays,
  describeRule,
  parseRecurrenceDurationInput as parseDurationInput,
  parseRRule,
  type ParsedRecurrenceRule,
  type RecurrenceFrequency,
  RECURRENCE_MONTH_LABELS as MONTH_LABELS,
  RECURRENCE_ORDINAL_LABELS as ORDINAL_LABELS,
  type WeekdayCode,
} from "@horologia/client-core/domain/recurrence";
import {
  addDays,
  formatDateDisplay,
  parseDateInput,
  toISODate,
} from "@horologia/client-core/domain/dates";
import type { components } from "@horologia/client-core/schema";
import { useTaskPatch } from "../../lib/mutations.ts";
import { useMenuSearch } from "../../lib/useMenuSearch.ts";
import {
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuRoot,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
} from "../../ui/DropdownMenu.tsx";
import { FieldPill } from "../FieldPill.tsx";
import { SearchableMenuContent, SearchableSubMenuContent } from "../SearchableMenuContent.tsx";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";

type TaskRecurrenceType = components["schemas"]["TaskRecurrenceType"];

type FreqCode = RecurrenceFrequency;
type ParsedRule = ParsedRecurrenceRule;

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

const WEEKDAY_CODES = ["MO", "TU", "WE", "TH", "FR", "SA", "SU"] as const;

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

// Toggle chip button classes for freq-specific grid selectors
const CHIP_ON = "bg-primary text-primary-content";
const CHIP_OFF = "bg-base-200 text-base-content hover:bg-base-300";

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
  triggerContent,
}: {
  recurrenceType: TaskRecurrenceType;
  currentRule: ParsedRule;
  onSave: (update: { recurrenceType: TaskRecurrenceType; recurrenceRule?: string | null }) => void;
  triggerContent: ReactNode;
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
      nthWeekday: freq === "MONTHLY" || freq === "YEARLY" ? currentRule.nthWeekday : [],
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
      nthWeekday: [],
      byweekday: [],
    };
    onSave({ recurrenceType, recurrenceRule: buildRRule(newRule) });
  }

  function toggleNthWeekday(entry: { ordinal: number; weekday: WeekdayCode }) {
    const exists = currentRule.nthWeekday.some(
      (nw) => nw.ordinal === entry.ordinal && nw.weekday === entry.weekday,
    );
    const newNth = exists
      ? currentRule.nthWeekday.filter(
          (nw) => nw.ordinal !== entry.ordinal || nw.weekday !== entry.weekday,
        )
      : [...currentRule.nthWeekday, entry];
    const newRule: ParsedRule = {
      ...currentRule,
      bymonthday: [],
      nthWeekday: newNth,
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
    <DropdownMenuSub {...search.menuProps}>
      <DropdownMenuSubTrigger>{triggerContent}</DropdownMenuSubTrigger>
      <SearchableSubMenuContent
        search={search}
        placeholder='e.g. "2 weeks", "3 months"'
        inputLabel="Frequency"
      >
        {isSearching && parsedDuration && (
          <DropdownMenuItem
            onSelect={(e) => {
              e.preventDefault();
              selectFreq(parsedDuration.freq, parsedDuration.interval);
            }}
          >
            <span>{parsedDuration.label}</span>
          </DropdownMenuItem>
        )}

        <DropdownMenuRadioGroup
          value={currentRule.freq}
          onValueChange={(v) => {
            const shortcut = FREQ_SHORTCUTS.find((s) => s.freq === v);
            if (!shortcut) return;
            selectFreq(
              shortcut.freq,
              currentRule.freq === shortcut.freq ? currentRule.interval : 1,
            );
          }}
        >
          {filteredShortcuts.map((shortcut) => (
            <DropdownMenuRadioItem key={shortcut.label} value={shortcut.freq}>
              <span>{shortcut.label}</span>
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>

        {isSearching && !parsedDuration && filteredShortcuts.length === 0 && (
          <div role="status" className="px-3 py-2 text-sm text-base-content/60">
            No matching options
          </div>
        )}

        {currentRule.freq === "WEEKLY" && (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>
                <span>
                  {currentRule.byweekday.length > 0
                    ? `On ${currentRule.byweekday.map((d) => WEEKDAY_SHORT_LABELS[d]).join(", ")}`
                    : "On days..."}
                </span>
                <ChevronRight className="ml-auto size-4" aria-hidden="true" />
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent className="p-2">
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
                        className={`flex size-7 items-center justify-center rounded-full text-xs font-medium transition-colors ${active ? CHIP_ON : CHIP_OFF}`}
                      >
                        {WEEKDAY_SHORT_LABELS[day]}
                      </button>
                    );
                  })}
                </div>
              </DropdownMenuSubContent>
            </DropdownMenuSub>
          </>
        )}

        {currentRule.freq === "MONTHLY" && (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>
                <span>
                  {currentRule.nthWeekday.length > 0
                    ? `On the ${currentRule.nthWeekday.map((nw) => `${ORDINAL_LABELS[nw.ordinal] ?? ""} ${WEEKDAY_LABELS[nw.weekday]}`).join(", ")}`
                    : currentRule.bymonthday.length > 0
                      ? `On day ${describeMonthDays(currentRule.bymonthday)}`
                      : "On day..."}
                </span>
                <ChevronRight className="ml-auto size-4" aria-hidden="true" />
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent className="p-2">
                <div className="mb-1.5 text-xs text-base-content/60">Day of month</div>
                <div className="grid grid-cols-7 gap-1">
                  {DAY_NUMBERS.map((d) => {
                    const hasLast =
                      currentRule.bymonthday.includes(-1) && currentRule.nthWeekday.length === 0;
                    const active =
                      currentRule.bymonthday.includes(d) && currentRule.nthWeekday.length === 0;
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
                            ? CHIP_ON
                            : lastDayHint
                              ? "bg-primary/20 text-base-content hover:bg-base-200"
                              : isShortMonthDay
                                ? "border border-dashed border-base-300 text-base-content hover:bg-base-200"
                                : CHIP_OFF
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
                        currentRule.bymonthday.includes(-1) && currentRule.nthWeekday.length === 0
                      }
                      className={`flex h-7 items-center rounded px-2 text-xs font-medium transition-colors ${
                        currentRule.bymonthday.includes(-1) && currentRule.nthWeekday.length === 0
                          ? CHIP_ON
                          : CHIP_OFF
                      }`}
                    >
                      Last
                    </button>
                  </div>
                </div>
                <div className="mt-1 text-xs text-base-content/60">
                  29–31 skipped in short months
                </div>
                <DropdownMenuSeparator />
                <div className="mb-1.5 text-xs text-base-content/60">Or on the Nth weekday</div>
                <div className="mb-1 flex items-center justify-between">
                  {ORDINALS.map((ord) => {
                    const active = currentRule.nthWeekday.some((nw) => nw.ordinal === ord);
                    return (
                      <button
                        key={ord}
                        type="button"
                        onClick={() => {
                          const firstEntry = currentRule.nthWeekday[0];
                          const weekday: WeekdayCode = firstEntry ? firstEntry.weekday : "MO";
                          toggleNthWeekday({ ordinal: ord, weekday });
                        }}
                        aria-label={`${ORDINAL_LABELS[ord]} week of month`}
                        aria-pressed={active}
                        className={`rounded px-2 py-1 text-xs font-medium transition-colors ${active ? CHIP_ON : CHIP_OFF}`}
                      >
                        {ORDINAL_LABELS[ord]}
                      </button>
                    );
                  })}
                </div>
                <div className="flex items-center justify-between">
                  {WEEKDAY_CODES.map((day) => {
                    const active = currentRule.nthWeekday.some((nw) => nw.weekday === day);
                    return (
                      <button
                        key={day}
                        type="button"
                        onClick={() => {
                          const firstEntry = currentRule.nthWeekday[0];
                          const ordinal = firstEntry ? firstEntry.ordinal : 1;
                          toggleNthWeekday({ ordinal, weekday: day });
                        }}
                        aria-label={WEEKDAY_LABELS[day]}
                        aria-pressed={active}
                        className={`flex size-7 items-center justify-center rounded-full text-xs font-medium transition-colors ${active ? CHIP_ON : CHIP_OFF}`}
                      >
                        {WEEKDAY_SHORT_LABELS[day]}
                      </button>
                    );
                  })}
                </div>
              </DropdownMenuSubContent>
            </DropdownMenuSub>
          </>
        )}

        {currentRule.freq === "YEARLY" && (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>
                <span>
                  {currentRule.bymonth.length > 0
                    ? `In ${currentRule.bymonth.map((m) => MONTH_SHORT_LABELS[m - 1]).join(", ")}`
                    : "In months..."}
                </span>
                <ChevronRight className="ml-auto size-4" aria-hidden="true" />
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent className="p-2">
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
                        className={`rounded px-1.5 py-1 text-xs font-medium transition-colors ${active ? CHIP_ON : CHIP_OFF}`}
                      >
                        {label}
                      </button>
                    );
                  })}
                </div>
              </DropdownMenuSubContent>
            </DropdownMenuSub>
          </>
        )}

        <UntilDateSubMenu
          currentUntil={currentRule.until}
          currentRule={currentRule}
          recurrenceType={recurrenceType}
          onSave={onSave}
          triggerContent={
            <>
              <span>
                {currentRule.until ? `Until ${currentRule.until}` : "Until (no end date)"}
              </span>
              <ChevronRight className="ml-auto size-4" aria-hidden="true" />
            </>
          }
        />
      </SearchableSubMenuContent>
    </DropdownMenuSub>
  );
}

// ─── Until Date Submenu ─────────────────────────────────────────────────────

function UntilDateSubMenu({
  currentUntil,
  currentRule,
  recurrenceType,
  onSave,
  triggerContent,
}: {
  currentUntil: string | null;
  currentRule: ParsedRule;
  recurrenceType: TaskRecurrenceType;
  onSave: (update: { recurrenceType: TaskRecurrenceType; recurrenceRule?: string | null }) => void;
  triggerContent: ReactNode;
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
    <DropdownMenuSub {...search.menuProps}>
      <DropdownMenuSubTrigger>{triggerContent}</DropdownMenuSubTrigger>
      <SearchableSubMenuContent
        search={search}
        placeholder='e.g. "dec 2026", "in 6 months"'
        inputLabel="Until"
      >
        {currentUntil && (
          <DropdownMenuItem
            className="text-error"
            onSelect={(e) => {
              e.preventDefault();
              selectUntil(null);
            }}
          >
            <X className="size-4" aria-hidden="true" />
            <span>No end date</span>
          </DropdownMenuItem>
        )}

        {search.query ? (
          parsedDate ? (
            <DropdownMenuItem
              onSelect={(e) => {
                e.preventDefault();
                selectUntil(parsedDate.value);
              }}
            >
              <Calendar className="size-4" aria-hidden="true" />
              <span>{parsedDate.label}</span>
              <span className="ml-auto text-xs text-base-content/60">{parsedDate.value}</span>
            </DropdownMenuItem>
          ) : (
            <div role="status" className="px-3 py-2 text-sm text-base-content/60">
              No matching dates
            </div>
          )
        ) : (
          shortcuts.map((shortcut) => {
            const date = addDays(today, shortcut.offsetDays);
            const isoDate = toISODate(date);
            return (
              <DropdownMenuItem
                key={shortcut.label}
                onSelect={(e) => {
                  e.preventDefault();
                  selectUntil(isoDate);
                }}
              >
                <span>{shortcut.label}</span>
                <span className="ml-auto text-xs text-base-content/60">
                  {formatDateDisplay(date)}
                </span>
              </DropdownMenuItem>
            );
          })
        )}
      </SearchableSubMenuContent>
    </DropdownMenuSub>
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
      <DropdownMenuRoot {...search.menuProps}>
        <FieldPill
          icon={<RefreshCw className="size-3.5" aria-hidden="true" />}
          label="Recurrence"
          value={displayValue}
        />
        <SearchableMenuContent
          search={search}
          placeholder="Search recurrence..."
          inputLabel="Search recurrence"
        >
          {typeItems.length === 0 ? (
            <div role="status" className="px-3 py-2 text-sm text-base-content/60">
              No matching types
            </div>
          ) : (
            typeItems.map((item) =>
              item.hasSubmenu ? (
                <FreqSubMenu
                  key={item.value}
                  recurrenceType={item.value}
                  currentRule={currentRule}
                  onSave={handleSave}
                  triggerContent={
                    <>
                      {recurrenceType === item.value ? (
                        <Check className="size-4" aria-hidden="true" />
                      ) : (
                        <span className="size-4" />
                      )}
                      <span>{item.label}</span>
                      <ChevronRight className="ml-auto size-4" aria-hidden="true" />
                    </>
                  }
                />
              ) : (
                <DropdownMenuItem
                  key={item.value}
                  onSelect={() => {
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
                  <span>{item.label}</span>
                </DropdownMenuItem>
              ),
            )
          )}
        </SearchableMenuContent>
      </DropdownMenuRoot>
      {currentRule.parseError && (
        <div className="text-xs text-warning">Could not parse recurrence rule</div>
      )}
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </>
  );
}

import { Calendar, Check, ChevronRight, Clock, X } from "lucide-react";
import { type ReactNode, useMemo } from "react";
import type { components } from "../../api/schema.d.ts";
import { addDays, formatDateDisplay, parseDateInput, toISODate } from "../../lib/dates.ts";
import { useTaskPatch } from "../../lib/mutations.ts";
import { useMenuSearch } from "../../lib/useMenuSearch.ts";
import {
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuRoot,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubTrigger,
} from "../../ui/DropdownMenu.tsx";
import { FieldPill } from "../FieldPill.tsx";
import { SearchableMenuContent, SearchableSubMenuContent } from "../SearchableMenuContent.tsx";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";

type Task = components["schemas"]["Task"];
type TaskOverdueAction = components["schemas"]["TaskOverdueAction"];
type TaskOverdueActionRule = components["schemas"]["TaskOverdueActionRule"];
type TaskRecurrenceType = components["schemas"]["TaskRecurrenceType"];
type TaskStatus = components["schemas"]["TaskStatus"];

const BROWSER_TIMEZONE = Intl.DateTimeFormat().resolvedOptions().timeZone;

// ─── Date Shortcuts ─────────────────────────────────────────────────────────

const DATE_SHORTCUTS = [
  { label: "Today", offsetDays: 0 },
  { label: "Tomorrow", offsetDays: 1 },
  { label: "In 3 days", offsetDays: 3 },
  { label: "In 1 week", offsetDays: 7 },
  { label: "In 2 weeks", offsetDays: 14 },
  { label: "In 1 month", offsetDays: 30 },
];

// ─── Overdue Action Helpers ─────────────────────────────────────────────────

interface WhenOption {
  label: string;
  after: number | null;
}

const WHEN_SHORTCUTS: WhenOption[] = [
  { label: "Immediately", after: null },
  { label: "After 1 day", after: 1 },
  { label: "After 3 days", after: 3 },
  { label: "After 7 days", after: 7 },
];

function parseDaysInput(input: string): WhenOption | null {
  const match = input.match(/^\s*(\d+)\s*(d|days?)?\s*$/i);
  if (match) {
    const n = parseInt(match[1]!, 10);
    if (n > 0) {
      return { label: `After ${n} day${n === 1 ? "" : "s"}`, after: n };
    }
  }
  return null;
}

interface ActionItem {
  value: TaskOverdueAction;
  label: string;
}

const ALL_ACTIONS: ActionItem[] = [
  { value: "advance_recurrence", label: "Advance to next recurrence" },
  { value: "set_status", label: "Set status" },
  { value: "clear_due_date", label: "Clear due date" },
];

function describeOverdueAction(rule: TaskOverdueActionRule): string {
  const whenPrefix =
    rule.after == null
      ? "When overdue"
      : rule.after === 1
        ? "1 day overdue"
        : `${rule.after} days overdue`;

  switch (rule.action) {
    case "advance_recurrence":
      return `${whenPrefix}: advance`;
    case "set_status":
      return rule.status ? `${whenPrefix}: set ${rule.status}` : `${whenPrefix}: set status`;
    case "clear_due_date":
      return `${whenPrefix}: clear due date`;
  }
}

// ─── Status Picker Submenu ──────────────────────────────────────────────────

function StatusPickerSubMenu({
  statuses,
  currentStatus,
  onSelect,
  triggerContent,
}: {
  statuses: TaskStatus[];
  currentStatus: string | undefined;
  onSelect: (status: string) => void;
  triggerContent: ReactNode;
}) {
  const search = useMenuSearch();

  const filtered = useMemo(
    () => statuses.filter((s) => s.name.toLowerCase().includes(search.query.toLowerCase())),
    [statuses, search.query],
  );

  return (
    <DropdownMenuSub {...search.menuProps}>
      <DropdownMenuSubTrigger>{triggerContent}</DropdownMenuSubTrigger>
      <SearchableSubMenuContent
        search={search}
        placeholder="Search statuses..."
        inputLabel="Search statuses"
      >
        {filtered.length === 0 ? (
          <div role="status" className="px-3 py-2 text-sm text-base-content/60">
            No matching statuses
          </div>
        ) : (
          <DropdownMenuRadioGroup
            value={currentStatus ?? ""}
            onValueChange={(v) => {
              if (v) onSelect(v);
            }}
          >
            {filtered.map((status) => (
              <DropdownMenuRadioItem key={status.name} value={status.name}>
                <span>{status.name}</span>
                <span className="ml-auto text-xs text-base-content/60">{status.category}</span>
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        )}
      </SearchableSubMenuContent>
    </DropdownMenuSub>
  );
}

// ─── When Submenu ───────────────────────────────────────────────────────────

function WhenSubMenu({
  action,
  currentRule,
  statuses,
  onSave,
  triggerContent,
}: {
  action: TaskOverdueAction;
  currentRule: TaskOverdueActionRule | null;
  statuses: TaskStatus[];
  onSave: (rule: TaskOverdueActionRule) => void;
  triggerContent: ReactNode;
}) {
  const search = useMenuSearch();
  const parsedDays = useMemo(() => parseDaysInput(search.query), [search.query]);

  const isCurrentAction = currentRule?.action === action;
  const currentAfter = isCurrentAction ? (currentRule.after ?? null) : null;
  const currentStatus =
    currentRule?.action === "set_status" && currentRule.status
      ? currentRule.status
      : (statuses[0]?.name ?? "");

  function selectAfter(after: number | null) {
    const rule: TaskOverdueActionRule = { after, action };
    if (action === "set_status") {
      rule.status = currentStatus;
    }
    onSave(rule);
  }

  const filteredShortcuts = useMemo(() => {
    if (!search.query) return WHEN_SHORTCUTS;
    return WHEN_SHORTCUTS.filter((s) => s.label.toLowerCase().includes(search.query.toLowerCase()));
  }, [search.query]);

  const isSearching = search.query.length > 0;

  return (
    <DropdownMenuSub {...search.menuProps}>
      <DropdownMenuSubTrigger>{triggerContent}</DropdownMenuSubTrigger>
      <SearchableSubMenuContent search={search} placeholder='e.g. "3 days"' inputLabel="When">
        {isSearching &&
          parsedDays &&
          !filteredShortcuts.some((s) => s.after === parsedDays.after) && (
            <DropdownMenuItem
              onSelect={(e) => {
                e.preventDefault();
                selectAfter(parsedDays.after);
              }}
            >
              <span>{parsedDays.label}</span>
            </DropdownMenuItem>
          )}

        {filteredShortcuts.map((option) => (
          <DropdownMenuItem
            key={option.label}
            className="pl-7 relative"
            onSelect={(e) => {
              e.preventDefault();
              selectAfter(option.after);
            }}
          >
            <span className="absolute left-2 flex h-3.5 w-3.5 items-center justify-center">
              {isCurrentAction && currentAfter === option.after && (
                <Check className="size-3.5" aria-hidden="true" />
              )}
            </span>
            <span>{option.label}</span>
          </DropdownMenuItem>
        ))}

        {isSearching && !parsedDays && filteredShortcuts.length === 0 && (
          <div role="status" className="px-3 py-2 text-sm text-base-content/60">
            No matching options
          </div>
        )}

        {action === "set_status" && (
          <>
            <DropdownMenuSeparator />
            <StatusPickerSubMenu
              statuses={statuses}
              currentStatus={currentStatus}
              onSelect={(status) => {
                onSave({
                  after: isCurrentAction ? currentAfter : null,
                  action: "set_status",
                  status,
                });
              }}
              triggerContent={
                <>
                  <span>{currentStatus ? `Status: ${currentStatus}` : "Pick status..."}</span>
                  <ChevronRight className="ml-auto size-4" aria-hidden="true" />
                </>
              }
            />
          </>
        )}
      </SearchableSubMenuContent>
    </DropdownMenuSub>
  );
}

// ─── Overdue Action Submenu ─────────────────────────────────────────────────

function OverdueActionSubMenu({
  overdueActionRule,
  hasRecurrence,
  statuses,
  onSave,
}: {
  overdueActionRule: TaskOverdueActionRule | null;
  hasRecurrence: boolean;
  statuses: TaskStatus[];
  onSave: (rule: TaskOverdueActionRule | null) => void;
}) {
  const search = useMenuSearch();

  const actionItems = useMemo(() => {
    let items = ALL_ACTIONS;
    if (!hasRecurrence) {
      items = items.filter((a) => a.value !== "advance_recurrence");
    }
    if (!search.query) return items;
    return items.filter((a) => a.label.toLowerCase().includes(search.query.toLowerCase()));
  }, [search.query, hasRecurrence]);

  return (
    <DropdownMenuSub {...search.menuProps}>
      <DropdownMenuSubTrigger>
        <Clock className="size-4" aria-hidden="true" />
        <span>
          {overdueActionRule ? describeOverdueAction(overdueActionRule) : "When overdue..."}
        </span>
        <ChevronRight className="ml-auto size-4" aria-hidden="true" />
      </DropdownMenuSubTrigger>
      <SearchableSubMenuContent
        search={search}
        placeholder="Search actions..."
        inputLabel="Search actions"
      >
        {overdueActionRule && (
          <DropdownMenuItem
            className="text-error"
            onSelect={(e) => {
              e.preventDefault();
              onSave(null);
            }}
          >
            <X className="size-4" aria-hidden="true" />
            <span>None</span>
          </DropdownMenuItem>
        )}

        {actionItems.length === 0 ? (
          <div role="status" className="px-3 py-2 text-sm text-base-content/60">
            No matching actions
          </div>
        ) : (
          actionItems.map((item) => (
            <WhenSubMenu
              key={item.value}
              action={item.value}
              currentRule={overdueActionRule}
              statuses={statuses}
              onSave={onSave}
              triggerContent={
                <>
                  {overdueActionRule?.action === item.value ? (
                    <Check className="size-4" aria-hidden="true" />
                  ) : (
                    <span className="size-4" />
                  )}
                  <span>{item.label}</span>
                  <ChevronRight className="ml-auto size-4" aria-hidden="true" />
                </>
              }
            />
          ))
        )}
      </SearchableSubMenuContent>
    </DropdownMenuSub>
  );
}

// ─── Main Component ─────────────────────────────────────────────────────────

export function DueDateMenuField({
  spaceSlug,
  taskId,
  due,
  overdueActionRule,
  recurrenceType,
  statuses,
}: {
  spaceSlug: string;
  taskId: string;
  due: Task["due"];
  overdueActionRule: TaskOverdueActionRule | null;
  recurrenceType: TaskRecurrenceType;
  statuses: TaskStatus[];
}) {
  const mutation = useTaskPatch(spaceSlug, taskId);
  const search = useMenuSearch();
  const parsedDate = useMemo(() => parseDateInput(search.query), [search.query]);
  const today = new Date();

  const displayValue = useMemo(() => {
    if (!due) return null;
    return new Date(due.at).toLocaleDateString();
  }, [due]);

  function selectDate(isoDate: string) {
    mutation.mutate({ due: { at: isoDate, timezone: BROWSER_TIMEZONE } });
  }

  function handleOverdueSave(rule: TaskOverdueActionRule | null) {
    mutation.mutate({ overdueActionRule: rule });
  }

  const hasRecurrence = recurrenceType !== "one_off";

  return (
    <>
      <DropdownMenuRoot {...search.menuProps}>
        <FieldPill
          icon={<Calendar className="size-3.5" aria-hidden="true" />}
          label="Due date"
          value={displayValue}
        />
        <SearchableMenuContent
          search={search}
          placeholder='e.g. "tomorrow", "next friday"'
          inputLabel="Search due date"
        >
          {due && (
            <DropdownMenuItem
              className="text-error"
              onSelect={() => {
                mutation.mutate({ due: null });
              }}
            >
              <X className="size-4" aria-hidden="true" />
              <span>Clear due date</span>
            </DropdownMenuItem>
          )}

          {search.query ? (
            parsedDate ? (
              <DropdownMenuItem onSelect={() => selectDate(parsedDate.value)}>
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
            DATE_SHORTCUTS.map((shortcut) => {
              const date = addDays(today, shortcut.offsetDays);
              const isoDate = toISODate(date);
              return (
                <DropdownMenuItem key={shortcut.label} onSelect={() => selectDate(isoDate)}>
                  <span>{shortcut.label}</span>
                  <span className="ml-auto text-xs text-base-content/60">
                    {formatDateDisplay(date)}
                  </span>
                </DropdownMenuItem>
              );
            })
          )}

          {due && (
            <>
              <DropdownMenuSeparator />
              <OverdueActionSubMenu
                overdueActionRule={overdueActionRule}
                hasRecurrence={hasRecurrence}
                statuses={statuses}
                onSave={handleOverdueSave}
              />
            </>
          )}
        </SearchableMenuContent>
      </DropdownMenuRoot>
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </>
  );
}

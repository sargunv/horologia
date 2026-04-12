import { Menu, Portal } from "@skeletonlabs/skeleton-react";
import { Calendar, Check, ChevronRight, Clock, X } from "lucide-react";
import { useMemo } from "react";
import type { components } from "../../api/schema.d.ts";
import { FieldPill } from "../FieldPill.tsx";
import { MENU_ITEM_CLASS, SearchableMenuContent } from "../SearchableMenuContent.tsx";
import { addDays, formatDateDisplay, parseDateInput, toISODate } from "../../lib/dates.ts";
import { useMenuSearch } from "../../lib/useMenuSearch.ts";
import { useTaskPatch } from "../../lib/mutations.ts";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";

type Task = components["schemas"]["Task"];
type TaskOverdueAction = components["schemas"]["TaskOverdueAction"];
type TaskOverdueActionRule = components["schemas"]["TaskOverdueActionRule"];
type TaskRecurrenceType = components["schemas"]["TaskRecurrenceType"];
type TaskStatus = components["schemas"]["TaskStatus"];

// Zag.js reads z-index from Menu.Content's computed style and propagates
// it to the positioner via --z-index. Apply z-index to Content (not
// Positioner) so Zag picks it up. Child portals render before parent
// portals in DOM, so deeper menus need higher z-index.
const Z_SUBMENU = "z-10";
const Z_DETAIL = "z-20";
const Z_DEEP = "z-30";

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

// ─── Status Picker Content ──────────────────────────────────────────────────

function StatusPickerContent({
  statuses,
  currentStatus,
  onSelect,
  className,
}: {
  statuses: TaskStatus[];
  currentStatus: string | undefined;
  onSelect: (status: string) => void;
  className?: string;
}) {
  const search = useMenuSearch();

  const filtered = useMemo(
    () => statuses.filter((s) => s.name.toLowerCase().includes(search.query.toLowerCase())),
    [statuses, search.query],
  );

  return (
    <SearchableMenuContent
      inputProps={search.inputProps}
      placeholder="Search statuses..."
      className={className}
    >
      {filtered.length === 0 ? (
        <div role="status" className="text-surface-500 px-3 py-2 text-sm">
          No matching statuses
        </div>
      ) : (
        filtered.map((status) => (
          <Menu.OptionItem
            key={status.name}
            type="radio"
            checked={currentStatus === status.name}
            value={status.name}
            onCheckedChange={(checked) => {
              if (checked) onSelect(status.name);
            }}
            className={MENU_ITEM_CLASS}
          >
            <Menu.ItemIndicator className="invisible data-[state=checked]:visible">
              <Check className="size-4" />
            </Menu.ItemIndicator>
            <Menu.ItemText>{status.name}</Menu.ItemText>
            <span className="text-surface-500 ml-auto text-xs">{status.category}</span>
          </Menu.OptionItem>
        ))
      )}
    </SearchableMenuContent>
  );
}

// ─── When Submenu ───────────────────────────────────────────────────────────

function WhenSubMenu({
  action,
  currentRule,
  statuses,
  onSave,
  className,
}: {
  action: TaskOverdueAction;
  currentRule: TaskOverdueActionRule | null;
  statuses: TaskStatus[];
  onSave: (rule: TaskOverdueActionRule) => void;
  className?: string;
}) {
  const search = useMenuSearch();
  const parsedDays = useMemo(() => parseDaysInput(search.query), [search.query]);

  const isCurrentAction = currentRule?.action === action;
  const currentAfter = isCurrentAction ? (currentRule.after ?? null) : null;
  const currentStatus =
    currentRule?.action === "set_status" && currentRule.status
      ? currentRule.status
      : // Spaces always have ≥2 statuses; this fallback is defensive only
        (statuses[0]?.name ?? "");

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
    <SearchableMenuContent
      inputProps={search.inputProps}
      placeholder='e.g. "3 days"'
      className={className}
    >
      {/* Parsed days from search */}
      {isSearching &&
        parsedDays &&
        !filteredShortcuts.some((s) => s.after === parsedDays.after) && (
          <Menu.Item
            value={`parsed-${parsedDays.after}`}
            className={MENU_ITEM_CLASS}
            onClick={() => selectAfter(parsedDays.after)}
          >
            <Menu.ItemText>{parsedDays.label}</Menu.ItemText>
          </Menu.Item>
        )}

      {/* When shortcuts */}
      {filteredShortcuts.map((option) => (
        <Menu.OptionItem
          key={option.label}
          type="radio"
          checked={isCurrentAction && currentAfter === option.after}
          value={option.label}
          onCheckedChange={(checked) => {
            if (checked) selectAfter(option.after);
          }}
          className={MENU_ITEM_CLASS}
        >
          <Menu.ItemIndicator className="invisible data-[state=checked]:visible">
            <Check className="size-4" />
          </Menu.ItemIndicator>
          <Menu.ItemText>{option.label}</Menu.ItemText>
        </Menu.OptionItem>
      ))}

      {isSearching && !parsedDays && filteredShortcuts.length === 0 && (
        <div role="status" className="text-surface-500 px-3 py-2 text-sm">
          No matching options
        </div>
      )}

      {/* Status picker for set_status */}
      {action === "set_status" && (
        <>
          <Menu.Separator />
          <Menu typeahead={false} closeOnSelect={false}>
            <Menu.TriggerItem value="status-picker" className="justify-start gap-2 text-sm">
              <Menu.ItemText>
                {currentStatus ? `Status: ${currentStatus}` : "Pick status..."}
              </Menu.ItemText>
              <Menu.ItemIndicator className="ml-auto">
                <ChevronRight className="size-4" />
              </Menu.ItemIndicator>
            </Menu.TriggerItem>
            <Portal>
              <Menu.Positioner>
                <StatusPickerContent
                  className={Z_DEEP}
                  statuses={statuses}
                  currentStatus={currentStatus}
                  onSelect={(status) => {
                    onSave({
                      after: isCurrentAction ? currentAfter : null,
                      action: "set_status",
                      status,
                    });
                  }}
                />
              </Menu.Positioner>
            </Portal>
          </Menu>
        </>
      )}
    </SearchableMenuContent>
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
    <Menu typeahead={false} closeOnSelect={false}>
      <Menu.TriggerItem value="overdue-action" className="justify-start gap-2 text-sm">
        <Clock className="size-4" aria-hidden="true" />
        <Menu.ItemText>
          {overdueActionRule ? describeOverdueAction(overdueActionRule) : "When overdue..."}
        </Menu.ItemText>
        <Menu.ItemIndicator className="ml-auto">
          <ChevronRight className="size-4" />
        </Menu.ItemIndicator>
      </Menu.TriggerItem>
      <Portal>
        <Menu.Positioner>
          <SearchableMenuContent
            inputProps={search.inputProps}
            placeholder="Search actions..."
            className={Z_SUBMENU}
          >
            {/* Clear option */}
            {overdueActionRule && (
              <Menu.Item
                value="none"
                className={`text-error-500 ${MENU_ITEM_CLASS}`}
                onClick={() => onSave(null)}
              >
                <X className="size-4" aria-hidden="true" />
                <Menu.ItemText>None</Menu.ItemText>
              </Menu.Item>
            )}

            {actionItems.length === 0 ? (
              <div role="status" className="text-surface-500 px-3 py-2 text-sm">
                No matching actions
              </div>
            ) : (
              actionItems.map((item) => (
                <Menu key={item.value} typeahead={false} closeOnSelect={false}>
                  <Menu.TriggerItem value={item.value} className="justify-start gap-2 text-sm">
                    {overdueActionRule?.action === item.value ? (
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
                      <WhenSubMenu
                        className={Z_DETAIL}
                        action={item.value}
                        currentRule={overdueActionRule}
                        statuses={statuses}
                        onSave={onSave}
                      />
                    </Menu.Positioner>
                  </Portal>
                </Menu>
              ))
            )}
          </SearchableMenuContent>
        </Menu.Positioner>
      </Portal>
    </Menu>
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
      <Menu {...search.menuProps} closeOnSelect={false}>
        <FieldPill
          icon={<Calendar className="size-3.5" aria-hidden="true" />}
          label="Due date"
          value={displayValue}
        />
        <Portal>
          <Menu.Positioner>
            <SearchableMenuContent
              inputProps={search.inputProps}
              placeholder='e.g. "tomorrow", "next friday"'
            >
              {due && (
                <Menu.Item
                  value="clear"
                  className={`text-error-500 ${MENU_ITEM_CLASS}`}
                  onClick={() => {
                    mutation.mutate({ due: null });
                  }}
                >
                  <X className="size-4" aria-hidden="true" />
                  <Menu.ItemText>Clear due date</Menu.ItemText>
                </Menu.Item>
              )}

              {search.query ? (
                parsedDate ? (
                  <Menu.Item
                    value={parsedDate.value}
                    className={MENU_ITEM_CLASS}
                    onClick={() => selectDate(parsedDate.value)}
                  >
                    <Calendar className="size-4" aria-hidden="true" />
                    <Menu.ItemText>{parsedDate.label}</Menu.ItemText>
                    <span className="text-surface-500 ml-auto text-xs">{parsedDate.value}</span>
                  </Menu.Item>
                ) : (
                  <div role="status" className="text-surface-500 px-3 py-2 text-sm">
                    No matching dates
                  </div>
                )
              ) : (
                DATE_SHORTCUTS.map((shortcut) => {
                  const date = addDays(today, shortcut.offsetDays);
                  const isoDate = toISODate(date);
                  return (
                    <Menu.Item
                      key={shortcut.label}
                      value={isoDate}
                      className={MENU_ITEM_CLASS}
                      onClick={() => selectDate(isoDate)}
                    >
                      <Menu.ItemText>{shortcut.label}</Menu.ItemText>
                      <span className="text-surface-500 ml-auto text-xs">
                        {formatDateDisplay(date)}
                      </span>
                    </Menu.Item>
                  );
                })
              )}

              {/* Overdue action submenu — only when a due date is set */}
              {due && (
                <>
                  <Menu.Separator />
                  <OverdueActionSubMenu
                    overdueActionRule={overdueActionRule}
                    hasRecurrence={hasRecurrence}
                    statuses={statuses}
                    onSave={handleOverdueSave}
                  />
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

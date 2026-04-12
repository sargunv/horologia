import { Menu, Portal } from "@skeletonlabs/skeleton-react";
import { createFileRoute } from "@tanstack/react-router";
import {
  Calendar,
  Check,
  ChevronRight,
  CircleAlert,
  Gauge,
  RefreshCw,
  Users,
  X,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { FieldPill } from "../../components/FieldPill.tsx";
import { SearchableMenuContent } from "../../components/SearchableMenuContent.tsx";
import { useMenuSearch } from "../../lib/useMenuSearch.ts";

export const Route = createFileRoute("/_authenticated/playground")({
  component: PlaygroundPage,
});

// ─── Example 1: Simple Search Menu (radio single-select) ──────────────

interface StatusOption {
  name: string;
  category: "initial" | "intermediate" | "completion";
}

const STATUSES: StatusOption[] = [
  { name: "Backlog", category: "initial" },
  { name: "Todo", category: "initial" },
  { name: "In Progress", category: "intermediate" },
  { name: "In Review", category: "intermediate" },
  { name: "Done", category: "completion" },
  { name: "Cancelled", category: "completion" },
];

function SimpleSearchExample() {
  const [selected, setSelected] = useState("In Progress");
  const search = useMenuSearch();

  const filtered = useMemo(
    () => STATUSES.filter((s) => s.name.toLowerCase().includes(search.query.toLowerCase())),
    [search.query],
  );

  return (
    <ExampleCard
      title="Simple Search Menu"
      description="Radio single-select with text filtering. Used for status, priority, effort."
    >
      <Menu {...search.menuProps}>
        <FieldPill
          icon={<CircleAlert className="size-3.5" aria-hidden="true" />}
          label="Status"
          value={selected}
        />
        <Portal>
          <Menu.Positioner>
            <SearchableMenuContent inputProps={search.inputProps} placeholder="Search statuses...">
              {filtered.length === 0 ? (
                <div className="text-surface-500 px-3 py-2 text-sm">No matching statuses</div>
              ) : (
                filtered.map((status) => (
                  <Menu.OptionItem
                    key={status.name}
                    type="radio"
                    checked={selected === status.name}
                    value={status.name}
                    onCheckedChange={(checked) => {
                      if (checked) setSelected(status.name);
                    }}
                    className="justify-start gap-2 text-sm"
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
          </Menu.Positioner>
        </Portal>
      </Menu>
    </ExampleCard>
  );
}

// ─── Example 2: Custom Parser Menu (date-like) ────────────────────────

interface DateShortcut {
  label: string;
  offsetDays: number;
}

const DATE_SHORTCUTS: DateShortcut[] = [
  { label: "Today", offsetDays: 0 },
  { label: "Tomorrow", offsetDays: 1 },
  { label: "In 1 week", offsetDays: 7 },
  { label: "In 2 weeks", offsetDays: 14 },
  { label: "In 1 month", offsetDays: 30 },
];

function addDays(date: Date, days: number): Date {
  const result = new Date(date);
  result.setDate(result.getDate() + days);
  return result;
}

function formatDate(date: Date): string {
  return date.toLocaleDateString(undefined, {
    weekday: "short",
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function toISODate(date: Date): string {
  return date.toISOString().slice(0, 10);
}

/**
 * Mock date parser. In production, use chrono-node.
 * Handles: "tomorrow", "today", "in N days/weeks/months", ISO dates.
 */
function parseDateInput(input: string): { label: string; value: string } | null {
  const lower = input.trim().toLowerCase();
  if (!lower) return null;

  const today = new Date();

  if (lower === "today") {
    return { label: formatDate(today), value: toISODate(today) };
  }
  if (lower === "tomorrow") {
    const d = addDays(today, 1);
    return { label: formatDate(d), value: toISODate(d) };
  }
  if (lower === "yesterday") {
    const d = addDays(today, -1);
    return { label: formatDate(d), value: toISODate(d) };
  }

  // "in N days/weeks/months"
  const relativeMatch = /^in\s+(\d+)\s+(day|days|week|weeks|month|months)$/.exec(lower);
  if (relativeMatch?.[1] && relativeMatch[2]) {
    const n = parseInt(relativeMatch[1], 10);
    const unit = relativeMatch[2].replace(/s$/, "");
    const d = new Date(today);
    if (unit === "day") d.setDate(d.getDate() + n);
    else if (unit === "week") d.setDate(d.getDate() + n * 7);
    else if (unit === "month") d.setMonth(d.getMonth() + n);
    return { label: formatDate(d), value: toISODate(d) };
  }

  // ISO date or other parseable format
  const parsed = new Date(input);
  if (!isNaN(parsed.getTime())) {
    return { label: formatDate(parsed), value: toISODate(parsed) };
  }

  return null;
}

function CustomParserExample() {
  const [selected, setSelected] = useState<string | null>(null);
  const search = useMenuSearch();

  const parsedDate = useMemo(() => parseDateInput(search.query), [search.query]);

  const today = useMemo(() => new Date(), []);

  return (
    <ExampleCard
      title="Custom Parser Menu"
      description='Text input parses natural language dates. Try "tomorrow", "in 2 weeks", or an ISO date.'
    >
      <Menu {...search.menuProps}>
        <FieldPill
          icon={<Calendar className="size-3.5" aria-hidden="true" />}
          label="Due date"
          value={selected}
        />
        <Portal>
          <Menu.Positioner>
            <SearchableMenuContent
              inputProps={search.inputProps}
              placeholder='e.g. "tomorrow", "in 2 weeks"'
            >
              {selected && (
                <Menu.Item
                  value="clear"
                  className="text-error-500 justify-start gap-2 text-sm"
                  closeOnSelect
                  onClick={() => setSelected(null)}
                >
                  <X className="size-4" aria-hidden="true" />
                  <Menu.ItemText>Clear due date</Menu.ItemText>
                </Menu.Item>
              )}

              {search.query ? (
                parsedDate ? (
                  <Menu.Item
                    value={parsedDate.value}
                    className="justify-start gap-2 text-sm"
                    closeOnSelect
                    onClick={() => setSelected(parsedDate.value)}
                  >
                    <Calendar className="size-4" aria-hidden="true" />
                    <Menu.ItemText>{parsedDate.label}</Menu.ItemText>
                    <span className="text-surface-500 ml-auto text-xs">{parsedDate.value}</span>
                  </Menu.Item>
                ) : (
                  <div className="text-surface-500 px-3 py-2 text-sm">No matching dates</div>
                )
              ) : (
                DATE_SHORTCUTS.map((shortcut) => {
                  const date = addDays(today, shortcut.offsetDays);
                  const value = toISODate(date);
                  return (
                    <Menu.Item
                      key={shortcut.label}
                      value={value}
                      className="justify-start gap-2 text-sm"
                      closeOnSelect
                      onClick={() => setSelected(value)}
                    >
                      <Menu.ItemText>{shortcut.label}</Menu.ItemText>
                      <span className="text-surface-500 ml-auto text-xs">{formatDate(date)}</span>
                    </Menu.Item>
                  );
                })
              )}
            </SearchableMenuContent>
          </Menu.Positioner>
        </Portal>
      </Menu>
    </ExampleCard>
  );
}

// ─── Example 3: Submenu Menu (recurrence-like) ────────────────────────

const FREQUENCIES = ["Daily", "Weekly", "Monthly", "Yearly"];

interface RecurrenceState {
  type: string;
  frequency: string | null;
}

function SubmenuExample() {
  const [recurrence, setRecurrence] = useState<RecurrenceState>({
    type: "completion_based",
    frequency: "Weekly",
  });
  const search = useMenuSearch();

  const displayValue = useMemo(() => {
    if (recurrence.type === "one_off") return "One-off";
    if (recurrence.type === "on_dependency") return "On dependency";
    const typeLabel =
      recurrence.type === "completion_based"
        ? "On completion"
        : recurrence.type === "fixed"
          ? "Fixed"
          : "Fixed (accum.)";
    return recurrence.frequency ? `${typeLabel}, ${recurrence.frequency}` : typeLabel;
  }, [recurrence]);

  const typeItems = useMemo(() => {
    const items = [
      { value: "one_off", label: "One-off", hasSubmenu: false },
      { value: "completion_based", label: "On completion", hasSubmenu: true },
      { value: "fixed", label: "Fixed", hasSubmenu: true },
      { value: "fixed_accumulating", label: "Fixed (accumulating)", hasSubmenu: true },
      { value: "on_dependency", label: "On dependency", hasSubmenu: false },
    ];
    if (!search.query) return items;
    return items.filter((item) => item.label.toLowerCase().includes(search.query.toLowerCase()));
  }, [search.query]);

  return (
    <ExampleCard
      title="Submenu Menu"
      description="Top-level type selection with nested frequency submenus. Used for recurrence."
    >
      <Menu {...search.menuProps}>
        <FieldPill
          icon={<RefreshCw className="size-3.5" aria-hidden="true" />}
          label="Recurrence"
          value={displayValue}
        />
        <Portal>
          <Menu.Positioner>
            <SearchableMenuContent
              inputProps={search.inputProps}
              placeholder="Search recurrence types..."
            >
              {typeItems.length === 0 ? (
                <div className="text-surface-500 px-3 py-2 text-sm">No matching types</div>
              ) : (
                typeItems.map((item) =>
                  item.hasSubmenu ? (
                    <Menu key={item.value}>
                      <Menu.TriggerItem value={item.value} className="justify-start gap-2 text-sm">
                        {recurrence.type === item.value && (
                          <Check className="size-4" aria-hidden="true" />
                        )}
                        {recurrence.type !== item.value && <span className="size-4" />}
                        <Menu.ItemText>{item.label}</Menu.ItemText>
                        <Menu.ItemIndicator className="ml-auto">
                          <ChevronRight className="size-4" />
                        </Menu.ItemIndicator>
                      </Menu.TriggerItem>
                      <Portal>
                        <Menu.Positioner>
                          <Menu.Content>
                            {FREQUENCIES.map((freq) => (
                              <Menu.Item
                                key={freq}
                                value={`${item.value}-${freq.toLowerCase()}`}
                                className="justify-start gap-2 text-sm"
                                closeOnSelect
                                onClick={() => setRecurrence({ type: item.value, frequency: freq })}
                              >
                                {recurrence.type === item.value &&
                                  recurrence.frequency === freq && (
                                    <Check className="size-4" aria-hidden="true" />
                                  )}
                                {(recurrence.type !== item.value ||
                                  recurrence.frequency !== freq) && <span className="size-4" />}
                                <Menu.ItemText>{freq}</Menu.ItemText>
                              </Menu.Item>
                            ))}
                          </Menu.Content>
                        </Menu.Positioner>
                      </Portal>
                    </Menu>
                  ) : (
                    <Menu.Item
                      key={item.value}
                      value={item.value}
                      className="justify-start gap-2 text-sm"
                      closeOnSelect
                      onClick={() => setRecurrence({ type: item.value, frequency: null })}
                    >
                      {recurrence.type === item.value && (
                        <Check className="size-4" aria-hidden="true" />
                      )}
                      {recurrence.type !== item.value && <span className="size-4" />}
                      <Menu.ItemText>{item.label}</Menu.ItemText>
                    </Menu.Item>
                  ),
                )
              )}
            </SearchableMenuContent>
          </Menu.Positioner>
        </Portal>
      </Menu>
    </ExampleCard>
  );
}

// ─── Example 4: Async Search Menu (multi-select with simulated delay) ─

interface MockMember {
  id: string;
  name: string;
  email: string;
}

const ALL_MEMBERS: MockMember[] = [
  { id: "U1", name: "Alice Chen", email: "alice@example.com" },
  { id: "U2", name: "Bob Martinez", email: "bob@example.com" },
  { id: "U3", name: "Charlie Kim", email: "charlie@example.com" },
  { id: "U4", name: "Diana Patel", email: "diana@example.com" },
  { id: "U5", name: "Eve Johnson", email: "eve@example.com" },
  { id: "U6", name: "Frank Liu", email: "frank@example.com" },
  { id: "U7", name: "Grace Thompson", email: "grace@example.com" },
  { id: "U8", name: "Henry Wilson", email: "henry@example.com" },
];

function simulateSearch(query: string): Promise<MockMember[]> {
  return new Promise((resolve) => {
    setTimeout(() => {
      if (!query) {
        resolve(ALL_MEMBERS);
        return;
      }
      const lower = query.toLowerCase();
      resolve(
        ALL_MEMBERS.filter(
          (m) => m.name.toLowerCase().includes(lower) || m.email.toLowerCase().includes(lower),
        ),
      );
    }, 400);
  });
}

function AsyncSearchExample() {
  const [selected, setSelected] = useState<string[]>(["U1", "U3"]);
  const [results, setResults] = useState<MockMember[]>(ALL_MEMBERS);
  const [loading, setLoading] = useState(false);
  const search = useMenuSearch();

  useEffect(() => {
    setLoading(true);
    const timer = setTimeout(() => {
      void simulateSearch(search.query).then((members) => {
        setResults(members);
        setLoading(false);
      });
    }, 100);
    return () => clearTimeout(timer);
  }, [search.query]);

  const handleCheckedChange = useCallback((memberId: string, checked: boolean) => {
    setSelected((prev) => (checked ? [...prev, memberId] : prev.filter((id) => id !== memberId)));
  }, []);

  const displayValue = useMemo(() => {
    if (selected.length === 0) return null;
    const names = selected.map((id) => ALL_MEMBERS.find((m) => m.id === id)?.name ?? id).join(", ");
    return names;
  }, [selected]);

  return (
    <ExampleCard
      title="Async Search Menu"
      description="Multi-select with simulated async search (400ms delay). Used for assignees, rotation pool."
    >
      <Menu
        typeahead={false}
        closeOnSelect={false}
        onOpenChange={(details) => {
          search.handleOpenChange(details);
          // In production, batch mutations on close:
          // if (!details.open) batchMutate(selected);
        }}
      >
        <FieldPill
          icon={<Users className="size-3.5" aria-hidden="true" />}
          label="Assignees"
          value={displayValue}
        />
        <Portal>
          <Menu.Positioner>
            <SearchableMenuContent inputProps={search.inputProps} placeholder="Search members...">
              {loading ? (
                <div className="text-surface-500 px-3 py-2 text-sm">Searching...</div>
              ) : results.length === 0 ? (
                <div className="text-surface-500 px-3 py-2 text-sm">No members found</div>
              ) : (
                results.map((member) => (
                  <Menu.OptionItem
                    key={member.id}
                    type="checkbox"
                    checked={selected.includes(member.id)}
                    value={member.id}
                    onCheckedChange={(checked) => handleCheckedChange(member.id, checked)}
                    className="justify-start gap-2 text-sm"
                  >
                    <Menu.ItemIndicator className="invisible data-[state=checked]:visible">
                      <Check className="size-4" />
                    </Menu.ItemIndicator>
                    <Menu.ItemText>{member.name}</Menu.ItemText>
                    <span className="text-surface-500 ml-auto text-xs">{member.email}</span>
                  </Menu.OptionItem>
                ))
              )}
            </SearchableMenuContent>
          </Menu.Positioner>
        </Portal>
      </Menu>
    </ExampleCard>
  );
}

// ─── Nullable field example (effort with "None") ──────────────────────

interface EffortOption {
  name: string;
}

const EFFORT_LEVELS: EffortOption[] = [
  { name: "Trivial" },
  { name: "Small" },
  { name: "Medium" },
  { name: "Large" },
  { name: "Extra Large" },
];

function NullableFieldExample() {
  const [selected, setSelected] = useState<string | null>("Medium");
  const search = useMenuSearch();

  const filtered = useMemo(() => {
    if (!search.query) return EFFORT_LEVELS;
    return EFFORT_LEVELS.filter((e) => e.name.toLowerCase().includes(search.query.toLowerCase()));
  }, [search.query]);

  return (
    <ExampleCard
      title="Nullable Field"
      description='Radio single-select with a "None" clear option. Used for effort, priority.'
    >
      <Menu {...search.menuProps}>
        <FieldPill
          icon={<Gauge className="size-3.5" aria-hidden="true" />}
          label="Effort"
          value={selected}
        />
        <Portal>
          <Menu.Positioner>
            <SearchableMenuContent
              inputProps={search.inputProps}
              placeholder="Search effort levels..."
            >
              {selected !== null && (
                <Menu.Item
                  value="none"
                  className="text-error-500 justify-start gap-2 text-sm"
                  closeOnSelect
                  onClick={() => setSelected(null)}
                >
                  <X className="size-4" aria-hidden="true" />
                  <Menu.ItemText>None</Menu.ItemText>
                </Menu.Item>
              )}
              {filtered.length === 0 ? (
                <div className="text-surface-500 px-3 py-2 text-sm">No matching levels</div>
              ) : (
                filtered.map((effort) => (
                  <Menu.OptionItem
                    key={effort.name}
                    type="radio"
                    checked={selected === effort.name}
                    value={effort.name}
                    onCheckedChange={(checked) => {
                      if (checked) setSelected(effort.name);
                    }}
                    className="justify-start gap-2 text-sm"
                  >
                    <Menu.ItemIndicator className="invisible data-[state=checked]:visible">
                      <Check className="size-4" />
                    </Menu.ItemIndicator>
                    <Menu.ItemText>{effort.name}</Menu.ItemText>
                  </Menu.OptionItem>
                ))
              )}
            </SearchableMenuContent>
          </Menu.Positioner>
        </Portal>
      </Menu>
    </ExampleCard>
  );
}

// ─── Shared layout ────────────────────────────────────────────────────

function ExampleCard({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <div className="card preset-outlined-surface-200-800 space-y-4 p-6">
      <div>
        <h2 className="h5">{title}</h2>
        <p className="text-surface-600-400 mt-1 text-sm">{description}</p>
      </div>
      <div className="flex flex-wrap items-center gap-2">{children}</div>
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────

function PlaygroundPage() {
  return (
    <div className="mx-auto max-w-3xl space-y-8 p-6">
      <div>
        <h1 className="h3">Searchable Menu Components</h1>
        <p className="text-surface-600-400 mt-2 text-sm">
          Building blocks for the task detail redesign. Every field interaction is a Menu with a
          search input at the top — consistent, keyboard-driven, composable.
        </p>
      </div>

      <SimpleSearchExample />
      <NullableFieldExample />
      <CustomParserExample />
      <SubmenuExample />
      <AsyncSearchExample />
    </div>
  );
}

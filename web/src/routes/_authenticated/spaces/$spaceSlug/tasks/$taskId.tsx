import { Dialog, Menu, Portal, TagsInput } from "@skeletonlabs/skeleton-react";
import {
  useMutation,
  useQueryClient,
  useSuspenseInfiniteQuery,
  useSuspenseQuery,
} from "@tanstack/react-query";
import { createFileRoute, createLink, useNavigate } from "@tanstack/react-router";
import {
  Activity,
  ArrowLeft,
  Calendar,
  Check,
  ChevronRight,
  CircleAlert,
  Copy,
  Ellipsis,
  Gauge,
  RefreshCw,
  SignalHigh,
  Trash2,
  Users,
  X,
} from "lucide-react";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { apiClient } from "../../../../../api/client.ts";
import type { components } from "../../../../../api/schema.d.ts";
import { FieldPill } from "../../../../../components/FieldPill.tsx";
import {
  MENU_ITEM_CLASS,
  SearchableMenuContent,
} from "../../../../../components/SearchableMenuContent.tsx";
import { RecurrenceMenuField } from "../../../../../components/task/RecurrenceMenuField.tsx";
import { TaskDescriptionEditor } from "../../../../../components/TaskDescriptionEditor.tsx";
import { ActivityFeed } from "../../../../../components/ActivityFeed.tsx";
import { ErrorAlert } from "../../../../../components/space-settings/ErrorAlert.tsx";
import { OverdueActionEditor } from "../../../../../components/task/OverdueActionEditor.tsx";
import { RelationsSection } from "../../../../../components/task/RelationsSection.tsx";
import { useSpaceMemberMap } from "../../../../../lib/hooks.ts";
import { addDays, formatDateDisplay, parseDateInput, toISODate } from "../../../../../lib/dates.ts";
import { useMenuSearch } from "../../../../../lib/useMenuSearch.ts";
import { useTaskPatch } from "../../../../../lib/mutations.ts";
import {
  spaceEffortLevelsQueryOptions,
  spaceMembersQueryOptions,
  spacePriorityLevelsQueryOptions,
  spaceQueryOptions,
  spaceTaskQueryOptions,
  spaceTaskStatusesQueryOptions,
  taskActivityInfiniteQueryOptions,
} from "../../../../../lib/queries.ts";

type Task = components["schemas"]["Task"];
type TaskOverdueActionRule = components["schemas"]["TaskOverdueActionRule"];
type TaskStatus = components["schemas"]["TaskStatus"];
type SpaceMember = components["schemas"]["SpaceMember"];

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug/tasks/$taskId")({
  loader: ({ context: { queryClient }, params: { spaceSlug, taskId } }) =>
    Promise.all([
      queryClient.ensureQueryData(spaceQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spaceTaskQueryOptions(spaceSlug, taskId)),
      queryClient.ensureQueryData(spaceMembersQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spaceTaskStatusesQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spaceEffortLevelsQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spacePriorityLevelsQueryOptions(spaceSlug)),
    ]),
  component: TaskDetailPage,
});

const BackLink = createLink("a");
const BreadcrumbLink = createLink("a");

// ─── Breadcrumb Bar ─────────────────────────────────────────────────────────

function TaskBreadcrumbBar({
  spaceName,
  spaceSlug,
  taskId,
}: {
  spaceName: string;
  spaceSlug: string;
  taskId: string;
}) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [deleteOpen, setDeleteOpen] = useState(false);
  const cancelRef = useRef<HTMLButtonElement>(null);

  const deleteMutation = useMutation({
    mutationFn: async () => {
      const { error } = await apiClient.DELETE("/spaces/{spaceSlug}/tasks/{taskId}", {
        params: { path: { spaceSlug, taskId } },
      });
      if (error) throw new Error(error.message ?? "Failed to delete task");
    },
    onSuccess: async () => {
      queryClient.removeQueries({ queryKey: ["spaces", spaceSlug, "tasks", taskId] });
      await queryClient.invalidateQueries({ queryKey: ["spaces", spaceSlug, "tasks", "list"] });
      await navigate({ to: "/spaces/$spaceSlug", params: { spaceSlug } });
    },
  });

  function handleCopyId() {
    void navigator.clipboard.writeText(taskId).catch(() => {});
  }

  return (
    <nav aria-label="Breadcrumb" className="flex items-center gap-2">
      <ol className="flex min-w-0 items-center gap-1 text-sm">
        <li>
          <BreadcrumbLink
            to="/spaces/$spaceSlug"
            params={{ spaceSlug }}
            className="text-surface-600-400 truncate hover:underline"
          >
            {spaceName}
          </BreadcrumbLink>
        </li>
        <li className="text-surface-500" aria-hidden="true">
          <ChevronRight className="size-3" />
        </li>
        <li>
          <BreadcrumbLink
            to="/spaces/$spaceSlug/tasks/$taskId"
            params={{ spaceSlug, taskId }}
            className="shrink-0 font-mono hover:underline"
          >
            {taskId}
          </BreadcrumbLink>
        </li>
      </ol>

      <Menu>
        <Menu.Trigger
          className="btn-icon btn-sm preset-tonal-surface ml-auto"
          aria-label="Task actions"
        >
          <Ellipsis className="size-3.5" />
        </Menu.Trigger>
        <Portal>
          <Menu.Positioner>
            <Menu.Content>
              <Menu.Item value="copy-id" className={MENU_ITEM_CLASS} onClick={handleCopyId}>
                <Copy className="size-4" aria-hidden="true" />
                <Menu.ItemText>Copy task ID</Menu.ItemText>
              </Menu.Item>
              <Menu.Item
                value="copy-url"
                className={MENU_ITEM_CLASS}
                onClick={() => {
                  void navigator.clipboard.writeText(window.location.href).catch(() => {});
                }}
              >
                <Copy className="size-4" aria-hidden="true" />
                <Menu.ItemText>Copy URL</Menu.ItemText>
              </Menu.Item>
              <Menu.Separator />
              <Menu.Item
                value="delete"
                className={`text-error-500 ${MENU_ITEM_CLASS}`}
                onClick={() => setDeleteOpen(true)}
              >
                <Trash2 className="size-4" aria-hidden="true" />
                <Menu.ItemText>Delete task</Menu.ItemText>
              </Menu.Item>
            </Menu.Content>
          </Menu.Positioner>
        </Portal>
      </Menu>

      <Dialog
        open={deleteOpen}
        onOpenChange={(d) => {
          setDeleteOpen(d.open);
          if (!d.open) deleteMutation.reset();
        }}
        role="alertdialog"
        initialFocusEl={() => cancelRef.current}
      >
        <Portal>
          <Dialog.Backdrop className="fixed inset-0 z-50 bg-surface-50-950/50" />
          <Dialog.Positioner className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <Dialog.Content className="card bg-surface-100-900 w-full max-w-md space-y-4 p-6">
              <Dialog.Title className="h4">Delete task</Dialog.Title>
              <Dialog.Description className="text-surface-600-400 text-sm">
                Are you sure you want to delete this task? This action cannot be undone.
              </Dialog.Description>
              <div role="alert" aria-live="assertive">
                {deleteMutation.error && <ErrorAlert message={deleteMutation.error.message} />}
              </div>
              <div className="flex gap-3">
                <button
                  type="button"
                  onClick={() => deleteMutation.mutate()}
                  disabled={deleteMutation.isPending}
                  className="btn preset-filled-error-500 flex-1"
                >
                  {deleteMutation.isPending ? "Deleting..." : "Delete"}
                </button>
                <Dialog.CloseTrigger
                  ref={cancelRef}
                  className="btn preset-outlined-surface-200-800"
                >
                  Cancel
                </Dialog.CloseTrigger>
              </div>
            </Dialog.Content>
          </Dialog.Positioner>
        </Portal>
      </Dialog>
    </nav>
  );
}

// ─── Editable Title ─────────────────────────────────────────────────────────

function EditableTitle({
  spaceSlug,
  taskId,
  value,
}: {
  spaceSlug: string;
  taskId: string;
  value: string;
}) {
  const mutation = useTaskPatch(spaceSlug, taskId);
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(value);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!editing) setDraft(value);
  }, [value, editing]);

  useEffect(() => {
    if (editing) inputRef.current?.focus();
  }, [editing]);

  function save() {
    setEditing(false);
    const trimmed = draft.trim();
    if (trimmed && trimmed !== value) {
      mutation.reset();
      mutation.mutate({ title: trimmed });
    } else {
      setDraft(value);
    }
  }

  function enterEditing() {
    mutation.reset();
    setEditing(true);
  }

  if (editing) {
    return (
      <div>
        <input
          ref={inputRef}
          type="text"
          aria-label="Task title"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={save}
          onKeyDown={(e) => {
            if (e.key === "Enter") save();
            if (e.key === "Escape") {
              setDraft(value);
              setEditing(false);
            }
          }}
          className="h4 w-full border-b-2 border-primary-500 bg-transparent outline-none"
          maxLength={500}
          disabled={mutation.isPending}
        />
        {mutation.error && <ErrorAlert message={mutation.error.message} />}
      </div>
    );
  }

  return (
    <div>
      <h1
        className="h4 -mx-1 cursor-pointer rounded px-1 transition-colors hover:bg-surface-100-900"
        onClick={enterEditing}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            enterEditing();
          }
        }}
        role="button"
        tabIndex={0}
      >
        {value}
      </h1>
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </div>
  );
}

// ─── Status Field ───────────────────────────────────────────────────────────

function StatusField({
  spaceSlug,
  taskId,
  value,
  statuses,
}: {
  spaceSlug: string;
  taskId: string;
  value: string;
  statuses: TaskStatus[];
}) {
  const mutation = useTaskPatch(spaceSlug, taskId);
  const search = useMenuSearch();

  const filtered = useMemo(
    () => statuses.filter((s) => s.name.toLowerCase().includes(search.query.toLowerCase())),
    [statuses, search.query],
  );

  return (
    <>
      <Menu {...search.menuProps} closeOnSelect={false}>
        <FieldPill
          icon={<CircleAlert className="size-3.5" aria-hidden="true" />}
          label="Status"
          value={value}
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
                    checked={value === status.name}
                    value={status.name}
                    onCheckedChange={(checked) => {
                      if (checked) {
                        mutation.reset();
                        mutation.mutate({ status: status.name });
                      }
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
          </Menu.Positioner>
        </Portal>
      </Menu>
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </>
  );
}

// ─── Nullable Select Field (effort, priority) ───────────────────────────────

function NullableMenuField({
  spaceSlug,
  taskId,
  field,
  value,
  options,
  label,
  icon,
}: {
  spaceSlug: string;
  taskId: string;
  field: "effort" | "priority";
  value: string | null;
  options: { name: string }[];
  label: string;
  icon: React.ReactNode;
}) {
  const mutation = useTaskPatch(spaceSlug, taskId);
  const search = useMenuSearch();

  const filtered = useMemo(
    () => options.filter((o) => o.name.toLowerCase().includes(search.query.toLowerCase())),
    [options, search.query],
  );

  return (
    <>
      <Menu {...search.menuProps} closeOnSelect={false}>
        <FieldPill icon={icon} label={label} value={value} />
        <Portal>
          <Menu.Positioner>
            <SearchableMenuContent
              inputProps={search.inputProps}
              placeholder={`Search ${label.toLowerCase()}...`}
            >
              {value !== null && (
                <Menu.Item
                  value="none"
                  className={`text-error-500 ${MENU_ITEM_CLASS}`}
                  onClick={() => {
                    mutation.reset();
                    mutation.mutate({ [field]: null });
                  }}
                >
                  <X className="size-4" aria-hidden="true" />
                  <Menu.ItemText>None</Menu.ItemText>
                </Menu.Item>
              )}
              {filtered.length === 0 ? (
                <div className="text-surface-500 px-3 py-2 text-sm">No matching options</div>
              ) : (
                filtered.map((option) => (
                  <Menu.OptionItem
                    key={option.name}
                    type="radio"
                    checked={value === option.name}
                    value={option.name}
                    onCheckedChange={(checked) => {
                      if (checked) {
                        mutation.reset();
                        mutation.mutate({ [field]: option.name });
                      }
                    }}
                    className={MENU_ITEM_CLASS}
                  >
                    <Menu.ItemIndicator className="invisible data-[state=checked]:visible">
                      <Check className="size-4" />
                    </Menu.ItemIndicator>
                    <Menu.ItemText>{option.name}</Menu.ItemText>
                  </Menu.OptionItem>
                ))
              )}
            </SearchableMenuContent>
          </Menu.Positioner>
        </Portal>
      </Menu>
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </>
  );
}

// ─── Member Multi-Select Field ──────────────────────────────────────────────

function MemberMenuField({
  spaceSlug,
  taskId,
  field,
  value,
  members,
  memberMap,
  label,
  icon,
}: {
  spaceSlug: string;
  taskId: string;
  field: "assigneeIds" | "rotationPool";
  value: string[];
  members: SpaceMember[];
  memberMap: Map<string, SpaceMember>;
  label: string;
  icon: React.ReactNode;
}) {
  const mutation = useTaskPatch(spaceSlug, taskId);
  const search = useMenuSearch();
  const [draft, setDraft] = useState(value);

  useEffect(() => {
    setDraft(value);
  }, [value]);

  const filtered = useMemo(
    () =>
      members.filter(
        (m) =>
          m.userName.toLowerCase().includes(search.query.toLowerCase()) ||
          m.userEmail.toLowerCase().includes(search.query.toLowerCase()),
      ),
    [members, search.query],
  );

  const displayValue = useMemo(() => {
    if (draft.length === 0) return null;
    return draft.map((id) => memberMap.get(id)?.userName ?? id).join(", ");
  }, [draft, memberMap]);

  const handleCheckedChange = useCallback((memberId: string, checked: boolean) => {
    setDraft((prev) => (checked ? [...prev, memberId] : prev.filter((id) => id !== memberId)));
  }, []);

  return (
    <>
      <Menu
        typeahead={false}
        closeOnSelect={false}
        onOpenChange={(details) => {
          search.handleOpenChange(details);
          const changed = draft.length !== value.length || draft.some((id, i) => id !== value[i]);
          if (!details.open && changed) {
            mutation.reset();
            mutation.mutate({ [field]: draft });
          }
        }}
      >
        <FieldPill icon={icon} label={label} value={displayValue} />
        <Portal>
          <Menu.Positioner>
            <SearchableMenuContent inputProps={search.inputProps} placeholder="Search members...">
              {filtered.length === 0 ? (
                <div className="text-surface-500 px-3 py-2 text-sm">No members found</div>
              ) : (
                filtered.map((member) => (
                  <Menu.OptionItem
                    key={member.userId}
                    type="checkbox"
                    checked={draft.includes(member.userId)}
                    value={member.userId}
                    onCheckedChange={(checked) => handleCheckedChange(member.userId, checked)}
                    className={MENU_ITEM_CLASS}
                  >
                    <Menu.ItemIndicator className="invisible data-[state=checked]:visible">
                      <Check className="size-4" />
                    </Menu.ItemIndicator>
                    <Menu.ItemText>{member.userName}</Menu.ItemText>
                    <span className="text-surface-500 ml-auto text-xs">{member.userEmail}</span>
                  </Menu.OptionItem>
                ))
              )}
            </SearchableMenuContent>
          </Menu.Positioner>
        </Portal>
      </Menu>
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </>
  );
}

// ─── Due Date Field ─────────────────────────────────────────────────────────

const BROWSER_TIMEZONE = Intl.DateTimeFormat().resolvedOptions().timeZone;

const DATE_SHORTCUTS = [
  { label: "Today", offsetDays: 0 },
  { label: "Tomorrow", offsetDays: 1 },
  { label: "In 1 week", offsetDays: 7 },
  { label: "In 2 weeks", offsetDays: 14 },
  { label: "In 1 month", offsetDays: 30 },
];

function DueDateField({
  spaceSlug,
  taskId,
  value,
}: {
  spaceSlug: string;
  taskId: string;
  value: Task["due"];
}) {
  const mutation = useTaskPatch(spaceSlug, taskId);
  const search = useMenuSearch();
  const parsedDate = useMemo(() => parseDateInput(search.query), [search.query]);
  const today = new Date();

  const displayValue = useMemo(() => {
    if (!value) return null;
    return new Date(value.at).toLocaleDateString();
  }, [value]);

  function selectDate(isoDate: string) {
    mutation.reset();
    mutation.mutate({ due: { at: isoDate, timezone: BROWSER_TIMEZONE } });
  }

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
              {value && (
                <Menu.Item
                  value="clear"
                  className={`text-error-500 ${MENU_ITEM_CLASS}`}
                  onClick={() => {
                    mutation.reset();
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
                  <div className="text-surface-500 px-3 py-2 text-sm">No matching dates</div>
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
            </SearchableMenuContent>
          </Menu.Positioner>
        </Portal>
      </Menu>
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </>
  );
}

// ─── Overdue Action Field ───────────────────────────────────────────────────

function OverdueActionField({
  spaceSlug,
  taskId,
  overdueActionRule,
  statuses,
}: {
  spaceSlug: string;
  taskId: string;
  overdueActionRule: TaskOverdueActionRule | null;
  statuses: TaskStatus[];
}) {
  const mutation = useTaskPatch(spaceSlug, taskId);

  return (
    <div className="flex flex-col gap-1">
      <OverdueActionEditor
        overdueActionRule={overdueActionRule}
        statuses={statuses}
        onSave={(val) => {
          mutation.reset();
          mutation.mutate({ overdueActionRule: val });
        }}
        disabled={mutation.isPending}
      />
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </div>
  );
}

// ─── Tags Field ─────────────────────────────────────────────────────────────

function TagsField({
  spaceSlug,
  taskId,
  value,
}: {
  spaceSlug: string;
  taskId: string;
  value: string[];
}) {
  const mutation = useTaskPatch(spaceSlug, taskId);

  return (
    <div className="flex flex-col gap-1">
      <TagsInput
        value={value}
        onValueChange={(e) => {
          mutation.reset();
          mutation.mutate({ tags: e.value });
        }}
        validate={(details) => {
          const trimmed = details.inputValue.trim();
          return trimmed.length > 0 && !value.includes(trimmed);
        }}
        blurBehavior="add"
        disabled={mutation.isPending}
      >
        <TagsInput.Control className="input preset-outlined-surface-200-800 flex flex-wrap items-center gap-1 py-1.5">
          {value.map((tag, i) => (
            <TagsInput.Item key={tag} index={i} value={tag}>
              <TagsInput.ItemPreview className="preset-tonal-surface rounded-base flex items-center gap-1 px-2 py-0.5 text-xs">
                <TagsInput.ItemText>{tag}</TagsInput.ItemText>
                <TagsInput.ItemDeleteTrigger className="cursor-pointer opacity-60 hover:opacity-100">
                  <X className="size-3" aria-hidden="true" />
                </TagsInput.ItemDeleteTrigger>
              </TagsInput.ItemPreview>
              <TagsInput.ItemInput className="outline-none" />
            </TagsInput.Item>
          ))}
          <TagsInput.Input
            placeholder={value.length === 0 ? "Add tags..." : ""}
            className="min-w-20 flex-1 bg-transparent text-sm outline-none"
          />
          <TagsInput.HiddenInput />
        </TagsInput.Control>
      </TagsInput>
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </div>
  );
}

// ─── Activity Feed ──────────────────────────────────────────────────────────

function TaskActivityFeed({ spaceSlug, taskId }: { spaceSlug: string; taskId: string }) {
  const memberMap = useSpaceMemberMap(spaceSlug);
  const {
    data: pages,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
  } = useSuspenseInfiniteQuery(taskActivityInfiniteQueryOptions(spaceSlug, taskId));
  const entries = useMemo(() => pages.pages.flatMap((p) => p.items), [pages]);

  return (
    <ActivityFeed
      entries={entries}
      hasNextPage={hasNextPage}
      fetchNextPage={fetchNextPage}
      isFetchingNextPage={isFetchingNextPage}
      memberMap={memberMap}
    />
  );
}

// ─── Page ───────────────────────────────────────────────────────────────────

function TaskDetailPage() {
  const { spaceSlug, taskId } = Route.useParams();
  const { data: space } = useSuspenseQuery(spaceQueryOptions(spaceSlug));
  const { data: task } = useSuspenseQuery(spaceTaskQueryOptions(spaceSlug, taskId));
  const { data: members } = useSuspenseQuery(spaceMembersQueryOptions(spaceSlug));
  const { data: statuses } = useSuspenseQuery(spaceTaskStatusesQueryOptions(spaceSlug));
  const { data: effortLevels } = useSuspenseQuery(spaceEffortLevelsQueryOptions(spaceSlug));
  const { data: priorityLevels } = useSuspenseQuery(spacePriorityLevelsQueryOptions(spaceSlug));
  const memberMap = useSpaceMemberMap(spaceSlug);

  return (
    <div className="space-y-4">
      <BackLink
        to="/spaces/$spaceSlug"
        params={{ spaceSlug }}
        className="text-surface-600-400 hover:text-surface-950-50 inline-flex items-center gap-1 text-sm transition-colors lg:hidden"
      >
        <ArrowLeft className="size-4" aria-hidden="true" />
        Back to {space.name}
      </BackLink>

      <TaskBreadcrumbBar spaceName={space.name} spaceSlug={spaceSlug} taskId={task.id} />

      <EditableTitle spaceSlug={spaceSlug} taskId={taskId} value={task.title} />

      <div className="flex flex-wrap items-center gap-2">
        <StatusField
          spaceSlug={spaceSlug}
          taskId={taskId}
          value={task.status}
          statuses={statuses}
        />
        <NullableMenuField
          spaceSlug={spaceSlug}
          taskId={taskId}
          field="priority"
          value={task.priority}
          options={priorityLevels}
          label="Priority"
          icon={<SignalHigh className="size-3.5" aria-hidden="true" />}
        />
        <NullableMenuField
          spaceSlug={spaceSlug}
          taskId={taskId}
          field="effort"
          value={task.effort}
          options={effortLevels}
          label="Effort"
          icon={<Gauge className="size-3.5" aria-hidden="true" />}
        />
        <MemberMenuField
          spaceSlug={spaceSlug}
          taskId={taskId}
          field="assigneeIds"
          value={task.assigneeIds}
          members={members}
          memberMap={memberMap}
          label="Assignees"
          icon={<Users className="size-3.5" aria-hidden="true" />}
        />
        <DueDateField spaceSlug={spaceSlug} taskId={taskId} value={task.due} />
        <RecurrenceMenuField
          spaceSlug={spaceSlug}
          taskId={taskId}
          recurrenceType={task.recurrenceType}
          recurrenceRule={task.recurrenceRule}
        />
        <MemberMenuField
          spaceSlug={spaceSlug}
          taskId={taskId}
          field="rotationPool"
          value={task.rotationPool}
          members={members}
          memberMap={memberMap}
          label="Rotation"
          icon={<RefreshCw className="size-3.5" aria-hidden="true" />}
        />
      </div>

      <TagsField spaceSlug={spaceSlug} taskId={taskId} value={task.tags} />

      <TaskDescriptionEditor spaceSlug={spaceSlug} taskId={taskId} value={task.description} />

      <div className="space-y-4">
        {task.recurrenceType !== "one_off" && task.due !== null && (
          <OverdueActionField
            spaceSlug={spaceSlug}
            taskId={taskId}
            overdueActionRule={task.overdueActionRule}
            statuses={statuses}
          />
        )}
      </div>

      <RelationsSection spaceSlug={spaceSlug} taskId={task.id} relations={task.relations} />

      <div>
        <h2 className="h5 mb-4 flex items-center gap-2">
          <Activity className="size-4" aria-hidden="true" />
          Activity
        </h2>
        <Suspense
          fallback={
            <div className="text-surface-500 py-6 text-center text-sm">Loading activity…</div>
          }
        >
          <TaskActivityFeed spaceSlug={spaceSlug} taskId={taskId} />
        </Suspense>
      </div>
    </div>
  );
}

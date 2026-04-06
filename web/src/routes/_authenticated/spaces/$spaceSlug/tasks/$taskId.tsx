import {
  Combobox,
  Dialog,
  Portal,
  TagsInput,
  parseDate,
  useListCollection,
} from "@skeletonlabs/skeleton-react";
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
  ChevronDown,
  CircleAlert,
  Clock,
  Gauge,
  RefreshCw,
  SignalHigh,
  Tag,
  Trash2,
  Users,
  X,
} from "lucide-react";
import { Suspense, useEffect, useMemo, useRef, useState } from "react";
import { apiClient } from "../../../../../api/client.ts";
import type { components } from "../../../../../api/schema.d.ts";
import { DateField } from "../../../../../components/DateField.tsx";
import { RecurrenceRuleEditor } from "../../../../../components/RecurrenceRuleEditor.tsx";
import { TimezoneCombobox } from "../../../../../components/TimezoneCombobox.tsx";
import { TaskDescriptionEditor } from "../../../../../components/TaskDescriptionEditor.tsx";
import { ActivityFeed } from "../../../../../components/ActivityFeed.tsx";
import { ErrorAlert } from "../../../../../components/space-settings/ErrorAlert.tsx";
import { OverdueActionEditor } from "../../../../../components/task/OverdueActionEditor.tsx";
import { RelationsSection } from "../../../../../components/task/RelationsSection.tsx";
import { useSpaceMemberMap } from "../../../../../lib/hooks.ts";
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

type TaskRecurrenceType = components["schemas"]["TaskRecurrenceType"];
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

function PropertyRow({
  label,
  icon,
  children,
}: {
  label: string;
  icon?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-start gap-4 border-b border-surface-200-800 py-3 last:border-b-0">
      <span className="text-surface-600-400 flex w-36 shrink-0 items-center gap-2 pt-1 text-sm">
        {icon}
        {label}
      </span>
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  );
}

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
          className="h3 w-full border-b-2 border-primary-500 bg-transparent outline-none"
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
        className="h3 cursor-pointer rounded px-1 -mx-1 hover:bg-surface-100-900 transition-colors"
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

  return (
    <div className="flex flex-col gap-1">
      <select
        value={value}
        aria-label="Status"
        onChange={(e) => {
          mutation.reset();
          mutation.mutate({ status: e.target.value });
        }}
        disabled={mutation.isPending}
        className="select preset-outlined-surface-200-800 w-full"
      >
        {!statuses.some((s) => s.name === value) && (
          <option value={value} disabled>
            {value} (removed)
          </option>
        )}
        {statuses.map((s) => (
          <option key={s.name} value={s.name}>
            {s.name}
          </option>
        ))}
      </select>
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </div>
  );
}

function NullableSelectField({
  spaceSlug,
  taskId,
  field,
  value,
  options,
  label,
}: {
  spaceSlug: string;
  taskId: string;
  field: "effort" | "priority";
  value: string | null;
  options: { name: string }[];
  label: string;
}) {
  const mutation = useTaskPatch(spaceSlug, taskId);

  return (
    <div className="flex flex-col gap-1">
      <select
        value={value ?? ""}
        aria-label={label}
        onChange={(e) => {
          mutation.reset();
          mutation.mutate({ [field]: e.target.value || null });
        }}
        disabled={mutation.isPending}
        className="select preset-outlined-surface-200-800 w-full"
      >
        <option value="">None</option>
        {options.map((o) => (
          <option key={o.name} value={o.name}>
            {o.name}
          </option>
        ))}
      </select>
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </div>
  );
}

function MemberMultiSelectField({
  spaceSlug,
  taskId,
  field,
  value,
  members,
  memberMap,
}: {
  spaceSlug: string;
  taskId: string;
  field: "assigneeIds" | "rotationPool";
  value: string[];
  members: SpaceMember[];
  memberMap: Map<string, SpaceMember>;
}) {
  const mutation = useTaskPatch(spaceSlug, taskId);
  const [inputValue, setInputValue] = useState("");

  const filteredMembers = useMemo(
    () =>
      inputValue
        ? members.filter(
            (m) =>
              m.userName.toLowerCase().includes(inputValue.toLowerCase()) ||
              m.userEmail.toLowerCase().includes(inputValue.toLowerCase()),
          )
        : members,
    [members, inputValue],
  );

  const collection = useListCollection({
    items: filteredMembers,
    itemToString: (m) => m.userName,
    itemToValue: (m) => m.userId,
  });

  return (
    <div className="flex flex-col gap-1">
      <Combobox
        multiple
        collection={collection}
        value={value}
        inputValue={inputValue}
        onInputValueChange={(e) => setInputValue(e.inputValue)}
        onValueChange={(e) => {
          mutation.reset();
          mutation.mutate({ [field]: e.value });
        }}
        loopFocus
        openOnClick
        disabled={mutation.isPending}
      >
        <Combobox.Control className="input-group preset-outlined-surface-200-800 grid grid-cols-[1fr_auto]">
          <Combobox.Input placeholder="Search members..." className="ig-input" />
          <Combobox.Trigger className="ig-btn preset-tonal-surface">
            <ChevronDown className="size-4" aria-hidden="true" />
          </Combobox.Trigger>
        </Combobox.Control>
        <Portal>
          <Combobox.Positioner className="z-50">
            <Combobox.Content className="card preset-outlined-surface-200-800 bg-surface-100-900 max-h-60 overflow-auto p-1">
              {filteredMembers.length === 0 ? (
                <div className="text-surface-600-400 px-3 py-2 text-sm">No members found</div>
              ) : (
                filteredMembers.map((m) => (
                  <Combobox.Item
                    key={m.userId}
                    item={m}
                    className="flex cursor-pointer items-center gap-2 rounded px-3 py-2 text-sm data-[highlighted]:bg-surface-200-800"
                  >
                    <Combobox.ItemIndicator>
                      <Check className="size-4" />
                    </Combobox.ItemIndicator>
                    <Combobox.ItemText>{m.userName}</Combobox.ItemText>
                    <span className="text-surface-500 ml-auto text-xs">{m.userEmail}</span>
                  </Combobox.Item>
                ))
              )}
            </Combobox.Content>
          </Combobox.Positioner>
        </Portal>
      </Combobox>
      {value.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {value.map((id) => (
            <span key={id} className="preset-tonal-surface rounded-base px-2 py-0.5 text-xs">
              {memberMap.get(id)?.userName ?? "Unknown member"}
            </span>
          ))}
        </div>
      )}
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </div>
  );
}

const BROWSER_TIMEZONE = Intl.DateTimeFormat().resolvedOptions().timeZone;

function safeParseDateString(value: string) {
  if (!value) return null;
  try {
    return parseDate(value);
  } catch {
    return null;
  }
}

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
  const [draftAt, setDraftAt] = useState(value?.at ?? "");
  const [draftTz, setDraftTz] = useState(value?.timezone ?? BROWSER_TIMEZONE);
  const [editing, setEditing] = useState(false);

  useEffect(() => {
    if (!editing) {
      setDraftAt(value?.at ?? "");
      setDraftTz(value?.timezone ?? BROWSER_TIMEZONE);
    }
  }, [value, editing]);

  function save(at: string, tz: string) {
    setEditing(false);
    if (at && tz) {
      if (at !== (value?.at ?? "") || tz !== (value?.timezone ?? "")) {
        mutation.reset();
        mutation.mutate({ due: { at, timezone: tz } });
      }
    } else if (!at && value) {
      mutation.reset();
      mutation.mutate({ due: null });
    }
  }

  const draftDateValue = useMemo(() => safeParseDateString(draftAt), [draftAt]);

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-2">
        <DateField
          value={draftDateValue}
          onChange={(dateValue) => {
            const at = dateValue?.toString() ?? "";
            setDraftAt(at);
            save(at, draftTz);
          }}
          onOpenChange={(open) => {
            if (open) setEditing(true);
          }}
          disabled={mutation.isPending}
          aria-label="Due date"
        />
        {value && (
          <button
            type="button"
            onClick={() => {
              setDraftAt("");
              mutation.reset();
              mutation.mutate({ due: null });
            }}
            disabled={mutation.isPending}
            className="btn btn-sm preset-outlined-surface-200-800"
            aria-label="Clear due date"
            title="Clear due date"
          >
            <X className="size-4" aria-hidden="true" />
          </button>
        )}
      </div>
      <TimezoneCombobox
        value={draftTz}
        onChange={(tz) => {
          setDraftTz(tz);
          save(draftAt, tz);
        }}
        onOpenChange={(open) => {
          if (open) setEditing(true);
        }}
        disabled={mutation.isPending}
        aria-label="Due date timezone"
      />
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </div>
  );
}

function RecurrenceField({
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

  return (
    <div className="flex flex-col gap-1">
      <RecurrenceRuleEditor
        recurrenceType={recurrenceType}
        recurrenceRule={recurrenceRule}
        onSave={(update) => {
          mutation.reset();
          mutation.mutate(update);
        }}
        disabled={mutation.isPending}
      />
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </div>
  );
}

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
        onSave={(value) => {
          mutation.reset();
          mutation.mutate({ overdueActionRule: value });
        }}
        disabled={mutation.isPending}
      />
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </div>
  );
}

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

function DeleteTaskSection({ spaceSlug, taskId }: { spaceSlug: string; taskId: string }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
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

  function handleOpenChange(details: { open: boolean }) {
    setOpen(details.open);
    if (!details.open) deleteMutation.reset();
  }

  return (
    <div className="mt-8 border-t border-surface-200-800 pt-6">
      <Dialog
        open={open}
        onOpenChange={handleOpenChange}
        role="alertdialog"
        initialFocusEl={() => cancelRef.current}
      >
        <Dialog.Trigger className="btn preset-filled-error-500 flex items-center gap-2">
          <Trash2 className="size-4" aria-hidden="true" />
          Delete task
        </Dialog.Trigger>
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
    </div>
  );
}

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
    <div className="mx-auto max-w-3xl p-6">
      <BackLink
        to="/spaces/$spaceSlug"
        params={{ spaceSlug }}
        className="text-surface-600-400 hover:text-surface-950-50 mb-4 inline-flex items-center gap-1 text-sm transition-colors"
      >
        <ArrowLeft className="size-4" aria-hidden="true" />
        Back to {space.name}
      </BackLink>

      <div className="flex items-start gap-3">
        <span className="text-surface-500 mt-1 shrink-0 font-mono text-sm">{task.id}</span>
        <div className="min-w-0 flex-1">
          <EditableTitle spaceSlug={spaceSlug} taskId={taskId} value={task.title} />
        </div>
      </div>

      <TaskDescriptionEditor spaceSlug={spaceSlug} taskId={taskId} value={task.description} />

      <div className="card preset-outlined-surface-200-800 mt-6 p-4">
        <PropertyRow label="Status" icon={<CircleAlert className="size-4" aria-hidden="true" />}>
          <StatusField
            spaceSlug={spaceSlug}
            taskId={taskId}
            value={task.status}
            statuses={statuses}
          />
        </PropertyRow>

        <PropertyRow label="Effort" icon={<Gauge className="size-4" aria-hidden="true" />}>
          <NullableSelectField
            spaceSlug={spaceSlug}
            taskId={taskId}
            field="effort"
            value={task.effort}
            options={effortLevels}
            label="Effort"
          />
        </PropertyRow>

        <PropertyRow label="Priority" icon={<SignalHigh className="size-4" aria-hidden="true" />}>
          <NullableSelectField
            spaceSlug={spaceSlug}
            taskId={taskId}
            field="priority"
            value={task.priority}
            options={priorityLevels}
            label="Priority"
          />
        </PropertyRow>

        <PropertyRow label="Assignees" icon={<Users className="size-4" aria-hidden="true" />}>
          <MemberMultiSelectField
            spaceSlug={spaceSlug}
            taskId={taskId}
            field="assigneeIds"
            value={task.assigneeIds}
            members={members}
            memberMap={memberMap}
          />
        </PropertyRow>

        <PropertyRow label="Due date" icon={<Calendar className="size-4" aria-hidden="true" />}>
          <DueDateField spaceSlug={spaceSlug} taskId={taskId} value={task.due} />
        </PropertyRow>

        <PropertyRow label="Recurrence" icon={<RefreshCw className="size-4" aria-hidden="true" />}>
          <RecurrenceField
            spaceSlug={spaceSlug}
            taskId={taskId}
            recurrenceType={task.recurrenceType}
            recurrenceRule={task.recurrenceRule}
          />
        </PropertyRow>

        {task.recurrenceType !== "one_off" && task.due !== null && (
          <PropertyRow
            label="Overdue action"
            icon={<Clock className="size-4" aria-hidden="true" />}
          >
            <OverdueActionField
              spaceSlug={spaceSlug}
              taskId={taskId}
              overdueActionRule={task.overdueActionRule}
              statuses={statuses}
            />
          </PropertyRow>
        )}

        <PropertyRow label="Tags" icon={<Tag className="size-4" aria-hidden="true" />}>
          <TagsField spaceSlug={spaceSlug} taskId={taskId} value={task.tags} />
        </PropertyRow>

        <PropertyRow label="Rotation pool" icon={<Users className="size-4" aria-hidden="true" />}>
          <MemberMultiSelectField
            spaceSlug={spaceSlug}
            taskId={taskId}
            field="rotationPool"
            value={task.rotationPool}
            members={members}
            memberMap={memberMap}
          />
        </PropertyRow>
      </div>

      <RelationsSection spaceSlug={spaceSlug} taskId={task.id} relations={task.relations} />

      <div className="mt-8">
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

      <DeleteTaskSection spaceSlug={spaceSlug} taskId={taskId} />
    </div>
  );
}

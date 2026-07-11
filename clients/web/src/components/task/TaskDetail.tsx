import {
  useMutation,
  useQueryClient,
  useSuspenseInfiniteQuery,
  useSuspenseQuery,
} from "@tanstack/react-query";
import {
  Activity,
  Check,
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
import { type ReactNode, Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { apiClient } from "../../api/client.ts";
import type { components } from "../../api/schema.d.ts";
import { useSpaceMemberMap } from "../../lib/hooks.ts";
import { getIcon } from "../../lib/level-icons.ts";
import { invalidateUserTaskLists, useTaskPatch } from "../../lib/mutations.ts";
import {
  spaceEffortLevelsQueryOptions,
  spaceMembersQueryOptions,
  spacePriorityLevelsQueryOptions,
  spaceTaskQueryOptions,
  spaceTaskStatusesQueryOptions,
  taskActivityInfiniteQueryOptions,
} from "../../lib/queries.ts";
import { notifyStaleData } from "../../lib/toaster.ts";
import { useMenuSearch } from "../../lib/useMenuSearch.ts";
import { toast } from "sonner";
import {
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogRoot,
} from "../../ui/AlertDialog.tsx";
import {
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuRoot,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../../ui/DropdownMenu.tsx";
import { TagsInput } from "../../ui/TagsInput.tsx";
import { ActivityFeed } from "../ActivityFeed.tsx";
import { DetailPaneHeader, DETAIL_PANE_TITLE_CLASS } from "../DetailPaneHeader.tsx";
import { FieldPill } from "../FieldPill.tsx";
import { SearchableMenuContent } from "../SearchableMenuContent.tsx";
import { TaskDescriptionEditor } from "../TaskDescriptionEditor.tsx";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";
import { DueDateMenuField } from "./DueDateMenuField.tsx";
import { RecurrenceMenuField } from "./RecurrenceMenuField.tsx";
import { TaskRelationChipRow, TaskRelationMenuField } from "./TaskRelationMenuField.tsx";

type TaskStatus = components["schemas"]["TaskStatus"];
type SpaceMember = components["schemas"]["SpaceMember"];

// ─── Action Bar ─────────────────────────────────────────────────────────────

function TaskActions({
  spaceSlug,
  taskId,
  onDeleteSuccess,
}: {
  spaceSlug: string;
  taskId: string;
  onDeleteSuccess: () => void;
}) {
  const queryClient = useQueryClient();
  const [deleteOpen, setDeleteOpen] = useState(false);

  const deleteMutation = useMutation({
    mutationFn: async () => {
      const { error } = await apiClient.DELETE("/spaces/{spaceSlug}/tasks/{taskId}", {
        params: { path: { spaceSlug, taskId } },
      });
      if (error) throw new Error(error.message ?? "Failed to delete task");
    },
    onSuccess: async () => {
      queryClient.removeQueries({
        queryKey: ["spaces", spaceSlug, "tasks", taskId],
      });
      try {
        await Promise.all([
          queryClient.invalidateQueries({
            queryKey: ["spaces", spaceSlug, "tasks", "list"],
          }),
          invalidateUserTaskLists(queryClient),
        ]);
      } catch (err) {
        console.error("Cache invalidation failed after mutation:", err);
        notifyStaleData();
      }
      onDeleteSuccess();
    },
  });

  function handleCopyId() {
    void navigator.clipboard.writeText(taskId).catch(() => toast.error("Failed to copy task ID"));
  }

  return (
    <>
      <DropdownMenuRoot>
        <DropdownMenuTrigger className="btn btn-soft btn-square btn-sm" aria-label="Task actions">
          <Ellipsis className="size-3.5" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem onSelect={handleCopyId}>
            <Copy className="size-4" aria-hidden="true" />
            <span>Copy task ID</span>
          </DropdownMenuItem>
          <DropdownMenuItem
            onSelect={() => {
              void navigator.clipboard
                .writeText(window.location.href)
                .catch(() => toast.error("Failed to copy URL"));
            }}
          >
            <Copy className="size-4" aria-hidden="true" />
            <span>Copy URL</span>
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem className="text-error" onSelect={() => setDeleteOpen(true)}>
            <Trash2 className="size-4" aria-hidden="true" />
            <span>Delete task</span>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenuRoot>

      <AlertDialogRoot
        open={deleteOpen}
        onOpenChange={(next) => {
          setDeleteOpen(next);
          if (!next) deleteMutation.reset();
        }}
      >
        <AlertDialogContent className="max-w-md space-y-4">
          <AlertDialogHeader title="Delete task" />
          <AlertDialogDescription>
            Are you sure you want to delete this task? This action cannot be undone.
          </AlertDialogDescription>
          <div role="alert" aria-live="assertive">
            {deleteMutation.error && <ErrorAlert message={deleteMutation.error.message} />}
          </div>
          <AlertDialogFooter>
            <AlertDialogAction asChild>
              <button
                type="button"
                onClick={(e) => {
                  e.preventDefault();
                  deleteMutation.mutate();
                }}
                disabled={deleteMutation.isPending}
                className="btn btn-error flex-1"
              >
                {deleteMutation.isPending ? "Deleting..." : "Delete"}
              </button>
            </AlertDialogAction>
            <AlertDialogCancel className="btn btn-soft">Cancel</AlertDialogCancel>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialogRoot>
    </>
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
          className={`w-full border-b-2 border-primary bg-transparent outline-none ${DETAIL_PANE_TITLE_CLASS}`}
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
        className={`w-fit max-w-full cursor-pointer rounded-field transition-colors hover:bg-base-200 ${DETAIL_PANE_TITLE_CLASS}`}
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

  const selectedStatus = useMemo(() => statuses.find((s) => s.name === value), [statuses, value]);
  let statusIcon: ReactNode = <CircleAlert className="size-3.5" aria-hidden="true" />;
  if (selectedStatus?.icon) {
    const Icon = getIcon(selectedStatus.icon);
    statusIcon = <Icon className="size-3.5" aria-hidden="true" />;
  }

  return (
    <>
      <DropdownMenuRoot {...search.menuProps}>
        <FieldPill icon={statusIcon} label="Status" value={value} />
        <SearchableMenuContent
          search={search}
          placeholder="Search statuses..."
          inputLabel="Search statuses"
        >
          {filtered.length === 0 ? (
            <div className="px-3 py-2 text-sm text-base-content/60">No matching statuses</div>
          ) : (
            <DropdownMenuRadioGroup
              value={value}
              onValueChange={(v) => {
                if (v) mutation.mutate({ status: v });
              }}
            >
              {filtered.map((status) => {
                const StatusIcon = status.icon ? getIcon(status.icon) : null;
                return (
                  <DropdownMenuRadioItem key={status.name} value={status.name}>
                    {StatusIcon && <StatusIcon className="size-4" aria-hidden="true" />}
                    <span>{status.name}</span>
                    <span className="ml-auto text-xs text-base-content/60">{status.category}</span>
                  </DropdownMenuRadioItem>
                );
              })}
            </DropdownMenuRadioGroup>
          )}
        </SearchableMenuContent>
      </DropdownMenuRoot>
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
  defaultIcon,
}: {
  spaceSlug: string;
  taskId: string;
  field: "effort" | "priority";
  value: string | null;
  options: { name: string; icon: string }[];
  label: string;
  defaultIcon: ReactNode;
}) {
  const mutation = useTaskPatch(spaceSlug, taskId);
  const search = useMenuSearch();

  const filtered = useMemo(
    () => options.filter((o) => o.name.toLowerCase().includes(search.query.toLowerCase())),
    [options, search.query],
  );

  const selectedOption = useMemo(
    () => (value ? options.find((o) => o.name === value) : undefined),
    [options, value],
  );
  let pillIcon: ReactNode = defaultIcon;
  if (selectedOption?.icon) {
    const Icon = getIcon(selectedOption.icon);
    pillIcon = <Icon className="size-3.5" aria-hidden="true" />;
  }

  return (
    <>
      <DropdownMenuRoot {...search.menuProps}>
        <FieldPill icon={pillIcon} label={label} value={value} />
        <SearchableMenuContent
          search={search}
          placeholder={`Search ${label.toLowerCase()}...`}
          inputLabel={`Search ${label.toLowerCase()}`}
        >
          {value !== null && (
            <DropdownMenuItem
              className="text-error"
              onSelect={() => {
                mutation.mutate({ [field]: null });
              }}
            >
              <X className="size-4" aria-hidden="true" />
              <span>None</span>
            </DropdownMenuItem>
          )}
          {filtered.length === 0 ? (
            <div className="px-3 py-2 text-sm text-base-content/60">No matching options</div>
          ) : (
            <DropdownMenuRadioGroup
              value={value ?? ""}
              onValueChange={(v) => {
                if (v) mutation.mutate({ [field]: v });
              }}
            >
              {filtered.map((option) => {
                const OptionIcon = option.icon ? getIcon(option.icon) : null;
                return (
                  <DropdownMenuRadioItem key={option.name} value={option.name}>
                    {OptionIcon && <OptionIcon className="size-4" aria-hidden="true" />}
                    <span>{option.name}</span>
                  </DropdownMenuRadioItem>
                );
              })}
            </DropdownMenuRadioGroup>
          )}
        </SearchableMenuContent>
      </DropdownMenuRoot>
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
  icon: ReactNode;
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

  const handleToggle = useCallback((memberId: string) => {
    setDraft((prev) =>
      prev.includes(memberId) ? prev.filter((id) => id !== memberId) : [...prev, memberId],
    );
  }, []);

  return (
    <>
      <DropdownMenuRoot
        onOpenChange={(open) => {
          search.menuProps.onOpenChange(open);
          const changed = draft.length !== value.length || !draft.every((id) => value.includes(id));
          if (!open && changed) {
            mutation.mutate({ [field]: draft });
          }
        }}
      >
        <FieldPill icon={icon} label={label} value={displayValue} />
        <SearchableMenuContent
          search={search}
          placeholder="Search members..."
          inputLabel="Search members"
        >
          {filtered.length === 0 ? (
            <div className="px-3 py-2 text-sm text-base-content/60">No matching members</div>
          ) : (
            filtered.map((member) => {
              const isChecked = draft.includes(member.userId);
              return (
                <DropdownMenuItem
                  key={member.userId}
                  className="pl-7 relative"
                  onSelect={(e) => {
                    e.preventDefault();
                    handleToggle(member.userId);
                  }}
                >
                  <span className="absolute left-2 flex h-3.5 w-3.5 items-center justify-center">
                    {isChecked && <Check className="size-3.5" aria-hidden="true" />}
                  </span>
                  <span>{member.userName}</span>
                  <span className="ml-auto text-xs text-base-content/60">{member.userEmail}</span>
                </DropdownMenuItem>
              );
            })
          )}
        </SearchableMenuContent>
      </DropdownMenuRoot>
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </>
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
        label="Tags"
        value={value}
        onValueChange={(v) => {
          mutation.mutate({ tags: v });
        }}
        placeholder={value.length === 0 ? "Add tags..." : ""}
        disabled={mutation.isPending}
      />
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
      variant="task"
    />
  );
}

// ─── Task Detail View ───────────────────────────────────────────────────────

export function TaskDetailView({
  spaceSlug,
  taskId,
  backLink,
  breadcrumb,
  onDeleteSuccess,
}: {
  spaceSlug: string;
  taskId: string;
  backLink: ReactNode;
  breadcrumb: ReactNode;
  onDeleteSuccess: () => void;
}) {
  const { data: task } = useSuspenseQuery(spaceTaskQueryOptions(spaceSlug, taskId));
  const { data: members } = useSuspenseQuery(spaceMembersQueryOptions(spaceSlug));
  const { data: statuses } = useSuspenseQuery(spaceTaskStatusesQueryOptions(spaceSlug));
  const { data: effortLevels } = useSuspenseQuery(spaceEffortLevelsQueryOptions(spaceSlug));
  const { data: priorityLevels } = useSuspenseQuery(spacePriorityLevelsQueryOptions(spaceSlug));
  const memberMap = useSpaceMemberMap(spaceSlug);

  return (
    <div className="space-y-4">
      <DetailPaneHeader
        backLink={backLink}
        breadcrumb={breadcrumb}
        actions={
          <TaskActions spaceSlug={spaceSlug} taskId={task.id} onDeleteSuccess={onDeleteSuccess} />
        }
        title={<EditableTitle spaceSlug={spaceSlug} taskId={taskId} value={task.title} />}
      />

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
          defaultIcon={<SignalHigh className="size-3.5" aria-hidden="true" />}
        />
        <NullableMenuField
          spaceSlug={spaceSlug}
          taskId={taskId}
          field="effort"
          value={task.effort}
          options={effortLevels}
          label="Effort"
          defaultIcon={<Gauge className="size-3.5" aria-hidden="true" />}
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
        <DueDateMenuField
          spaceSlug={spaceSlug}
          taskId={taskId}
          due={task.due}
          overdueActionRule={task.overdueActionRule}
          recurrenceType={task.recurrenceType}
          statuses={statuses}
        />
        <RecurrenceMenuField
          spaceSlug={spaceSlug}
          taskId={taskId}
          recurrenceType={task.recurrenceType}
          recurrenceRule={task.recurrenceRule}
        />
        <TaskRelationMenuField spaceSlug={spaceSlug} taskId={task.id} relations={task.relations} />
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

      <TaskRelationChipRow spaceSlug={spaceSlug} taskId={task.id} relations={task.relations} />

      <TagsField spaceSlug={spaceSlug} taskId={taskId} value={task.tags} />

      <TaskDescriptionEditor spaceSlug={spaceSlug} taskId={taskId} value={task.description} />

      <div>
        <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold">
          <Activity className="size-4" aria-hidden="true" />
          Activity
        </h2>
        <Suspense
          fallback={
            <div className="py-6 text-center text-sm text-base-content/60">Loading activity…</div>
          }
        >
          <TaskActivityFeed spaceSlug={spaceSlug} taskId={taskId} />
        </Suspense>
      </div>
    </div>
  );
}

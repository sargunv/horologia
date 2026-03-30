import { Dialog, Portal } from "@skeletonlabs/skeleton-react";
import {
  useMutation,
  useQueryClient,
  useSuspenseInfiniteQuery,
  useSuspenseQuery,
} from "@tanstack/react-query";
import { createFileRoute, createLink, useNavigate } from "@tanstack/react-router";
import {
  Calendar,
  ChevronDown,
  CircleAlert,
  Gauge,
  ListChecks,
  Plus,
  Settings,
  SignalHigh,
  Tag,
  Users,
} from "lucide-react";
import { type FormEvent, useMemo, useState } from "react";
import { apiClient } from "../../../../api/client.ts";
import type { components } from "../../../../api/schema.d.ts";
import { useSpaceMemberMap } from "../../../../lib/hooks.ts";
import {
  spaceQueryOptions,
  spaceTaskStatusesQueryOptions,
  spaceTasksInfiniteQueryOptions,
} from "../../../../lib/queries.ts";

type Task = components["schemas"]["Task"];
type TaskStatus = components["schemas"]["TaskStatus"];
type TaskStatusCategory = components["schemas"]["TaskStatusCategory"];
type SpaceMember = components["schemas"]["SpaceMember"];

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug/")({
  loader: ({ context: { queryClient }, params: { spaceSlug } }) =>
    queryClient.ensureInfiniteQueryData(spaceTasksInfiniteQueryOptions(spaceSlug)),
  component: SpacePage,
});

const SettingsLink = createLink("a");
const TaskLink = createLink("a");

const statusCategoryPreset: Record<TaskStatusCategory, string> = {
  initial: "preset-tonal-surface",
  intermediate: "preset-tonal-warning",
  completion: "preset-tonal-success",
};

function StatusBadge({
  status,
  statusMap,
}: {
  status: string;
  statusMap: Map<string, TaskStatus>;
}) {
  const taskStatus = statusMap.get(status);
  const preset = taskStatus ? statusCategoryPreset[taskStatus.category] : "preset-tonal-surface";
  return (
    <span className={`rounded-base px-2 py-0.5 text-xs font-medium whitespace-nowrap ${preset}`}>
      {status}
    </span>
  );
}

function TaskRow({
  task,
  spaceSlug,
  statusMap,
  memberMap,
}: {
  task: Task;
  spaceSlug: string;
  statusMap: Map<string, TaskStatus>;
  memberMap: Map<string, SpaceMember>;
}) {
  const assigneeNames = task.assigneeIds.map((id) => memberMap.get(id)?.userName ?? id).join(", ");

  return (
    <TaskLink
      to="/spaces/$spaceSlug/tasks/$taskId"
      params={{ spaceSlug, taskId: task.id }}
      className="group flex items-center gap-4 border-b border-surface-200-800 px-4 py-3 transition-colors last:border-b-0 hover:bg-surface-100-900"
    >
      <div className="flex min-w-0 flex-1 items-center gap-3">
        <span className="text-surface-500 shrink-0 font-mono text-xs">{task.id}</span>
        <span className="truncate font-medium">{task.title}</span>
      </div>

      <div className="flex shrink-0 items-center gap-3">
        <StatusBadge status={task.status} statusMap={statusMap} />

        {task.assigneeIds.length > 0 && (
          <span
            className="text-surface-600-400 flex items-center gap-1 text-xs"
            title={assigneeNames}
          >
            <Users className="size-3.5" />
            <span className="max-w-24 truncate">{assigneeNames}</span>
          </span>
        )}

        {task.due && (
          <span className="text-surface-600-400 flex items-center gap-1 text-xs whitespace-nowrap">
            <Calendar className="size-3.5" />
            {task.due.at}
          </span>
        )}

        {task.effort && (
          <span className="text-surface-600-400 flex items-center gap-1 text-xs whitespace-nowrap">
            <Gauge className="size-3.5" />
            {task.effort}
          </span>
        )}

        {task.priority && (
          <span className="text-surface-600-400 flex items-center gap-1 text-xs whitespace-nowrap">
            <SignalHigh className="size-3.5" />
            {task.priority}
          </span>
        )}

        {task.tags.length > 0 && (
          <span
            className="text-surface-600-400 flex items-center gap-1 text-xs"
            title={task.tags.join(", ")}
          >
            <Tag className="size-3.5" />
            <span className="max-w-20 truncate">{task.tags.join(", ")}</span>
          </span>
        )}
      </div>
    </TaskLink>
  );
}

function CreateTaskDialog({ spaceSlug }: { spaceSlug: string }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState("");

  const createMutation = useMutation({
    mutationFn: async (body: { title: string }) => {
      const { data, error } = await apiClient.POST("/spaces/{spaceSlug}/tasks", {
        params: { path: { spaceSlug } },
        body,
      });
      if (error) throw new Error(error.message ?? "Failed to create task");
      return data;
    },
    onSuccess: async (data) => {
      setOpen(false);
      setTitle("");
      try {
        await queryClient.invalidateQueries({
          queryKey: ["spaces", spaceSlug, "tasks", "list"],
        });
      } catch (err) {
        console.error("Failed to refresh after task creation:", err);
      }
      await navigate({
        to: "/spaces/$spaceSlug/tasks/$taskId",
        params: { spaceSlug, taskId: data.id },
      });
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmed = title.trim();
    if (!trimmed || createMutation.isPending) return;
    createMutation.mutate({ title: trimmed });
  }

  function handleOpenChange(details: { open: boolean }) {
    setOpen(details.open);
    if (!details.open) {
      setTitle("");
      createMutation.reset();
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <Dialog.Trigger className="btn preset-filled-primary-500 flex items-center gap-2">
        <Plus className="size-4" />
        Create task
      </Dialog.Trigger>
      <Portal>
        <Dialog.Backdrop className="fixed inset-0 z-50 bg-surface-50-950/50" />
        <Dialog.Positioner className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <Dialog.Content className="card bg-surface-100-900 w-full max-w-md space-y-4 p-6">
            <Dialog.Title className="h4">Create task</Dialog.Title>
            <Dialog.Description className="text-surface-600-400 text-sm">
              Add a new task to this space.
            </Dialog.Description>
            <form onSubmit={handleSubmit} className="flex flex-col gap-4">
              <label className="flex flex-col gap-1">
                <span className="text-surface-600-400 text-sm font-medium">Title</span>
                <input
                  type="text"
                  required
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  className="input preset-outlined-surface-200-800 w-full"
                  placeholder="What needs to be done?"
                  maxLength={500}
                  disabled={createMutation.isPending}
                />
              </label>

              <div role="alert" aria-live="assertive">
                {createMutation.error && (
                  <div className="preset-filled-error-500 flex items-center gap-2 rounded-base px-3 py-2 text-sm">
                    <CircleAlert className="size-4 shrink-0" aria-hidden="true" />
                    {createMutation.error.message}
                  </div>
                )}
              </div>

              <div className="flex gap-3">
                <button
                  type="submit"
                  disabled={createMutation.isPending || !title.trim()}
                  className="btn preset-filled-primary-500 flex-1"
                >
                  {createMutation.isPending ? "Creating..." : "Create task"}
                </button>
                <Dialog.CloseTrigger className="btn preset-outlined-surface-200-800">
                  Cancel
                </Dialog.CloseTrigger>
              </div>
            </form>
          </Dialog.Content>
        </Dialog.Positioner>
      </Portal>
    </Dialog>
  );
}

function SpacePage() {
  const { spaceSlug } = Route.useParams();
  const { data: space } = useSuspenseQuery(spaceQueryOptions(spaceSlug));
  const { data: statuses } = useSuspenseQuery(spaceTaskStatusesQueryOptions(spaceSlug));
  const memberMap = useSpaceMemberMap(spaceSlug);

  const statusMap = useMemo(() => new Map(statuses.map((s) => [s.name, s])), [statuses]);

  const {
    data: taskPages,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    error,
    isError,
  } = useSuspenseInfiniteQuery(spaceTasksInfiniteQueryOptions(spaceSlug));

  const tasks = useMemo(() => taskPages.pages.flatMap((p) => p.items), [taskPages]);

  return (
    <div className="p-6">
      <div className="flex items-center justify-between">
        <h1 className="h3">{space.name}</h1>
        <div className="flex items-center gap-2">
          <CreateTaskDialog spaceSlug={spaceSlug} />
          <SettingsLink
            to="/spaces/$spaceSlug/settings"
            params={{ spaceSlug }}
            className="btn preset-outlined-surface-200-800 flex items-center gap-2"
          >
            <Settings className="size-4" />
            Settings
          </SettingsLink>
        </div>
      </div>

      <div className="mt-6">
        {tasks.length > 0 ? (
          <div className="card preset-outlined-surface-200-800 divide-surface-200-800 overflow-hidden">
            {tasks.map((task) => (
              <TaskRow
                key={task.id}
                task={task}
                spaceSlug={spaceSlug}
                statusMap={statusMap}
                memberMap={memberMap}
              />
            ))}
          </div>
        ) : (
          <div className="card preset-outlined-surface-200-800 flex flex-col items-center gap-3 p-12 text-center">
            <ListChecks className="text-surface-400 size-12" />
            <div>
              <p className="font-medium">No tasks yet</p>
              <p className="text-surface-600-400 mt-1 text-sm">
                Tasks in this space will appear here.
              </p>
            </div>
          </div>
        )}

        {isError && (
          <p className="text-error-500 mt-4 text-center text-sm">
            Failed to load more tasks: {error?.message ?? "Unknown error"}
          </p>
        )}

        {hasNextPage && (
          <div className="mt-4 flex justify-center">
            <button
              className="btn preset-outlined-surface-200-800 flex items-center gap-2"
              onClick={() => fetchNextPage()}
              disabled={isFetchingNextPage}
            >
              {isFetchingNextPage ? (
                "Loading..."
              ) : (
                <>
                  <ChevronDown className="size-4" />
                  Load more
                </>
              )}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

import { Dialog, Portal } from "@skeletonlabs/skeleton-react";
import { toaster } from "../../../../lib/toaster.ts";
import {
  useMutation,
  useQueryClient,
  useSuspenseInfiniteQuery,
  useSuspenseQuery,
} from "@tanstack/react-query";
import { createFileRoute, createLink, useNavigate } from "@tanstack/react-router";
import { ChevronDown, CircleAlert, ListChecks, Plus, Settings } from "lucide-react";
import { type FormEvent, useMemo, useState } from "react";
import { apiClient } from "../../../../api/client.ts";
import { TaskRow } from "../../../../components/task/TaskRow.tsx";
import {
  spaceQueryOptions,
  spaceTaskStatusesQueryOptions,
  spaceTasksInfiniteQueryOptions,
} from "../../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug/")({
  loader: ({ context: { queryClient }, params: { spaceSlug } }) =>
    queryClient.ensureInfiniteQueryData(spaceTasksInfiniteQueryOptions(spaceSlug)),
  component: SpacePage,
});

const SettingsLink = createLink("a");

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
      } catch {
        toaster.warning({
          title: "Data may be out of date",
          description: "Refresh the page to see the latest changes.",
        });
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
              <TaskRow key={task.id} task={task} spaceSlug={spaceSlug} statusMap={statusMap} />
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

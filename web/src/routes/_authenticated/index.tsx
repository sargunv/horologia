import { useQueries, useSuspenseInfiniteQuery, useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { ChevronDown, ListChecks } from "lucide-react";
import { useMemo } from "react";
import type { components } from "../../api/schema.d.ts";
import { TaskRow } from "../../components/task/TaskRow.tsx";
import {
  currentUserQueryOptions,
  spaceMembersQueryOptions,
  spacesQueryOptions,
  spaceTaskStatusesQueryOptions,
  userTasksInfiniteQueryOptions,
} from "../../lib/queries.ts";

type TaskStatus = components["schemas"]["TaskStatus"];
type Space = components["schemas"]["Space"];

// ── Route ──────────────────────────────────────────────────────────────────

export const Route = createFileRoute("/_authenticated/")({
  async loader({ context: { queryClient } }) {
    const [user, spaces] = await Promise.all([
      queryClient.ensureQueryData(currentUserQueryOptions),
      queryClient.ensureQueryData(spacesQueryOptions),
    ]);
    await Promise.all([
      queryClient.ensureInfiniteQueryData(userTasksInfiniteQueryOptions(user.id)),
      ...spaces.map((s: Space) =>
        queryClient.ensureQueryData(spaceTaskStatusesQueryOptions(s.slug)),
      ),
      ...spaces.map((s: Space) => queryClient.ensureQueryData(spaceMembersQueryOptions(s.slug))),
    ]);
  },
  component: MyTasksPage,
});

// ── Cross-space status map ─────────────────────────────────────────────────

function useAllSpaceStatusMaps(spaces: Space[]): Map<string, Map<string, TaskStatus>> {
  return useQueries({
    queries: spaces.map((s) => spaceTaskStatusesQueryOptions(s.slug)),
    combine(results) {
      const map = new Map<string, Map<string, TaskStatus>>();
      spaces.forEach((space, i) => {
        const statuses = results[i]?.data ?? [];
        map.set(space.slug, new Map(statuses.map((s) => [s.name, s])));
      });
      return map;
    },
  });
}

// ── Page ───────────────────────────────────────────────────────────────────

function MyTasksPage() {
  const { data: user } = useSuspenseQuery(currentUserQueryOptions);
  const { data: spaces } = useSuspenseQuery(spacesQueryOptions);

  const {
    data: taskPages,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
  } = useSuspenseInfiniteQuery(userTasksInfiniteQueryOptions(user.id));

  const tasks = useMemo(() => taskPages.pages.flatMap((p) => p.items), [taskPages]);

  const allStatusMaps = useAllSpaceStatusMaps(spaces);

  return (
    <div className="p-6">
      <h1 className="h3">My Tasks</h1>
      <p className="text-surface-600-400 mt-1">All tasks assigned to you across spaces.</p>

      <div className="mt-6">
        {tasks.length > 0 ? (
          <div className="card preset-outlined-surface-200-800 divide-surface-200-800 overflow-hidden">
            {tasks.map((task) => (
              <TaskRow
                key={`${task.spaceSlug}/${task.id}`}
                task={task}
                spaceSlug={task.spaceSlug}
                statusMap={allStatusMaps.get(task.spaceSlug) ?? new Map()}
                spaceLabel={spaces.find((s) => s.slug === task.spaceSlug)?.name ?? task.spaceSlug}
              />
            ))}
          </div>
        ) : (
          <div className="card preset-outlined-surface-200-800 flex flex-col items-center gap-3 p-12 text-center">
            <ListChecks className="text-surface-400 size-12" />
            <div>
              <p className="font-medium">No tasks assigned to you</p>
              <p className="text-surface-600-400 mt-1 text-sm">
                Tasks assigned to you across all spaces will appear here.
              </p>
            </div>
          </div>
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
                  Load more <ChevronDown className="size-4" />
                </>
              )}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

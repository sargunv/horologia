import {
  useQueries,
  useSuspenseInfiniteQuery,
  useSuspenseQuery,
} from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { ChevronDown, ListChecks } from "lucide-react";
import { Suspense, useMemo } from "react";
import type { components } from "../../api/schema.d.ts";
import { TaskRow } from "../../components/task/TaskRow.tsx";
import { useSpaceMemberMap } from "../../lib/hooks.ts";
import {
  currentUserQueryOptions,
  spaceMembersQueryOptions,
  spacesQueryOptions,
  spaceTaskStatusesQueryOptions,
  userTasksInfiniteQueryOptions,
} from "../../lib/queries.ts";

type Task = components["schemas"]["Task"];
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
        queryClient.ensureQueryData(spaceTaskStatusesQueryOptions(s.slug))
      ),
      ...spaces.map((s: Space) =>
        queryClient.ensureQueryData(spaceMembersQueryOptions(s.slug))
      ),
    ]);
  },
  component: MyTasksPage,
});

// ── Cross-space status map ─────────────────────────────────────────────────

function useAllSpaceStatusMaps(
  spaces: Space[]
): Map<string, Map<string, TaskStatus>> {
  return useQueries({
    queries: spaces.map((s) => spaceTaskStatusesQueryOptions(s.slug)),
    combine(results) {
      const map = new Map<string, Map<string, TaskStatus>>();
      spaces.forEach((space, i) => {
        const data = results[i]?.data;
        if (data) {
          map.set(space.slug, new Map(data.map((s) => [s.name, s])));
        }
      });
      return map;
    },
  });
}

// ── TaskRowWithMembers ─────────────────────────────────────────────────────
// Wrapper so useSpaceMemberMap is called once per rendered row (correct hook usage)

function TaskRowWithMembers({
  task,
  statusMap,
  spaceLabel,
}: {
  task: Task;
  statusMap: Map<string, TaskStatus>;
  spaceLabel: string;
}) {
  const memberMap = useSpaceMemberMap(task.spaceSlug);
  return (
    <TaskRow
      task={task}
      spaceSlug={task.spaceSlug}
      statusMap={statusMap}
      memberMap={memberMap}
      spaceLabel={spaceLabel}
    />
  );
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

  const tasks = useMemo(
    () => taskPages.pages.flatMap((p) => p.items),
    [taskPages]
  );

  const allStatusMaps = useAllSpaceStatusMaps(spaces);

  const spaceNameMap = useMemo(
    () => new Map(spaces.map((s) => [s.slug, s.name])),
    [spaces]
  );

  return (
    <div className="p-6">
      <h1 className="h3">My Tasks</h1>
      <p className="text-surface-600-400 mt-1">
        All tasks assigned to you across spaces.
      </p>

      <div className="mt-6">
        {tasks.length > 0 ? (
          <div className="card preset-outlined-surface-200-800 divide-surface-200-800 overflow-hidden">
            <Suspense>
              {tasks.map((task) => (
                <TaskRowWithMembers
                  key={`${task.spaceSlug}/${task.id}`}
                  task={task}
                  statusMap={allStatusMaps.get(task.spaceSlug) ?? new Map()}
                  spaceLabel={spaceNameMap.get(task.spaceSlug) ?? task.spaceSlug}
                />
              ))}
            </Suspense>
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
              {isFetchingNextPage ? "Loading..." : (
                <>Load more <ChevronDown className="size-4" /></>
              )}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

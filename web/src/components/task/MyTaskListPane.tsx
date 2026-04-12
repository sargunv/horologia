import { useQueries, useSuspenseInfiniteQuery, useSuspenseQuery } from "@tanstack/react-query";
import { createLink } from "@tanstack/react-router";
import { Activity, ChevronDown, ListChecks } from "lucide-react";
import { useMemo } from "react";
import type { components } from "../../api/schema.d.ts";
import {
  currentUserQueryOptions,
  spacesQueryOptions,
  spaceTaskStatusesQueryOptions,
  userTasksInfiniteQueryOptions,
} from "../../lib/queries.ts";
import { TaskRow } from "./TaskRow.tsx";

const ActivityLink = createLink("a");

type TaskStatus = components["schemas"]["TaskStatus"];
type Space = components["schemas"]["Space"];

const EMPTY_STATUS_MAP = new Map<string, TaskStatus>();

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

export function MyTaskListPane() {
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
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <h2 className="h5 truncate">My Tasks</h2>
        <ActivityLink
          to="/activity"
          className="btn-icon btn-sm preset-tonal-surface"
          aria-label="Activity"
        >
          <Activity className="size-4" aria-hidden="true" />
        </ActivityLink>
      </div>

      {tasks.length > 0 ? (
        <div className="card preset-outlined-surface-200-800 divide-surface-200-800 overflow-hidden">
          {tasks.map((task) => (
            <TaskRow
              key={`${task.spaceSlug}/${task.id}`}
              task={task}
              spaceSlug={task.spaceSlug}
              statusMap={allStatusMaps.get(task.spaceSlug) ?? EMPTY_STATUS_MAP}
              to="/tasks/$spaceSlug/$taskId"
              compact
            />
          ))}
        </div>
      ) : (
        <div className="card preset-outlined-surface-200-800 flex flex-col items-center gap-3 p-12 text-center">
          <ListChecks className="text-surface-400 size-12" aria-hidden="true" />
          <div>
            <p className="font-medium">No tasks assigned to you</p>
            <p className="text-surface-600-400 mt-1 text-sm">
              Tasks assigned to you across all spaces will appear here.
            </p>
          </div>
        </div>
      )}

      {hasNextPage && (
        <div className="flex justify-center">
          <button
            className="btn preset-outlined-surface-200-800 flex items-center gap-2"
            onClick={() => fetchNextPage()}
            disabled={isFetchingNextPage}
          >
            {isFetchingNextPage ? (
              "Loading\u2026"
            ) : (
              <>
                <ChevronDown className="size-4" aria-hidden="true" />
                Load more
              </>
            )}
          </button>
        </div>
      )}
    </div>
  );
}

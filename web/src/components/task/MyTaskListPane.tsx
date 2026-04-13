import { Portal, Tooltip } from "@skeletonlabs/skeleton-react";
import { useQueries, useSuspenseInfiniteQuery, useSuspenseQuery } from "@tanstack/react-query";
import { createLink } from "@tanstack/react-router";
import { Activity, ChevronDown, ListChecks } from "lucide-react";
import { useMemo } from "react";
import type { components } from "../../api/schema.d.ts";
import {
  currentUserQueryOptions,
  spaceEffortLevelsQueryOptions,
  spacePriorityLevelsQueryOptions,
  spacesQueryOptions,
  spaceTaskStatusesQueryOptions,
  userTasksInfiniteQueryOptions,
} from "../../lib/queries.ts";
import { TaskRow } from "./TaskRow.tsx";

const ActivityLink = createLink("a");

type TaskStatus = components["schemas"]["TaskStatus"];
type TaskEffortLevel = components["schemas"]["TaskEffortLevel"];
type TaskPriorityLevel = components["schemas"]["TaskPriorityLevel"];
type Space = components["schemas"]["Space"];

const EMPTY_STATUS_MAP = new Map<string, TaskStatus>();
const EMPTY_LEVELS: TaskEffortLevel[] = [];
const EMPTY_PRIORITY_LEVELS: TaskPriorityLevel[] = [];

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

function useAllSpaceEffortLevels(spaces: Space[]): Map<string, TaskEffortLevel[]> {
  return useQueries({
    queries: spaces.map((s) => spaceEffortLevelsQueryOptions(s.slug)),
    combine(results) {
      const map = new Map<string, TaskEffortLevel[]>();
      spaces.forEach((space, i) => {
        map.set(space.slug, results[i]?.data ?? []);
      });
      return map;
    },
  });
}

function useAllSpacePriorityLevels(spaces: Space[]): Map<string, TaskPriorityLevel[]> {
  return useQueries({
    queries: spaces.map((s) => spacePriorityLevelsQueryOptions(s.slug)),
    combine(results) {
      const map = new Map<string, TaskPriorityLevel[]>();
      spaces.forEach((space, i) => {
        map.set(space.slug, results[i]?.data ?? []);
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
  const allEffortLevels = useAllSpaceEffortLevels(spaces);
  const allPriorityLevels = useAllSpacePriorityLevels(spaces);

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <h2 className="h5 truncate">My Tasks</h2>
        <Tooltip>
          <Tooltip.Trigger className="btn-icon btn-sm preset-tonal-surface" aria-label="Activity">
            <ActivityLink to="/activity">
              <Activity className="size-4" aria-hidden="true" />
            </ActivityLink>
          </Tooltip.Trigger>
          <Portal>
            <Tooltip.Positioner>
              <Tooltip.Content className="preset-filled-surface-800-200 rounded px-2 py-1 text-xs shadow">
                Activity
              </Tooltip.Content>
            </Tooltip.Positioner>
          </Portal>
        </Tooltip>
      </div>

      {tasks.length > 0 ? (
        <div className="card preset-outlined-surface-200-800 divide-surface-200-800 overflow-hidden">
          {tasks.map((task) => (
            <TaskRow
              key={`${task.spaceSlug}/${task.id}`}
              task={task}
              spaceSlug={task.spaceSlug}
              statusMap={allStatusMaps.get(task.spaceSlug) ?? EMPTY_STATUS_MAP}
              effortLevels={allEffortLevels.get(task.spaceSlug) ?? EMPTY_LEVELS}
              priorityLevels={allPriorityLevels.get(task.spaceSlug) ?? EMPTY_PRIORITY_LEVELS}
              to="/tasks/$spaceSlug/$taskId"
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

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
import { TooltipContent, TooltipRoot, TooltipTrigger } from "../../ui/Tooltip.tsx";
import { TaskRow } from "./TaskRow.tsx";

const ActivityLink = createLink("a");

type TaskStatus = components["schemas"]["TaskStatus"];
type TaskEffortLevel = components["schemas"]["TaskEffortLevel"];
type TaskPriorityLevel = components["schemas"]["TaskPriorityLevel"];
type Space = components["schemas"]["Space"];

const EMPTY_STATUS_MAP = new Map<string, TaskStatus>();
const EMPTY_EFFORT_LEVELS: TaskEffortLevel[] = [];
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
      spaces.forEach((space, i) => map.set(space.slug, results[i]?.data ?? []));
      return map;
    },
  });
}

function useAllSpacePriorityLevels(spaces: Space[]): Map<string, TaskPriorityLevel[]> {
  return useQueries({
    queries: spaces.map((s) => spacePriorityLevelsQueryOptions(s.slug)),
    combine(results) {
      const map = new Map<string, TaskPriorityLevel[]>();
      spaces.forEach((space, i) => map.set(space.slug, results[i]?.data ?? []));
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
        <h2 className="truncate text-lg font-semibold">My Tasks</h2>
        <TooltipRoot>
          <TooltipTrigger asChild>
            <ActivityLink
              to="/activity"
              className="btn btn-soft btn-square btn-sm"
              aria-label="Activity"
            >
              <Activity className="size-4" aria-hidden="true" />
            </ActivityLink>
          </TooltipTrigger>
          <TooltipContent>Activity</TooltipContent>
        </TooltipRoot>
      </div>

      {tasks.length > 0 ? (
        <div className="overflow-hidden rounded-box border border-base-300 divide-y divide-base-300">
          {tasks.map((task) => (
            <TaskRow
              key={`${task.spaceSlug}/${task.id}`}
              task={task}
              spaceSlug={task.spaceSlug}
              statusMap={allStatusMaps.get(task.spaceSlug) ?? EMPTY_STATUS_MAP}
              effortLevels={allEffortLevels.get(task.spaceSlug) ?? EMPTY_EFFORT_LEVELS}
              priorityLevels={allPriorityLevels.get(task.spaceSlug) ?? EMPTY_PRIORITY_LEVELS}
              to="/tasks/$spaceSlug/$taskId"
            />
          ))}
        </div>
      ) : (
        <div className="flex flex-col items-center gap-3 rounded-box border border-base-300 p-12 text-center">
          <ListChecks className="size-12 text-base-content/40" aria-hidden="true" />
          <div>
            <p className="font-medium">No tasks assigned to you</p>
            <p className="mt-1 text-sm text-base-content/70">
              Tasks assigned to you across all spaces will appear here.
            </p>
          </div>
        </div>
      )}

      {hasNextPage && (
        <div className="flex justify-center">
          <button
            className="btn btn-soft flex items-center gap-2"
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

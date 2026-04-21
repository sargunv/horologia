import { useSuspenseInfiniteQuery, useSuspenseQuery } from "@tanstack/react-query";
import { createLink } from "@tanstack/react-router";
import { Activity, ChevronDown, ListChecks, Plus, Settings } from "lucide-react";
import { useMemo } from "react";
import {
  spaceEffortLevelsQueryOptions,
  spacePriorityLevelsQueryOptions,
  spaceQueryOptions,
  spaceTaskStatusesQueryOptions,
  spaceTasksInfiniteQueryOptions,
} from "../../lib/queries.ts";
import { TooltipContent, TooltipRoot, TooltipTrigger } from "../../ui/Tooltip.tsx";
import { TaskRow } from "./TaskRow.tsx";

const SettingsLink = createLink("a");
const ActivityLink = createLink("a");
const CreateTaskLink = createLink("a");

export function TaskListPane({ spaceSlug }: { spaceSlug: string }) {
  const { data: space } = useSuspenseQuery(spaceQueryOptions(spaceSlug));
  const { data: statuses } = useSuspenseQuery(spaceTaskStatusesQueryOptions(spaceSlug));
  const { data: effortLevels } = useSuspenseQuery(spaceEffortLevelsQueryOptions(spaceSlug));
  const { data: priorityLevels } = useSuspenseQuery(spacePriorityLevelsQueryOptions(spaceSlug));
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
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <h2 className="truncate text-lg font-semibold">{space.name}</h2>
        <div className="flex shrink-0 items-center gap-1">
          <TooltipRoot>
            <TooltipTrigger asChild>
              <ActivityLink
                to="/spaces/$spaceSlug/activity"
                params={{ spaceSlug }}
                className="btn btn-soft btn-square btn-sm"
                aria-label="Activity"
              >
                <Activity className="size-4" aria-hidden="true" />
              </ActivityLink>
            </TooltipTrigger>
            <TooltipContent>Activity</TooltipContent>
          </TooltipRoot>
          <TooltipRoot>
            <TooltipTrigger asChild>
              <SettingsLink
                to="/spaces/$spaceSlug/settings"
                params={{ spaceSlug }}
                className="btn btn-soft btn-square btn-sm"
                aria-label="Settings"
              >
                <Settings className="size-4" aria-hidden="true" />
              </SettingsLink>
            </TooltipTrigger>
            <TooltipContent>Settings</TooltipContent>
          </TooltipRoot>
        </div>
      </div>

      <CreateTaskLink
        to="/spaces/$spaceSlug/tasks/new"
        params={{ spaceSlug }}
        className="flex w-full items-center justify-center gap-2 rounded-box border-2 border-dashed border-base-300 p-3 text-sm text-base-content/60 transition-colors hover:border-base-content/40 hover:text-base-content/80"
      >
        <Plus className="size-4" aria-hidden="true" />
        Create task
      </CreateTaskLink>

      {tasks.length > 0 ? (
        <div className="overflow-hidden rounded-box border border-base-300 divide-y divide-base-300">
          {tasks.map((task) => (
            <TaskRow
              key={task.id}
              task={task}
              spaceSlug={spaceSlug}
              statusMap={statusMap}
              effortLevels={effortLevels}
              priorityLevels={priorityLevels}
            />
          ))}
        </div>
      ) : (
        <div className="flex flex-col items-center gap-3 rounded-box border border-base-300 p-12 text-center">
          <ListChecks className="size-12 text-base-content/40" aria-hidden="true" />
          <div>
            <p className="font-medium">No tasks yet</p>
            <p className="mt-1 text-sm text-base-content/70">
              Tasks in this space will appear here.
            </p>
          </div>
        </div>
      )}

      {isError && (
        <p className="text-center text-sm text-error">
          Failed to load more tasks: {error?.message ?? "Unknown error"}
        </p>
      )}

      {hasNextPage && (
        <div className="flex justify-center">
          <button
            className="btn btn-soft flex items-center gap-2"
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
  );
}

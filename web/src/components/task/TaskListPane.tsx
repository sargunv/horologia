import { Portal, Tooltip } from "@skeletonlabs/skeleton-react";
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
import { TaskRow } from "./TaskRow.tsx";

/** Pick tooltip-relevant attrs (id, data-*, aria-*) from a button-typed attrs bag for use on anchor elements. */
function tooltipAttrs(attrs: Record<string, unknown>): Record<string, unknown> {
  const result: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(attrs)) {
    if (k === "id" || k.startsWith("data-") || k.startsWith("aria-")) {
      result[k] = v;
    }
  }
  return result;
}

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
        <h2 className="h5 truncate">{space.name}</h2>
        <div className="flex shrink-0 items-center gap-1">
          <Tooltip>
            <Tooltip.Trigger
              element={(attrs) => (
                <ActivityLink
                  {...tooltipAttrs(attrs)}
                  to="/spaces/$spaceSlug/activity"
                  params={{ spaceSlug }}
                  className="btn-icon btn-sm preset-tonal-surface"
                  aria-label="Activity"
                >
                  <Activity className="size-4" aria-hidden="true" />
                </ActivityLink>
              )}
            />
            <Portal>
              <Tooltip.Positioner>
                <Tooltip.Content className="preset-filled-surface-800-200 rounded px-2 py-1 text-xs shadow">
                  Activity
                </Tooltip.Content>
              </Tooltip.Positioner>
            </Portal>
          </Tooltip>
          <Tooltip>
            <Tooltip.Trigger
              element={(attrs) => (
                <SettingsLink
                  {...tooltipAttrs(attrs)}
                  to="/spaces/$spaceSlug/settings"
                  params={{ spaceSlug }}
                  className="btn-icon btn-sm preset-tonal-surface"
                  aria-label="Settings"
                >
                  <Settings className="size-4" aria-hidden="true" />
                </SettingsLink>
              )}
            />
            <Portal>
              <Tooltip.Positioner>
                <Tooltip.Content className="preset-filled-surface-800-200 rounded px-2 py-1 text-xs shadow">
                  Settings
                </Tooltip.Content>
              </Tooltip.Positioner>
            </Portal>
          </Tooltip>
        </div>
      </div>

      <CreateTaskLink
        to="/spaces/$spaceSlug/tasks/new"
        params={{ spaceSlug }}
        className="flex w-full items-center justify-center gap-2 rounded-base border-2 border-dashed border-surface-300-700 p-3 text-sm text-surface-500 transition-colors hover:border-surface-400-600 hover:text-surface-700-300"
      >
        <Plus className="size-4" aria-hidden="true" />
        Create task
      </CreateTaskLink>

      {tasks.length > 0 ? (
        <div className="card preset-outlined-surface-200-800 divide-surface-200-800 overflow-hidden">
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
        <div className="card preset-outlined-surface-200-800 flex flex-col items-center gap-3 p-12 text-center">
          <ListChecks className="text-surface-400 size-12" aria-hidden="true" />
          <div>
            <p className="font-medium">No tasks yet</p>
            <p className="text-surface-600-400 mt-1 text-sm">
              Tasks in this space will appear here.
            </p>
          </div>
        </div>
      )}

      {isError && (
        <p className="text-error-500 text-center text-sm">
          Failed to load more tasks: {error?.message ?? "Unknown error"}
        </p>
      )}

      {hasNextPage && (
        <div className="flex justify-center">
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
  );
}

import { Link } from "@tanstack/react-router";
import { Calendar, Gauge, SignalHigh, Tag, Users } from "lucide-react";
import { useMemo } from "react";
import type { components } from "../../api/schema.d.ts";
import { useSpaceMemberMap } from "../../lib/hooks.ts";
import { getIcon } from "../../lib/level-icons.ts";
import { StatusBadge } from "./StatusBadge.tsx";

type Task = components["schemas"]["Task"];
type TaskStatus = components["schemas"]["TaskStatus"];
type TaskEffortLevel = components["schemas"]["TaskEffortLevel"];
type TaskPriorityLevel = components["schemas"]["TaskPriorityLevel"];

export function TaskRow({
  task,
  spaceSlug,
  statusMap,
  effortLevels,
  priorityLevels,
  spaceLabel,
  compact,
  to = "/spaces/$spaceSlug/tasks/$taskId",
}: {
  task: Task;
  spaceSlug: string;
  statusMap: Map<string, TaskStatus>;
  /** Effort levels for resolving per-level icons. Falls back to default icon when absent. */
  effortLevels?: TaskEffortLevel[];
  /** Priority levels for resolving per-level icons. Falls back to default icon when absent. */
  priorityLevels?: TaskPriorityLevel[];
  spaceLabel?: string;
  /** Compact mode for narrow list panes — shows only ID, title, and status badge */
  compact?: boolean;
  /** Override the link target route (default: space task detail) */
  to?: "/spaces/$spaceSlug/tasks/$taskId" | "/tasks/$spaceSlug/$taskId";
}) {
  const memberMap = useSpaceMemberMap(spaceSlug);
  const assigneeNames = task.assigneeIds.map((id) => memberMap.get(id)?.userName ?? id).join(", ");

  const EffortIcon = useMemo(() => {
    if (!task.effort || !effortLevels) return Gauge;
    const level = effortLevels.find((l) => l.name === task.effort);
    return level?.icon ? getIcon(level.icon) : Gauge;
  }, [task.effort, effortLevels]);

  const PriorityIcon = useMemo(() => {
    if (!task.priority || !priorityLevels) return SignalHigh;
    const level = priorityLevels.find((l) => l.name === task.priority);
    return level?.icon ? getIcon(level.icon) : SignalHigh;
  }, [task.priority, priorityLevels]);

  return (
    <Link
      to={to}
      params={{ spaceSlug, taskId: task.id }}
      className="group flex items-center gap-3 border-b border-surface-200-800 px-3 py-2.5 transition-colors last:border-b-0 hover:bg-surface-100-900 data-[status=active]:bg-surface-200-800"
    >
      <div className="flex min-w-0 flex-1 items-center gap-2">
        <span className="text-surface-500 shrink-0 font-mono text-xs">{task.id}</span>
        <span className="truncate text-sm font-medium">{task.title}</span>
        {spaceLabel && (
          <span className="chip preset-tonal-surface shrink-0 text-xs">{spaceLabel}</span>
        )}
      </div>

      <div className="flex shrink-0 items-center gap-2">
        <StatusBadge status={task.status} statusMap={statusMap} />

        {!compact && task.assigneeIds.length > 0 && (
          <span
            className="text-surface-600-400 flex items-center gap-1 text-xs"
            title={assigneeNames}
          >
            <Users className="size-3.5" aria-hidden="true" />
            <span className="max-w-24 truncate">{assigneeNames}</span>
          </span>
        )}

        {!compact && task.due && (
          <span className="text-surface-600-400 flex items-center gap-1 text-xs whitespace-nowrap">
            <Calendar className="size-3.5" aria-hidden="true" />
            {new Date(task.due.at).toLocaleDateString()}
          </span>
        )}

        {!compact && task.effort && (
          <span className="text-surface-600-400 flex items-center gap-1 text-xs whitespace-nowrap">
            <EffortIcon className="size-3.5" aria-hidden="true" />
            {task.effort}
          </span>
        )}

        {!compact && task.priority && (
          <span className="text-surface-600-400 flex items-center gap-1 text-xs whitespace-nowrap">
            <PriorityIcon className="size-3.5" aria-hidden="true" />
            {task.priority}
          </span>
        )}

        {!compact && task.tags.length > 0 && (
          <span
            className="text-surface-600-400 flex items-center gap-1 text-xs"
            title={task.tags.join(", ")}
          >
            <Tag className="size-3.5" aria-hidden="true" />
            <span className="max-w-20 truncate">{task.tags.join(", ")}</span>
          </span>
        )}
      </div>
    </Link>
  );
}

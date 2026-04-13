import { Avatar, Portal, Tooltip } from "@skeletonlabs/skeleton-react";
import { Link } from "@tanstack/react-router";
import { type ReactNode, useMemo } from "react";
import type { components } from "../../api/schema.d.ts";
import { useSpaceMemberMap } from "../../lib/hooks.ts";
import { getIcon } from "../../lib/level-icons.ts";

type Task = components["schemas"]["Task"];
type TaskStatus = components["schemas"]["TaskStatus"];
type TaskEffortLevel = components["schemas"]["TaskEffortLevel"];
type TaskPriorityLevel = components["schemas"]["TaskPriorityLevel"];

/** Color classes for status icon by category. */
const STATUS_ICON_COLOR: Record<string, string> = {
  initial: "text-surface-500",
  intermediate: "text-warning-500",
  completion: "text-success-500",
};

/** Extract up to two initials from a name. */
function initials(name: string): string {
  return name
    .split(/\s+/)
    .slice(0, 2)
    .map((w) => w[0])
    .join("")
    .toUpperCase();
}

/** Inline tooltip wrapper for icon-only elements in the task row. */
function IconTooltip({ label, children }: { label: string; children: ReactNode }) {
  return (
    <Tooltip openDelay={200} closeDelay={0}>
      <Tooltip.Trigger
        element={(attrs) => (
          <span {...attrs} aria-label={label}>
            {children}
          </span>
        )}
      />
      <Portal>
        <Tooltip.Positioner>
          <Tooltip.Content className="preset-filled-surface-800-200 rounded px-2 py-1 text-xs shadow">
            {label}
          </Tooltip.Content>
        </Tooltip.Positioner>
      </Portal>
    </Tooltip>
  );
}

export function TaskRow({
  task,
  spaceSlug,
  statusMap,
  effortLevels,
  priorityLevels,
  to = "/spaces/$spaceSlug/tasks/$taskId",
}: {
  task: Task;
  spaceSlug: string;
  statusMap: Map<string, TaskStatus>;
  /** Effort levels for resolving per-level icons. */
  effortLevels: TaskEffortLevel[];
  /** Priority levels for resolving per-level icons. */
  priorityLevels: TaskPriorityLevel[];
  /** Override the link target route (default: space task detail) */
  to?: "/spaces/$spaceSlug/tasks/$taskId" | "/tasks/$spaceSlug/$taskId";
}) {
  const memberMap = useSpaceMemberMap(spaceSlug);

  const status = statusMap.get(task.status);
  const StatusIcon = status?.icon ? getIcon(status.icon) : getIcon("circle");
  const statusColor = status
    ? (STATUS_ICON_COLOR[status.category] ?? "text-surface-500")
    : "text-surface-500";

  const effortLevel = task.effort ? effortLevels.find((l) => l.name === task.effort) : undefined;
  const EffortIcon = effortLevel?.icon ? getIcon(effortLevel.icon) : null;

  const priorityLevel = task.priority
    ? priorityLevels.find((l) => l.name === task.priority)
    : undefined;
  const PriorityIcon = priorityLevel?.icon ? getIcon(priorityLevel.icon) : null;

  const assignees = useMemo(
    () =>
      task.assigneeIds
        .map((id) => memberMap.get(id))
        .filter((m): m is NonNullable<typeof m> => m != null),
    [task.assigneeIds, memberMap],
  );

  return (
    <Link
      to={to}
      params={{ spaceSlug, taskId: task.id }}
      className="group flex items-center gap-2 border-b border-surface-200-800 px-3 py-2 transition-colors last:border-b-0 hover:bg-surface-100-900 data-[status=active]:bg-surface-200-800"
    >
      <IconTooltip label={task.status}>
        <StatusIcon className={`size-4 ${statusColor}`} aria-hidden="true" />
      </IconTooltip>

      <span className="min-w-0 flex-1 truncate text-sm">{task.title}</span>

      {EffortIcon && task.effort && (
        <IconTooltip label={task.effort}>
          <EffortIcon className="text-surface-500 size-3.5" aria-hidden="true" />
        </IconTooltip>
      )}

      {PriorityIcon && task.priority && (
        <IconTooltip label={task.priority}>
          <PriorityIcon className="text-surface-500 size-3.5" aria-hidden="true" />
        </IconTooltip>
      )}

      {assignees.length > 0 && (
        <div className="flex shrink-0 -space-x-1.5">
          {assignees.slice(0, 3).map((member) => (
            <IconTooltip key={member.userId} label={member.userName}>
              <Avatar className="size-5 border border-surface-100-900 text-[0.5rem]">
                <Avatar.Fallback>{initials(member.userName)}</Avatar.Fallback>
              </Avatar>
            </IconTooltip>
          ))}
          {assignees.length > 3 && (
            <IconTooltip label={`+${String(assignees.length - 3)} more`}>
              <Avatar className="size-5 border border-surface-100-900 text-[0.5rem]">
                <Avatar.Fallback>+{assignees.length - 3}</Avatar.Fallback>
              </Avatar>
            </IconTooltip>
          )}
        </div>
      )}
    </Link>
  );
}

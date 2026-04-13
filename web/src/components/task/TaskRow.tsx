import { Avatar } from "@skeletonlabs/skeleton-react";
import { Link } from "@tanstack/react-router";
import { useMemo } from "react";
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
  effortLevels?: TaskEffortLevel[];
  /** Priority levels for resolving per-level icons. */
  priorityLevels?: TaskPriorityLevel[];
  /** Override the link target route (default: space task detail) */
  to?: "/spaces/$spaceSlug/tasks/$taskId" | "/tasks/$spaceSlug/$taskId";
}) {
  const memberMap = useSpaceMemberMap(spaceSlug);

  const status = statusMap.get(task.status);
  const StatusIcon = useMemo(() => {
    return status?.icon ? getIcon(status.icon) : getIcon("circle");
  }, [status]);
  const statusColor = status
    ? (STATUS_ICON_COLOR[status.category] ?? "text-surface-500")
    : "text-surface-500";

  const EffortIcon = useMemo(() => {
    if (!task.effort || !effortLevels) return null;
    const level = effortLevels.find((l) => l.name === task.effort);
    return level?.icon ? getIcon(level.icon) : null;
  }, [task.effort, effortLevels]);

  const PriorityIcon = useMemo(() => {
    if (!task.priority || !priorityLevels) return null;
    const level = priorityLevels.find((l) => l.name === task.priority);
    return level?.icon ? getIcon(level.icon) : null;
  }, [task.priority, priorityLevels]);

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
      <StatusIcon className={`size-4 shrink-0 ${statusColor}`} aria-label={task.status} />

      <span className="min-w-0 flex-1 truncate text-sm">{task.title}</span>

      {assignees.length > 0 && (
        <div className="flex shrink-0 -space-x-1.5">
          {assignees.slice(0, 3).map((member) => (
            <Avatar
              key={member.userId}
              className="size-5 border border-surface-100-900 text-[0.5rem]"
              title={member.userName}
            >
              <Avatar.Fallback>{initials(member.userName)}</Avatar.Fallback>
            </Avatar>
          ))}
          {assignees.length > 3 && (
            <Avatar
              className="size-5 border border-surface-100-900 text-[0.5rem]"
              title={`+${String(assignees.length - 3)} more`}
            >
              <Avatar.Fallback>+{assignees.length - 3}</Avatar.Fallback>
            </Avatar>
          )}
        </div>
      )}

      {EffortIcon && (
        <EffortIcon
          className="text-surface-500 size-3.5 shrink-0"
          aria-label={task.effort ?? undefined}
        />
      )}

      {PriorityIcon && (
        <PriorityIcon
          className="text-surface-500 size-3.5 shrink-0"
          aria-label={task.priority ?? undefined}
        />
      )}
    </Link>
  );
}

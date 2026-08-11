import { Link } from "@tanstack/react-router";
import { useMemo } from "react";
import type { components } from "@horologia/client-core/schema";
import { useSpaceMemberMap } from "../../lib/hooks.ts";
import { getIcon } from "../../lib/level-icons.ts";
import { computeStaleness } from "@horologia/client-core/domain/staleness";
import { Avatar } from "../../ui/Avatar.tsx";

type Task = components["schemas"]["Task"];
type TaskStatus = components["schemas"]["TaskStatus"];
type TaskEffortLevel = components["schemas"]["TaskEffortLevel"];
type TaskPriorityLevel = components["schemas"]["TaskPriorityLevel"];

/** Color classes for status icon by category. */
const STATUS_ICON_COLOR: Record<string, string> = {
  initial: "text-base-content/60",
  intermediate: "text-warning",
  completion: "text-success",
};

const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });

/** Format a due date (YYYY-MM-DD) relative to today. */
function formatRelativeDue(dateStr: string): string {
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const due = new Date(dateStr + "T00:00:00");
  const diffDays = Math.round((due.getTime() - today.getTime()) / 86_400_000);

  const absDays = Math.abs(diffDays);
  let value: number;
  let unit: Intl.RelativeTimeFormatUnit;
  if (absDays < 7) {
    value = diffDays;
    unit = "day";
  } else if (absDays < 30) {
    value = Math.round(diffDays / 7);
    unit = "week";
  } else if (absDays < 365) {
    value = Math.round(diffDays / 30);
    unit = "month";
  } else {
    value = Math.round(diffDays / 365);
    unit = "year";
  }

  return rtf.format(value, unit);
}

function stalenessLabel(ratio: number): string {
  if (ratio >= 0.95 && ratio <= 1.05) return "Due now";
  if (ratio > 1) return `${(ratio - 1).toFixed(1)} cycles overdue`;
  return `${Math.round(ratio * 100)}% through cycle`;
}

/**
 * Map a staleness ratio to an HSL color string.
 *
 * 0  (fresh): deep green — hsl(130, 65%, 35%)
 * 1  (due):   pale green — hsl(120, 40%, 60%)
 * 2+ (very):  red        — hsl(0, 65%, 45%), clamped
 */
function stalenessColor(ratio: number): string {
  const t = Math.max(0, Math.min(ratio, 2));
  if (t <= 1) {
    const hue = 130 - t * 50;
    const sat = 65 - t * 25;
    const lgt = 35 + t * 25;
    return `hsl(${Math.round(hue)}, ${Math.round(sat)}%, ${Math.round(lgt)}%)`;
  }
  const p = t - 1;
  const hue = 80 - p * 80;
  const sat = 40 + p * 25;
  const lgt = 60 - p * 15;
  return `hsl(${Math.round(hue)}, ${Math.round(sat)}%, ${Math.round(lgt)}%)`;
}

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
  effortLevels: TaskEffortLevel[];
  priorityLevels: TaskPriorityLevel[];
  to?: "/spaces/$spaceSlug/tasks/$taskId" | "/tasks/$spaceSlug/$taskId";
}) {
  const memberMap = useSpaceMemberMap(spaceSlug);

  const status = statusMap.get(task.status);
  const StatusIcon = status?.icon ? getIcon(status.icon) : getIcon("circle");
  const statusColor = status
    ? (STATUS_ICON_COLOR[status.category] ?? "text-base-content/60")
    : "text-base-content/60";

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

  const { recurrenceType, recurrenceRule, lastCompletedAt, createdAt } = task;
  const staleness = useMemo(
    () =>
      computeStaleness(
        { recurrenceType, recurrenceRule, lastCompletedAt, createdAt },
        status?.category,
      ),
    [recurrenceType, recurrenceRule, lastCompletedAt, createdAt, status?.category],
  );

  const ariaLabel = useMemo(() => {
    const parts: string[] = [task.title];
    const details: string[] = [];
    details.push(`status ${task.status}`);
    if (task.effort) details.push(`effort ${task.effort}`);
    if (task.priority) details.push(`priority ${task.priority}`);
    if (assignees.length > 0) {
      details.push(`assigned to ${assignees.map((a) => a.userName).join(", ")}`);
    }
    return `${parts.join(" ")} — ${details.join(", ")}`;
  }, [task.title, task.status, task.effort, task.priority, assignees]);

  return (
    <Link
      to={to}
      params={{ spaceSlug, taskId: task.id }}
      aria-label={ariaLabel}
      className="catalogue-row group relative flex items-center gap-2 border-b border-base-300 px-3 py-2 transition-colors last:border-b-0 hover:bg-base-200 data-[status=active]:bg-base-200"
    >
      {staleness != null && (
        <div
          className="absolute inset-y-0 left-0 w-1"
          style={{ backgroundColor: stalenessColor(staleness) }}
          aria-hidden="true"
        />
      )}
      <StatusIcon className={`size-4 ${statusColor}`} aria-hidden="true" />

      <div className="min-w-0 flex-1">
        <span className="block truncate text-sm">{task.title}</span>
        {task.due && (
          <span className="block truncate text-xs text-base-content/60">
            {formatRelativeDue(task.due.at)}
          </span>
        )}
      </div>

      {EffortIcon && task.effort && (
        <EffortIcon className="size-3.5 text-base-content/60" aria-hidden="true" />
      )}

      {PriorityIcon && task.priority && (
        <PriorityIcon className="size-3.5 text-base-content/60" aria-hidden="true" />
      )}

      {assignees.length > 0 && (
        <div className="flex shrink-0 -space-x-1.5" aria-hidden="true">
          {assignees.slice(0, 3).map((member) => (
            <Avatar
              key={member.userId}
              size="xs"
              fallback={initials(member.userName)}
              className="border border-base-100"
            />
          ))}
          {assignees.length > 3 && (
            <Avatar
              size="xs"
              fallback={`+${assignees.length - 3}`}
              className="border border-base-100"
            />
          )}
        </div>
      )}
      {staleness != null && <span className="sr-only">{stalenessLabel(staleness)}</span>}
    </Link>
  );
}

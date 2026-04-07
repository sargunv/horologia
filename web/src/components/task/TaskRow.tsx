import { createLink } from "@tanstack/react-router";
import { Calendar, Gauge, SignalHigh, Tag, Users } from "lucide-react";
import type { components } from "../../api/schema.d.ts";
import { useSpaceMemberMap } from "../../lib/hooks.ts";
import { StatusBadge } from "./StatusBadge.tsx";

type Task = components["schemas"]["Task"];
type TaskStatus = components["schemas"]["TaskStatus"];

const TaskLink = createLink("a");

export function TaskRow({
  task,
  spaceSlug,
  statusMap,
  spaceLabel,
}: {
  task: Task;
  spaceSlug: string;
  statusMap: Map<string, TaskStatus>;
  spaceLabel?: string;
}) {
  const memberMap = useSpaceMemberMap(spaceSlug);
  const assigneeNames = task.assigneeIds.map((id) => memberMap.get(id)?.userName ?? id).join(", ");

  return (
    <TaskLink
      to="/spaces/$spaceSlug/tasks/$taskId"
      params={{ spaceSlug, taskId: task.id }}
      className="group flex items-center gap-4 border-b border-surface-200-800 px-4 py-3 transition-colors last:border-b-0 hover:bg-surface-100-900"
    >
      <div className="flex min-w-0 flex-1 items-center gap-3">
        <span className="text-surface-500 shrink-0 font-mono text-xs">{task.id}</span>
        <span className="truncate font-medium">{task.title}</span>
        {spaceLabel && (
          <span className="chip preset-tonal-surface shrink-0 text-xs">{spaceLabel}</span>
        )}
      </div>

      <div className="flex shrink-0 items-center gap-3">
        <StatusBadge status={task.status} statusMap={statusMap} />

        {task.assigneeIds.length > 0 && (
          <span
            className="text-surface-600-400 flex items-center gap-1 text-xs"
            title={assigneeNames}
          >
            <Users className="size-3.5" />
            <span className="max-w-24 truncate">{assigneeNames}</span>
          </span>
        )}

        {task.due && (
          <span className="text-surface-600-400 flex items-center gap-1 text-xs whitespace-nowrap">
            <Calendar className="size-3.5" />
            {new Date(task.due.at).toLocaleDateString()}
          </span>
        )}

        {task.effort && (
          <span className="text-surface-600-400 flex items-center gap-1 text-xs whitespace-nowrap">
            <Gauge className="size-3.5" />
            {task.effort}
          </span>
        )}

        {task.priority && (
          <span className="text-surface-600-400 flex items-center gap-1 text-xs whitespace-nowrap">
            <SignalHigh className="size-3.5" />
            {task.priority}
          </span>
        )}

        {task.tags.length > 0 && (
          <span
            className="text-surface-600-400 flex items-center gap-1 text-xs"
            title={task.tags.join(", ")}
          >
            <Tag className="size-3.5" />
            <span className="max-w-20 truncate">{task.tags.join(", ")}</span>
          </span>
        )}
      </div>
    </TaskLink>
  );
}

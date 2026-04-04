import type { components } from "../../api/schema.d.ts";

type TaskStatus = components["schemas"]["TaskStatus"];
type TaskStatusCategory = components["schemas"]["TaskStatusCategory"];

export const statusCategoryPreset: Record<TaskStatusCategory, string> = {
  initial: "preset-tonal-surface",
  intermediate: "preset-tonal-warning",
  completion: "preset-tonal-success",
};

export function StatusBadge({
  status,
  statusMap,
}: {
  status: string;
  statusMap: Map<string, TaskStatus>;
}) {
  const taskStatus = statusMap.get(status);
  const preset = taskStatus ? statusCategoryPreset[taskStatus.category] : "preset-tonal-surface";
  return (
    <span className={`rounded-base px-2 py-0.5 text-xs font-medium whitespace-nowrap ${preset}`}>
      {status}
    </span>
  );
}

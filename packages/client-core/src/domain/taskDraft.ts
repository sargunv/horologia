import type { components } from "../api/schema.d.ts";

type Task = components["schemas"]["Task"];
type TaskRecurrenceType = components["schemas"]["TaskRecurrenceType"];
type TaskOverdueAction = components["schemas"]["TaskOverdueAction"];

export interface TaskDraft {
  title: string;
  description: string;
  status: string;
  effort: string;
  priority: string;
  assigneeIds: string;
  rotationPool: string;
  tags: string;
  dueDate: string;
  timezone: string;
  recurrenceType: TaskRecurrenceType;
  recurrenceRule: string;
  overdueAction: TaskOverdueAction | "";
  overdueAfter: string;
  overdueStatus: string;
}

export function taskDraftFromTask(task?: Task, timezone = "UTC"): TaskDraft {
  const overdueAfter = task?.overdueActionRule?.after;
  return {
    title: task?.title ?? "",
    description: task?.description ?? "",
    status: task?.status ?? "",
    effort: task?.effort ?? "",
    priority: task?.priority ?? "",
    assigneeIds: task?.assigneeIds.join(", ") ?? "",
    rotationPool: task?.rotationPool.join(", ") ?? "",
    tags: task?.tags.join(", ") ?? "",
    dueDate: task?.due?.at ?? "",
    timezone: task?.due?.timezone ?? timezone,
    recurrenceType: task?.recurrenceType ?? "one_off",
    recurrenceRule: task?.recurrenceRule ?? "",
    overdueAction: task?.overdueActionRule?.action ?? "",
    overdueAfter: overdueAfter === null || overdueAfter === undefined ? "" : String(overdueAfter),
    overdueStatus: task?.overdueActionRule?.status ?? "",
  };
}

export function taskUpdateFromDraft(draft: TaskDraft): components["schemas"]["TaskUpdate"] {
  const overdueActionRule = draft.overdueAction
    ? {
        action: draft.overdueAction,
        after: draft.overdueAfter.trim() ? Number(draft.overdueAfter) : null,
        ...(draft.overdueAction === "set_status" && draft.overdueStatus
          ? { status: draft.overdueStatus }
          : {}),
      }
    : null;
  return {
    title: draft.title.trim(),
    description: draft.description,
    ...(draft.status ? { status: draft.status } : {}),
    effort: draft.effort || null,
    priority: draft.priority || null,
    assigneeIds: splitTaskValues(draft.assigneeIds),
    rotationPool: splitTaskValues(draft.rotationPool),
    tags: splitTaskValues(draft.tags),
    due: draft.dueDate.trim()
      ? { at: draft.dueDate.trim(), timezone: draft.timezone.trim() || "UTC" }
      : null,
    recurrenceType: draft.recurrenceType,
    recurrenceRule: taskRecurrenceUsesRule(draft.recurrenceType)
      ? draft.recurrenceRule || null
      : null,
    overdueActionRule,
  };
}

export function taskCreateFromDraft(draft: TaskDraft): components["schemas"]["TaskCreate"] {
  const update = taskUpdateFromDraft(draft);
  return {
    title: draft.title.trim(),
    description: draft.description,
    ...(update.status ? { status: update.status } : {}),
    ...(update.effort ? { effort: update.effort } : {}),
    ...(update.priority ? { priority: update.priority } : {}),
    recurrenceType: draft.recurrenceType,
    ...(update.recurrenceRule ? { recurrenceRule: update.recurrenceRule } : {}),
    assigneeIds: splitTaskValues(draft.assigneeIds),
    rotationPool: splitTaskValues(draft.rotationPool),
    tags: splitTaskValues(draft.tags),
    due: draft.dueDate.trim()
      ? { at: draft.dueDate.trim(), timezone: draft.timezone.trim() || "UTC" }
      : null,
    overdueActionRule: update.overdueActionRule ?? null,
  };
}

export function splitTaskValues(value: string): string[] {
  return [
    ...new Set(
      value
        .split(",")
        .map((part) => part.trim())
        .filter(Boolean),
    ),
  ];
}

export function taskRecurrenceUsesRule(type: TaskRecurrenceType): boolean {
  return (
    type === "completion_based" ||
    type === "fixed_non_accumulating" ||
    type === "fixed_accumulating"
  );
}

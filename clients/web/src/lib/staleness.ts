import { RRule } from "rrule";
import type { components } from "../api/schema.d.ts";

type Task = components["schemas"]["Task"];
type TaskStatusCategory = components["schemas"]["TaskStatusCategory"];

/**
 * Compute the next occurrence of an RRULE after a given date.
 * Sets dtstart to the anchor date so occurrences are relative to it.
 * Returns null if the rule cannot be parsed or has no future occurrences.
 */
export function computeNextOccurrence(rruleStr: string, afterDate: Date): Date | null {
  try {
    const normalized = rruleStr.startsWith("RRULE:") ? rruleStr : `RRULE:${rruleStr}`;
    const parsed = RRule.parseString(normalized);
    const rule = new RRule({ ...parsed, dtstart: afterDate });
    return rule.after(afterDate, false);
  } catch {
    return null;
  }
}

/**
 * Compute the staleness ratio for a task.
 *
 * staleness = (now - anchor) / (nextDue - anchor)
 *
 * where anchor = lastCompletedAt ?? createdAt, and nextDue is the next
 * rrule occurrence after anchor.
 *
 * Returns null when staleness does not apply:
 * - one_off recurrence type
 * - no recurrence rule
 * - completion-category status (task is "done")
 * - unable to compute interval from the rrule
 */
export function computeStaleness(
  task: Pick<Task, "recurrenceType" | "recurrenceRule" | "lastCompletedAt" | "createdAt">,
  statusCategory: TaskStatusCategory | undefined,
  now: Date = new Date(),
): number | null {
  if (task.recurrenceType === "one_off") return null;
  if (!task.recurrenceRule) return null;
  if (statusCategory === "completion") return null;

  const anchor = new Date(task.lastCompletedAt ?? task.createdAt);
  const nextDue = computeNextOccurrence(task.recurrenceRule, anchor);
  if (!nextDue) return null;

  const intervalMs = nextDue.getTime() - anchor.getTime();
  if (intervalMs <= 0) return null;

  return (now.getTime() - anchor.getTime()) / intervalMs;
}

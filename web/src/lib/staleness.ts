import { RRule } from "rrule";

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
  task: {
    recurrenceType: string;
    recurrenceRule: string | null;
    lastCompletedAt: string | null;
    createdAt: string;
  },
  statusCategory: string | undefined,
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

/**
 * Map a staleness ratio to an RGB color string.
 *
 * 0.0 = green (task is fresh)
 * 0.5 = yellow (halfway through cycle)
 * 1.0+ = red (task is due or overdue)
 *
 * Ratio is clamped to [0, 1] for color purposes.
 */
export function stalenessColor(ratio: number): string {
  const t = Math.max(0, Math.min(ratio, 1));

  let r: number;
  let g: number;
  let b: number;

  if (t <= 0.5) {
    // Green (76, 175, 80) → Yellow (255, 235, 59)
    const p = t / 0.5;
    r = Math.round(76 + (255 - 76) * p);
    g = Math.round(175 + (235 - 175) * p);
    b = Math.round(80 + (59 - 80) * p);
  } else {
    // Yellow (255, 235, 59) → Red (244, 67, 54)
    const p = (t - 0.5) / 0.5;
    r = Math.round(255 + (244 - 255) * p);
    g = Math.round(235 + (67 - 235) * p);
    b = Math.round(59 + (54 - 59) * p);
  }

  return `rgb(${r}, ${g}, ${b})`;
}

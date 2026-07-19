import type { components } from "../api/schema.d.ts";

type TaskStatusCategory = components["schemas"]["TaskStatusCategory"];

export function namedItemsValidationError(
  items: readonly { name: string }[],
  label: string,
  minItems = 0,
): string | null {
  for (const item of items) {
    const name = item.name.trim();
    if (!name) return `All ${label.toLowerCase()}s must have a name.`;
    if (name.length > 100) return `${label} names must be 100 characters or fewer.`;
  }
  const names = items.map((item) => item.name.trim().toLowerCase());
  if (new Set(names).size !== names.length) return `${label} names must be unique.`;
  if (items.length < minItems) {
    return `There must be at least ${minItems} ${label.toLowerCase()}${minItems === 1 ? "" : "s"}.`;
  }
  return null;
}

export function taskStatusesValidationError(
  items: readonly { name: string; category: TaskStatusCategory }[],
): string | null {
  const namesError = namedItemsValidationError(items, "Status");
  if (namesError) return namesError;
  if (items.filter((item) => item.category === "initial").length !== 1) {
    return "There must be exactly one initial status.";
  }
  if (!items.some((item) => item.category === "completion")) {
    return "There must be at least one completion status.";
  }
  return null;
}

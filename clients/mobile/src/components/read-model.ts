import { formatDuration, formatYield } from "@horologia/client-core/domain/recipe-inputs";

import type { DetailSection, Recipe, Task } from "@/components/native-views.types";

export function errorMessage(error: unknown, fallback: string): string {
  return error instanceof Error && error.message ? error.message : fallback;
}

export function formatTimestamp(value: string | number): string {
  const date = typeof value === "number" ? new Date(value) : new Date(value);
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

export function isStaleTimestamp(value: string | number, now = Date.now()): boolean {
  const timestamp = typeof value === "number" ? value : Date.parse(value);
  return !Number.isFinite(timestamp) || now - timestamp > 15 * 60 * 1000;
}

export function taskSubtitle(task: Task): string {
  const parts = [task.status];
  if (task.due) parts.push(`Due ${formatDate(task.due.at)}`);
  if (task.priority) parts.push(task.priority);
  return parts.join(" · ");
}

export function taskAccessibilityLabel(task: Task): string {
  const parts = [task.title, `status ${task.status}`];
  if (task.due) parts.push(`due ${formatDate(task.due.at)}`);
  if (task.effort) parts.push(`effort ${task.effort}`);
  if (task.priority) parts.push(`priority ${task.priority}`);
  return parts.join(", ");
}

export function taskDetailSections(task: Task): DetailSection[] {
  const overview = [
    { label: "Space", value: task.spaceSlug },
    { label: "Status", value: task.status },
    ...(task.effort ? [{ label: "Effort", value: task.effort }] : []),
    ...(task.priority ? [{ label: "Priority", value: task.priority }] : []),
    ...(task.due
      ? [{ label: "Due", value: `${formatDate(task.due.at)} · ${task.due.timezone}` }]
      : []),
    { label: "Recurrence", value: recurrenceLabel(task) },
  ];
  const sections: DetailSection[] = [{ title: "Details", properties: overview }];
  if (task.description.trim())
    sections.push({ title: "Description", paragraphs: [task.description] });
  if (task.tags.length > 0) sections.push({ title: "Tags", paragraphs: [task.tags.join(" · ")] });
  if (task.assigneeIds.length > 0) {
    sections.push({ title: "Assignees", paragraphs: task.assigneeIds });
  }
  if (task.relations.length > 0) {
    sections.push({
      title: "Related tasks",
      paragraphs: task.relations.map(
        (relation) => `${relation.kind.replaceAll("_", " ")} · ${relation.relatedTaskId}`,
      ),
    });
  }
  sections.push({
    title: "Record",
    properties: [
      { label: "Task ID", value: task.id },
      { label: "Created", value: formatTimestamp(task.createdAt) },
      { label: "Updated", value: formatTimestamp(task.updatedAt) },
    ],
  });
  return sections;
}

export function recipeSubtitle(recipe: {
  prepMinutes: number | null;
  cookMinutes: number | null;
  yield: Recipe["yield"];
}): string {
  const duration = (recipe.prepMinutes ?? 0) + (recipe.cookMinutes ?? 0);
  const parts: string[] = [];
  if (duration > 0) parts.push(formatDuration(duration));
  if (recipe.yield) parts.push(formatYield(recipe.yield.amount, recipe.yield.unit));
  return parts.join(" · ") || "Recipe";
}

export function recipeDetailSections(recipe: Recipe): DetailSection[] {
  const details = [
    { label: "Space", value: recipe.spaceSlug },
    ...(recipe.yield
      ? [{ label: "Yield", value: formatYield(recipe.yield.amount, recipe.yield.unit) }]
      : []),
    ...(recipe.prepMinutes ? [{ label: "Prep", value: formatDuration(recipe.prepMinutes) }] : []),
    ...(recipe.cookMinutes ? [{ label: "Cook", value: formatDuration(recipe.cookMinutes) }] : []),
  ];
  const sections: DetailSection[] = [{ title: "Details", properties: details }];
  if (recipe.tags.length > 0)
    sections.push({ title: "Tags", paragraphs: [recipe.tags.join(" · ")] });
  if (recipe.description.trim()) {
    sections.push({ title: "Description", paragraphs: [recipe.description] });
  }
  for (const section of recipe.ingredientSections) {
    sections.push({
      title: section.title || "Ingredients",
      paragraphs: section.ingredients.map(formatIngredient),
    });
  }
  for (const section of recipe.instructionSections) {
    sections.push({
      title: section.title || "Instructions",
      paragraphs: section.steps.map((step) => step.body),
      numbered: true,
    });
  }
  sections.push({
    title: "Record",
    properties: [
      { label: "Recipe ID", value: recipe.id },
      { label: "Created", value: formatTimestamp(recipe.createdAt) },
      { label: "Updated", value: formatTimestamp(recipe.updatedAt) },
    ],
  });
  return sections;
}

function recurrenceLabel(task: Task): string {
  const kind = task.recurrenceType.replaceAll("_", " ");
  return task.recurrenceRule ? `${kind} · ${task.recurrenceRule}` : kind;
}

function formatIngredient(
  ingredient: Recipe["ingredientSections"][number]["ingredients"][number],
): string {
  const range =
    ingredient.quantity == null
      ? ""
      : ingredient.quantityMax == null
        ? String(ingredient.quantity)
        : `${String(ingredient.quantity)}–${String(ingredient.quantityMax)}`;
  return [range, ingredient.unit, ingredient.item].filter(Boolean).join(" ");
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(
    new Date(`${value}T00:00:00`),
  );
}

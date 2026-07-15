export type SortableData =
  | { type: "ingredient-section" }
  | { type: "ingredient"; sectionKey: string }
  | { type: "instruction-section" }
  | { type: "step"; sectionKey: string };

export function readSortableData(
  data: Record<string, unknown> | undefined,
): SortableData | undefined {
  if (!data || typeof data["type"] !== "string") return undefined;
  if (data["type"] === "ingredient-section" || data["type"] === "instruction-section") {
    return { type: data["type"] };
  }
  if (
    (data["type"] === "ingredient" || data["type"] === "step") &&
    typeof data["sectionKey"] === "string"
  ) {
    return { type: data["type"], sectionKey: data["sectionKey"] };
  }
  return undefined;
}

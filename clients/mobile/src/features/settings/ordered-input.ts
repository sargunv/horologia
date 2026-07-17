import type { components } from "@horologia/client-core/schema";

type Schema = components["schemas"];
type StatusCategory = Schema["TaskStatusCategory"];

export function parseStatuses(value: string): Schema["TaskStatusInput"][] {
  return lines(value).map((line) => {
    const [name = "", category = "", icon = ""] = line.split("|").map((part) => part.trim());
    if (!name) throw new Error("Every status needs a name.");
    if (!isStatusCategory(category)) throw new Error(`${name} needs a valid category.`);
    return { name, category, icon };
  });
}

export function parseLevels(value: string): Schema["TaskEffortLevelInput"][] {
  return lines(value).map((line) => {
    const [name = "", icon = ""] = line.split("|").map((part) => part.trim());
    if (!name) throw new Error("Every level needs a name.");
    return { name, icon };
  });
}

function lines(value: string): string[] {
  return value
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);
}

function isStatusCategory(value: string): value is StatusCategory {
  return value === "initial" || value === "intermediate" || value === "completion";
}

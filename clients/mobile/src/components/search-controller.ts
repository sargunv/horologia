import type { components } from "@horologia/client-core/schema";

import type { SearchResultItem } from "./native-views.types";
import { type RouteScope, routes } from "../navigation/routes";

type TaskSearchResult = components["schemas"]["TaskSearchResult"];
type RecipeSearchResult = components["schemas"]["RecipeSummary"];

export function createSearchResults({
  tasks,
  recipes,
}: {
  tasks: TaskSearchResult[];
  recipes: RecipeSearchResult[];
}): SearchResultItem[] {
  return [
    ...tasks.map((task) => ({
      kind: "task" as const,
      id: task.id,
      spaceSlug: task.spaceSlug,
      title: task.title,
      meta: task.status,
    })),
    ...recipes.map((recipe) => ({
      kind: "recipe" as const,
      id: recipe.id,
      spaceSlug: recipe.spaceSlug,
      title: recipe.name,
      meta: recipe.tags.slice(0, 2).join(" · ") || "Recipe",
    })),
  ];
}

export function searchResultRoute(scope: RouteScope, result: SearchResultItem) {
  return result.kind === "task"
    ? routes.taskDetail(scope, result.spaceSlug, result.id)
    : routes.recipeDetail(scope, result.spaceSlug, result.id);
}

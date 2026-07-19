import { Link } from "@tanstack/react-router";
import { Clock3, CookingPot } from "lucide-react";
import { formatDuration } from "@horologia/client-core/domain/recipe-inputs";
import type { components } from "@horologia/client-core/schema";

type RecipeSummary = components["schemas"]["RecipeSummary"];

function durationLabel(recipe: RecipeSummary): string | null {
  const minutes = (recipe.prepMinutes ?? 0) + (recipe.cookMinutes ?? 0);
  return minutes > 0 ? formatDuration(minutes) : null;
}

export function RecipeRow({
  recipe,
  spaceName,
  scoped = false,
}: {
  recipe: RecipeSummary;
  spaceName?: string | undefined;
  scoped?: boolean;
}) {
  const duration = durationLabel(recipe);
  const yieldLabel = recipe.yield
    ? `${recipe.yield.amount.toLocaleString()} ${recipe.yield.unit}`
    : null;
  const to = scoped ? "/spaces/$spaceSlug/recipes/$recipeId" : "/recipes/$spaceSlug/$recipeId";

  return (
    <Link
      to={to}
      params={{ spaceSlug: recipe.spaceSlug, recipeId: recipe.id }}
      className="group block border-b border-base-300 px-3 py-3 transition-colors last:border-b-0 hover:bg-base-200 data-[status=active]:bg-base-200"
    >
      <span className="block truncate text-sm">{recipe.name}</span>
      <span className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-base-content/60">
        {spaceName && <span>{spaceName}</span>}
        {duration && (
          <span className="inline-flex items-center gap-1">
            <Clock3 className="size-3" aria-hidden="true" />
            {duration}
          </span>
        )}
        {yieldLabel && (
          <span className="inline-flex items-center gap-1">
            <CookingPot className="size-3" aria-hidden="true" />
            {yieldLabel}
          </span>
        )}
      </span>
      {recipe.tags.length > 0 && (
        <span className="mt-2 flex flex-wrap gap-1">
          {recipe.tags.slice(0, 3).map((tag) => (
            <span key={tag} className="badge badge-soft badge-sm">
              {tag}
            </span>
          ))}
          {recipe.tags.length > 3 && (
            <span className="badge badge-ghost badge-sm">+{recipe.tags.length - 3}</span>
          )}
        </span>
      )}
    </Link>
  );
}

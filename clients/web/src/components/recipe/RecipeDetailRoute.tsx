import { useSuspenseQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { ArrowLeft, ChevronRight } from "lucide-react";
import { AnchorLink } from "../../lib/links.ts";
import { spaceQueryOptions } from "../../lib/queries.ts";
import { RecipeDetailView } from "./RecipeDetail.tsx";

export function RecipeDetailRoute({
  spaceSlug,
  recipeId,
  scoped,
}: {
  spaceSlug: string;
  recipeId: string;
  scoped: boolean;
}) {
  const { data: space } = useSuspenseQuery(spaceQueryOptions(spaceSlug));
  const navigate = useNavigate();

  if (scoped) {
    return (
      <RecipeDetailView
        spaceSlug={spaceSlug}
        recipeId={recipeId}
        backLink={
          <AnchorLink
            to="/spaces/$spaceSlug/recipes"
            params={{ spaceSlug }}
            className="inline-flex items-center gap-1 text-sm text-base-content/70 transition-colors hover:text-base-content lg:hidden"
          >
            <ArrowLeft className="size-4" aria-hidden="true" />
            Back to {space.name} recipes
          </AnchorLink>
        }
        breadcrumb={
          <ol className="flex min-w-0 items-center gap-1 text-sm">
            <li>
              <AnchorLink
                to="/spaces/$spaceSlug/recipes"
                params={{ spaceSlug }}
                className="truncate text-base-content/70 hover:underline"
              >
                {space.name} recipes
              </AnchorLink>
            </li>
            <li className="text-base-content/60" aria-hidden="true">
              <ChevronRight className="size-3" />
            </li>
            <li>
              <AnchorLink
                to="/spaces/$spaceSlug/recipes/$recipeId"
                params={{ spaceSlug, recipeId }}
                className="shrink-0 font-mono hover:underline"
                aria-current="page"
              >
                {recipeId}
              </AnchorLink>
            </li>
          </ol>
        }
        onDeleteSuccess={() =>
          void navigate({ to: "/spaces/$spaceSlug/recipes", params: { spaceSlug } })
        }
      />
    );
  }

  return (
    <RecipeDetailView
      spaceSlug={spaceSlug}
      recipeId={recipeId}
      backLink={
        <AnchorLink
          to="/recipes"
          className="inline-flex items-center gap-1 text-sm text-base-content/70 transition-colors hover:text-base-content lg:hidden"
        >
          <ArrowLeft className="size-4" aria-hidden="true" />
          Back to recipes
        </AnchorLink>
      }
      breadcrumb={
        <ol className="flex min-w-0 items-center gap-1 text-sm">
          <li>
            <AnchorLink to="/recipes" className="truncate text-base-content/70 hover:underline">
              Recipes
            </AnchorLink>
          </li>
          <li className="text-base-content/60" aria-hidden="true">
            <ChevronRight className="size-3" />
          </li>
          <li>
            <AnchorLink
              to="/recipes/$spaceSlug/$recipeId"
              params={{ spaceSlug, recipeId }}
              className="shrink-0 font-mono hover:underline"
              aria-current="page"
            >
              {recipeId}
            </AnchorLink>
          </li>
        </ol>
      }
      onDeleteSuccess={() => void navigate({ to: "/recipes" })}
    />
  );
}

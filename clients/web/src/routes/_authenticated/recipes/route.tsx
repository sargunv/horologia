import { Outlet, createFileRoute, useRouterState } from "@tanstack/react-router";
import { CookingPot } from "lucide-react";
import { Suspense } from "react";
import { ListDetailLayout } from "../../../components/ListDetailLayout.tsx";
import { RecipeListPane } from "../../../components/recipe/RecipeListPane.tsx";
import { recipesInfiniteQueryOptions, spacesQueryOptions } from "../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/recipes")({
  loader: ({ context: { queryClient } }) =>
    Promise.all([
      queryClient.ensureQueryData(spacesQueryOptions),
      queryClient.ensureInfiniteQueryData(recipesInfiniteQueryOptions()),
    ]),
  component: RecipesLayout,
});

function RecipesLayout() {
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const isIndex = pathname === "/recipes" || pathname === "/recipes/";
  return (
    <div className="p-6">
      <ListDetailLayout
        list={
          <Suspense
            fallback={
              <div className="py-8 text-center text-sm text-base-content/60">Loading recipes…</div>
            }
          >
            <RecipeListPane />
          </Suspense>
        }
        detail={
          !isIndex ? (
            <Suspense
              fallback={
                <div className="p-6 text-center text-sm text-base-content/60">Loading…</div>
              }
            >
              <Outlet />
            </Suspense>
          ) : null
        }
        emptyState={
          <>
            <CookingPot className="size-12 text-base-content/40" aria-hidden="true" />
            <span className="text-sm text-base-content/60">Select a recipe to view details</span>
          </>
        }
      />
    </div>
  );
}

import { Outlet, createFileRoute, useRouterState } from "@tanstack/react-router";
import { CookingPot, ListChecks } from "lucide-react";
import { Suspense } from "react";
import { ListDetailLayout } from "../../../../components/ListDetailLayout.tsx";
import { TaskListPane } from "../../../../components/task/TaskListPane.tsx";
import { RecipeListPane } from "../../../../components/recipe/RecipeListPane.tsx";
import { spaceQueryOptions } from "../../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug")({
  loader: ({ context: { queryClient }, params: { spaceSlug } }) =>
    queryClient.ensureQueryData(spaceQueryOptions(spaceSlug)),
  component: SpaceLayout,
});

function SpaceLayout() {
  const { spaceSlug } = Route.useParams();
  const pathname = useRouterState({ select: (s) => s.location.pathname });

  const isSpaceIndex = pathname === `/spaces/${spaceSlug}` || pathname === `/spaces/${spaceSlug}/`;
  const isRecipeModule = pathname.startsWith(`/spaces/${spaceSlug}/recipes`);
  const isRecipeIndex =
    pathname === `/spaces/${spaceSlug}/recipes` || pathname === `/spaces/${spaceSlug}/recipes/`;

  return (
    <div className="p-6">
      <ListDetailLayout
        list={
          <Suspense>
            {isRecipeModule ? (
              <RecipeListPane lockedSpaceSlug={spaceSlug} />
            ) : (
              <TaskListPane spaceSlug={spaceSlug} />
            )}
          </Suspense>
        }
        detail={
          !(isRecipeModule ? isRecipeIndex : isSpaceIndex) ? (
            <Suspense
              fallback={
                <div className="text-base-content/60 p-6 text-center text-sm">Loading...</div>
              }
            >
              <Outlet />
            </Suspense>
          ) : null
        }
        emptyState={
          <>
            {isRecipeModule ? (
              <>
                <CookingPot className="text-base-content/40 size-12" aria-hidden="true" />
                <span className="text-base-content/60 text-sm">
                  Select a recipe to view details
                </span>
              </>
            ) : (
              <>
                <ListChecks className="text-base-content/40 size-12" aria-hidden="true" />
                <span className="text-base-content/60 text-sm">Select a task to view details</span>
              </>
            )}
          </>
        }
      />
    </div>
  );
}

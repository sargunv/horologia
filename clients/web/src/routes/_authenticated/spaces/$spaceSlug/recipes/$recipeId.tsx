import { createFileRoute } from "@tanstack/react-router";
import { RecipeDetailRoute } from "../../../../../components/recipe/RecipeDetailRoute.tsx";
import {
  recipeQueryOptions,
  spaceMembersQueryOptions,
  spaceQueryOptions,
} from "../../../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug/recipes/$recipeId")({
  loader: ({ context: { queryClient }, params: { spaceSlug, recipeId } }) =>
    Promise.all([
      queryClient.ensureQueryData(spaceQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(recipeQueryOptions(spaceSlug, recipeId)),
      queryClient.ensureQueryData(spaceMembersQueryOptions(spaceSlug)),
    ]),
  component: SpaceRecipeDetailPage,
});

function SpaceRecipeDetailPage() {
  const { spaceSlug, recipeId } = Route.useParams();
  return <RecipeDetailRoute spaceSlug={spaceSlug} recipeId={recipeId} scoped />;
}

import { createFileRoute } from "@tanstack/react-router";
import { RecipeDetailRoute } from "../../../../components/recipe/RecipeDetailRoute.tsx";
import {
  recipeQueryOptions,
  spaceMembersQueryOptions,
  spaceQueryOptions,
} from "../../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/recipes/$spaceSlug/$recipeId")({
  loader: ({ context: { queryClient }, params: { spaceSlug, recipeId } }) =>
    Promise.all([
      queryClient.ensureQueryData(recipeQueryOptions(spaceSlug, recipeId)),
      queryClient.ensureQueryData(spaceMembersQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spaceQueryOptions(spaceSlug)),
    ]),
  component: RecipeDetailPage,
});

function RecipeDetailPage() {
  const { spaceSlug, recipeId } = Route.useParams();
  return <RecipeDetailRoute spaceSlug={spaceSlug} recipeId={recipeId} scoped={false} />;
}

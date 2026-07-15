import { createFileRoute } from "@tanstack/react-router";
import { RecipeCreateView } from "../../../../../components/recipe/RecipeCreateView.tsx";
import { spaceQueryOptions } from "../../../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug/recipes/new")({
  loader: ({ context: { queryClient }, params: { spaceSlug } }) =>
    queryClient.ensureQueryData(spaceQueryOptions(spaceSlug)),
  component: CreateSpaceRecipePage,
});

function CreateSpaceRecipePage() {
  const { spaceSlug } = Route.useParams();
  return <RecipeCreateView spaceSlug={spaceSlug} scoped />;
}

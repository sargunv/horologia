import { createFileRoute } from "@tanstack/react-router";
import { RecipeCreateView } from "../../../../components/recipe/RecipeCreateView.tsx";
import { spaceQueryOptions } from "../../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/recipes/new/$spaceSlug")({
  loader: ({ context: { queryClient }, params: { spaceSlug } }) =>
    queryClient.ensureQueryData(spaceQueryOptions(spaceSlug)),
  component: CreateRecipePage,
});

function CreateRecipePage() {
  const { spaceSlug } = Route.useParams();
  return <RecipeCreateView spaceSlug={spaceSlug} scoped={false} />;
}

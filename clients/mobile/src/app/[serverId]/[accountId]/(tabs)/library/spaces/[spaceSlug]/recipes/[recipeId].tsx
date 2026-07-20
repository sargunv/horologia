import { useLocalSearchParams } from "expo-router";

import { RecipeDetailController } from "@/components/library-screens";

export default function SpaceRecipeDetailScreen() {
  const { spaceSlug, recipeId } = useLocalSearchParams<{ spaceSlug: string; recipeId: string }>();
  return <RecipeDetailController spaceSlug={spaceSlug} recipeId={recipeId} />;
}

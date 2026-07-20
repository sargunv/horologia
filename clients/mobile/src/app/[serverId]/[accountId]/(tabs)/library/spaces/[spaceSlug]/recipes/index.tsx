import { useLocalSearchParams } from "expo-router";

import { RecipesScreen } from "@/components/library-screens";

export default function SpaceRecipesScreen() {
  const { spaceSlug } = useLocalSearchParams<{ spaceSlug: string }>();
  return <RecipesScreen spaceSlug={spaceSlug} />;
}

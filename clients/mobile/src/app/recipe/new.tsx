import { useLocalSearchParams } from "expo-router";

import { useSession } from "@/auth/session-context";
import { RecipeEditor } from "@/components/recipe-editor";
import { ScreenState } from "@/components/screen-state";

export default function NewRecipeScreen() {
  const { spaceSlug } = useLocalSearchParams<{ spaceSlug?: string }>();
  const session = useSession();
  if (!session.profile || !session.client) return <ScreenState loading title="Opening editor" />;
  return (
    <RecipeEditor
      client={session.client}
      profile={session.profile}
      {...(spaceSlug ? { initialSpaceSlug: spaceSlug } : {})}
    />
  );
}

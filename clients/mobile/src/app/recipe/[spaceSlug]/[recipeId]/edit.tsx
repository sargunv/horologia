import { createQueries, type HorologiaClient, type ServerProfile } from "@horologia/client-core";
import { useQuery } from "@tanstack/react-query";
import { useLocalSearchParams } from "expo-router";
import { useMemo } from "react";

import { useSession } from "@/auth/session-context";
import { RecipeEditor } from "@/components/recipe-editor";
import { ScreenState } from "@/components/screen-state";

export default function EditRecipeScreen() {
  const { spaceSlug, recipeId } = useLocalSearchParams<{ spaceSlug: string; recipeId: string }>();
  const session = useSession();
  if (!session.profile || !session.client || !spaceSlug || !recipeId) return <ScreenState loading title="Opening editor" />;
  return <AuthenticatedEditor client={session.client} profile={session.profile} recipeId={recipeId} spaceSlug={spaceSlug} />;
}

function AuthenticatedEditor({ client, profile, recipeId, spaceSlug }: { client: HorologiaClient; profile: ServerProfile; recipeId: string; spaceSlug: string }) {
  const queries = useMemo(() => createQueries({ serverId: profile.id, apiClient: client, appClient: client }), [client, profile.id]);
  const recipe = useQuery(queries.recipeQueryOptions(spaceSlug, recipeId));
  if (recipe.isPending) return <ScreenState loading title="Opening editor" />;
  if (recipe.isError) return <ScreenState detail={recipe.error.message} title="Recipe unavailable" />;
  return <RecipeEditor client={client} profile={profile} recipe={recipe.data} />;
}

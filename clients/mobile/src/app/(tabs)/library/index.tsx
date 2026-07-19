import { createQueries, type HorologiaClient } from "@horologia/client-core";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { Button, Host, List, ListItem, Text } from "@expo/ui";
import { useRouter } from "expo-router";
import { useMemo } from "react";

import { useSession } from "@/auth/session-context";
import { ScreenState } from "@/components/screen-state";

export default function LibraryScreen() {
  const session = useSession();
  if (!session.profile || !session.client) return <ScreenState loading title="Opening library" />;
  return <Library client={session.client} serverId={session.profile.id} />;
}

function Library({ client, serverId }: { client: HorologiaClient; serverId: string }) {
  const router = useRouter();
  const queries = useMemo(() => createQueries({ serverId, apiClient: client }), [client, serverId]);
  const spaces = useQuery(queries.spacesQueryOptions);
  const recipes = useInfiniteQuery(queries.recipesInfiniteQueryOptions());
  if (spaces.isPending || recipes.isPending) return <ScreenState loading title="Loading library" />;
  if (spaces.isError || recipes.isError) {
    return (
      <ScreenState
        detail="Pull the library again when your server is reachable."
        title="Library unavailable"
      />
    );
  }
  const recipeItems = recipes.data.pages.flatMap((page) => page.items);
  return (
    <Host style={{ flex: 1 }} useViewportSizeMeasurement>
      <List
        onRefresh={async () => {
          await Promise.all([spaces.refetch(), recipes.refetch()]);
        }}
      >
        <ListItem
          supportingText="Spaces, recipes, and household context"
          trailing={
            <Button label="New space" onPress={() => router.push("/space/new")} variant="text" />
          }
        >
          <Text>Spaces</Text>
        </ListItem>
        {spaces.data.map((space) => (
          <ListItem
            key={space.slug}
            onPress={() =>
              router.push({ pathname: "/space/[spaceSlug]", params: { spaceSlug: space.slug } })
            }
            supportingText={space.description || space.slug}
          >
            <Text>{space.name}</Text>
          </ListItem>
        ))}
        <ListItem
          trailing={
            <Button label="New recipe" onPress={() => router.push("/recipe/new")} variant="text" />
          }
        >
          <Text>Recipes</Text>
        </ListItem>
        {recipeItems.map((recipe) => (
          <ListItem
            key={`${recipe.spaceSlug}/${recipe.id}`}
            onPress={() =>
              router.push({
                pathname: "/recipe/[spaceSlug]/[recipeId]",
                params: { spaceSlug: recipe.spaceSlug, recipeId: recipe.id },
              })
            }
            supportingText={`${recipe.spaceSlug} · ${(recipe.prepMinutes ?? 0) + (recipe.cookMinutes ?? 0)} min`}
          >
            <Text>{recipe.name}</Text>
          </ListItem>
        ))}
        {recipeItems.length === 0 ? (
          <ListItem supportingText="Add the first household favorite">
            <Text>No recipes yet</Text>
          </ListItem>
        ) : null}
        {recipes.hasNextPage ? (
          <ListItem
            onPress={() => {
              if (!recipes.isFetchingNextPage) void recipes.fetchNextPage();
            }}
            supportingText={recipes.isFetchingNextPage ? "Loading…" : undefined}
          >
            <Text>Load more</Text>
          </ListItem>
        ) : null}
      </List>
    </Host>
  );
}

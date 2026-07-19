import { createQueries, type HorologiaClient } from "@horologia/client-core";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { Button, Host, List, ListItem, Text } from "@expo/ui";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useMemo } from "react";

import { useSession } from "@/auth/session-context";
import { ScreenState } from "@/components/screen-state";

export default function SpaceScreen() {
  const { spaceSlug } = useLocalSearchParams<{ spaceSlug: string }>();
  const session = useSession();
  if (!session.profile || !session.client || !spaceSlug) {
    return <ScreenState loading title="Opening space" />;
  }
  return (
    <SpaceDetail client={session.client} serverId={session.profile.id} spaceSlug={spaceSlug} />
  );
}

function SpaceDetail({
  client,
  serverId,
  spaceSlug,
}: {
  client: HorologiaClient;
  serverId: string;
  spaceSlug: string;
}) {
  const router = useRouter();
  const queries = useMemo(
    () => createQueries({ serverId, apiClient: client }),
    [client, serverId],
  );
  const space = useQuery(queries.spaceQueryOptions(spaceSlug));
  const tasks = useInfiniteQuery(queries.spaceTasksInfiniteQueryOptions(spaceSlug));
  const recipes = useInfiniteQuery(queries.recipesInfiniteQueryOptions(spaceSlug));
  const activity = useInfiniteQuery(queries.spaceActivityInfiniteQueryOptions(spaceSlug));
  if (space.isPending || tasks.isPending || recipes.isPending) {
    return <ScreenState loading title="Loading space" />;
  }
  if (space.isError || tasks.isError || recipes.isError) {
    return (
      <ScreenState detail="Try again when your server is reachable." title="Space unavailable" />
    );
  }
  const taskItems = tasks.data.pages.flatMap((page) => page.items);
  const recipeItems = recipes.data.pages.flatMap((page) => page.items);
  const activityItems = activity.data?.pages.flatMap((page) => page.items) ?? [];
  return (
    <Host style={{ flex: 1 }} useViewportSizeMeasurement>
      <List
        onRefresh={async () => {
          await Promise.all([space.refetch(), tasks.refetch(), recipes.refetch(), activity.refetch()]);
        }}
      >
        <ListItem
          supportingText={space.data.description || space.data.slug}
          trailing={
            <Button
              label="Settings"
              onPress={() =>
                router.push({ pathname: "/space/[spaceSlug]/settings", params: { spaceSlug } })
              }
              variant="text"
            />
          }
        >
          <Text>{space.data.name}</Text>
        </ListItem>
        <ListItem
          trailing={
            <Button
              label="New task"
              onPress={() => router.push({ pathname: "/task/new", params: { spaceSlug } })}
              variant="text"
            />
          }
        >
          <Text>Tasks</Text>
        </ListItem>
        {taskItems.map((task) => (
          <ListItem
            key={task.id}
            onPress={() =>
              router.push({
                pathname: "/task/[spaceSlug]/[taskId]",
                params: { spaceSlug, taskId: task.id },
              })
            }
            supportingText={task.status}
          >
            <Text>{task.title}</Text>
          </ListItem>
        ))}
        <ListItem
          trailing={
            <Button
              label="New recipe"
              onPress={() => router.push({ pathname: "/recipe/new", params: { spaceSlug } })}
              variant="text"
            />
          }
        >
          <Text>Recipes</Text>
        </ListItem>
        {recipeItems.map((recipe) => (
          <ListItem
            key={recipe.id}
            onPress={() =>
              router.push({
                pathname: "/recipe/[spaceSlug]/[recipeId]",
                params: { spaceSlug, recipeId: recipe.id },
              })
            }
            supportingText={`${(recipe.prepMinutes ?? 0) + (recipe.cookMinutes ?? 0)} min`}
          >
            <Text>{recipe.name}</Text>
          </ListItem>
        ))}
        <ListItem>
          <Text>Activity</Text>
        </ListItem>
        {activityItems.slice(0, 20).map((entry) => (
          <ListItem
            key={entry.id}
            supportingText={new Date(entry.createdAt).toLocaleString()}
          >
            <Text>{`${entry.action} ${entry.entityType}`}</Text>
          </ListItem>
        ))}
        {activityItems.length === 0 ? (
          <ListItem>
            <Text>No activity yet</Text>
          </ListItem>
        ) : null}
      </List>
    </Host>
  );
}

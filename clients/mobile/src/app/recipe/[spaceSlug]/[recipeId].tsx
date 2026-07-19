import {
  createLibraryCommands,
  createQueries,
  type HorologiaClient,
  type ServerProfile,
} from "@horologia/client-core";
import type { components } from "@horologia/client-core/schema";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button, ListItem, Text } from "@expo/ui";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useMemo } from "react";
import { Alert, Share } from "react-native";

import { useSession } from "@/auth/session-context";
import { FormSection } from "@/components/forms";
import { NativeFormScreen } from "@/components/native-screen";
import { ScreenState } from "@/components/screen-state";

type Recipe = components["schemas"]["Recipe"];

export default function RecipeScreen() {
  const { spaceSlug, recipeId } = useLocalSearchParams<{
    spaceSlug: string;
    recipeId: string;
  }>();
  const session = useSession();
  if (!session.profile || !session.client || !spaceSlug || !recipeId) {
    return <ScreenState loading title="Loading recipe" />;
  }
  return (
    <RecipeDetail
      client={session.client}
      profile={session.profile}
      recipeId={recipeId}
      spaceSlug={spaceSlug}
    />
  );
}

function RecipeDetail({
  client,
  profile,
  recipeId,
  spaceSlug,
}: {
  client: HorologiaClient;
  profile: ServerProfile;
  recipeId: string;
  spaceSlug: string;
}) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const queries = useMemo(
    () => createQueries({ serverId: profile.id, apiClient: client }),
    [client, profile.id],
  );
  const recipe = useQuery(queries.recipeQueryOptions(spaceSlug, recipeId));
  const activity = useInfiniteQuery(
    queries.recipeActivityInfiniteQueryOptions(spaceSlug, recipeId),
  );
  const commands = createLibraryCommands({
    serverId: profile.id,
    apiClient: client,
    queryClient,
  });
  const deletion = useMutation({
    mutationFn: () => commands.deleteRecipe(spaceSlug, recipeId),
    onSuccess() {
      router.replace("/(tabs)/library");
    },
  });
  if (recipe.isPending) return <ScreenState loading title="Loading recipe" />;
  if (recipe.isError) {
    return <ScreenState detail={recipe.error.message} title="Recipe unavailable" />;
  }
  const shareUrl = new URL(
    `spaces/${encodeURIComponent(spaceSlug)}/recipes/${encodeURIComponent(recipeId)}`,
    `${profile.baseUrl.replace(/\/+$/u, "")}/`,
  ).toString();
  return (
    <RecipeView
      activity={activity.data?.pages.flatMap((page) => page.items) ?? []}
      onDelete={() =>
        Alert.alert("Delete recipe?", "This cannot be undone.", [
          { text: "Cancel", style: "cancel" },
          { text: "Delete", style: "destructive", onPress: () => deletion.mutate() },
        ])
      }
      onEdit={() =>
        router.push({
          pathname: "/recipe/[spaceSlug]/[recipeId]/edit",
          params: { spaceSlug, recipeId },
        })
      }
      onShare={() =>
        void Share.share({ title: recipe.data.name, message: shareUrl, url: shareUrl })
      }
      recipe={recipe.data}
    />
  );
}

function RecipeView({
  recipe,
  activity,
  onEdit,
  onShare,
  onDelete,
}: {
  recipe: Recipe;
  activity: components["schemas"]["ActivityLogEntry"][];
  onEdit: () => void;
  onShare: () => void;
  onDelete: () => void;
}) {
  return (
    <NativeFormScreen>
      <FormSection title={recipe.name}>
        <Text>{recipe.spaceSlug}</Text>
        <Button label="Edit" onPress={onEdit} variant="filled" />
        <Button label="Share" onPress={onShare} variant="filled" />
        <Button label="Delete" onPress={onDelete} variant="text" />
      </FormSection>
      <FormSection title="Recipe details">
        <Metric
          label="Yield"
          value={recipe.yield ? `${recipe.yield.amount} ${recipe.yield.unit}` : "—"}
        />
        <Metric
          label="Prep"
          value={recipe.prepMinutes === null ? "—" : `${recipe.prepMinutes} min`}
        />
        <Metric
          label="Cook"
          value={recipe.cookMinutes === null ? "—" : `${recipe.cookMinutes} min`}
        />
        {recipe.description ? <Text>{recipe.description}</Text> : null}
        {recipe.tags.length ? <Text>{recipe.tags.join(" · ")}</Text> : null}
      </FormSection>
      <FormSection title="Ingredients">
        {recipe.ingredientSections.flatMap((section, sectionIndex) => [
          ...(section.title ? [<Text key={`title/${sectionIndex}`}>{section.title}</Text>] : []),
          ...section.ingredients.map((ingredient, index) => (
            <Text key={`${sectionIndex}/${ingredient.item}/${index}`}>
              {`• ${formatIngredient(ingredient)}`}
            </Text>
          )),
        ])}
      </FormSection>
      <FormSection title="Instructions">
        {recipe.instructionSections.flatMap((section, sectionIndex) => [
          ...(section.title ? [<Text key={`title/${sectionIndex}`}>{section.title}</Text>] : []),
          ...section.steps.map((step, index) => (
            <Text key={`${sectionIndex}/${step.body}/${index}`}>{`${index + 1}. ${step.body}`}</Text>
          )),
        ])}
      </FormSection>
      <FormSection title="Activity">
        {activity.map((entry) => (
          <ListItem key={entry.id} supportingText={new Date(entry.createdAt).toLocaleString()}>
            <Text>{`${entry.action} ${entry.entityType}`}</Text>
          </ListItem>
        ))}
        {activity.length === 0 ? <Text>No activity yet.</Text> : null}
      </FormSection>
    </NativeFormScreen>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <ListItem supportingText={value}>
      <Text>{label}</Text>
    </ListItem>
  );
}

function formatIngredient(
  item: Recipe["ingredientSections"][number]["ingredients"][number],
): string {
  const amount =
    item.quantity === null
      ? ""
      : item.quantityMax === null
        ? String(item.quantity)
        : `${item.quantity}–${item.quantityMax}`;
  return [amount, item.unit, item.item].filter(Boolean).join(" ");
}

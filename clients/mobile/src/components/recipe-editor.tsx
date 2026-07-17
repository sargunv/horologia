import {
  createLibraryCommands,
  createQueries,
  recipeCreateFromDraft,
  recipeDraftFromRecipe,
  validateRecipeDraft,
  type HorologiaClient,
  type RecipeDraft,
  type ServerProfile,
} from "@horologia/client-core";
import type { components } from "@horologia/client-core/schema";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import { useMemo, useState } from "react";
import { ScrollView, StyleSheet, Text } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ActionButton, ChoiceChips, FormField, FormSection } from "@/components/forms";
import { colors } from "@/design/tokens";

type Recipe = components["schemas"]["Recipe"];

export function RecipeEditor({
  client,
  profile,
  recipe,
  initialSpaceSlug,
}: {
  client: HorologiaClient;
  profile: ServerProfile;
  recipe?: Recipe;
  initialSpaceSlug?: string;
}) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const queries = useMemo(
    () => createQueries({ serverId: profile.id, apiClient: client, appClient: client }),
    [client, profile.id],
  );
  const spaces = useQuery(queries.spacesQueryOptions);
  const defaultSpace = recipe?.spaceSlug ?? initialSpaceSlug ?? spaces.data?.[0]?.slug ?? "";
  const [spaceSlug, setSpaceSlug] = useState(defaultSpace);
  const activeSpace = spaceSlug || defaultSpace;
  const [draft, setDraft] = useState<RecipeDraft>(() => recipeDraftFromRecipe(recipe));
  const [cacheWarning, setCacheWarning] = useState<string | null>(null);
  const commands = createLibraryCommands({
    serverId: profile.id,
    apiClient: client,
    queryClient,
    onCacheError() {
      setCacheWarning("Saved, but a list may need to be refreshed.");
    },
  });
  const mutation = useMutation({
    mutationFn: async () => {
      if (!activeSpace) throw new Error("Choose a space.");
      if (recipe) {
        const result = validateRecipeDraft(draft);
        if (!result.body) throw new Error(result.errors.join(" "));
        return commands.updateRecipe(activeSpace, recipe.id, result.body);
      }
      const result = recipeCreateFromDraft(draft);
      if (!result.body) throw new Error(result.errors.join(" "));
      return commands.createRecipe(activeSpace, result.body);
    },
    onSuccess(saved) {
      router.replace({
        pathname: "/recipe/[spaceSlug]/[recipeId]",
        params: { spaceSlug: saved.spaceSlug, recipeId: saved.id },
      });
    },
  });

  function set<K extends keyof RecipeDraft>(key: K, value: RecipeDraft[K]) {
    setDraft((current) => ({ ...current, [key]: value }));
  }

  return (
    <SafeAreaView edges={["left", "right", "bottom"]} style={styles.safeArea}>
      <ScrollView
        automaticallyAdjustKeyboardInsets
        contentContainerStyle={styles.scroll}
        keyboardDismissMode="on-drag"
        keyboardShouldPersistTaps="handled"
      >
        <Text accessibilityRole="header" style={styles.heading}>
          {recipe ? "Edit recipe" : "New recipe"}
        </Text>
        <Text style={styles.detail}>
          Simple fields, Markdown where it helps, and line order as the section order.
        </Text>
        {!recipe && spaces.data ? (
          <FormSection title="Space">
            <ChoiceChips
              label="Space"
              onChange={setSpaceSlug}
              options={spaces.data.map((space) => ({ value: space.slug, label: space.name }))}
              value={activeSpace}
            />
          </FormSection>
        ) : null}
        <FormSection title="Recipe">
          <FormField
            label="Name"
            maxLength={500}
            onChangeText={(value) => set("name", value)}
            testID="recipe-name-input"
            value={draft.name}
          />
          <FormField
            label="Description (Markdown)"
            maxLength={10_000}
            multiline
            onChangeText={(value) => set("description", value)}
            placeholder="A **favorite** weeknight dinner."
            testID="recipe-description-input"
            value={draft.description}
          />
          <FormField
            label="Yield"
            onChangeText={(value) => set("yield", value)}
            placeholder="4 servings"
            value={draft.yield}
          />
          <FormField
            label="Prep time"
            onChangeText={(value) => set("prepTime", value)}
            placeholder="20 min"
            value={draft.prepTime}
          />
          <FormField
            label="Cook time"
            onChangeText={(value) => set("cookTime", value)}
            placeholder="1h 15m"
            value={draft.cookTime}
          />
          <FormField
            label="Tags (comma separated)"
            onChangeText={(value) => set("tags", value)}
            value={draft.tags}
          />
        </FormSection>
        <FormSection title="Ingredients">
          <Text style={styles.hint}>
            Use `## Section` headings and `quantity | ingredient` lines. Reorder lines to reorder
            sections and ingredients.
          </Text>
          <FormField
            label="Ingredient sections"
            multiline
            onChangeText={(value) => set("ingredients", value)}
            placeholder={"## Dough\n2 cups | flour\n1 tsp | salt"}
            value={draft.ingredients}
          />
        </FormSection>
        <FormSection title="Instructions">
          <Text style={styles.hint}>
            Use `## Section` headings and one `- step` per line. Markdown is preserved in
            descriptions and step bodies.
          </Text>
          <FormField
            label="Instruction sections"
            multiline
            onChangeText={(value) => set("instructions", value)}
            placeholder={"## Make\n- Mix everything\n- Bake until golden"}
            value={draft.instructions}
          />
        </FormSection>
        {mutation.error ? (
          <Text accessibilityRole="alert" style={styles.error}>
            {mutation.error.message}
          </Text>
        ) : null}
        {cacheWarning ? <Text style={styles.warning}>{cacheWarning}</Text> : null}
        <ActionButton
          disabled={!draft.name.trim() || !activeSpace || mutation.isPending}
          label={mutation.isPending ? "Saving…" : recipe ? "Save recipe" : "Create recipe"}
          onPress={() => mutation.mutate()}
        />
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: { backgroundColor: colors.canvas, flex: 1 },
  scroll: {
    alignSelf: "center",
    gap: 14,
    maxWidth: 760,
    padding: 18,
    paddingBottom: 48,
    width: "100%",
  },
  heading: { color: colors.ink, fontSize: 30, fontWeight: "700" },
  detail: { color: colors.muted, fontSize: 14, lineHeight: 20 },
  hint: { color: colors.muted, fontSize: 13, lineHeight: 19 },
  error: { color: colors.danger, fontSize: 14, fontWeight: "600", lineHeight: 20 },
  warning: { color: colors.accent, fontSize: 14, fontWeight: "600" },
});

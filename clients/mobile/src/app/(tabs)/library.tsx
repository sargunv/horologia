import { createQueries, type HorologiaClient } from "@horologia/client-core";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import { useMemo } from "react";
import { Pressable, ScrollView, StyleSheet, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useSession } from "@/auth/session-context";
import { ScreenState } from "@/components/screen-state";
import { colors } from "@/design/tokens";

export default function LibraryScreen() {
  const session = useSession();
  if (!session.profile || !session.client) return <ScreenState loading title="Opening library" />;
  return <Library client={session.client} serverId={session.profile.id} />;
}

function Library({ client, serverId }: { client: HorologiaClient; serverId: string }) {
  const router = useRouter();
  const queries = useMemo(
    () => createQueries({ serverId, apiClient: client, appClient: client }),
    [client, serverId],
  );
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
    <SafeAreaView edges={["left", "right"]} style={styles.safeArea}>
      <ScrollView contentContainerStyle={styles.scroll}>
        <View style={styles.header}>
          <View>
            <Text accessibilityRole="header" style={styles.heading}>
              Library
            </Text>
            <Text style={styles.subheading}>Spaces, recipes, and household context.</Text>
          </View>
          <Pressable
            accessibilityRole="button"
            onPress={() => router.push("/space/new")}
            style={styles.addButton}
          >
            <Text style={styles.addText}>New space</Text>
          </Pressable>
        </View>
        <Text accessibilityRole="header" style={styles.sectionTitle}>
          Spaces
        </Text>
        <View style={styles.grid}>
          {spaces.data.map((space) => (
            <Pressable
              accessibilityRole="button"
              key={space.slug}
              onPress={() =>
                router.push({ pathname: "/space/[spaceSlug]", params: { spaceSlug: space.slug } })
              }
              style={styles.spaceCard}
            >
              <Text style={styles.cardTitle}>{space.name}</Text>
              <Text numberOfLines={2} style={styles.cardDetail}>
                {space.description || space.slug}
              </Text>
            </Pressable>
          ))}
        </View>
        <View style={styles.sectionHeader}>
          <Text accessibilityRole="header" style={styles.sectionTitle}>
            Recipes
          </Text>
          <Pressable accessibilityRole="button" onPress={() => router.push("/recipe/new")}>
            <Text style={styles.link}>New recipe</Text>
          </Pressable>
        </View>
        {recipeItems.map((recipe) => (
          <Pressable
            accessibilityRole="button"
            key={`${recipe.spaceSlug}/${recipe.id}`}
            onPress={() =>
              router.push({
                pathname: "/recipe/[spaceSlug]/[recipeId]",
                params: { spaceSlug: recipe.spaceSlug, recipeId: recipe.id },
              })
            }
            style={styles.recipeRow}
          >
            <View style={styles.rowBody}>
              <Text style={styles.recipeTitle}>{recipe.name}</Text>
              <Text style={styles.meta}>
                {recipe.spaceSlug} · {(recipe.prepMinutes ?? 0) + (recipe.cookMinutes ?? 0)} min
              </Text>
            </View>
            <Text style={styles.chevron}>›</Text>
          </Pressable>
        ))}
        {recipeItems.length === 0 ? (
          <Text style={styles.empty}>No recipes yet. Add the first favorite.</Text>
        ) : null}
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: { backgroundColor: colors.canvas, flex: 1 },
  scroll: {
    alignSelf: "center",
    gap: 12,
    maxWidth: 900,
    padding: 18,
    paddingBottom: 44,
    width: "100%",
  },
  header: { alignItems: "center", flexDirection: "row", justifyContent: "space-between" },
  heading: { color: colors.ink, fontSize: 30, fontWeight: "700" },
  subheading: { color: colors.muted, fontSize: 14, marginTop: 3 },
  addButton: {
    backgroundColor: colors.accent,
    borderRadius: 13,
    paddingHorizontal: 14,
    paddingVertical: 11,
  },
  addText: { color: colors.surface, fontSize: 14, fontWeight: "700" },
  sectionHeader: {
    alignItems: "center",
    flexDirection: "row",
    justifyContent: "space-between",
    marginTop: 8,
  },
  sectionTitle: { color: colors.ink, fontSize: 20, fontWeight: "700", marginTop: 8 },
  link: { color: colors.accent, fontSize: 14, fontWeight: "700" },
  grid: { flexDirection: "row", flexWrap: "wrap", gap: 10 },
  spaceCard: {
    backgroundColor: colors.surface,
    borderRadius: 16,
    minHeight: 100,
    minWidth: 220,
    padding: 16,
    flexGrow: 1,
    flexBasis: "45%",
  },
  cardTitle: { color: colors.ink, fontSize: 18, fontWeight: "700" },
  cardDetail: { color: colors.muted, fontSize: 14, lineHeight: 19, marginTop: 7 },
  recipeRow: {
    alignItems: "center",
    backgroundColor: colors.surface,
    borderRadius: 15,
    flexDirection: "row",
    padding: 15,
  },
  rowBody: { flex: 1 },
  recipeTitle: { color: colors.ink, fontSize: 17, fontWeight: "600" },
  meta: { color: colors.muted, fontSize: 13, marginTop: 4 },
  chevron: { color: colors.muted, fontSize: 27 },
  empty: { color: colors.muted, fontSize: 14, paddingVertical: 20, textAlign: "center" },
});

import { useQuery } from "@tanstack/react-query";
import { useLocalSearchParams } from "expo-router";
import { ScrollView, StyleSheet, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useSession } from "@/auth/session-context";
import { ScreenState } from "@/components/screen-state";
import { colors } from "@/design/tokens";

export default function RecipePreviewScreen() {
  const { spaceSlug, recipeId } = useLocalSearchParams<{
    spaceSlug: string;
    recipeId: string;
  }>();
  const session = useSession();
  const recipe = useQuery({
    queryKey: [session.profile?.id ?? "", "spaces", spaceSlug, "recipes", recipeId],
    enabled: session.client !== null && Boolean(spaceSlug && recipeId),
    queryFn: async () => {
      if (!session.client) throw new Error("Not signed in");
      const { data, error } = await session.client.GET(
        "/spaces/{spaceSlug}/recipes/{recipeId}",
        {
          params: { path: { spaceSlug, recipeId } },
        },
      );
      if (error) throw error;
      return data;
    },
  });
  if (recipe.isPending) return <ScreenState loading title="Loading recipe" />;
  if (recipe.isError) return <ScreenState detail={recipe.error.message} title="Recipe unavailable" />;
  return (
    <SafeAreaView edges={["left", "right", "bottom"]} style={styles.safeArea}>
      <ScrollView contentContainerStyle={styles.scroll}>
        <Text style={styles.space}>{recipe.data.spaceSlug.toUpperCase()}</Text>
        <Text accessibilityRole="header" style={styles.title}>
          {recipe.data.name}
        </Text>
        {recipe.data.description ? <Text style={styles.description}>{recipe.data.description}</Text> : null}
        <View style={styles.card}>
          <Text style={styles.meta}>
            {(recipe.data.prepMinutes ?? 0) + (recipe.data.cookMinutes ?? 0)} minutes · {recipe.data.tags.join(" · ") || "No tags"}
          </Text>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: { backgroundColor: colors.canvas, flex: 1 },
  scroll: { alignSelf: "center", maxWidth: 760, padding: 20, width: "100%" },
  space: { color: colors.accent, fontSize: 12, fontWeight: "800", letterSpacing: 1.2 },
  title: { color: colors.ink, fontSize: 32, fontWeight: "700", marginTop: 7 },
  description: { color: colors.ink, fontSize: 16, lineHeight: 24, marginTop: 18 },
  card: { backgroundColor: colors.surface, borderRadius: 15, marginTop: 18, padding: 16 },
  meta: { color: colors.muted, fontSize: 14 },
});

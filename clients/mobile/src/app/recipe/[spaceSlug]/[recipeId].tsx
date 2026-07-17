import { createLibraryCommands, createQueries, type HorologiaClient, type ServerProfile } from "@horologia/client-core";
import type { components } from "@horologia/client-core/schema";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useMemo } from "react";
import { Alert, Pressable, ScrollView, Share, StyleSheet, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useSession } from "@/auth/session-context";
import { ScreenState } from "@/components/screen-state";
import { colors } from "@/design/tokens";

type Recipe = components["schemas"]["Recipe"];

export default function RecipeScreen() {
  const { spaceSlug, recipeId } = useLocalSearchParams<{ spaceSlug: string; recipeId: string }>();
  const session = useSession();
  if (!session.profile || !session.client || !spaceSlug || !recipeId) return <ScreenState loading title="Loading recipe" />;
  return <RecipeDetail client={session.client} profile={session.profile} recipeId={recipeId} spaceSlug={spaceSlug} />;
}

function RecipeDetail({ client, profile, recipeId, spaceSlug }: { client: HorologiaClient; profile: ServerProfile; recipeId: string; spaceSlug: string }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const queries = useMemo(() => createQueries({ serverId: profile.id, apiClient: client, appClient: client }), [client, profile.id]);
  const recipe = useQuery(queries.recipeQueryOptions(spaceSlug, recipeId));
  const activity = useInfiniteQuery(queries.recipeActivityInfiniteQueryOptions(spaceSlug, recipeId));
  const commands = createLibraryCommands({ serverId: profile.id, apiClient: client, queryClient });
  const deletion = useMutation({
    mutationFn: () => commands.deleteRecipe(spaceSlug, recipeId),
    onSuccess() { router.replace("/(tabs)/library"); },
  });
  if (recipe.isPending) return <ScreenState loading title="Loading recipe" />;
  if (recipe.isError) return <ScreenState detail={recipe.error.message} title="Recipe unavailable" />;
  const shareUrl = new URL(`spaces/${encodeURIComponent(spaceSlug)}/recipes/${encodeURIComponent(recipeId)}`, `${profile.baseUrl.replace(/\/+$/u, "")}/`).toString();
  return <RecipeView recipe={recipe.data} activity={activity.data?.pages.flatMap((page) => page.items) ?? []} onEdit={() => router.push({ pathname: "/recipe/[spaceSlug]/[recipeId]/edit", params: { spaceSlug, recipeId } })} onShare={() => void Share.share({ title: recipe.data.name, message: shareUrl, url: shareUrl })} onDelete={() => Alert.alert("Delete recipe?", "This cannot be undone.", [{ text: "Cancel", style: "cancel" }, { text: "Delete", style: "destructive", onPress: () => deletion.mutate() }])} />;
}

function RecipeView({ recipe, activity, onEdit, onShare, onDelete }: { recipe: Recipe; activity: components["schemas"]["ActivityLogEntry"][]; onEdit: () => void; onShare: () => void; onDelete: () => void }) {
  return (
    <SafeAreaView edges={["left", "right", "bottom"]} style={styles.safeArea}>
      <ScrollView contentContainerStyle={styles.scroll}>
        <Text style={styles.space}>{recipe.spaceSlug.toUpperCase()}</Text>
        <Text accessibilityRole="header" style={styles.title}>{recipe.name}</Text>
        <View style={styles.actions}>
          <Action label="Edit" onPress={onEdit} primary />
          <Action label="Share" onPress={onShare} />
          <Action label="Delete" onPress={onDelete} destructive />
        </View>
        <View style={styles.metrics}>
          <Metric label="Yield" value={recipe.yield ? `${recipe.yield.amount} ${recipe.yield.unit}` : "—"} />
          <Metric label="Prep" value={recipe.prepMinutes === null ? "—" : `${recipe.prepMinutes} min`} />
          <Metric label="Cook" value={recipe.cookMinutes === null ? "—" : `${recipe.cookMinutes} min`} />
        </View>
        {recipe.description ? <Section title="Description"><Text style={styles.body}>{recipe.description}</Text></Section> : null}
        <Section title="Ingredients">
          {recipe.ingredientSections.map((section, sectionIndex) => (
            <View key={`${section.title}/${sectionIndex}`} style={styles.subsection}>
              {section.title ? <Text style={styles.subtitle}>{section.title}</Text> : null}
              {section.ingredients.map((ingredient, index) => (
                <Text key={`${ingredient.item}/${index}`} style={styles.body}>• {formatIngredient(ingredient)}</Text>
              ))}
            </View>
          ))}
        </Section>
        <Section title="Instructions">
          {recipe.instructionSections.map((section, sectionIndex) => (
            <View key={`${section.title}/${sectionIndex}`} style={styles.subsection}>
              {section.title ? <Text style={styles.subtitle}>{section.title}</Text> : null}
              {section.steps.map((step, index) => <Text key={`${step.body}/${index}`} style={styles.body}>{index + 1}. {step.body}</Text>)}
            </View>
          ))}
        </Section>
        {recipe.tags.length ? <Text style={styles.tags}>{recipe.tags.join(" · ")}</Text> : null}
        <Section title="Activity">
          {activity.map((entry) => <View key={entry.id} style={styles.activity}><Text style={styles.body}>{entry.action} {entry.entityType}</Text><Text style={styles.meta}>{new Date(entry.createdAt).toLocaleString()}</Text></View>)}
          {activity.length === 0 ? <Text style={styles.meta}>No activity yet.</Text> : null}
        </Section>
      </ScrollView>
    </SafeAreaView>
  );
}

function Action({ label, onPress, primary = false, destructive = false }: { label: string; onPress: () => void; primary?: boolean; destructive?: boolean }) {
  return <Pressable accessibilityRole="button" onPress={onPress} style={[styles.action, primary && styles.primaryAction]}><Text style={[styles.actionText, primary && styles.primaryActionText, destructive && styles.destructive]}>{label}</Text></Pressable>;
}

function Metric({ label, value }: { label: string; value: string }) { return <View style={styles.metric}><Text style={styles.label}>{label}</Text><Text style={styles.metricValue}>{value}</Text></View>; }
function Section({ title, children }: { title: string; children: React.ReactNode }) { return <View style={styles.section}><Text accessibilityRole="header" style={styles.sectionTitle}>{title}</Text>{children}</View>; }
function formatIngredient(item: Recipe["ingredientSections"][number]["ingredients"][number]): string { const amount = item.quantity === null ? "" : item.quantityMax === null ? String(item.quantity) : `${item.quantity}–${item.quantityMax}`; return [amount, item.unit, item.item].filter(Boolean).join(" "); }

const styles = StyleSheet.create({
  safeArea: { backgroundColor: colors.canvas, flex: 1 },
  scroll: { alignSelf: "center", gap: 13, maxWidth: 800, padding: 20, paddingBottom: 48, width: "100%" },
  space: { color: colors.accent, fontSize: 12, fontWeight: "800", letterSpacing: 1.2 },
  title: { color: colors.ink, fontSize: 32, fontWeight: "700", marginTop: -5 },
  actions: { flexDirection: "row", gap: 8 },
  action: { borderRadius: 12, paddingHorizontal: 15, paddingVertical: 10 },
  primaryAction: { backgroundColor: colors.accent },
  actionText: { color: colors.accent, fontSize: 14, fontWeight: "700" },
  primaryActionText: { color: colors.surface },
  destructive: { color: colors.danger },
  metrics: { flexDirection: "row", gap: 9 },
  metric: { backgroundColor: colors.surface, borderRadius: 14, flex: 1, padding: 14 },
  label: { color: colors.muted, fontSize: 12, fontWeight: "700", textTransform: "uppercase" },
  metricValue: { color: colors.ink, fontSize: 16, fontWeight: "600", marginTop: 5 },
  section: { backgroundColor: colors.surface, borderRadius: 16, gap: 8, padding: 17 },
  sectionTitle: { color: colors.ink, fontSize: 19, fontWeight: "700" },
  subsection: { gap: 6, marginTop: 3 },
  subtitle: { color: colors.accent, fontSize: 15, fontWeight: "700" },
  body: { color: colors.ink, fontSize: 16, lineHeight: 24 },
  tags: { color: colors.accent, fontSize: 14, fontWeight: "600" },
  activity: { borderBottomColor: colors.outline, borderBottomWidth: StyleSheet.hairlineWidth, paddingVertical: 6 },
  meta: { color: colors.muted, fontSize: 13, marginTop: 2 },
});

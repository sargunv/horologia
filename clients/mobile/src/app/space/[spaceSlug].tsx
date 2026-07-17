import { createQueries, type HorologiaClient } from "@horologia/client-core";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useMemo } from "react";
import { Pressable, ScrollView, StyleSheet, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useSession } from "@/auth/session-context";
import { ScreenState } from "@/components/screen-state";
import { colors } from "@/design/tokens";

export default function SpaceScreen() {
  const { spaceSlug } = useLocalSearchParams<{ spaceSlug: string }>();
  const session = useSession();
  if (!session.profile || !session.client || !spaceSlug) return <ScreenState loading title="Opening space" />;
  return <SpaceDetail client={session.client} serverId={session.profile.id} spaceSlug={spaceSlug} />;
}

function SpaceDetail({ client, serverId, spaceSlug }: { client: HorologiaClient; serverId: string; spaceSlug: string }) {
  const router = useRouter();
  const queries = useMemo(() => createQueries({ serverId, apiClient: client, appClient: client }), [client, serverId]);
  const space = useQuery(queries.spaceQueryOptions(spaceSlug));
  const tasks = useInfiniteQuery(queries.spaceTasksInfiniteQueryOptions(spaceSlug));
  const recipes = useInfiniteQuery(queries.recipesInfiniteQueryOptions(spaceSlug));
  const activity = useInfiniteQuery(queries.spaceActivityInfiniteQueryOptions(spaceSlug));
  if (space.isPending || tasks.isPending || recipes.isPending) return <ScreenState loading title="Loading space" />;
  if (space.isError || tasks.isError || recipes.isError) return <ScreenState title="Space unavailable" detail="Try again when your server is reachable." />;
  const taskItems = tasks.data.pages.flatMap((page) => page.items);
  const recipeItems = recipes.data.pages.flatMap((page) => page.items);
  const activityItems = activity.data?.pages.flatMap((page) => page.items) ?? [];
  return (
    <SafeAreaView edges={["left", "right", "bottom"]} style={styles.safeArea}>
      <ScrollView contentContainerStyle={styles.scroll}>
        <View style={styles.hero}>
          <View style={styles.heroBody}>
            <Text accessibilityRole="header" style={styles.heading}>{space.data.name}</Text>
            <Text style={styles.detail}>{space.data.description || space.data.slug}</Text>
          </View>
          <Pressable accessibilityRole="button" onPress={() => router.push({ pathname: "/task/new", params: { spaceSlug } })} style={styles.button}>
            <Text style={styles.buttonText}>New task</Text>
          </Pressable>
        </View>
        <Section title="Tasks">
          {taskItems.map((task) => (
            <Row key={task.id} title={task.title} meta={task.status} onPress={() => router.push({ pathname: "/task/[spaceSlug]/[taskId]", params: { spaceSlug, taskId: task.id } })} />
          ))}
        </Section>
        <Section title="Recipes" action={<Pressable onPress={() => router.push({ pathname: "/recipe/new", params: { spaceSlug } })}><Text style={styles.link}>New recipe</Text></Pressable>}>
          {recipeItems.map((recipe) => (
            <Row key={recipe.id} title={recipe.name} meta={`${(recipe.prepMinutes ?? 0) + (recipe.cookMinutes ?? 0)} min`} onPress={() => router.push({ pathname: "/recipe/[spaceSlug]/[recipeId]", params: { spaceSlug, recipeId: recipe.id } })} />
          ))}
        </Section>
        <Section title="Activity">
          {activityItems.slice(0, 20).map((entry) => (
            <View key={entry.id} style={styles.activity}>
              <Text style={styles.rowTitle}>{entry.action} {entry.entityType}</Text>
              <Text style={styles.meta}>{new Date(entry.createdAt).toLocaleString()}</Text>
            </View>
          ))}
          {activityItems.length === 0 ? <Text style={styles.empty}>No activity yet.</Text> : null}
        </Section>
      </ScrollView>
    </SafeAreaView>
  );
}

function Section({ title, action, children }: { title: string; action?: React.ReactNode; children: React.ReactNode }) {
  return <View style={styles.section}><View style={styles.sectionHeader}><Text accessibilityRole="header" style={styles.sectionTitle}>{title}</Text>{action}</View>{children}</View>;
}

function Row({ title, meta, onPress }: { title: string; meta: string; onPress: () => void }) {
  return <Pressable accessibilityRole="button" onPress={onPress} style={styles.row}><View style={styles.heroBody}><Text style={styles.rowTitle}>{title}</Text><Text style={styles.meta}>{meta}</Text></View><Text style={styles.chevron}>›</Text></Pressable>;
}

const styles = StyleSheet.create({
  safeArea: { backgroundColor: colors.canvas, flex: 1 },
  scroll: { alignSelf: "center", gap: 14, maxWidth: 900, padding: 18, paddingBottom: 44, width: "100%" },
  hero: { alignItems: "center", flexDirection: "row", gap: 14 },
  heroBody: { flex: 1 },
  heading: { color: colors.ink, fontSize: 31, fontWeight: "700" },
  detail: { color: colors.muted, fontSize: 14, marginTop: 4 },
  button: { backgroundColor: colors.accent, borderRadius: 13, paddingHorizontal: 14, paddingVertical: 11 },
  buttonText: { color: colors.surface, fontWeight: "700" },
  section: { backgroundColor: colors.surface, borderRadius: 17, gap: 4, padding: 16 },
  sectionHeader: { alignItems: "center", flexDirection: "row", justifyContent: "space-between", marginBottom: 5 },
  sectionTitle: { color: colors.ink, fontSize: 20, fontWeight: "700" },
  link: { color: colors.accent, fontSize: 14, fontWeight: "700" },
  row: { alignItems: "center", borderBottomColor: colors.outline, borderBottomWidth: StyleSheet.hairlineWidth, flexDirection: "row", paddingVertical: 12 },
  rowTitle: { color: colors.ink, fontSize: 16, fontWeight: "600" },
  meta: { color: colors.muted, fontSize: 13, marginTop: 3 },
  chevron: { color: colors.muted, fontSize: 26 },
  activity: { borderBottomColor: colors.outline, borderBottomWidth: StyleSheet.hairlineWidth, paddingVertical: 10 },
  empty: { color: colors.muted, fontSize: 14, paddingVertical: 12 },
});

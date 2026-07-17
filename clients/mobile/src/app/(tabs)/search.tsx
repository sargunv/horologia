import { createQueries } from "@horologia/client-core";
import { useQuery } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import { useMemo, useState } from "react";
import { Pressable, ScrollView, StyleSheet, Text, TextInput, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useSession } from "@/auth/session-context";
import { ScreenState } from "@/components/screen-state";
import { colors } from "@/design/tokens";

export default function SearchScreen() {
  const session = useSession();
  if (!session.profile || !session.client) return <ScreenState loading title="Opening search" />;
  return <AuthenticatedSearch client={session.client} serverId={session.profile.id} />;
}

function AuthenticatedSearch({
  client,
  serverId,
}: {
  client: NonNullable<ReturnType<typeof useSession>["client"]>;
  serverId: string;
}) {
  const router = useRouter();
  const [search, setSearch] = useState("");
  const queries = useMemo(
    () => createQueries({ serverId, apiClient: client, appClient: client }),
    [client, serverId],
  );
  const enabled = search.trim().length >= 2;
  const tasks = useQuery({ ...queries.taskSearchQueryOptions({ query: search }), enabled });
  const recipes = useQuery({ ...queries.recipeSearchQueryOptions({ query: search }), enabled });

  return (
    <SafeAreaView edges={["left", "right"]} style={styles.safeArea}>
      <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
        <Text accessibilityRole="header" style={styles.heading}>
          Search
        </Text>
        <TextInput
          accessibilityLabel="Search tasks and recipes"
          autoCapitalize="none"
          autoCorrect={false}
          onChangeText={setSearch}
          placeholder="Tasks and recipes across every space"
          placeholderTextColor={colors.muted}
          style={styles.input}
          value={search}
        />
        {!enabled ? <Text style={styles.hint}>Type at least two characters.</Text> : null}
        {enabled ? (
          <>
            <ResultSection title="Tasks">
              {(tasks.data ?? []).map((task) => (
                <Result
                  key={`${task.spaceSlug}/${task.id}`}
                  meta={`${task.spaceSlug} · ${task.status}`}
                  onPress={() =>
                    router.push({
                      pathname: "/task/[spaceSlug]/[taskId]",
                      params: { spaceSlug: task.spaceSlug, taskId: task.id },
                    })
                  }
                  title={task.title}
                />
              ))}
              {!tasks.isPending && tasks.data?.length === 0 ? (
                <Text style={styles.hint}>No matching tasks.</Text>
              ) : null}
            </ResultSection>
            <ResultSection title="Recipes">
              {(recipes.data ?? []).map((recipe) => (
                <Result
                  key={`${recipe.spaceSlug}/${recipe.id}`}
                  meta={recipe.spaceSlug}
                  onPress={() =>
                    router.push({
                      pathname: "/recipe/[spaceSlug]/[recipeId]",
                      params: { spaceSlug: recipe.spaceSlug, recipeId: recipe.id },
                    })
                  }
                  title={recipe.name}
                />
              ))}
              {!recipes.isPending && recipes.data?.length === 0 ? (
                <Text style={styles.hint}>No matching recipes.</Text>
              ) : null}
            </ResultSection>
          </>
        ) : null}
      </ScrollView>
    </SafeAreaView>
  );
}

function ResultSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <View style={styles.section}>
      <Text accessibilityRole="header" style={styles.sectionTitle}>
        {title}
      </Text>
      {children}
    </View>
  );
}

function Result({ title, meta, onPress }: { title: string; meta: string; onPress: () => void }) {
  return (
    <Pressable accessibilityRole="button" onPress={onPress} style={styles.result}>
      <Text style={styles.resultTitle}>{title}</Text>
      <Text style={styles.resultMeta}>{meta}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  safeArea: { backgroundColor: colors.canvas, flex: 1 },
  scroll: { alignSelf: "center", gap: 14, maxWidth: 760, padding: 18, width: "100%" },
  heading: { color: colors.ink, fontSize: 30, fontWeight: "700" },
  input: {
    backgroundColor: colors.surface,
    borderColor: colors.outline,
    borderRadius: 15,
    borderWidth: StyleSheet.hairlineWidth,
    color: colors.ink,
    fontSize: 16,
    minHeight: 52,
    paddingHorizontal: 15,
  },
  section: { gap: 8 },
  sectionTitle: { color: colors.ink, fontSize: 19, fontWeight: "700", marginTop: 4 },
  result: { backgroundColor: colors.surface, borderRadius: 14, padding: 14 },
  resultTitle: { color: colors.ink, fontSize: 16, fontWeight: "600" },
  resultMeta: { color: colors.muted, fontSize: 13, marginTop: 4 },
  hint: { color: colors.muted, fontSize: 14 },
});

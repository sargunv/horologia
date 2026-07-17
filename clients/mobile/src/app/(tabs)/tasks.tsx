import {
  createQueries,
  projectMyTasksWidgetSnapshot,
  type HorologiaClient,
  type ServerProfile,
} from "@horologia/client-core";
import type { components } from "@horologia/client-core/schema";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import { useEffect, useMemo, useState } from "react";
import {
  ActivityIndicator,
  FlatList,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useSession } from "@/auth/session-context";
import { ScreenState } from "@/components/screen-state";
import { colors } from "@/design/tokens";
import { type CachedMyTasks, loadCachedMyTasks, saveCachedMyTasks } from "@/persistence/database";
import { publishWidgetSnapshot } from "@/widgets/publishWidgetSnapshot";

type Task = components["schemas"]["Task"];

export default function TasksScreen() {
  const session = useSession();
  if (!session.profile || !session.accountId || !session.client) {
    return <ScreenState loading title="Opening My Tasks" />;
  }
  return (
    <AuthenticatedTasks
      accountId={session.accountId}
      client={session.client}
      profile={session.profile}
    />
  );
}

function AuthenticatedTasks(props: {
  accountId: string;
  client: HorologiaClient;
  profile: ServerProfile;
}) {
  const router = useRouter();
  const [cache, setCache] = useState<CachedMyTasks | null>(null);
  const queries = useMemo(
    () =>
      createQueries({
        serverId: props.profile.id,
        apiClient: props.client,
        appClient: props.client,
      }),
    [props.client, props.profile.id],
  );
  const query = useInfiniteQuery(queries.userTasksInfiniteQueryOptions(props.accountId));
  const liveTasks = useMemo(
    () => query.data?.pages.flatMap((page) => page.items) ?? null,
    [query.data],
  );

  useEffect(() => {
    let active = true;
    async function hydrateCache() {
      const next = await loadCachedMyTasks(props.profile.id, props.accountId);
      if (active) setCache(next);
    }
    void hydrateCache();
    return () => {
      active = false;
    };
  }, [props.accountId, props.profile.id]);

  useEffect(() => {
    if (!liveTasks) return;
    const tasksToPersist = liveTasks;
    const generatedAt = new Date().toISOString();
    const nextCache = { tasks: tasksToPersist, updatedAt: generatedAt };
    setCache(nextCache);
    async function persistTasks() {
      await Promise.all([
        saveCachedMyTasks(props.profile.id, props.accountId, tasksToPersist, generatedAt),
        publishWidgetSnapshot(
          projectMyTasksWidgetSnapshot({
            serverId: props.profile.id,
            accountId: props.accountId,
            generatedAt,
            tasks: tasksToPersist,
          }),
        ),
      ]);
    }
    void persistTasks().catch((error: unknown) => {
      console.warn("Could not update the durable task snapshot", error);
    });
  }, [liveTasks, props.accountId, props.profile.id]);

  const showingCache = cache !== null && (liveTasks === null || query.isRefetchError);
  const tasks = liveTasks ?? cache?.tasks ?? [];

  if (query.isPending && !cache) {
    return <ScreenState loading title="Loading your tasks" detail="Across every space" />;
  }
  if (query.isError && !cache) {
    return (
      <ScreenState
        actionLabel="Try again"
        detail={errorMessage(query.error)}
        onAction={() => void query.refetch()}
        title="My Tasks couldn't load"
      />
    );
  }

  return (
    <SafeAreaView edges={["left", "right"]} style={styles.safeArea}>
      <FlatList
        accessibilityLabel="My Tasks"
        contentContainerStyle={[styles.list, tasks.length === 0 && styles.emptyList]}
        data={tasks}
        keyExtractor={(task) => `${task.spaceSlug}/${task.id}`}
        ListEmptyComponent={
          <View style={styles.empty}>
            <Text style={styles.emptyGlyph}>✓</Text>
            <Text style={styles.emptyTitle}>You're all caught up</Text>
            <Text style={styles.emptyDetail}>Tasks assigned to you will appear here.</Text>
          </View>
        }
        ListFooterComponent={
          query.isFetchingNextPage ? (
            <ActivityIndicator accessibilityLabel="Loading more tasks" style={styles.footer} />
          ) : null
        }
        ListHeaderComponentStyle={styles.headerFrame}
        onEndReached={() => {
          if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
        }}
        onEndReachedThreshold={0.35}
        refreshControl={
          <RefreshControl refreshing={query.isRefetching} onRefresh={() => void query.refetch()} />
        }
        renderItem={({ item }) => (
          <TaskRow
            onPress={() =>
              router.push({
                pathname: "/task/[spaceSlug]/[taskId]",
                params: { spaceSlug: item.spaceSlug, taskId: item.id },
              })
            }
            task={item}
          />
        )}
        ListHeaderComponent={
          <>
            <View style={styles.header}>
              <View>
                <Text accessibilityRole="header" style={styles.heading}>
                  My Tasks
                </Text>
                <Text style={styles.subheading}>
                  {tasks.length === 1
                    ? "1 task assigned to you"
                    : `${tasks.length} tasks assigned to you`}
                </Text>
              </View>
              <View style={styles.headerActions}>
                {query.isFetching && !query.isFetchingNextPage ? (
                  <ActivityIndicator accessibilityLabel="Refreshing tasks" />
                ) : null}
                <Pressable
                  accessibilityLabel="Create task"
                  accessibilityRole="button"
                  onPress={() => router.push("/task/new")}
                  style={styles.addButton}
                >
                  <Text style={styles.addButtonText}>＋</Text>
                </Pressable>
              </View>
            </View>
            {showingCache ? (
              <View accessibilityRole="alert" style={styles.staleBanner}>
                <Text style={styles.staleText}>
                  Offline · showing tasks saved {formatCachedAt(cache.updatedAt)}
                </Text>
              </View>
            ) : null}
          </>
        }
      />
    </SafeAreaView>
  );
}

function TaskRow({ task, onPress }: { task: Task; onPress: () => void }) {
  const due = task.due ? describeDue(task.due.at) : null;
  return (
    <Pressable
      accessibilityHint="Opens task details"
      accessibilityLabel={`${task.title}, ${task.status}${due ? `, ${due.label}` : ""}`}
      accessibilityRole="button"
      onPress={onPress}
      style={({ pressed }) => [styles.row, pressed && styles.rowPressed]}
    >
      <View style={styles.rowBody}>
        <Text numberOfLines={2} style={styles.taskTitle}>
          {task.title}
        </Text>
        <View style={styles.metadata}>
          <Text style={styles.space}>{task.spaceSlug}</Text>
          <Text style={styles.separator}>·</Text>
          <Text style={styles.status}>{task.status}</Text>
          {due ? (
            <>
              <Text style={styles.separator}>·</Text>
              <Text style={due.overdue ? styles.overdue : styles.due}>{due.label}</Text>
            </>
          ) : null}
        </View>
      </View>
      <Text accessibilityElementsHidden style={styles.chevron}>
        ›
      </Text>
    </Pressable>
  );
}

function describeDue(value: string): { label: string; overdue: boolean } {
  const today = new Date();
  const todayKey = [today.getFullYear(), today.getMonth() + 1, today.getDate()]
    .map((part, index) => part.toString().padStart(index === 0 ? 4 : 2, "0"))
    .join("-");
  const parsed = new Date(`${value}T12:00:00`);
  const date = parsed.toLocaleDateString(undefined, { month: "short", day: "numeric" });
  if (value < todayKey) return { label: `Overdue ${date}`, overdue: true };
  if (value === todayKey) return { label: "Due today", overdue: false };
  return { label: `Due ${date}`, overdue: false };
}

function formatCachedAt(value: string): string {
  return new Date(value).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
}

function errorMessage(error: Error): string {
  return error.message || "Check your connection and try again.";
}

const styles = StyleSheet.create({
  safeArea: { backgroundColor: colors.canvas, flex: 1 },
  list: { alignSelf: "center", maxWidth: 760, padding: 16, paddingBottom: 40, width: "100%" },
  emptyList: { flexGrow: 1 },
  headerFrame: { marginBottom: 12 },
  header: { alignItems: "center", flexDirection: "row", justifyContent: "space-between" },
  heading: { color: colors.ink, fontSize: 30, fontWeight: "700", letterSpacing: -0.7 },
  subheading: { color: colors.muted, fontSize: 14, marginTop: 3 },
  headerActions: { alignItems: "center", flexDirection: "row", gap: 10 },
  addButton: {
    alignItems: "center",
    backgroundColor: colors.accent,
    borderRadius: 999,
    height: 42,
    justifyContent: "center",
    width: 42,
  },
  addButtonText: { color: colors.surface, fontSize: 23, fontWeight: "500", lineHeight: 26 },
  staleBanner: { backgroundColor: colors.accentSoft, borderRadius: 12, marginTop: 12, padding: 10 },
  staleText: { color: colors.accent, fontSize: 13, fontWeight: "600" },
  row: {
    alignItems: "center",
    backgroundColor: colors.surface,
    borderColor: colors.outline,
    borderRadius: 16,
    borderWidth: StyleSheet.hairlineWidth,
    flexDirection: "row",
    marginBottom: 10,
    minHeight: 76,
    paddingHorizontal: 16,
    paddingVertical: 13,
  },
  rowPressed: { opacity: 0.66 },
  rowBody: { flex: 1 },
  taskTitle: { color: colors.ink, fontSize: 17, fontWeight: "600", lineHeight: 22 },
  metadata: { alignItems: "center", flexDirection: "row", flexWrap: "wrap", marginTop: 7 },
  space: { color: colors.accent, fontSize: 13, fontWeight: "600" },
  separator: { color: colors.outline, marginHorizontal: 6 },
  status: { color: colors.muted, fontSize: 13 },
  due: { color: colors.muted, fontSize: 13 },
  overdue: { color: colors.danger, fontSize: 13, fontWeight: "600" },
  chevron: { color: colors.muted, fontSize: 29, marginLeft: 12 },
  empty: { alignItems: "center", flex: 1, justifyContent: "center", padding: 32 },
  emptyGlyph: { color: colors.accent, fontSize: 48, fontWeight: "300" },
  emptyTitle: { color: colors.ink, fontSize: 21, fontWeight: "700", marginTop: 14 },
  emptyDetail: { color: colors.muted, fontSize: 15, marginTop: 7, textAlign: "center" },
  footer: { padding: 18 },
});

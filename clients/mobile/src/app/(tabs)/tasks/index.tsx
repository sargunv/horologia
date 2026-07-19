import { createQueries, type HorologiaClient, type ServerProfile } from "@horologia/client-core";
import type { components } from "@horologia/client-core/schema";
import { useInfiniteQuery } from "@tanstack/react-query";
import { Button, Host, List, ListItem, Text } from "@expo/ui";
import { useRouter } from "expo-router";
import { useEffect, useMemo, useState } from "react";

import { useSession } from "@/auth/session-context";
import { ScreenState } from "@/components/screen-state";
import { type CachedMyTasks, loadCachedMyTasks } from "@/persistence/database";
import { saveMyTasksSnapshot } from "@/widgets/saveMyTasksSnapshot";

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
    void saveMyTasksSnapshot({
      profile: props.profile,
      accountId: props.accountId,
      tasks: tasksToPersist,
      hasMore: query.hasNextPage,
    })
      .then(setCache)
      .catch((error: unknown) => {
        console.warn("Could not update the durable task snapshot", error);
      });
  }, [liveTasks, props.accountId, props.profile, query.hasNextPage]);

  const showingCache = cache !== null && (liveTasks === null || query.isRefetchError);
  const tasks = liveTasks ?? cache?.tasks ?? [];
  const hasMore = liveTasks ? query.hasNextPage : (cache?.hasMore ?? false);

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
    <Host style={{ flex: 1 }} useViewportSizeMeasurement>
      <List onRefresh={async () => void (await query.refetch())}>
        <ListItem
          supportingText={
            query.isFetching && !query.isFetchingNextPage
              ? "Refreshing…"
              : tasks.length === 1
                ? `1${hasMore ? "+" : ""} task assigned to you`
                : `${tasks.length}${hasMore ? "+" : ""} tasks assigned to you`
          }
          trailing={
            <Button label="New task" onPress={() => router.push("/task/new")} variant="text" />
          }
        >
          <Text>My Tasks</Text>
        </ListItem>
        {showingCache ? (
          <ListItem supportingText={`Saved ${formatCachedAt(cache.updatedAt)}`}>
            <Text>Offline copy</Text>
          </ListItem>
        ) : null}
        {tasks.map((item) => (
          <TaskRow
            key={`${item.spaceSlug}/${item.id}`}
            onPress={() =>
              router.push({
                pathname: "/task/[spaceSlug]/[taskId]",
                params: { spaceSlug: item.spaceSlug, taskId: item.id },
              })
            }
            task={item}
          />
        ))}
        {tasks.length === 0 ? (
          <ListItem supportingText="Tasks assigned to you will appear here.">
            <Text>You're all caught up</Text>
          </ListItem>
        ) : null}
        {query.hasNextPage ? (
          <ListItem
            onPress={() => {
              if (!query.isFetchingNextPage) void query.fetchNextPage();
            }}
            supportingText={query.isFetchingNextPage ? "Loading…" : undefined}
          >
            <Text>Load more</Text>
          </ListItem>
        ) : null}
      </List>
    </Host>
  );
}

function TaskRow({ task, onPress }: { task: Task; onPress: () => void }) {
  const due = task.due ? describeDue(task.due.at) : null;
  return (
    <ListItem
      onPress={onPress}
      supportingText={`${task.spaceSlug} · ${task.status}${due ? ` · ${due}` : ""}`}
    >
      <Text numberOfLines={2}>{task.title}</Text>
    </ListItem>
  );
}

function describeDue(value: string): string {
  const today = new Date();
  const todayKey = [today.getFullYear(), today.getMonth() + 1, today.getDate()]
    .map((part, index) => part.toString().padStart(index === 0 ? 4 : 2, "0"))
    .join("-");
  const parsed = new Date(`${value}T12:00:00`);
  const date = parsed.toLocaleDateString(undefined, { month: "short", day: "numeric" });
  if (value < todayKey) return `Overdue ${date}`;
  if (value === todayKey) return "Due today";
  return `Due ${date}`;
}

function formatCachedAt(value: string): string {
  return new Date(value).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
}

function errorMessage(error: Error): string {
  return error.message || "Check your connection and try again.";
}

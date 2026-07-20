import { useInfiniteQuery } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useState } from "react";

import type { Task } from "@/components/native-views.types";
import { errorMessage, formatTimestamp, isStaleTimestamp } from "@/components/read-model";
import { useSession } from "@/auth/session-context";
import { loadCachedMyTasks, type CachedMyTasks } from "@/persistence/database";
import { useAppRuntime } from "@/runtime/app-runtime";
import { selectMyTasksSource } from "@/runtime/my-tasks-controller";
import { saveMyTasksSnapshot } from "@/widgets/saveMyTasksSnapshot";

export interface MyTasksReadModel {
  tasks: Task[];
  source: "network" | "cache";
  timestamp: string | null;
  isStale: boolean;
  isInitialLoading: boolean;
  isRefreshing: boolean;
  isLoadingMore: boolean;
  initialError: string | null;
  loadMoreError: string | null;
  canLoadMore: boolean;
  cachedHasMore: boolean;
  refresh: () => Promise<void>;
  loadMore: () => Promise<void>;
  retry: () => Promise<void>;
}

export function useMyTasks(): MyTasksReadModel {
  const session = useSession();
  const runtime = useAppRuntime();
  if (session.status !== "signed-in") throw new Error("My Tasks requires an active session");
  const profile = session.profile;
  const accountId = session.accountId;
  const [cache, setCache] = useState<CachedMyTasks | null>(null);
  const [cacheLoaded, setCacheLoaded] = useState(false);

  useEffect(() => {
    let active = true;
    void (async () => {
      try {
        const next = await loadCachedMyTasks(profile.id, accountId);
        if (active) setCache(next);
      } finally {
        if (active) setCacheLoaded(true);
      }
    })();
    return () => {
      active = false;
    };
  }, [accountId, profile.id]);

  const query = useInfiniteQuery(runtime.queries.userTasksInfiniteQueryOptions(accountId));
  const { fetchNextPage, refetch } = query;
  const relevantCache = query.data ? null : cache;
  const selection = useMemo(
    () =>
      selectMyTasksSource({
        networkPages: query.data?.pages,
        cache: relevantCache,
        queryHasNextPage: query.hasNextPage,
      }),
    [query.data?.pages, query.hasNextPage, relevantCache],
  );
  const networkTasks = selection.source === "network" ? selection.tasks : null;
  const networkHasMore = selection.networkNextCursor !== null;

  useEffect(() => {
    if (networkTasks === null) return;
    let active = true;
    void (async () => {
      try {
        const saved = await saveMyTasksSnapshot({
          profile,
          accountId,
          tasks: networkTasks,
          hasMore: networkHasMore,
          isCurrent: () => active,
        });
        if (!active) return;
        setCache(saved);
      } catch {
        // The accepted network response remains authoritative if local snapshot persistence fails.
      }
    })();
    return () => {
      active = false;
    };
  }, [accountId, networkHasMore, networkTasks, profile, query.dataUpdatedAt]);

  const timestampValue =
    selection.source === "cache" ? (cache?.updatedAt ?? null) : query.dataUpdatedAt || null;
  const initialQueryError = !query.isFetchNextPageError && query.error ? query.error : null;

  const refresh = useCallback(async () => {
    await refetch();
  }, [refetch]);
  const loadMore = useCallback(async () => {
    if (selection.canLoadMore) await fetchNextPage();
  }, [fetchNextPage, selection.canLoadMore]);

  return {
    tasks: selection.tasks,
    source: selection.source,
    timestamp: timestampValue ? formatTimestamp(timestampValue) : null,
    isStale: timestampValue ? isStaleTimestamp(timestampValue) : false,
    isInitialLoading: !cacheLoaded || (query.isPending && !cache),
    isRefreshing: query.isRefetching && !query.isFetchingNextPage,
    isLoadingMore: query.isFetchingNextPage,
    initialError: initialQueryError
      ? errorMessage(initialQueryError, "Could not load your tasks.")
      : null,
    loadMoreError: query.isFetchNextPageError
      ? errorMessage(query.error, "Could not load more tasks.")
      : null,
    canLoadMore: selection.canLoadMore,
    cachedHasMore: selection.cachedHasMore,
    refresh,
    loadMore,
    retry: refresh,
  };
}

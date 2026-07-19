import { infiniteQueryOptions, queryOptions } from "@tanstack/react-query";
import type { HorologiaClient } from "../api/client";
import type { components } from "../api/schema.d.ts";
import { createQueryKeys } from "./queryKeys";

export type AuthConfig = components["schemas"]["AuthConfig"];
export type LinkPendingInfo = components["schemas"]["AuthLinkPendingResponse"];

export interface QueryContext {
  serverId: string;
  apiClient: HorologiaClient;
}

export interface AppQueryContext {
  serverId: string;
  appClient: HorologiaClient;
}

export function createAppQueries({ serverId, appClient }: AppQueryContext) {
  const keys = createQueryKeys(serverId);
  const authConfigQueryOptions = queryOptions({
    queryKey: keys.authConfig,
    queryFn: async (): Promise<AuthConfig> => {
      const { data, error } = await appClient.GET("/app/auth/config");
      if (error) throw error;
      return data;
    },
    staleTime: Infinity,
  });

  const linkPendingQueryOptions = queryOptions({
    queryKey: keys.linkPending,
    queryFn: async (): Promise<LinkPendingInfo | null> => {
      const { data, error, response } = await appClient.GET("/app/auth/link/pending");
      if (response.status === 404) return null;
      if (error) throw error;
      return data;
    },
    retry: false,
    staleTime: Infinity,
  });

  return { authConfigQueryOptions, linkPendingQueryOptions };
}

export function createQueries({ serverId, apiClient }: QueryContext) {
  const keys = createQueryKeys(serverId);

  const currentUserQueryOptions = queryOptions({
    queryKey: keys.currentUser,
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/users/me");
      if (error) throw error;
      return data;
    },
  });

  const serverInfoQueryOptions = queryOptions({
    queryKey: keys.serverInfo,
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/server-info");
      if (error) throw error;
      return data;
    },
    staleTime: Infinity,
  });

  const usersQueryOptions = queryOptions({
    queryKey: keys.users,
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/users");
      if (error) throw error;
      return data.items;
    },
  });

  const userQueryOptions = (userId: string) =>
    queryOptions({
      queryKey: keys.user(userId),
      queryFn: async () => {
        const { data, error } = await apiClient.GET("/users/{userId}", {
          params: { path: { userId } },
        });
        if (error) throw error;
        return data;
      },
    });

  const spacesQueryOptions = queryOptions({
    queryKey: keys.spaces,
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/spaces");
      if (error) throw error;
      return data.items;
    },
  });

  const spaceQueryOptions = (spaceSlug: string) =>
    queryOptions({
      queryKey: keys.space(spaceSlug),
      queryFn: async () => {
        const { data, error } = await apiClient.GET("/spaces/{spaceSlug}", {
          params: { path: { spaceSlug } },
        });
        if (error) throw error;
        return data;
      },
    });

  const spaceMembersQueryOptions = (spaceSlug: string) =>
    queryOptions({
      queryKey: keys.spaceMembers(spaceSlug),
      queryFn: async () => {
        const { data, error } = await apiClient.GET("/spaces/{spaceSlug}/members", {
          params: { path: { spaceSlug } },
        });
        if (error) throw error;
        return data.items;
      },
    });

  const spaceTaskStatusesQueryOptions = (spaceSlug: string) =>
    queryOptions({
      queryKey: keys.spaceTaskStatuses(spaceSlug),
      queryFn: async () => {
        const { data, error } = await apiClient.GET("/spaces/{spaceSlug}/task-statuses", {
          params: { path: { spaceSlug } },
        });
        if (error) throw error;
        return data.items;
      },
    });

  const spaceEffortLevelsQueryOptions = (spaceSlug: string) =>
    queryOptions({
      queryKey: keys.spaceEffortLevels(spaceSlug),
      queryFn: async () => {
        const { data, error } = await apiClient.GET("/spaces/{spaceSlug}/task-effort-levels", {
          params: { path: { spaceSlug } },
        });
        if (error) throw error;
        return data.items;
      },
    });

  const spacePriorityLevelsQueryOptions = (spaceSlug: string) =>
    queryOptions({
      queryKey: keys.spacePriorityLevels(spaceSlug),
      queryFn: async () => {
        const { data, error } = await apiClient.GET("/spaces/{spaceSlug}/task-priority-levels", {
          params: { path: { spaceSlug } },
        });
        if (error) throw error;
        return data.items;
      },
    });

  const spaceTagsQueryOptions = (spaceSlug: string) =>
    queryOptions({
      queryKey: keys.spaceTags(spaceSlug),
      queryFn: async () => {
        const { data, error } = await apiClient.GET("/spaces/{spaceSlug}/tags", {
          params: { path: { spaceSlug } },
        });
        if (error) throw error;
        return data.items;
      },
    });

  const spaceTasksInfiniteQueryOptions = (spaceSlug: string) => {
    const initialPageParam: string | null = null;
    return infiniteQueryOptions({
      queryKey: keys.spaceTasks(spaceSlug),
      queryFn: async ({ pageParam }: { pageParam: string | null }) => {
        const { data, error } = await apiClient.GET("/spaces/{spaceSlug}/tasks", {
          params: {
            path: { spaceSlug },
            query: { ...(pageParam ? { cursor: pageParam } : {}), limit: 50 },
          },
        });
        if (error) throw error;
        return data;
      },
      initialPageParam,
      getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    });
  };

  const authTokensQueryOptions = queryOptions({
    queryKey: keys.authTokens,
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/auth/tokens");
      if (error) throw error;
      return data.items;
    },
  });

  const spaceTaskQueryOptions = (spaceSlug: string, taskId: string) =>
    queryOptions({
      queryKey: keys.spaceTask(spaceSlug, taskId),
      queryFn: async () => {
        const { data, error } = await apiClient.GET("/spaces/{spaceSlug}/tasks/{taskId}", {
          params: { path: { spaceSlug, taskId } },
        });
        if (error) throw error;
        return data;
      },
    });

  const recipesInfiniteQueryOptions = (spaceSlug?: string) => {
    const initialPageParam: string | null = null;
    return infiniteQueryOptions({
      queryKey: keys.recipeList(spaceSlug),
      queryFn: async ({ pageParam }: { pageParam: string | null }) => {
        const { data, error } = await apiClient.GET("/recipes", {
          params: {
            query: {
              ...(spaceSlug ? { spaceSlug } : {}),
              ...(pageParam ? { cursor: pageParam } : {}),
              limit: 50,
            },
          },
        });
        if (error) throw error;
        return data;
      },
      initialPageParam,
      getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    });
  };

  const recipeQueryOptions = (spaceSlug: string, recipeId: string) =>
    queryOptions({
      queryKey: keys.recipe(spaceSlug, recipeId),
      queryFn: async () => {
        const { data, error } = await apiClient.GET("/spaces/{spaceSlug}/recipes/{recipeId}", {
          params: { path: { spaceSlug, recipeId } },
        });
        if (error) throw error;
        return data;
      },
    });

  const recipeActivityInfiniteQueryOptions = (spaceSlug: string, recipeId: string) => {
    const initialPageParam: string | null = null;
    return infiniteQueryOptions({
      queryKey: keys.recipeActivity(spaceSlug, recipeId),
      queryFn: async ({ pageParam }: { pageParam: string | null }) => {
        const { data, error } = await apiClient.GET(
          "/spaces/{spaceSlug}/recipes/{recipeId}/activity",
          {
            params: {
              path: { spaceSlug, recipeId },
              query: { ...(pageParam ? { cursor: pageParam } : {}), limit: 50 },
            },
          },
        );
        if (error) throw error;
        return data;
      },
      initialPageParam,
      getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    });
  };

  const recipeSearchQueryOptions = ({
    query,
    spaceSlug,
    limit = 10,
  }: {
    query: string;
    spaceSlug?: string;
    limit?: number;
  }) =>
    queryOptions({
      queryKey: keys.recipeSearch(query, spaceSlug, limit),
      queryFn: async () => {
        const { data, error } = await apiClient.GET("/recipes/search", {
          params: {
            query: { q: query, ...(spaceSlug ? { spaceSlug } : {}), limit },
          },
        });
        if (error) throw error;
        return data.items;
      },
      staleTime: 10_000,
    });

  const userTasksInfiniteQueryOptions = (userId: string) => {
    const initialPageParam: string | null = null;
    return infiniteQueryOptions({
      queryKey: keys.userTasks(userId),
      queryFn: async ({ pageParam }: { pageParam: string | null }) => {
        const { data, error } = await apiClient.GET("/users/{userId}/tasks", {
          params: {
            path: { userId },
            query: { ...(pageParam ? { cursor: pageParam } : {}), limit: 50 },
          },
        });
        if (error) throw error;
        return data;
      },
      initialPageParam,
      getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    });
  };

  const taskActivityInfiniteQueryOptions = (spaceSlug: string, taskId: string) => {
    const initialPageParam: string | null = null;
    return infiniteQueryOptions({
      queryKey: keys.taskActivity(spaceSlug, taskId),
      queryFn: async ({ pageParam }: { pageParam: string | null }) => {
        const { data, error } = await apiClient.GET("/spaces/{spaceSlug}/tasks/{taskId}/activity", {
          params: {
            path: { spaceSlug, taskId },
            query: { ...(pageParam ? { cursor: pageParam } : {}), limit: 50 },
          },
        });
        if (error) throw error;
        return data;
      },
      initialPageParam,
      getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    });
  };

  const taskSearchQueryOptions = ({
    query,
    spaceSlug,
    excludeTaskId,
    limit = 10,
  }: {
    query: string;
    spaceSlug?: string;
    excludeTaskId?: string;
    limit?: number;
  }) =>
    queryOptions({
      queryKey: keys.taskSearch(query, spaceSlug, excludeTaskId, limit),
      queryFn: async () => {
        const { data, error } = await apiClient.GET("/tasks/search", {
          params: {
            query: {
              q: query,
              ...(spaceSlug ? { spaceSlug } : {}),
              ...(excludeTaskId ? { excludeTaskId } : {}),
              limit,
            },
          },
        });
        if (error) throw error;
        return data.items;
      },
      staleTime: 10_000,
    });

  const spaceActivityInfiniteQueryOptions = (spaceSlug: string) => {
    const initialPageParam: string | null = null;
    return infiniteQueryOptions({
      queryKey: keys.spaceActivity(spaceSlug),
      queryFn: async ({ pageParam }: { pageParam: string | null }) => {
        const { data, error } = await apiClient.GET("/spaces/{spaceSlug}/activity", {
          params: {
            path: { spaceSlug },
            query: { ...(pageParam ? { cursor: pageParam } : {}), limit: 50 },
          },
        });
        if (error) throw error;
        return data;
      },
      initialPageParam,
      getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    });
  };

  const userActivityInfiniteQueryOptions = (userId: string) => {
    const initialPageParam: string | null = null;
    return infiniteQueryOptions({
      queryKey: keys.userActivity(userId),
      queryFn: async ({ pageParam }: { pageParam: string | null }) => {
        const { data, error } = await apiClient.GET("/users/{userId}/activity", {
          params: {
            path: { userId },
            query: { ...(pageParam ? { cursor: pageParam } : {}), limit: 50 },
          },
        });
        if (error) throw error;
        return data;
      },
      initialPageParam,
      getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    });
  };

  return {
    currentUserQueryOptions,
    serverInfoQueryOptions,
    usersQueryOptions,
    userQueryOptions,
    spacesQueryOptions,
    spaceQueryOptions,
    spaceMembersQueryOptions,
    spaceTaskStatusesQueryOptions,
    spaceEffortLevelsQueryOptions,
    spacePriorityLevelsQueryOptions,
    spaceTagsQueryOptions,
    spaceTasksInfiniteQueryOptions,
    authTokensQueryOptions,
    spaceTaskQueryOptions,
    recipesInfiniteQueryOptions,
    recipeQueryOptions,
    recipeActivityInfiniteQueryOptions,
    recipeSearchQueryOptions,
    userTasksInfiniteQueryOptions,
    taskActivityInfiniteQueryOptions,
    taskSearchQueryOptions,
    spaceActivityInfiniteQueryOptions,
    userActivityInfiniteQueryOptions,
  };
}

/** Query options surface returned by {@link createQueries}. */
export type Queries = ReturnType<typeof createQueries>;

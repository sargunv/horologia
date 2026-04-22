import { infiniteQueryOptions, queryOptions } from "@tanstack/react-query";
import { apiClient, appClient } from "../api/client.ts";
import type { components } from "../api/schema.d.ts";

export type AuthConfig = components["schemas"]["AuthConfig"];

export const authConfigQueryOptions = queryOptions({
  queryKey: ["authConfig"],
  queryFn: async (): Promise<AuthConfig> => {
    const { data, error } = await appClient.GET("/app/auth/config");
    if (error) throw error;
    return data;
  },
  staleTime: Infinity,
});

export type LinkPendingInfo = components["schemas"]["AuthLinkPendingResponse"];

export const linkPendingQueryOptions = queryOptions({
  queryKey: ["linkPending"],
  queryFn: async (): Promise<LinkPendingInfo | null> => {
    const { data, error, response } = await appClient.GET("/app/auth/link/pending");
    if (response.status === 404) return null;
    if (error) throw error;
    return data;
  },
  retry: false,
  staleTime: Infinity,
});

export const currentUserQueryOptions = queryOptions({
  queryKey: ["currentUser"],
  queryFn: async () => {
    const { data, error } = await apiClient.GET("/users/me");
    if (error) throw error;
    return data;
  },
});

export const usersQueryOptions = queryOptions({
  queryKey: ["users"],
  queryFn: async () => {
    const { data, error } = await apiClient.GET("/users");
    if (error) throw error;
    return data.items;
  },
});

export const userQueryOptions = (userId: string) =>
  queryOptions({
    queryKey: ["users", userId],
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/users/{userId}", {
        params: { path: { userId } },
      });
      if (error) throw error;
      return data;
    },
  });

export const spacesQueryOptions = queryOptions({
  queryKey: ["spaces"],
  queryFn: async () => {
    const { data, error } = await apiClient.GET("/spaces");
    if (error) throw error;
    return data.items;
  },
});

export const spaceQueryOptions = (spaceSlug: string) =>
  queryOptions({
    queryKey: ["spaces", spaceSlug],
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/spaces/{spaceSlug}", {
        params: { path: { spaceSlug } },
      });
      if (error) throw error;
      return data;
    },
  });

export const spaceMembersQueryOptions = (spaceSlug: string) =>
  queryOptions({
    queryKey: ["spaces", spaceSlug, "members"],
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/spaces/{spaceSlug}/members", {
        params: { path: { spaceSlug } },
      });
      if (error) throw error;
      return data.items;
    },
  });

export const spaceTaskStatusesQueryOptions = (spaceSlug: string) =>
  queryOptions({
    queryKey: ["spaces", spaceSlug, "taskStatuses"],
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/spaces/{spaceSlug}/task-statuses", {
        params: { path: { spaceSlug } },
      });
      if (error) throw error;
      return data.items;
    },
  });

export const spaceEffortLevelsQueryOptions = (spaceSlug: string) =>
  queryOptions({
    queryKey: ["spaces", spaceSlug, "effortLevels"],
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/spaces/{spaceSlug}/task-effort-levels", {
        params: { path: { spaceSlug } },
      });
      if (error) throw error;
      return data.items;
    },
  });

export const spacePriorityLevelsQueryOptions = (spaceSlug: string) =>
  queryOptions({
    queryKey: ["spaces", spaceSlug, "priorityLevels"],
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/spaces/{spaceSlug}/task-priority-levels", {
        params: { path: { spaceSlug } },
      });
      if (error) throw error;
      return data.items;
    },
  });

export const spaceTagsQueryOptions = (spaceSlug: string) =>
  queryOptions({
    queryKey: ["spaces", spaceSlug, "tags"],
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/spaces/{spaceSlug}/tags", {
        params: { path: { spaceSlug } },
      });
      if (error) throw error;
      return data.items;
    },
  });

export const spaceTasksInfiniteQueryOptions = (spaceSlug: string) => {
  const initialPageParam: string | null = null;
  return infiniteQueryOptions({
    queryKey: ["spaces", spaceSlug, "tasks", "list"],
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

export const authTokensQueryOptions = queryOptions({
  queryKey: ["authTokens"],
  queryFn: async () => {
    const { data, error } = await apiClient.GET("/auth/tokens");
    if (error) throw error;
    return data.items;
  },
});

export const spaceTaskQueryOptions = (spaceSlug: string, taskId: string) =>
  queryOptions({
    queryKey: ["spaces", spaceSlug, "tasks", taskId],
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/spaces/{spaceSlug}/tasks/{taskId}", {
        params: { path: { spaceSlug, taskId } },
      });
      if (error) throw error;
      return data;
    },
  });

export const userTasksInfiniteQueryOptions = (userId: string) => {
  const initialPageParam: string | null = null;
  return infiniteQueryOptions({
    queryKey: ["users", userId, "tasks", "list"],
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

export const taskActivityInfiniteQueryOptions = (spaceSlug: string, taskId: string) => {
  const initialPageParam: string | null = null;
  return infiniteQueryOptions({
    queryKey: ["spaces", spaceSlug, "tasks", taskId, "activity"],
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

export const taskSearchQueryOptions = ({
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
    queryKey: ["tasks", "search", query, spaceSlug ?? null, excludeTaskId ?? null, limit],
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

export const spaceActivityInfiniteQueryOptions = (spaceSlug: string) => {
  const initialPageParam: string | null = null;
  return infiniteQueryOptions({
    queryKey: ["spaces", spaceSlug, "activity"],
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

export const userActivityInfiniteQueryOptions = (userId: string) => {
  const initialPageParam: string | null = null;
  return infiniteQueryOptions({
    queryKey: ["users", userId, "activity"],
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

import { infiniteQueryOptions, queryOptions } from "@tanstack/react-query";
import * as v from "valibot";
import { apiClient } from "../api/client.ts";

const AuthConfigSchema = v.object({
  oidc: v.object({
    enabled: v.boolean(),
    label: v.string(),
    autoRedirect: v.boolean(),
  }),
  password: v.object({
    enabled: v.boolean(),
  }),
});

export type AuthConfig = v.InferOutput<typeof AuthConfigSchema>;

export const authConfigQueryOptions = queryOptions({
  queryKey: ["authConfig"],
  queryFn: async (): Promise<AuthConfig> => {
    const res = await fetch("/api/auth/config");
    if (!res.ok) throw new Error("Failed to fetch auth config");
    const raw: unknown = await res.json();
    return v.parse(AuthConfigSchema, raw);
  },
  staleTime: Infinity,
});

const LinkPendingInfoSchema = v.object({
  email: v.string(),
  name: v.string(),
});

export type LinkPendingInfo = v.InferOutput<typeof LinkPendingInfoSchema>;

export const linkPendingQueryOptions = queryOptions({
  queryKey: ["linkPending"],
  queryFn: async (): Promise<LinkPendingInfo | null> => {
    const res = await fetch("/api/auth/link/pending", { credentials: "include" });
    if (res.status === 404) return null;
    if (!res.ok) throw new Error("Failed to fetch link state");
    const raw: unknown = await res.json();
    return v.parse(LinkPendingInfoSchema, raw);
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

import { infiniteQueryOptions, queryOptions } from "@tanstack/react-query";
import { apiClient } from "../api/client.ts";

export interface AuthConfig {
  oidc: {
    enabled: boolean;
    label: string;
    autoRedirect: boolean;
  };
  password: {
    enabled: boolean;
  };
}

export const authConfigQueryOptions = queryOptions({
  queryKey: ["authConfig"],
  queryFn: async (): Promise<AuthConfig> => {
    const res = await fetch("/api/auth/config");
    if (!res.ok) throw new Error("Failed to fetch auth config");
    return res.json();
  },
  staleTime: Infinity,
});

export interface LinkPendingInfo {
  email: string;
  name: string;
}

export const linkPendingQueryOptions = queryOptions({
  queryKey: ["linkPending"],
  queryFn: async (): Promise<LinkPendingInfo | null> => {
    const res = await fetch("/api/auth/link/pending", { credentials: "include" });
    if (res.status === 404) return null;
    if (!res.ok) throw new Error("Failed to fetch link state");
    return res.json();
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

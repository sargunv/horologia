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

function parseAuthConfig(raw: unknown): AuthConfig {
  if (raw !== null && typeof raw === "object" && "oidc" in raw && "password" in raw) {
    const { oidc, password } = raw;
    if (
      oidc !== null &&
      typeof oidc === "object" &&
      "enabled" in oidc &&
      typeof oidc.enabled === "boolean" &&
      "label" in oidc &&
      typeof oidc.label === "string" &&
      "autoRedirect" in oidc &&
      typeof oidc.autoRedirect === "boolean" &&
      password !== null &&
      typeof password === "object" &&
      "enabled" in password &&
      typeof password.enabled === "boolean"
    ) {
      return {
        oidc: {
          enabled: oidc.enabled,
          label: oidc.label,
          autoRedirect: oidc.autoRedirect,
        },
        password: { enabled: password.enabled },
      };
    }
  }
  throw new Error("Invalid auth config response");
}

export const authConfigQueryOptions = queryOptions({
  queryKey: ["authConfig"],
  queryFn: async (): Promise<AuthConfig> => {
    const res = await fetch("/api/auth/config");
    if (!res.ok) throw new Error("Failed to fetch auth config");
    const raw: unknown = await res.json();
    return parseAuthConfig(raw);
  },
  staleTime: Infinity,
});

export interface LinkPendingInfo {
  email: string;
  name: string;
}

function parseLinkPendingInfo(raw: unknown): LinkPendingInfo {
  if (
    raw !== null &&
    typeof raw === "object" &&
    "email" in raw &&
    typeof raw.email === "string" &&
    "name" in raw &&
    typeof raw.name === "string"
  ) {
    return { email: raw.email, name: raw.name };
  }
  throw new Error("Invalid link pending info response");
}

export const linkPendingQueryOptions = queryOptions({
  queryKey: ["linkPending"],
  queryFn: async (): Promise<LinkPendingInfo | null> => {
    const res = await fetch("/api/auth/link/pending", { credentials: "include" });
    if (res.status === 404) return null;
    if (!res.ok) throw new Error("Failed to fetch link state");
    const raw: unknown = await res.json();
    return parseLinkPendingInfo(raw);
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

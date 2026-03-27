import { queryOptions } from "@tanstack/react-query";
import { apiClient } from "../api/client.ts";

export interface AuthConfig {
  oidc: {
    enabled: boolean;
    label: string;
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

export const currentUserQueryOptions = queryOptions({
  queryKey: ["currentUser"],
  queryFn: async () => {
    const { data, error } = await apiClient.GET("/users/me");
    if (error) throw error;
    return data;
  },
});

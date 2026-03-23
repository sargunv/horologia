import { queryOptions } from "@tanstack/react-query";
import { apiClient } from "../api/client.ts";

export function spacesQueryOptions() {
  return queryOptions({
    queryKey: ["spaces"],
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/spaces");
      if (error) throw error;
      return data;
    },
  });
}

export function spaceQueryOptions(spaceSlug: string) {
  return queryOptions({
    queryKey: ["spaces", spaceSlug],
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/spaces/{spaceSlug}", {
        params: { path: { spaceSlug } },
      });
      if (error) throw error;
      return data;
    },
  });
}

export function spaceMembersQueryOptions(spaceSlug: string) {
  return queryOptions({
    queryKey: ["spaces", spaceSlug, "members"],
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/spaces/{spaceSlug}/members", {
        params: { path: { spaceSlug } },
      });
      if (error) throw error;
      return data;
    },
  });
}

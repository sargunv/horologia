import { queryOptions } from "@tanstack/react-query";
import { apiClient } from "../api/client.ts";

export function spaceTasksQueryOptions(spaceSlug: string) {
  return queryOptions({
    queryKey: ["spaces", spaceSlug, "tasks"],
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/spaces/{spaceSlug}/tasks", {
        params: { path: { spaceSlug } },
      });
      if (error) throw error;
      return data;
    },
  });
}

export function taskQueryOptions(taskId: string) {
  return queryOptions({
    queryKey: ["tasks", taskId],
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/tasks/{taskId}", {
        params: { path: { taskId } },
      });
      if (error) throw error;
      return data;
    },
  });
}

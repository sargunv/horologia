import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "../api/client.ts";
import type { components } from "../api/schema.d.ts";

type TaskUpdate = components["schemas"]["TaskUpdate"];

export function useTaskPatch(spaceSlug: string, taskId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: TaskUpdate) => {
      const { data, error } = await apiClient.PATCH("/spaces/{spaceSlug}/tasks/{taskId}", {
        params: { path: { spaceSlug, taskId } },
        body,
      });
      if (error) throw new Error(error.message ?? "Failed to update task");
      return data;
    },
    onSuccess: async () => {
      try {
        await Promise.all([
          queryClient.invalidateQueries({ queryKey: ["spaces", spaceSlug, "tasks", taskId] }),
          queryClient.invalidateQueries({ queryKey: ["spaces", spaceSlug, "tasks", "list"] }),
        ]);
      } catch (err) {
        console.error("Failed to refresh after task update:", err);
      }
    },
  });
}

export function useUserPatch(userId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: components["schemas"]["UserUpdate"]) => {
      const { data, error } = await apiClient.PATCH("/users/{userId}", {
        params: { path: { userId } },
        body,
      });
      if (error) throw new Error(error.message ?? "Failed to update user");
      return data;
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["users", userId] }),
        queryClient.invalidateQueries({ queryKey: ["users"] }),
        queryClient.invalidateQueries({ queryKey: ["currentUser"] }),
      ]);
    },
  });
}

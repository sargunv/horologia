import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "../api/client.ts";
import type { components } from "../api/schema.d.ts";
import { notifyStaleData } from "./toaster.ts";

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
        console.error("Cache invalidation failed after mutation:", err);
        notifyStaleData();
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

type TaskRelationCreate = components["schemas"]["TaskRelationCreate"];
type TaskRelationKind = components["schemas"]["TaskRelationKind"];

export function useAddRelation(spaceSlug: string, taskId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: TaskRelationCreate) => {
      const { data, error } = await apiClient.POST("/spaces/{spaceSlug}/tasks/{taskId}/relations", {
        params: { path: { spaceSlug, taskId } },
        body,
      });
      if (error) throw new Error(error.message ?? "Failed to add relation");
      return data;
    },
    onSuccess: async () => {
      try {
        await Promise.all([
          queryClient.invalidateQueries({ queryKey: ["spaces", spaceSlug, "tasks", taskId] }),
          queryClient.invalidateQueries({ queryKey: ["spaces", spaceSlug, "tasks", "list"] }),
        ]);
      } catch (err) {
        console.error("Cache invalidation failed after mutation:", err);
        notifyStaleData();
      }
    },
  });
}

export function useDeleteRelation(spaceSlug: string, taskId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      kind,
      relatedTaskId,
    }: {
      kind: TaskRelationKind;
      relatedTaskId: string;
    }) => {
      const { error } = await apiClient.DELETE(
        "/spaces/{spaceSlug}/tasks/{taskId}/relations/{kind}/{relatedTaskId}",
        {
          params: { path: { spaceSlug, taskId, kind, relatedTaskId } },
        },
      );
      if (error) throw new Error(error.message ?? "Failed to remove relation");
    },
    onSuccess: async () => {
      try {
        await Promise.all([
          queryClient.invalidateQueries({ queryKey: ["spaces", spaceSlug, "tasks", taskId] }),
          queryClient.invalidateQueries({ queryKey: ["spaces", spaceSlug, "tasks", "list"] }),
        ]);
      } catch (err) {
        console.error("Cache invalidation failed after mutation:", err);
        notifyStaleData();
      }
    },
  });
}

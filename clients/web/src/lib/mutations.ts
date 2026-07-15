import { type QueryClient, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "../api/client.ts";
import type { components } from "../api/schema.d.ts";
import { notifyStaleData } from "./toaster.ts";

type TaskUpdate = components["schemas"]["TaskUpdate"];
type RecipeUpdate = components["schemas"]["RecipeUpdate"];

/** Invalidate all user task list queries (for the "My Tasks" view). */
export async function invalidateUserTaskLists(queryClient: QueryClient) {
  await queryClient.invalidateQueries({
    predicate: (query) =>
      query.queryKey[0] === "users" &&
      query.queryKey[2] === "tasks" &&
      query.queryKey[3] === "list",
  });
}

export async function invalidateRecipeLists(queryClient: QueryClient) {
  await queryClient.invalidateQueries({ queryKey: ["recipes", "list"] });
}

export function useRecipePatch(spaceSlug: string, recipeId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: RecipeUpdate) => {
      const { data, error } = await apiClient.PATCH("/spaces/{spaceSlug}/recipes/{recipeId}", {
        params: { path: { spaceSlug, recipeId } },
        body,
      });
      if (error) throw new Error(error.message ?? "Failed to update recipe");
      return data;
    },
    onSuccess: async () => {
      try {
        await Promise.all([
          queryClient.invalidateQueries({
            queryKey: ["spaces", spaceSlug, "recipes", recipeId],
          }),
          invalidateRecipeLists(queryClient),
        ]);
      } catch (err) {
        console.error("Cache invalidation failed after mutation:", err);
        notifyStaleData();
      }
    },
  });
}

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
          invalidateUserTaskLists(queryClient),
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
      try {
        await Promise.all([
          queryClient.invalidateQueries({ queryKey: ["users", userId] }),
          queryClient.invalidateQueries({ queryKey: ["users"] }),
          queryClient.invalidateQueries({ queryKey: ["currentUser"] }),
        ]);
      } catch (err) {
        console.error("Cache invalidation failed after mutation:", err);
        notifyStaleData();
      }
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
          invalidateUserTaskLists(queryClient),
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
          invalidateUserTaskLists(queryClient),
        ]);
      } catch (err) {
        console.error("Cache invalidation failed after mutation:", err);
        notifyStaleData();
      }
    },
  });
}

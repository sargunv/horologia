import { createTaskCommands } from "@horologia/client-core/commands/tasks";
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
      query.queryKey[0] === window.location.origin &&
      query.queryKey[1] === "users" &&
      query.queryKey[3] === "tasks" &&
      query.queryKey[4] === "list",
  });
}

export async function invalidateRecipeLists(queryClient: QueryClient) {
  await queryClient.invalidateQueries({ queryKey: [window.location.origin, "recipes", "list"] });
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
            queryKey: [window.location.origin, "spaces", spaceSlug, "recipes", recipeId],
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
  const commands = createTaskCommands({
    serverId: window.location.origin,
    apiClient,
    queryClient,
    onCacheError(error) {
      console.error("Cache invalidation failed after mutation:", error);
      notifyStaleData();
    },
  });
  return useMutation({
    mutationFn: (body: TaskUpdate) => commands.update(spaceSlug, taskId, body),
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
          queryClient.invalidateQueries({ queryKey: [window.location.origin, "users", userId] }),
          queryClient.invalidateQueries({ queryKey: [window.location.origin, "users"] }),
          queryClient.invalidateQueries({ queryKey: [window.location.origin, "currentUser"] }),
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
  const commands = createTaskCommands({
    serverId: window.location.origin,
    apiClient,
    queryClient,
    onCacheError(error) {
      console.error("Cache invalidation failed after mutation:", error);
      notifyStaleData();
    },
  });
  return useMutation({
    mutationFn: (body: TaskRelationCreate) => commands.addRelation(spaceSlug, taskId, body),
  });
}

export function useDeleteRelation(spaceSlug: string, taskId: string) {
  const queryClient = useQueryClient();
  const commands = createTaskCommands({
    serverId: window.location.origin,
    apiClient,
    queryClient,
    onCacheError(error) {
      console.error("Cache invalidation failed after mutation:", error);
      notifyStaleData();
    },
  });
  return useMutation({
    mutationFn: async ({
      kind,
      relatedTaskId,
    }: {
      kind: TaskRelationKind;
      relatedTaskId: string;
    }) => commands.deleteRelation(spaceSlug, taskId, kind, relatedTaskId),
  });
}

import { createTaskCommands } from "@horologia/client-core/commands/tasks";
import { createLibraryCommands } from "@horologia/client-core/commands/library";
import { createSettingsCommands } from "@horologia/client-core/commands/settings";
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
  const commands = createLibraryCommands({
    serverId: window.location.origin,
    apiClient,
    queryClient,
    onCacheError(error) {
      console.error("Cache invalidation failed after mutation:", error);
      notifyStaleData();
    },
  });
  return useMutation({
    mutationFn: (body: RecipeUpdate) => commands.updateRecipe(spaceSlug, recipeId, body),
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
  const commands = createSettingsCommands({
    serverId: window.location.origin,
    apiClient,
    queryClient,
    onCacheError(error) {
      console.error("Cache invalidation failed after mutation:", error);
      notifyStaleData();
    },
  });
  return useMutation({
    mutationFn: (body: components["schemas"]["UserUpdate"]) => commands.updateUser(userId, body),
  });
}

export function useSettingsCommands() {
  const queryClient = useQueryClient();
  return createSettingsCommands({
    serverId: window.location.origin,
    apiClient,
    queryClient,
    onCacheError(error) {
      console.error("Cache invalidation failed after mutation:", error);
      notifyStaleData();
    },
  });
}

export function useLibraryCommands() {
  const queryClient = useQueryClient();
  return createLibraryCommands({
    serverId: window.location.origin,
    apiClient,
    queryClient,
    onCacheError(error) {
      console.error("Cache invalidation failed after mutation:", error);
      notifyStaleData();
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

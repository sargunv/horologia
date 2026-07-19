import { createTaskCommands } from "@horologia/client-core/commands/tasks";
import { createLibraryCommands } from "@horologia/client-core/commands/library";
import { createSettingsCommands } from "@horologia/client-core/commands/settings";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "../api/client.ts";
import type { components } from "@horologia/client-core/schema";
import { notifyStaleData } from "./toaster.ts";

type TaskUpdate = components["schemas"]["TaskUpdate"];
type RecipeUpdate = components["schemas"]["RecipeUpdate"];

function useCommandContext() {
  const queryClient = useQueryClient();
  return {
    serverId: window.location.origin,
    apiClient,
    queryClient,
    onCacheError(error: unknown) {
      console.error("Cache invalidation failed after mutation:", error);
      notifyStaleData();
    },
  };
}

export function useRecipePatch(spaceSlug: string, recipeId: string) {
  const commands = useLibraryCommands();
  return useMutation({
    mutationFn: (body: RecipeUpdate) => commands.updateRecipe(spaceSlug, recipeId, body),
  });
}

export function useTaskPatch(spaceSlug: string, taskId: string) {
  const commands = useTaskCommands();
  return useMutation({
    mutationFn: (body: TaskUpdate) => commands.update(spaceSlug, taskId, body),
  });
}

export function useUserPatch(userId: string) {
  const commands = useSettingsCommands();
  return useMutation({
    mutationFn: (body: components["schemas"]["UserUpdate"]) => commands.updateUser(userId, body),
  });
}

export function useSettingsCommands() {
  return createSettingsCommands(useCommandContext());
}

export function useLibraryCommands() {
  return createLibraryCommands(useCommandContext());
}

export function useTaskCommands() {
  return createTaskCommands(useCommandContext());
}

type TaskRelationCreate = components["schemas"]["TaskRelationCreate"];
type TaskRelationKind = components["schemas"]["TaskRelationKind"];

export function useAddRelation(spaceSlug: string, taskId: string) {
  const commands = useTaskCommands();
  return useMutation({
    mutationFn: (body: TaskRelationCreate) => commands.addRelation(spaceSlug, taskId, body),
  });
}

export function useDeleteRelation(spaceSlug: string, taskId: string) {
  const commands = useTaskCommands();
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

import type { QueryClient } from "@tanstack/react-query";

import { getApiErrorMessage, type HorologiaClient } from "../api/client";
import type { components } from "../api/schema.d.ts";

export interface LibraryCommandContext {
  serverId: string;
  apiClient: HorologiaClient;
  queryClient: QueryClient;
  onCacheError?(error: unknown): void;
}

export function createLibraryCommands(context: LibraryCommandContext) {
  async function synchronize(operation: () => Promise<void>) {
    try {
      await operation();
    } catch (error) {
      context.onCacheError?.(error);
    }
  }

  async function invalidateSpaces(spaceSlug?: string) {
    await Promise.all([
      context.queryClient.invalidateQueries({ queryKey: [context.serverId, "spaces"] }),
      ...(spaceSlug
        ? [
            context.queryClient.invalidateQueries({
              queryKey: [context.serverId, "spaces", spaceSlug],
            }),
          ]
        : []),
    ]);
  }

  async function invalidateRecipes(spaceSlug: string, recipeId?: string) {
    await Promise.all([
      context.queryClient.invalidateQueries({ queryKey: [context.serverId, "recipes", "list"] }),
      context.queryClient.invalidateQueries({ queryKey: [context.serverId, "recipes", "search"] }),
      ...(recipeId
        ? [
            context.queryClient.invalidateQueries({
              queryKey: [context.serverId, "spaces", spaceSlug, "recipes", recipeId],
            }),
          ]
        : []),
    ]);
  }

  return {
    async createSpace(body: components["schemas"]["SpaceCreate"]) {
      const { data, error } = await context.apiClient.POST("/spaces", { body });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to create space"));
      await synchronize(() => invalidateSpaces(data.slug));
      return data;
    },

    async updateSpace(spaceSlug: string, body: components["schemas"]["SpaceUpdate"]) {
      const { data, error } = await context.apiClient.PATCH("/spaces/{spaceSlug}", {
        params: { path: { spaceSlug } },
        body,
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to update space"));
      await synchronize(() => invalidateSpaces(spaceSlug));
      return data;
    },

    async deleteSpace(spaceSlug: string) {
      const { error } = await context.apiClient.DELETE("/spaces/{spaceSlug}", {
        params: { path: { spaceSlug } },
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to delete space"));
      context.queryClient.removeQueries({
        queryKey: [context.serverId, "spaces", spaceSlug],
      });
      await synchronize(() => invalidateSpaces());
    },

    async createRecipe(spaceSlug: string, body: components["schemas"]["RecipeCreate"]) {
      const { data, error } = await context.apiClient.POST("/spaces/{spaceSlug}/recipes", {
        params: { path: { spaceSlug } },
        body,
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to create recipe"));
      await synchronize(() => invalidateRecipes(spaceSlug, data.id));
      return data;
    },

    async updateRecipe(
      spaceSlug: string,
      recipeId: string,
      body: components["schemas"]["RecipeUpdate"],
    ) {
      const { data, error } = await context.apiClient.PATCH(
        "/spaces/{spaceSlug}/recipes/{recipeId}",
        { params: { path: { spaceSlug, recipeId } }, body },
      );
      if (error) throw new Error(getApiErrorMessage(error, "Failed to update recipe"));
      await synchronize(() => invalidateRecipes(spaceSlug, recipeId));
      return data;
    },

    async deleteRecipe(spaceSlug: string, recipeId: string) {
      const { error } = await context.apiClient.DELETE("/spaces/{spaceSlug}/recipes/{recipeId}", {
        params: { path: { spaceSlug, recipeId } },
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to delete recipe"));
      context.queryClient.removeQueries({
        queryKey: [context.serverId, "spaces", spaceSlug, "recipes", recipeId],
      });
      await synchronize(() => invalidateRecipes(spaceSlug));
    },
  };
}

export type LibraryCommands = ReturnType<typeof createLibraryCommands>;

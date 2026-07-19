import { getApiErrorMessage } from "../api/client";
import type { components } from "../api/schema.d.ts";
import { createQueryKeys } from "../queries/queryKeys";
import { type CommandContext, synchronizeCache } from "./context";

export function createLibraryCommands(context: CommandContext) {
  const keys = createQueryKeys(context.serverId);

  async function invalidateSpaces(spaceSlug?: string) {
    await Promise.all([
      context.queryClient.invalidateQueries({ queryKey: keys.spaces }),
      ...(spaceSlug
        ? [
            context.queryClient.invalidateQueries({
              queryKey: keys.space(spaceSlug),
            }),
          ]
        : []),
    ]);
  }

  async function invalidateRecipes(spaceSlug: string, recipeId?: string) {
    await Promise.all([
      context.queryClient.invalidateQueries({ queryKey: keys.recipeLists }),
      context.queryClient.invalidateQueries({ queryKey: keys.recipeSearches }),
      ...(recipeId
        ? [
            context.queryClient.invalidateQueries({
              queryKey: keys.recipe(spaceSlug, recipeId),
            }),
          ]
        : []),
    ]);
  }

  return {
    async createSpace(body: components["schemas"]["SpaceCreate"]) {
      const { data, error } = await context.apiClient.POST("/spaces", { body });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to create space"));
      await synchronizeCache(context, () => invalidateSpaces(data.slug));
      return data;
    },

    async updateSpace(spaceSlug: string, body: components["schemas"]["SpaceUpdate"]) {
      const { data, error } = await context.apiClient.PATCH("/spaces/{spaceSlug}", {
        params: { path: { spaceSlug } },
        body,
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to update space"));
      await synchronizeCache(context, () => invalidateSpaces(spaceSlug));
      return data;
    },

    async deleteSpace(spaceSlug: string) {
      const { error } = await context.apiClient.DELETE("/spaces/{spaceSlug}", {
        params: { path: { spaceSlug } },
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to delete space"));
      context.queryClient.removeQueries({
        queryKey: keys.space(spaceSlug),
      });
      await synchronizeCache(context, () => invalidateSpaces());
    },

    async createRecipe(spaceSlug: string, body: components["schemas"]["RecipeCreate"]) {
      const { data, error } = await context.apiClient.POST("/spaces/{spaceSlug}/recipes", {
        params: { path: { spaceSlug } },
        body,
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to create recipe"));
      await synchronizeCache(context, () => invalidateRecipes(spaceSlug, data.id));
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
      await synchronizeCache(context, () => invalidateRecipes(spaceSlug, recipeId));
      return data;
    },

    async deleteRecipe(spaceSlug: string, recipeId: string) {
      const { error } = await context.apiClient.DELETE("/spaces/{spaceSlug}/recipes/{recipeId}", {
        params: { path: { spaceSlug, recipeId } },
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to delete recipe"));
      context.queryClient.removeQueries({
        queryKey: keys.recipe(spaceSlug, recipeId),
      });
      await synchronizeCache(context, () => invalidateRecipes(spaceSlug));
    },
  };
}

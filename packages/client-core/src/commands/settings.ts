import { getApiErrorMessage } from "../api/client";
import type { components } from "../api/schema.d.ts";
import { createQueryKeys } from "../queries/queryKeys";
import { type CommandContext, synchronizeCache } from "./context";

export function createSettingsCommands(context: CommandContext) {
  const keys = createQueryKeys(context.serverId);

  async function synchronize(queryKeys: readonly (readonly unknown[])[]) {
    await synchronizeCache(context, () =>
      Promise.all(queryKeys.map((queryKey) => context.queryClient.invalidateQueries({ queryKey }))),
    );
  }

  return {
    async updateUser(userId: string, body: components["schemas"]["UserUpdate"]) {
      const { data, error } = await context.apiClient.PATCH("/users/{userId}", {
        params: { path: { userId } },
        body,
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to update account"));
      await synchronize([keys.currentUser, keys.users, keys.user(userId)]);
      return data;
    },

    async deleteUser(userId: string) {
      const { error } = await context.apiClient.DELETE("/users/{userId}", {
        params: { path: { userId } },
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to delete account"));
      context.queryClient.clear();
    },

    async createToken(name: string) {
      const { data, error } = await context.apiClient.POST("/auth/tokens", {
        body: { name },
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to create API token"));
      await synchronize([keys.authTokens]);
      return data;
    },

    async revokeToken(tokenId: string) {
      const { error } = await context.apiClient.DELETE("/auth/tokens/{tokenId}", {
        params: { path: { tokenId } },
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to revoke API token"));
      await synchronize([keys.authTokens]);
    },

    async createMember(spaceSlug: string, body: components["schemas"]["SpaceMemberCreate"]) {
      const { data, error } = await context.apiClient.POST("/spaces/{spaceSlug}/members", {
        params: { path: { spaceSlug } },
        body,
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to add member"));
      await synchronize([keys.spaceMembers(spaceSlug)]);
      return data;
    },

    async updateMember(
      spaceSlug: string,
      userId: string,
      body: components["schemas"]["SpaceMemberUpdate"],
    ) {
      const { data, error } = await context.apiClient.PATCH(
        "/spaces/{spaceSlug}/members/{userId}",
        { params: { path: { spaceSlug, userId } }, body },
      );
      if (error) throw new Error(getApiErrorMessage(error, "Failed to update member"));
      await synchronize([keys.spaceMembers(spaceSlug)]);
      return data;
    },

    async deleteMember(spaceSlug: string, userId: string) {
      const { error } = await context.apiClient.DELETE("/spaces/{spaceSlug}/members/{userId}", {
        params: { path: { spaceSlug, userId } },
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to remove member"));
      await synchronize([keys.spaceMembers(spaceSlug)]);
    },

    async replaceTaskStatuses(
      spaceSlug: string,
      items: components["schemas"]["TaskStatusInput"][],
    ) {
      const { data, error } = await context.apiClient.PUT("/spaces/{spaceSlug}/task-statuses", {
        params: { path: { spaceSlug } },
        body: { items },
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to update task statuses"));
      await synchronize([keys.spaceTaskStatuses(spaceSlug)]);
      return data;
    },

    async replaceEffortLevels(
      spaceSlug: string,
      items: components["schemas"]["TaskEffortLevelInput"][],
    ) {
      const { data, error } = await context.apiClient.PUT(
        "/spaces/{spaceSlug}/task-effort-levels",
        { params: { path: { spaceSlug } }, body: { items } },
      );
      if (error) throw new Error(getApiErrorMessage(error, "Failed to update effort levels"));
      await synchronize([keys.spaceEffortLevels(spaceSlug)]);
      return data;
    },

    async replacePriorityLevels(
      spaceSlug: string,
      items: components["schemas"]["TaskPriorityLevelInput"][],
    ) {
      const { data, error } = await context.apiClient.PUT(
        "/spaces/{spaceSlug}/task-priority-levels",
        { params: { path: { spaceSlug } }, body: { items } },
      );
      if (error) throw new Error(getApiErrorMessage(error, "Failed to update priority levels"));
      await synchronize([keys.spacePriorityLevels(spaceSlug)]);
      return data;
    },

    async createTag(spaceSlug: string, name: string) {
      const { data, error } = await context.apiClient.POST("/spaces/{spaceSlug}/tags", {
        params: { path: { spaceSlug } },
        body: { name },
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to create tag"));
      await synchronize([keys.spaceTags(spaceSlug)]);
      return data;
    },

    async updateTag(spaceSlug: string, tagName: string, name: string) {
      const { data, error } = await context.apiClient.PATCH("/spaces/{spaceSlug}/tags/{tagName}", {
        params: { path: { spaceSlug, tagName } },
        body: { name },
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to rename tag"));
      await synchronize([keys.spaceTags(spaceSlug)]);
      return data;
    },

    async deleteTag(spaceSlug: string, tagName: string) {
      const { error } = await context.apiClient.DELETE("/spaces/{spaceSlug}/tags/{tagName}", {
        params: { path: { spaceSlug, tagName } },
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to delete tag"));
      await synchronize([keys.spaceTags(spaceSlug)]);
    },
  };
}

import type { QueryClient } from "@tanstack/react-query";

import { getApiErrorMessage, type HorologiaClient } from "../api/client";
import type { components } from "../api/schema.d.ts";

export interface SettingsCommandContext {
  serverId: string;
  apiClient: HorologiaClient;
  queryClient: QueryClient;
  onCacheError?(error: unknown): void;
}

export function createSettingsCommands(context: SettingsCommandContext) {
  async function synchronize(queryKeys: readonly (readonly unknown[])[]) {
    try {
      await Promise.all(
        queryKeys.map((queryKey) => context.queryClient.invalidateQueries({ queryKey })),
      );
    } catch (error) {
      context.onCacheError?.(error);
    }
  }

  const spaceKey = (spaceSlug: string, resource: string) => [
    context.serverId,
    "spaces",
    spaceSlug,
    resource,
  ];

  return {
    async updateUser(userId: string, body: components["schemas"]["UserUpdate"]) {
      const { data, error } = await context.apiClient.PATCH("/users/{userId}", {
        params: { path: { userId } },
        body,
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to update account"));
      await synchronize([
        [context.serverId, "currentUser"],
        [context.serverId, "users"],
        [context.serverId, "users", userId],
      ]);
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
      await synchronize([[context.serverId, "authTokens"]]);
      return data;
    },

    async revokeToken(tokenId: string) {
      const { error } = await context.apiClient.DELETE("/auth/tokens/{tokenId}", {
        params: { path: { tokenId } },
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to revoke API token"));
      await synchronize([[context.serverId, "authTokens"]]);
    },

    async createMember(spaceSlug: string, body: components["schemas"]["SpaceMemberCreate"]) {
      const { data, error } = await context.apiClient.POST("/spaces/{spaceSlug}/members", {
        params: { path: { spaceSlug } },
        body,
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to add member"));
      await synchronize([spaceKey(spaceSlug, "members")]);
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
      await synchronize([spaceKey(spaceSlug, "members")]);
      return data;
    },

    async deleteMember(spaceSlug: string, userId: string) {
      const { error } = await context.apiClient.DELETE("/spaces/{spaceSlug}/members/{userId}", {
        params: { path: { spaceSlug, userId } },
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to remove member"));
      await synchronize([spaceKey(spaceSlug, "members")]);
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
      await synchronize([spaceKey(spaceSlug, "taskStatuses")]);
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
      await synchronize([spaceKey(spaceSlug, "effortLevels")]);
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
      await synchronize([spaceKey(spaceSlug, "priorityLevels")]);
      return data;
    },

    async createTag(spaceSlug: string, name: string) {
      const { data, error } = await context.apiClient.POST("/spaces/{spaceSlug}/tags", {
        params: { path: { spaceSlug } },
        body: { name },
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to create tag"));
      await synchronize([spaceKey(spaceSlug, "tags")]);
      return data;
    },

    async updateTag(spaceSlug: string, tagName: string, name: string) {
      const { data, error } = await context.apiClient.PATCH("/spaces/{spaceSlug}/tags/{tagName}", {
        params: { path: { spaceSlug, tagName } },
        body: { name },
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to rename tag"));
      await synchronize([spaceKey(spaceSlug, "tags")]);
      return data;
    },

    async deleteTag(spaceSlug: string, tagName: string) {
      const { error } = await context.apiClient.DELETE("/spaces/{spaceSlug}/tags/{tagName}", {
        params: { path: { spaceSlug, tagName } },
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to delete tag"));
      await synchronize([spaceKey(spaceSlug, "tags")]);
    },
  };
}

export type SettingsCommands = ReturnType<typeof createSettingsCommands>;

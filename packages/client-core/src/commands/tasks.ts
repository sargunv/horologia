import type { QueryClient } from "@tanstack/react-query";

import { getApiErrorMessage, type HorologiaClient } from "../api/client";
import type { components } from "../api/schema.d.ts";

export type TaskCreate = components["schemas"]["TaskCreate"];
export type TaskUpdate = components["schemas"]["TaskUpdate"];
export type TaskRelationCreate = components["schemas"]["TaskRelationCreate"];
export type TaskRelationKind = components["schemas"]["TaskRelationKind"];

export interface TaskCommandContext {
  serverId: string;
  apiClient: HorologiaClient;
  queryClient: QueryClient;
  onCacheError?(error: unknown): void;
}

export function createTaskCommands(context: TaskCommandContext) {
  async function synchronize(operation: () => Promise<void>) {
    try {
      await operation();
    } catch (error) {
      context.onCacheError?.(error);
    }
  }

  async function invalidateTaskLists(spaceSlug: string) {
    await Promise.all([
      context.queryClient.invalidateQueries({
        queryKey: [context.serverId, "spaces", spaceSlug, "tasks", "list"],
      }),
      context.queryClient.invalidateQueries({
        predicate: (query) =>
          query.queryKey[0] === context.serverId &&
          query.queryKey[1] === "users" &&
          query.queryKey[3] === "tasks" &&
          query.queryKey[4] === "list",
      }),
      context.queryClient.invalidateQueries({
        queryKey: [context.serverId, "tasks", "search"],
      }),
    ]);
  }

  async function invalidateTask(spaceSlug: string, taskId: string) {
    await Promise.all([
      context.queryClient.invalidateQueries({
        queryKey: [context.serverId, "spaces", spaceSlug, "tasks", taskId],
      }),
      invalidateTaskLists(spaceSlug),
    ]);
  }

  return {
    async create(spaceSlug: string, body: TaskCreate) {
      const { data, error } = await context.apiClient.POST("/spaces/{spaceSlug}/tasks", {
        params: { path: { spaceSlug } },
        body,
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to create task"));
      await synchronize(() => invalidateTaskLists(spaceSlug));
      return data;
    },

    async update(spaceSlug: string, taskId: string, body: TaskUpdate) {
      const { data, error } = await context.apiClient.PATCH("/spaces/{spaceSlug}/tasks/{taskId}", {
        params: { path: { spaceSlug, taskId } },
        body,
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to update task"));
      await synchronize(() => invalidateTask(spaceSlug, taskId));
      return data;
    },

    async delete(spaceSlug: string, taskId: string) {
      const { error } = await context.apiClient.DELETE("/spaces/{spaceSlug}/tasks/{taskId}", {
        params: { path: { spaceSlug, taskId } },
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to delete task"));
      context.queryClient.removeQueries({
        queryKey: [context.serverId, "spaces", spaceSlug, "tasks", taskId],
      });
      await synchronize(() => invalidateTaskLists(spaceSlug));
    },

    async addRelation(spaceSlug: string, taskId: string, body: TaskRelationCreate) {
      const { data, error } = await context.apiClient.POST(
        "/spaces/{spaceSlug}/tasks/{taskId}/relations",
        { params: { path: { spaceSlug, taskId } }, body },
      );
      if (error) throw new Error(getApiErrorMessage(error, "Failed to add relation"));
      await synchronize(() => invalidateTask(spaceSlug, taskId));
      return data;
    },

    async deleteRelation(
      spaceSlug: string,
      taskId: string,
      kind: TaskRelationKind,
      relatedTaskId: string,
    ) {
      const { error } = await context.apiClient.DELETE(
        "/spaces/{spaceSlug}/tasks/{taskId}/relations/{kind}/{relatedTaskId}",
        { params: { path: { spaceSlug, taskId, kind, relatedTaskId } } },
      );
      if (error) throw new Error(getApiErrorMessage(error, "Failed to remove relation"));
      await synchronize(() => invalidateTask(spaceSlug, taskId));
    },
  };
}

export type TaskCommands = ReturnType<typeof createTaskCommands>;

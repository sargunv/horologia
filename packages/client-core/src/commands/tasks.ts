import { getApiErrorMessage } from "../api/client";
import type { components } from "../api/schema.d.ts";
import { createQueryKeys } from "../queries/queryKeys";
import { type CommandContext, synchronizeCache } from "./context";

export type TaskCreate = components["schemas"]["TaskCreate"];
export type TaskUpdate = components["schemas"]["TaskUpdate"];
export type TaskRelationCreate = components["schemas"]["TaskRelationCreate"];
export type TaskRelationKind = components["schemas"]["TaskRelationKind"];

export function createTaskCommands(context: CommandContext) {
  const keys = createQueryKeys(context.serverId);

  async function invalidateTaskLists(spaceSlug: string) {
    await Promise.all([
      context.queryClient.invalidateQueries({
        queryKey: keys.spaceTasks(spaceSlug),
      }),
      context.queryClient.invalidateQueries({
        predicate: (query) => keys.isUserTaskList(query.queryKey),
      }),
      context.queryClient.invalidateQueries({
        queryKey: keys.taskSearches,
      }),
    ]);
  }

  async function invalidateTask(spaceSlug: string, taskId: string) {
    await Promise.all([
      context.queryClient.invalidateQueries({
        queryKey: keys.spaceTask(spaceSlug, taskId),
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
      await synchronizeCache(context, () => invalidateTaskLists(spaceSlug));
      return data;
    },

    async update(spaceSlug: string, taskId: string, body: TaskUpdate) {
      const { data, error } = await context.apiClient.PATCH("/spaces/{spaceSlug}/tasks/{taskId}", {
        params: { path: { spaceSlug, taskId } },
        body,
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to update task"));
      await synchronizeCache(context, () => invalidateTask(spaceSlug, taskId));
      return data;
    },

    async delete(spaceSlug: string, taskId: string) {
      const { error } = await context.apiClient.DELETE("/spaces/{spaceSlug}/tasks/{taskId}", {
        params: { path: { spaceSlug, taskId } },
      });
      if (error) throw new Error(getApiErrorMessage(error, "Failed to delete task"));
      context.queryClient.removeQueries({
        queryKey: keys.spaceTask(spaceSlug, taskId),
      });
      await synchronizeCache(context, () => invalidateTaskLists(spaceSlug));
    },

    async addRelation(spaceSlug: string, taskId: string, body: TaskRelationCreate) {
      const { data, error } = await context.apiClient.POST(
        "/spaces/{spaceSlug}/tasks/{taskId}/relations",
        { params: { path: { spaceSlug, taskId } }, body },
      );
      if (error) throw new Error(getApiErrorMessage(error, "Failed to add relation"));
      await synchronizeCache(context, () => invalidateTask(spaceSlug, taskId));
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
      await synchronizeCache(context, () => invalidateTask(spaceSlug, taskId));
    },
  };
}

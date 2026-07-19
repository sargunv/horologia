export interface WidgetTaskV1 {
  id: string;
  spaceSlug: string;
  title: string;
  due: string | null;
  status: string;
}

export interface WidgetSnapshotV1 {
  version: 1;
  serverId: string;
  accountId: string;
  generatedAt: string;
  taskCount: number;
  hasMore: boolean;
  tasks: WidgetTaskV1[];
}

export function projectMyTasksWidgetSnapshot(input: {
  serverId: string;
  accountId: string;
  generatedAt: string;
  tasks: Array<{
    id: string;
    spaceSlug: string;
    title: string;
    due: { at: string } | null;
    status: string;
  }>;
  hasMore?: boolean;
  limit?: number;
}): WidgetSnapshotV1 {
  return {
    version: 1,
    serverId: input.serverId,
    accountId: input.accountId,
    generatedAt: input.generatedAt,
    taskCount: input.tasks.length,
    hasMore: input.hasMore ?? false,
    tasks: input.tasks.slice(0, input.limit ?? 12).map((task) => ({
      id: task.id,
      spaceSlug: task.spaceSlug,
      title: task.title,
      due: task.due?.at ?? null,
      status: task.status,
    })),
  };
}

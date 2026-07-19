import { projectMyTasksWidgetSnapshot, type ServerProfile } from "@horologia/client-core";
import type { components } from "@horologia/client-core/schema";

import { saveCachedMyTasks, type CachedMyTasks } from "@/persistence/database";
import { publishWidgetSnapshot } from "@/widgets/publishWidgetSnapshot";

type Task = components["schemas"]["Task"];

export async function saveMyTasksSnapshot({
  profile,
  accountId,
  tasks,
  hasMore,
}: {
  profile: ServerProfile;
  accountId: string;
  tasks: Task[];
  hasMore: boolean;
}): Promise<CachedMyTasks> {
  const generatedAt = new Date().toISOString();
  const cache = { tasks, updatedAt: generatedAt, hasMore };
  await Promise.all([
    saveCachedMyTasks(profile.id, accountId, tasks, generatedAt, hasMore),
    publishWidgetSnapshot(
      projectMyTasksWidgetSnapshot({
        serverId: profile.id,
        accountId,
        generatedAt,
        tasks,
        hasMore,
      }),
    ),
  ]);
  return cache;
}

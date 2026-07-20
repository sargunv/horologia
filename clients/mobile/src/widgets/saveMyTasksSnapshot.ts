import { projectMyTasksWidgetSnapshot, type ServerProfile } from "@horologia/client-core";
import type { components } from "@horologia/client-core/schema";

import {
  clearCachedMyTasksVersion,
  saveCachedMyTasks,
  type CachedMyTasks,
} from "@/persistence/database";
import { publishWidgetSnapshot } from "@/widgets/publishWidgetSnapshot";

type Task = components["schemas"]["Task"];

export async function saveMyTasksSnapshot({
  profile,
  accountId,
  tasks,
  hasMore,
  isCurrent = () => true,
}: {
  profile: ServerProfile;
  accountId: string;
  tasks: Task[];
  hasMore: boolean;
  isCurrent?: () => boolean;
}): Promise<CachedMyTasks | null> {
  if (!isCurrent()) return null;
  const generatedAt = new Date().toISOString();
  const cache = { tasks, updatedAt: generatedAt, hasMore };
  const saved = await saveCachedMyTasks(profile.id, accountId, tasks, generatedAt, hasMore);
  if (!saved) return null;
  if (!isCurrent()) {
    await clearCachedMyTasksVersion(profile.id, accountId, generatedAt);
    return null;
  }
  await publishWidgetSnapshot(
    projectMyTasksWidgetSnapshot({
      serverId: profile.id,
      accountId,
      generatedAt,
      tasks,
      hasMore,
    }),
  );
  if (!isCurrent()) {
    await clearCachedMyTasksVersion(profile.id, accountId, generatedAt);
    return null;
  }
  return cache;
}

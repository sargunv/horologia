import {
  createQueries,
  projectMyTasksWidgetSnapshot,
  type HorologiaClient,
  type ServerProfile,
} from "@horologia/client-core";
import type { QueryClient } from "@tanstack/react-query";

import { saveCachedMyTasks } from "@/persistence/database";
import { publishWidgetSnapshot } from "@/widgets/publishWidgetSnapshot";

export async function refreshMyTasksWidget({
  profile,
  accountId,
  client,
  queryClient,
}: {
  profile: ServerProfile;
  accountId: string;
  client: HorologiaClient;
  queryClient: QueryClient;
}) {
  const queries = createQueries({ serverId: profile.id, apiClient: client, appClient: client });
  const data = await queryClient.fetchInfiniteQuery(
    queries.userTasksInfiniteQueryOptions(accountId),
  );
  const tasks = data.pages.flatMap((page) => page.items);
  const generatedAt = new Date().toISOString();
  await Promise.all([
    saveCachedMyTasks(profile.id, accountId, tasks, generatedAt),
    publishWidgetSnapshot(
      projectMyTasksWidgetSnapshot({
        serverId: profile.id,
        accountId,
        generatedAt,
        tasks,
      }),
    ),
  ]);
}

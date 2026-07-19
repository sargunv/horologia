import { createQueries, type HorologiaClient, type ServerProfile } from "@horologia/client-core";
import type { QueryClient } from "@tanstack/react-query";

import { saveMyTasksSnapshot } from "@/widgets/saveMyTasksSnapshot";

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
  const queries = createQueries({ serverId: profile.id, apiClient: client });
  const data = await queryClient.fetchInfiniteQuery(
    queries.userTasksInfiniteQueryOptions(accountId),
  );
  const tasks = data.pages.flatMap((page) => page.items);
  await saveMyTasksSnapshot({
    profile,
    accountId,
    tasks,
    hasMore: data.pages.at(-1)?.nextCursor !== null,
  });
}

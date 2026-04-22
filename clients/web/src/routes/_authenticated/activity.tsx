import { useSuspenseInfiniteQuery, useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useMemo } from "react";
import type { ActivityMember } from "../../components/ActivityFeed.tsx";
import { ActivityFeed } from "../../components/ActivityFeed.tsx";
import { currentUserQueryOptions, userActivityInfiniteQueryOptions } from "../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/activity")({
  async loader({ context: { queryClient } }) {
    const user = await queryClient.ensureQueryData(currentUserQueryOptions);
    await queryClient.ensureInfiniteQueryData(userActivityInfiniteQueryOptions(user.id));
  },
  component: UserActivityPage,
});

function UserActivityPage() {
  const { data: user } = useSuspenseQuery(currentUserQueryOptions);

  const {
    data: pages,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
  } = useSuspenseInfiniteQuery(userActivityInfiniteQueryOptions(user.id));

  const entries = useMemo(() => pages.pages.flatMap((p) => p.items), [pages]);

  // Build a minimal member map so actor names resolve to the current user's name
  const memberMap = useMemo<Map<string, ActivityMember>>(
    () => new Map([[user.id, { userName: user.name }]]),
    [user.id, user.name],
  );

  return (
    <div className="mx-auto max-w-3xl p-6">
      <h1 className="h3 mb-6">My Activity</h1>
      <ActivityFeed
        entries={entries}
        hasNextPage={hasNextPage}
        fetchNextPage={fetchNextPage}
        isFetchingNextPage={isFetchingNextPage}
        memberMap={memberMap}
        showSpace
      />
    </div>
  );
}

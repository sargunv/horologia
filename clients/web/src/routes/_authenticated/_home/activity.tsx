import { useSuspenseInfiniteQuery, useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute, createLink } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { useMemo } from "react";
import type { ActivityMember } from "../../../components/ActivityFeed.tsx";
import { ActivityFeed } from "../../../components/ActivityFeed.tsx";
import { currentUserQueryOptions, userActivityInfiniteQueryOptions } from "../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/_home/activity")({
  async loader({ context: { queryClient } }) {
    const user = await queryClient.ensureQueryData(currentUserQueryOptions);
    await queryClient.ensureInfiniteQueryData(userActivityInfiniteQueryOptions(user.id));
  },
  component: UserActivityPage,
});

const BackLink = createLink("a");

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
    <div className="space-y-4">
      <BackLink
        to="/"
        className="text-base-content/70 hover:text-base-content inline-flex items-center gap-1 text-sm transition-colors lg:hidden"
      >
        <ArrowLeft className="size-4" aria-hidden="true" />
        Back to My Tasks
      </BackLink>

      <h1 className="h3">My Activity</h1>
      <ActivityFeed
        entries={entries}
        hasNextPage={hasNextPage}
        fetchNextPage={fetchNextPage}
        isFetchingNextPage={isFetchingNextPage}
        memberMap={memberMap}
        showSpace
        variant="compact"
      />
    </div>
  );
}

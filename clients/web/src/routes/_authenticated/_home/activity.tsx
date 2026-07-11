import { useSuspenseInfiniteQuery, useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute, createLink } from "@tanstack/react-router";
import { ArrowLeft, ChevronRight } from "lucide-react";
import { useMemo } from "react";
import type { ActivityMember } from "../../../components/ActivityFeed.tsx";
import { ActivityFeed } from "../../../components/ActivityFeed.tsx";
import {
  DetailPaneHeader,
  DETAIL_PANE_TITLE_CLASS,
} from "../../../components/DetailPaneHeader.tsx";
import { currentUserQueryOptions, userActivityInfiniteQueryOptions } from "../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/_home/activity")({
  async loader({ context: { queryClient } }) {
    const user = await queryClient.ensureQueryData(currentUserQueryOptions);
    await queryClient.ensureInfiniteQueryData(userActivityInfiniteQueryOptions(user.id));
  },
  component: UserActivityPage,
});

const BackLink = createLink("a");
const BreadcrumbLink = createLink("a");

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
      <DetailPaneHeader
        backLink={
          <BackLink
            to="/"
            className="text-base-content/70 hover:text-base-content inline-flex items-center gap-1 text-sm transition-colors lg:hidden"
          >
            <ArrowLeft className="size-4" aria-hidden="true" />
            Back to My Tasks
          </BackLink>
        }
        breadcrumb={
          <ol className="flex min-w-0 items-center gap-1 text-sm">
            <li>
              <BreadcrumbLink to="/" className="text-base-content/70 truncate hover:underline">
                My Tasks
              </BreadcrumbLink>
            </li>
            <li className="text-base-content/60" aria-hidden="true">
              <ChevronRight className="size-3" />
            </li>
            <li className="shrink-0" aria-current="page">
              Activity
            </li>
          </ol>
        }
        title={<h1 className={DETAIL_PANE_TITLE_CLASS}>My Activity</h1>}
      />
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

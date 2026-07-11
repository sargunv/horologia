import { useSuspenseInfiniteQuery, useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute, createLink } from "@tanstack/react-router";
import { ArrowLeft, ChevronRight } from "lucide-react";
import { useMemo } from "react";
import { ActivityFeed } from "../../../../components/ActivityFeed.tsx";
import {
  DetailPaneHeader,
  DETAIL_PANE_TITLE_CLASS,
} from "../../../../components/DetailPaneHeader.tsx";
import { useSpaceMemberMap } from "../../../../lib/hooks.ts";
import {
  spaceActivityInfiniteQueryOptions,
  spaceMembersQueryOptions,
  spaceQueryOptions,
} from "../../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug/activity")({
  loader: ({ context: { queryClient }, params: { spaceSlug } }) =>
    Promise.all([
      queryClient.ensureQueryData(spaceQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spaceMembersQueryOptions(spaceSlug)),
      queryClient.ensureInfiniteQueryData(spaceActivityInfiniteQueryOptions(spaceSlug)),
    ]),
  component: SpaceActivityPage,
});

const BackLink = createLink("a");
const BreadcrumbLink = createLink("a");

function SpaceActivityPage() {
  const { spaceSlug } = Route.useParams();
  const { data: space } = useSuspenseQuery(spaceQueryOptions(spaceSlug));
  const memberMap = useSpaceMemberMap(spaceSlug);

  const {
    data: pages,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
  } = useSuspenseInfiniteQuery(spaceActivityInfiniteQueryOptions(spaceSlug));

  const entries = useMemo(() => pages.pages.flatMap((p) => p.items), [pages]);

  return (
    <div className="space-y-4">
      <DetailPaneHeader
        backLink={
          <BackLink
            to="/spaces/$spaceSlug"
            params={{ spaceSlug }}
            className="text-base-content/70 hover:text-base-content inline-flex items-center gap-1 text-sm transition-colors lg:hidden"
          >
            <ArrowLeft className="size-4" aria-hidden="true" />
            Back to {space.name}
          </BackLink>
        }
        breadcrumb={
          <ol className="flex min-w-0 items-center gap-1 text-sm">
            <li>
              <BreadcrumbLink
                to="/spaces/$spaceSlug"
                params={{ spaceSlug }}
                className="text-base-content/70 truncate hover:underline"
              >
                {space.name}
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
        title={<h1 className={DETAIL_PANE_TITLE_CLASS}>Activity</h1>}
      />

      <ActivityFeed
        entries={entries}
        hasNextPage={hasNextPage}
        fetchNextPage={fetchNextPage}
        isFetchingNextPage={isFetchingNextPage}
        memberMap={memberMap}
        variant="compact"
      />
    </div>
  );
}

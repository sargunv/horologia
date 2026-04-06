import { useSuspenseInfiniteQuery, useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute, createLink } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { useMemo } from "react";
import { ActivityFeed } from "../../../../components/ActivityFeed.tsx";
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
    <div className="mx-auto max-w-3xl p-6">
      <BackLink
        to="/spaces/$spaceSlug"
        params={{ spaceSlug }}
        className="text-surface-600-400 hover:text-surface-950-50 mb-4 inline-flex items-center gap-1 text-sm transition-colors"
      >
        <ArrowLeft className="size-4" aria-hidden="true" />
        Back to {space.name}
      </BackLink>

      <h1 className="h3 mb-6">{space.name} — Activity</h1>

      <ActivityFeed
        entries={entries}
        hasNextPage={hasNextPage}
        fetchNextPage={fetchNextPage}
        isFetchingNextPage={isFetchingNextPage}
        memberMap={memberMap}
      />
    </div>
  );
}

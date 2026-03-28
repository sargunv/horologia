import { Outlet, createFileRoute } from "@tanstack/react-router";
import {
  spaceEffortLevelsQueryOptions,
  spaceMembersQueryOptions,
  spacePriorityLevelsQueryOptions,
  spaceQueryOptions,
  spaceTaskStatusesQueryOptions,
} from "../../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug")({
  loader: ({ context: { queryClient }, params: { spaceSlug } }) =>
    Promise.all([
      queryClient.ensureQueryData(spaceQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spaceMembersQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spaceTaskStatusesQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spaceEffortLevelsQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spacePriorityLevelsQueryOptions(spaceSlug)),
    ]),
  component: SpaceLayout,
});

function SpaceLayout() {
  return <Outlet />;
}

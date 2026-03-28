import { Outlet, createFileRoute } from "@tanstack/react-router";
import { spaceQueryOptions } from "../../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug")({
  loader: ({ context: { queryClient }, params: { spaceSlug } }) =>
    queryClient.ensureQueryData(spaceQueryOptions(spaceSlug)),
  component: SpaceLayout,
});

function SpaceLayout() {
  return <Outlet />;
}

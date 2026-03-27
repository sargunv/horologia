import { Outlet, createFileRoute, redirect } from "@tanstack/react-router";
import { currentUserQueryOptions } from "../lib/queries.ts";

export const Route = createFileRoute("/_authenticated")({
  async beforeLoad({ context, location }) {
    try {
      await context.queryClient.ensureQueryData(currentUserQueryOptions);
    } catch {
      throw redirect({
        to: "/login",
        search: { redirect: location.pathname },
      });
    }
  },
  component: () => <Outlet />,
});

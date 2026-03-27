import { useQuery } from "@tanstack/react-query";
import { Outlet, createFileRoute, redirect } from "@tanstack/react-router";
import { AppShell } from "../components/AppShell.tsx";
import { currentUserQueryOptions, spacesQueryOptions } from "../lib/queries.ts";

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
    // Pre-fetch spaces for the nav (non-blocking)
    void context.queryClient.ensureQueryData(spacesQueryOptions);
  },
  component: AuthenticatedLayout,
});

function AuthenticatedLayout() {
  const { data: user } = useQuery(currentUserQueryOptions);
  const { data: spaces } = useQuery(spacesQueryOptions);

  return (
    <AppShell user={user!} spaces={spaces ?? []}>
      <Outlet />
    </AppShell>
  );
}

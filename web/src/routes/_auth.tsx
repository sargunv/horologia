import { Outlet, createFileRoute, redirect } from "@tanstack/react-router";
import { AppShell } from "../components/layout/app-shell.tsx";
import { meQueryOptions } from "../queries/auth.ts";

export const Route = createFileRoute("/_auth")({
  beforeLoad: async ({ context, location }) => {
    try {
      const user = await context.queryClient.ensureQueryData(meQueryOptions());
      return { user };
    } catch {
      throw redirect({
        to: "/login",
        search: { redirect: location.href },
      });
    }
  },
  component: AuthLayout,
});

function AuthLayout() {
  const { user } = Route.useRouteContext();
  return (
    <AppShell user={user}>
      <Outlet />
    </AppShell>
  );
}

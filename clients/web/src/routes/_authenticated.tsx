import { useQuery } from "@tanstack/react-query";
import { Outlet, createFileRoute, redirect } from "@tanstack/react-router";
import { useEffect } from "react";
import type { components } from "@horologia/client-core/schema";
import { AppShell } from "../components/AppShell.tsx";
import { currentUserQueryOptions, spacesQueryOptions } from "../lib/queries.ts";
import { useTheme } from "../lib/theme.tsx";

type User = components["schemas"]["User"];

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
    <>
      {user && <ThemePreferenceSync user={user} />}
      <AppShell user={user!} spaces={spaces ?? []}>
        <Outlet />
      </AppShell>
    </>
  );
}

function ThemePreferenceSync({ user }: { user: User }) {
  const { syncPreference } = useTheme();

  useEffect(() => {
    syncPreference({
      mode: user.appearanceMode,
      lightTheme: user.appearanceLightTheme,
      darkTheme: user.appearanceDarkTheme,
    });
  }, [syncPreference, user.appearanceMode, user.appearanceLightTheme, user.appearanceDarkTheme]);

  return null;
}

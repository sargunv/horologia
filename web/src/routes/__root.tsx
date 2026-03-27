import type { QueryClient } from "@tanstack/react-query";
import {
  Outlet,
  createRootRouteWithContext,
  useRouter,
  useRouterState,
} from "@tanstack/react-router";
import { useEffect } from "react";

interface RouterContext {
  queryClient: QueryClient;
}

export const Route = createRootRouteWithContext<RouterContext>()({
  component: RootLayout,
});

function RootLayout() {
  const router = useRouter();
  const pathname = useRouterState({ select: (s) => s.location.pathname });

  useEffect(() => {
    function onUnauthorized() {
      if (pathname === "/login") return;
      void router.navigate({
        to: "/login",
        search: { redirect: pathname },
      });
    }
    window.addEventListener("tend:unauthorized", onUnauthorized);
    return () => window.removeEventListener("tend:unauthorized", onUnauthorized);
  }, [router, pathname]);

  return <Outlet />;
}

import type { QueryClient } from "@tanstack/react-query";
import {
  Link,
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
  notFoundComponent: NotFoundPage,
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
    window.addEventListener("horologia:unauthorized", onUnauthorized);
    return () => window.removeEventListener("horologia:unauthorized", onUnauthorized);
  }, [router, pathname]);

  return <Outlet />;
}

function NotFoundPage() {
  return (
    <div className="flex min-h-svh items-center justify-center p-4">
      <div className="card preset-outlined-surface-200-800 flex w-full max-w-md flex-col items-center gap-4 p-8 text-center">
        <h1 className="h1">Page not found</h1>
        <p className="text-surface-600-400 text-sm">
          The page you requested does not exist or is not available here.
        </p>
        <Link to="/" className="btn preset-filled-primary-500">
          Go home
        </Link>
      </div>
    </div>
  );
}

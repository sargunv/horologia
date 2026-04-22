import type { QueryClient } from "@tanstack/react-query";
import {
  Link,
  Outlet,
  createRootRouteWithContext,
  useRouter,
  useRouterState,
} from "@tanstack/react-router";
import { useEffect } from "react";
import { Card } from "../ui/Card.tsx";

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
      <Card className="flex w-full max-w-md flex-col items-center gap-4 p-8 text-center">
        <h1 className="text-xl font-semibold">Page not found</h1>
        <p className="text-base-content/70 text-sm">
          The page you requested does not exist or is not available here.
        </p>
        <Link to="/" className="btn btn-primary">
          Go home
        </Link>
      </Card>
    </div>
  );
}

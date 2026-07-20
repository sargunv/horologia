import { Redirect, Slot, useLocalSearchParams } from "expo-router";

import { useSession } from "@/auth/session-context";
import { parseRouteScope, routes, routeScopesMatch } from "@/navigation/routes";
import { AppRuntimeProvider } from "@/runtime/app-runtime";

export default function AuthenticatedScopeLayout() {
  const session = useSession();
  const params = useLocalSearchParams<{
    serverId?: string | string[];
    accountId?: string | string[];
  }>();
  const routeScope = parseRouteScope(params);

  if (session.status === "restoring") return null;
  if (session.status !== "signed-in") {
    return <Redirect href={routes.root()} />;
  }

  const sessionScope = {
    serverId: session.profile.id,
    accountId: session.accountId,
  };
  if (!routeScope || !routeScopesMatch(routeScope, sessionScope)) {
    return <Redirect href={routes.tasks(sessionScope)} />;
  }

  return (
    <AppRuntimeProvider scope={routeScope}>
      <Slot />
    </AppRuntimeProvider>
  );
}

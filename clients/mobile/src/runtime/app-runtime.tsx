import { QueryClientProvider } from "@tanstack/react-query";
import { createContext, type PropsWithChildren, useContext, useEffect, useState } from "react";

import { useSession } from "@/auth/session-context";
import { type RouteScope, routeScopeKey, routeScopesMatch } from "@/navigation/routes";
import {
  type AppRuntime,
  type CreateAppRuntimeOptions,
  createAppRuntime,
  disposeAppRuntime,
} from "@/runtime/create-app-runtime";

const AppRuntimeContext = createContext<AppRuntime | null>(null);

export function AppRuntimeProvider({ scope, children }: PropsWithChildren<{ scope: RouteScope }>) {
  const session = useSession();
  if (session.status !== "signed-in") {
    throw new Error("Authenticated route scope requires an active session");
  }
  const sessionScope = {
    serverId: session.profile.id,
    accountId: session.accountId,
  };
  if (!routeScopesMatch(scope, sessionScope)) {
    throw new Error("Authenticated route scope does not match the active session");
  }

  return (
    <ScopedAppRuntime key={routeScopeKey(scope)} scope={scope} apiClient={session.client}>
      {children}
    </ScopedAppRuntime>
  );
}

function ScopedAppRuntime({
  scope,
  apiClient,
  children,
}: PropsWithChildren<CreateAppRuntimeOptions>) {
  const [runtime] = useState(() => createAppRuntime({ scope, apiClient }));

  useEffect(
    () => () => {
      disposeAppRuntime(runtime);
    },
    [runtime],
  );

  return (
    <AppRuntimeContext.Provider value={runtime}>
      <QueryClientProvider client={runtime.queryClient}>{children}</QueryClientProvider>
    </AppRuntimeContext.Provider>
  );
}

export function useAppRuntime(): AppRuntime {
  const runtime = useContext(AppRuntimeContext);
  if (!runtime) throw new Error("useAppRuntime must be used within AppRuntimeProvider");
  return runtime;
}

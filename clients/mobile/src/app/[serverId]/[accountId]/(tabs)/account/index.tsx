import { useQuery } from "@tanstack/react-query";
import Constants from "expo-constants";
import { useState } from "react";

import { useSession } from "@/auth/session-context";
import { AccountView } from "@/components/native-views";
import { errorMessage } from "@/components/read-model";
import { useAppRuntime } from "@/runtime/app-runtime";

export default function AccountScreen() {
  const session = useSession();
  const runtime = useAppRuntime();
  const userQuery = useQuery(runtime.queries.currentUserQueryOptions);
  const serverInfoQuery = useQuery(runtime.queries.serverInfoQueryOptions);
  const [isSigningOut, setIsSigningOut] = useState(false);
  const [signOutError, setSignOutError] = useState<string | null>(null);

  if (session.status !== "signed-in") return null;

  async function signOut() {
    setIsSigningOut(true);
    setSignOutError(null);
    try {
      await session.signOut();
    } catch (error) {
      setSignOutError(errorMessage(error, "Could not finish signing out."));
      setIsSigningOut(false);
    }
  }

  const queryError = userQuery.error ?? serverInfoQuery.error;
  return (
    <AccountView
      user={userQuery.data ?? null}
      serverUrl={session.profile.baseUrl}
      serverInfo={serverInfoQuery.data ?? null}
      appVersion={Constants.expoConfig?.version ?? "Unknown"}
      isLoading={userQuery.isLoading || serverInfoQuery.isLoading}
      error={queryError ? errorMessage(queryError, "Could not load account details.") : null}
      isSigningOut={isSigningOut}
      signOutError={signOutError}
      onRetry={() => {
        void Promise.all([userQuery.refetch(), serverInfoQuery.refetch()]);
      }}
      onSignOut={() => {
        void signOut();
      }}
    />
  );
}

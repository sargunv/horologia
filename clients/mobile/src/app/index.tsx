import { Redirect } from "expo-router";

import { useSession } from "@/auth/session-context";
import { OnboardingView } from "@/components/native-views";
import { routes } from "@/navigation/routes";

export default function IndexScreen() {
  const session = useSession();

  if (session.status === "signed-in") {
    return (
      <Redirect
        href={routes.tasks({
          serverId: session.profile.id,
          accountId: session.accountId,
        })}
      />
    );
  }

  return (
    <OnboardingView
      status={session.status}
      detail={session.detail}
      serverName={session.profile?.displayName ?? null}
      initialServerUrl={session.profile?.baseUrl ?? ""}
      onConnect={(serverUrl) => {
        void session.connect(serverUrl);
      }}
      onSignIn={() => {
        void session.signIn();
      }}
      onCancel={() => {
        void session.cancelAuthorization();
      }}
      onRecover={() => {
        void session.recover();
      }}
    />
  );
}

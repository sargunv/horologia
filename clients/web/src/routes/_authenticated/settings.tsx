import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { DangerZoneCard } from "../../components/settings/DangerZoneCard.tsx";
import { PasswordCard } from "../../components/settings/PasswordCard.tsx";
import { ProfileCard } from "../../components/settings/ProfileCard.tsx";
import { TokensCard } from "../../components/settings/TokensCard.tsx";
import {
  authConfigQueryOptions,
  authTokensQueryOptions,
  currentUserQueryOptions,
} from "../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/settings")({
  loader: ({ context: { queryClient } }) =>
    Promise.all([
      queryClient.ensureQueryData(currentUserQueryOptions),
      queryClient.ensureQueryData(authConfigQueryOptions),
      queryClient.ensureQueryData(authTokensQueryOptions),
    ]),
  component: SettingsPage,
});

function SettingsPage() {
  const { data: user } = useQuery(currentUserQueryOptions);
  const { data: authConfig } = useQuery(authConfigQueryOptions);
  if (!user) return null;

  return (
    <div className="mx-auto max-w-2xl space-y-6 p-6">
      <div>
        <h1 className="text-2xl font-semibold">Account settings</h1>
        <p className="mt-1 text-sm text-base-content/70">
          Manage your profile and account preferences.
        </p>
      </div>
      <ProfileCard userId={user.id} name={user.name} email={user.email} />
      {authConfig?.password.enabled && (
        <PasswordCard userId={user.id} hasPassword={user.hasPassword} />
      )}
      <TokensCard />
      <DangerZoneCard userId={user.id} email={user.email} />
    </div>
  );
}

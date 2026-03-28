import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute, createLink } from "@tanstack/react-router";
import { ArrowLeft, Gauge, Shield, SignalHigh } from "lucide-react";
import { DangerZoneSection } from "../../../../components/space-settings/DangerZoneSection.tsx";
import { GeneralSettingsSection } from "../../../../components/space-settings/GeneralSettingsSection.tsx";
import { MembersSection } from "../../../../components/space-settings/MembersSection.tsx";
import { SettingsSection } from "../../../../components/space-settings/SettingsSection.tsx";
import {
  currentUserQueryOptions,
  spaceMembersQueryOptions,
  spaceQueryOptions,
} from "../../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug/settings")({
  loader: ({ context: { queryClient }, params: { spaceSlug } }) =>
    Promise.all([
      queryClient.ensureQueryData(spaceQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spaceMembersQueryOptions(spaceSlug)),
    ]),
  component: SpaceSettingsPage,
});

const BackLink = createLink("a");

function SpaceSettingsPage() {
  const { spaceSlug } = Route.useParams();
  const { data: space } = useSuspenseQuery(spaceQueryOptions(spaceSlug));
  const { data: members } = useSuspenseQuery(spaceMembersQueryOptions(spaceSlug));
  const { data: currentUser } = useSuspenseQuery(currentUserQueryOptions);

  const isAdmin = members.some((m) => m.userId === currentUser.id && m.role === "admin");

  return (
    <div className="mx-auto max-w-3xl p-6">
      <BackLink
        to="/spaces/$spaceSlug"
        params={{ spaceSlug }}
        className="text-surface-600-400 hover:text-surface-950-50 mb-4 inline-flex items-center gap-1 text-sm transition-colors"
      >
        <ArrowLeft className="size-4" />
        Back to {space.name}
      </BackLink>

      <h1 className="h3">Space Settings</h1>
      <p className="text-surface-600-400 mt-1">Manage settings for {space.name}.</p>

      <div className="mt-6 flex flex-col gap-4">
        <GeneralSettingsSection space={space} />

        <MembersSection
          spaceSlug={spaceSlug}
          members={members}
          isAdmin={isAdmin}
          currentUserId={currentUser.id}
        />

        <SettingsSection
          icon={<Shield className="size-5" />}
          title="Task Statuses"
          description="Configure the workflow statuses for tasks in this space."
        >
          <p className="text-surface-500 text-sm">Coming soon.</p>
        </SettingsSection>

        <SettingsSection
          icon={<Gauge className="size-5" />}
          title="Effort Levels"
          description="Define effort levels for estimating task complexity."
        >
          <p className="text-surface-500 text-sm">Coming soon.</p>
        </SettingsSection>

        <SettingsSection
          icon={<SignalHigh className="size-5" />}
          title="Priority Levels"
          description="Configure priority levels for organizing tasks."
        >
          <p className="text-surface-500 text-sm">Coming soon.</p>
        </SettingsSection>

        {isAdmin && <DangerZoneSection space={space} />}
      </div>
    </div>
  );
}

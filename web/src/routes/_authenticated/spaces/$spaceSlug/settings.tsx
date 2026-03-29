import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute, createLink } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { DangerZoneSection } from "../../../../components/space-settings/DangerZoneSection.tsx";
import { EffortLevelsSection } from "../../../../components/space-settings/EffortLevelsSection.tsx";
import { GeneralSettingsSection } from "../../../../components/space-settings/GeneralSettingsSection.tsx";
import { MembersSection } from "../../../../components/space-settings/MembersSection.tsx";
import { PriorityLevelsSection } from "../../../../components/space-settings/PriorityLevelsSection.tsx";
import { TagsSection } from "../../../../components/space-settings/TagsSection.tsx";
import { TaskStatusesSection } from "../../../../components/space-settings/TaskStatusesSection.tsx";
import {
  currentUserQueryOptions,
  spaceEffortLevelsQueryOptions,
  spaceMembersQueryOptions,
  spacePriorityLevelsQueryOptions,
  spaceQueryOptions,
  spaceTagsQueryOptions,
  spaceTaskStatusesQueryOptions,
} from "../../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug/settings")({
  loader: ({ context: { queryClient }, params: { spaceSlug } }) =>
    Promise.all([
      queryClient.ensureQueryData(currentUserQueryOptions),
      queryClient.ensureQueryData(spaceTagsQueryOptions(spaceSlug)),
    ]),
  component: SpaceSettingsPage,
});

const BackLink = createLink("a");

function SpaceSettingsPage() {
  const { spaceSlug } = Route.useParams();
  const { data: space } = useSuspenseQuery(spaceQueryOptions(spaceSlug));
  const { data: members } = useSuspenseQuery(spaceMembersQueryOptions(spaceSlug));
  const { data: taskStatuses } = useSuspenseQuery(spaceTaskStatusesQueryOptions(spaceSlug));
  const { data: effortLevels } = useSuspenseQuery(spaceEffortLevelsQueryOptions(spaceSlug));
  const { data: priorityLevels } = useSuspenseQuery(spacePriorityLevelsQueryOptions(spaceSlug));
  const { data: tags } = useSuspenseQuery(spaceTagsQueryOptions(spaceSlug));
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

        {isAdmin && <TaskStatusesSection spaceSlug={spaceSlug} taskStatuses={taskStatuses} />}

        {isAdmin && <EffortLevelsSection spaceSlug={spaceSlug} effortLevels={effortLevels} />}

        {isAdmin && <PriorityLevelsSection spaceSlug={spaceSlug} priorityLevels={priorityLevels} />}

        {isAdmin && <TagsSection spaceSlug={spaceSlug} tags={tags} />}

        {isAdmin && <DangerZoneSection space={space} />}
      </div>
    </div>
  );
}

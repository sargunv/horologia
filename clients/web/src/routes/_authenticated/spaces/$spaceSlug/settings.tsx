import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute, createLink } from "@tanstack/react-router";
import { ArrowLeft, ChevronRight } from "lucide-react";
import {
  DetailPaneHeader,
  DETAIL_PANE_TITLE_CLASS,
} from "../../../../components/DetailPaneHeader.tsx";
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
const BreadcrumbLink = createLink("a");

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
    <div className="space-y-4">
      <DetailPaneHeader
        backLink={
          <BackLink
            to="/spaces/$spaceSlug"
            params={{ spaceSlug }}
            className="text-base-content/70 hover:text-base-content inline-flex items-center gap-1 text-sm transition-colors lg:hidden"
          >
            <ArrowLeft className="size-4" aria-hidden="true" />
            Back to {space.name}
          </BackLink>
        }
        breadcrumb={
          <ol className="flex min-w-0 items-center gap-1 text-sm">
            <li>
              <BreadcrumbLink
                to="/spaces/$spaceSlug"
                params={{ spaceSlug }}
                className="text-base-content/70 truncate hover:underline"
              >
                {space.name}
              </BreadcrumbLink>
            </li>
            <li className="text-base-content/60" aria-hidden="true">
              <ChevronRight className="size-3" />
            </li>
            <li className="shrink-0" aria-current="page">
              Settings
            </li>
          </ol>
        }
        title={<h1 className={DETAIL_PANE_TITLE_CLASS}>Space settings</h1>}
      />

      <div className="flex flex-col gap-4">
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

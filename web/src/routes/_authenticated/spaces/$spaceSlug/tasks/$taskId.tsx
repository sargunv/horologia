import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { ArrowLeft, ChevronRight } from "lucide-react";
import { TaskDetailView } from "../../../../../components/task/TaskDetail.tsx";
import { AnchorLink } from "../../../../../lib/links.ts";
import {
  spaceEffortLevelsQueryOptions,
  spaceMembersQueryOptions,
  spacePriorityLevelsQueryOptions,
  spaceQueryOptions,
  spaceTaskQueryOptions,
  spaceTaskStatusesQueryOptions,
} from "../../../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug/tasks/$taskId")({
  loader: ({ context: { queryClient }, params: { spaceSlug, taskId } }) =>
    Promise.all([
      queryClient.ensureQueryData(spaceQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spaceTaskQueryOptions(spaceSlug, taskId)),
      queryClient.ensureQueryData(spaceMembersQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spaceTaskStatusesQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spaceEffortLevelsQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spacePriorityLevelsQueryOptions(spaceSlug)),
    ]),
  component: TaskDetailPage,
});

function TaskDetailPage() {
  const { spaceSlug, taskId } = Route.useParams();
  const { data: space } = useSuspenseQuery(spaceQueryOptions(spaceSlug));
  const navigate = useNavigate();

  return (
    <TaskDetailView
      spaceSlug={spaceSlug}
      taskId={taskId}
      backLink={
        <AnchorLink
          to="/spaces/$spaceSlug"
          params={{ spaceSlug }}
          className="text-surface-600-400 hover:text-surface-950-50 inline-flex items-center gap-1 text-sm transition-colors lg:hidden"
        >
          <ArrowLeft className="size-4" aria-hidden="true" />
          Back to {space.name}
        </AnchorLink>
      }
      breadcrumb={
        <ol className="flex min-w-0 items-center gap-1 text-sm">
          <li>
            <AnchorLink
              to="/spaces/$spaceSlug"
              params={{ spaceSlug }}
              className="text-surface-600-400 truncate hover:underline"
            >
              {space.name}
            </AnchorLink>
          </li>
          <li className="text-surface-500" aria-hidden="true">
            <ChevronRight className="size-3" />
          </li>
          <li>
            <AnchorLink
              to="/spaces/$spaceSlug/tasks/$taskId"
              params={{ spaceSlug, taskId }}
              className="shrink-0 font-mono hover:underline"
              aria-current="page"
            >
              {taskId}
            </AnchorLink>
          </li>
        </ol>
      }
      onDeleteSuccess={() => {
        void navigate({ to: "/spaces/$spaceSlug", params: { spaceSlug } });
      }}
    />
  );
}

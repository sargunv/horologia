import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { ArrowLeft, ChevronRight } from "lucide-react";
import { TaskDetailView } from "../../../components/task/TaskDetail.tsx";
import { AnchorLink } from "../../../lib/links.ts";
import {
  spaceEffortLevelsQueryOptions,
  spaceMembersQueryOptions,
  spacePriorityLevelsQueryOptions,
  spaceQueryOptions,
  spaceTaskQueryOptions,
  spaceTaskStatusesQueryOptions,
} from "../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/_home/tasks/$spaceSlug/$taskId")({
  loader: ({ context: { queryClient }, params: { spaceSlug, taskId } }) =>
    Promise.all([
      queryClient.ensureQueryData(spaceQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spaceTaskQueryOptions(spaceSlug, taskId)),
      queryClient.ensureQueryData(spaceMembersQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spaceTaskStatusesQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spaceEffortLevelsQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spacePriorityLevelsQueryOptions(spaceSlug)),
    ]),
  component: HomeTaskDetailPage,
});

function HomeTaskDetailPage() {
  const { spaceSlug, taskId } = Route.useParams();
  const navigate = useNavigate();

  return (
    <TaskDetailView
      spaceSlug={spaceSlug}
      taskId={taskId}
      backLink={
        <AnchorLink
          to="/"
          className="text-base-content/70 hover:text-base-content inline-flex items-center gap-1 text-sm transition-colors lg:hidden"
        >
          <ArrowLeft className="size-4" aria-hidden="true" />
          Back to My Tasks
        </AnchorLink>
      }
      breadcrumb={
        <ol className="flex min-w-0 items-center gap-1 text-sm">
          <li>
            <AnchorLink to="/" className="text-base-content/70 truncate hover:underline">
              My Tasks
            </AnchorLink>
          </li>
          <li className="text-base-content/60" aria-hidden="true">
            <ChevronRight className="size-3" />
          </li>
          <li>
            <AnchorLink
              to="/tasks/$spaceSlug/$taskId"
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
        void navigate({ to: "/" });
      }}
    />
  );
}

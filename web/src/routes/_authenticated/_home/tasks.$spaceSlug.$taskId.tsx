import { createFileRoute, createLink, useNavigate } from "@tanstack/react-router";
import { ArrowLeft, ChevronRight } from "lucide-react";
import { TaskDetailView } from "../../../components/task/TaskDetail.tsx";
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

const BackLink = createLink("a");
const BreadcrumbLink = createLink("a");

function HomeTaskDetailPage() {
  const { spaceSlug, taskId } = Route.useParams();
  const navigate = useNavigate();

  return (
    <TaskDetailView
      spaceSlug={spaceSlug}
      taskId={taskId}
      backLink={
        <BackLink
          to="/"
          className="text-surface-600-400 hover:text-surface-950-50 inline-flex items-center gap-1 text-sm transition-colors lg:hidden"
        >
          <ArrowLeft className="size-4" aria-hidden="true" />
          Back to My Tasks
        </BackLink>
      }
      breadcrumb={
        <ol className="flex min-w-0 items-center gap-1 text-sm">
          <li>
            <BreadcrumbLink to="/" className="text-surface-600-400 truncate hover:underline">
              My Tasks
            </BreadcrumbLink>
          </li>
          <li className="text-surface-500" aria-hidden="true">
            <ChevronRight className="size-3" />
          </li>
          <li>
            <BreadcrumbLink
              to="/tasks/$spaceSlug/$taskId"
              params={{ spaceSlug, taskId }}
              className="shrink-0 font-mono hover:underline"
            >
              {taskId}
            </BreadcrumbLink>
          </li>
        </ol>
      }
      onDeleteSuccess={() => {
        void navigate({ to: "/" });
      }}
    />
  );
}

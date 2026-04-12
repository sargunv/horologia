import { Outlet, createFileRoute, useRouterState } from "@tanstack/react-router";
import { ListChecks } from "lucide-react";
import { Suspense } from "react";
import { ListDetailLayout } from "../../../../components/ListDetailLayout.tsx";
import { TaskListPane } from "../../../../components/task/TaskListPane.tsx";
import {
  spaceEffortLevelsQueryOptions,
  spaceMembersQueryOptions,
  spacePriorityLevelsQueryOptions,
  spaceQueryOptions,
  spaceTaskStatusesQueryOptions,
  spaceTasksInfiniteQueryOptions,
} from "../../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug")({
  loader: ({ context: { queryClient }, params: { spaceSlug } }) =>
    Promise.all([
      queryClient.ensureQueryData(spaceQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spaceMembersQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spaceTaskStatusesQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spaceEffortLevelsQueryOptions(spaceSlug)),
      queryClient.ensureQueryData(spacePriorityLevelsQueryOptions(spaceSlug)),
      queryClient.ensureInfiniteQueryData(spaceTasksInfiniteQueryOptions(spaceSlug)),
    ]),
  component: SpaceLayout,
});

function SpaceLayout() {
  const { spaceSlug } = Route.useParams();
  const pathname = useRouterState({ select: (s) => s.location.pathname });

  // Task list-detail layout for the space index and task detail routes.
  // Activity and settings routes render as standalone pages.
  const isTaskRoute =
    pathname === `/spaces/${spaceSlug}` ||
    pathname === `/spaces/${spaceSlug}/` ||
    pathname.startsWith(`/spaces/${spaceSlug}/tasks/`);

  if (!isTaskRoute) {
    return <Outlet />;
  }

  const isTaskDetail = pathname.startsWith(`/spaces/${spaceSlug}/tasks/`);

  return (
    <div className="p-6">
      <ListDetailLayout
        list={
          <Suspense>
            <TaskListPane spaceSlug={spaceSlug} />
          </Suspense>
        }
        detail={
          isTaskDetail ? (
            <Suspense
              fallback={<div className="text-surface-500 p-6 text-center text-sm">Loading...</div>}
            >
              <Outlet />
            </Suspense>
          ) : null
        }
        emptyState={
          <>
            <ListChecks className="text-surface-400 size-12" />
            <span className="text-surface-500 text-sm">Select a task to view details</span>
          </>
        }
      />
    </div>
  );
}

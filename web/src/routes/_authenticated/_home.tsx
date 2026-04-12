import { Outlet, createFileRoute, useRouterState } from "@tanstack/react-router";
import { ListChecks } from "lucide-react";
import { Suspense } from "react";
import type { components } from "../../api/schema.d.ts";
import { ListDetailLayout } from "../../components/ListDetailLayout.tsx";
import { MyTaskListPane } from "../../components/task/MyTaskListPane.tsx";
import {
  currentUserQueryOptions,
  spaceMembersQueryOptions,
  spacesQueryOptions,
  spaceTaskStatusesQueryOptions,
  userTasksInfiniteQueryOptions,
} from "../../lib/queries.ts";

type Space = components["schemas"]["Space"];

export const Route = createFileRoute("/_authenticated/_home")({
  async loader({ context: { queryClient } }) {
    const [user, spaces] = await Promise.all([
      queryClient.ensureQueryData(currentUserQueryOptions),
      queryClient.ensureQueryData(spacesQueryOptions),
    ]);
    await Promise.all([
      queryClient.ensureInfiniteQueryData(userTasksInfiniteQueryOptions(user.id)),
      ...spaces.map((s: Space) =>
        queryClient.ensureQueryData(spaceTaskStatusesQueryOptions(s.slug)),
      ),
      ...spaces.map((s: Space) => queryClient.ensureQueryData(spaceMembersQueryOptions(s.slug))),
    ]);
  },
  component: HomeLayout,
});

function HomeLayout() {
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const isTaskDetail = pathname.startsWith("/tasks/");

  return (
    <div className="p-6">
      <ListDetailLayout
        list={
          <Suspense>
            <MyTaskListPane />
          </Suspense>
        }
        detail={
          isTaskDetail ? (
            <Suspense
              fallback={<div className="text-surface-500 p-6 text-center text-sm">Loading…</div>}
            >
              <Outlet />
            </Suspense>
          ) : null
        }
        emptyState={
          <>
            <ListChecks className="text-surface-400 size-12" aria-hidden="true" />
            <span className="text-surface-500 text-sm">Select a task to view details</span>
          </>
        }
      />
    </div>
  );
}

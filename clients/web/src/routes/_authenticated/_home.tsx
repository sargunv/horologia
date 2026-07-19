import { Outlet, createFileRoute, useMatchRoute } from "@tanstack/react-router";
import { ListChecks } from "lucide-react";
import { Suspense } from "react";
import type { components } from "@horologia/client-core/schema";
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
  const matchRoute = useMatchRoute();
  const hasDetail =
    !!matchRoute({ to: "/tasks/$spaceSlug/$taskId" }) || !!matchRoute({ to: "/activity" });

  return (
    <div className="p-6">
      <ListDetailLayout
        list={
          <Suspense>
            <MyTaskListPane />
          </Suspense>
        }
        detail={
          hasDetail ? (
            <Suspense
              fallback={
                <div className="text-base-content/60 p-6 text-center text-sm">Loading…</div>
              }
            >
              <Outlet />
            </Suspense>
          ) : null
        }
        emptyState={
          <>
            <ListChecks className="text-base-content/40 size-12" aria-hidden="true" />
            <span className="text-base-content/60 text-sm">Select a task to view details</span>
          </>
        }
      />
    </div>
  );
}

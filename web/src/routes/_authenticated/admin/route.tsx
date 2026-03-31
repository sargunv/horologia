import {
  Outlet,
  createFileRoute,
  redirect,
  useNavigate,
  useRouterState,
} from "@tanstack/react-router";
import { Shield } from "lucide-react";
import { Tabs } from "@skeletonlabs/skeleton-react";
import { currentUserQueryOptions, usersQueryOptions } from "../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/admin")({
  async beforeLoad({ context }) {
    const user = await context.queryClient.ensureQueryData(currentUserQueryOptions);
    if (!user.isOwner) {
      throw redirect({ to: "/" });
    }
    void context.queryClient.ensureQueryData(usersQueryOptions);
  },
  component: AdminLayout,
});

function AdminLayout() {
  const navigate = useNavigate();
  const pathname = useRouterState({ select: (s) => s.location.pathname });

  const activeTab = "users";
  void pathname; // Will derive from pathname when more tabs are added.

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center gap-3">
        <Shield className="text-surface-600-400 size-6" aria-hidden="true" />
        <h1 className="h3 flex-1">Admin</h1>
      </div>

      <Tabs
        value={activeTab}
        onValueChange={(details) => {
          if (details.value === "users") void navigate({ to: "/admin/users" });
        }}
      >
        <Tabs.List>
          <Tabs.Trigger value="users">Users</Tabs.Trigger>
          <Tabs.Indicator />
        </Tabs.List>
      </Tabs>

      <div className="mt-6">
        <Outlet />
      </div>
    </div>
  );
}

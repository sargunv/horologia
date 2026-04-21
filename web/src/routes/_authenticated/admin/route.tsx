import {
  Outlet,
  createFileRoute,
  redirect,
  useNavigate,
  useRouterState,
} from "@tanstack/react-router";
import { Shield } from "lucide-react";
import { currentUserQueryOptions, usersQueryOptions } from "../../../lib/queries.ts";
import { TabsList, TabsRoot, TabsTrigger } from "../../../ui/Tabs.tsx";

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

const TAB_ROUTES: Record<string, string> = {
  users: "/admin/users",
  about: "/admin/about",
};

function AdminLayout() {
  const navigate = useNavigate();
  const pathname = useRouterState({ select: (s) => s.location.pathname });

  const activeTab = pathname.startsWith("/admin/about") ? "about" : "users";

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center gap-3">
        <Shield className="size-6 text-base-content/70" aria-hidden="true" />
        <h1 className="flex-1 text-2xl font-semibold">Admin</h1>
      </div>

      <TabsRoot
        value={activeTab}
        onValueChange={(value) => {
          const route = TAB_ROUTES[value];
          if (route) void navigate({ to: route });
        }}
      >
        <TabsList>
          <TabsTrigger value="users">Users</TabsTrigger>
          <TabsTrigger value="about">About</TabsTrigger>
        </TabsList>
      </TabsRoot>

      <div className="mt-6">
        <Outlet />
      </div>
    </div>
  );
}

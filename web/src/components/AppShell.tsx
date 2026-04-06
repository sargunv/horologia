import { Navigation, Toast } from "@skeletonlabs/skeleton-react";
import { createLink } from "@tanstack/react-router";
import { Activity, CircleUser, House, Layers, LayoutGrid, Plus, X } from "lucide-react";
import type { ReactNode } from "react";
import type { components } from "../api/schema.d.ts";
import { toaster } from "../lib/toaster.ts";
import { UserMenu } from "./UserMenu.tsx";

type User = components["schemas"]["User"];
type Space = components["schemas"]["Space"];

const NavLink = createLink(Navigation.TriggerAnchor);
const PlainLink = createLink("a");

function DesktopSidebar({ user, spaces }: { user: User; spaces: Space[] }) {
  return (
    <Navigation
      layout="sidebar"
      className="hidden shrink-0 border-r border-surface-200-800 md:flex md:flex-col"
    >
      <Navigation.Header>
        <NavLink to="/" className="flex items-center gap-2 px-1">
          <span className="text-lg font-bold">Tend</span>
        </NavLink>
      </Navigation.Header>

      <Navigation.Content className="flex-1">
        <Navigation.Group>
          <Navigation.Menu>
            <NavLink
              to="/"
              activeProps={{ className: "preset-filled-primary-500" }}
              activeOptions={{ exact: true }}
            >
              <House className="size-5" />
              <Navigation.TriggerText>Home</Navigation.TriggerText>
            </NavLink>
            <NavLink to="/activity" activeProps={{ className: "preset-filled-primary-500" }}>
              <Activity className="size-5" />
              <Navigation.TriggerText>Activity</Navigation.TriggerText>
            </NavLink>
          </Navigation.Menu>
        </Navigation.Group>

        <Navigation.Group>
          <div className="flex items-center justify-between">
            <PlainLink to="/spaces" className="unstyled">
              <Navigation.Label>Spaces</Navigation.Label>
            </PlainLink>
            <PlainLink
              to="/spaces/new"
              className="text-surface-600-400 hover:text-surface-900-100 transition-colors"
              aria-label="Create space"
            >
              <Plus className="size-4" />
            </PlainLink>
          </div>
          <Navigation.Menu>
            {spaces.map((space) => (
              <NavLink
                key={space.slug}
                to="/spaces/$spaceSlug"
                params={{ spaceSlug: space.slug }}
                activeProps={{ className: "preset-filled-primary-500" }}
              >
                <LayoutGrid className="size-5" />
                <Navigation.TriggerText>{space.name}</Navigation.TriggerText>
              </NavLink>
            ))}
            {spaces.length === 0 && <div className="text-surface-500 text-sm">No spaces yet</div>}
          </Navigation.Menu>
        </Navigation.Group>
      </Navigation.Content>

      <Navigation.Footer>
        <UserMenu user={user} />
      </Navigation.Footer>
    </Navigation>
  );
}

function MobileBar() {
  return (
    <Navigation
      layout="bar"
      className="fixed inset-x-0 bottom-0 z-40 border-t border-surface-200-800 md:hidden"
    >
      <Navigation.Menu>
        <NavLink
          to="/"
          activeProps={{ className: "text-primary-500" }}
          activeOptions={{ exact: true }}
        >
          <House className="size-5" />
          <Navigation.TriggerText>Home</Navigation.TriggerText>
        </NavLink>
        <NavLink to="/activity" activeProps={{ className: "text-primary-500" }}>
          <Activity className="size-5" />
          <Navigation.TriggerText>Activity</Navigation.TriggerText>
        </NavLink>
        <NavLink to="/spaces" activeProps={{ className: "text-primary-500" }}>
          <Layers className="size-5" />
          <Navigation.TriggerText>Spaces</Navigation.TriggerText>
        </NavLink>
        <NavLink to="/settings" activeProps={{ className: "text-primary-500" }}>
          <CircleUser className="size-5" />
          <Navigation.TriggerText>Account</Navigation.TriggerText>
        </NavLink>
      </Navigation.Menu>
    </Navigation>
  );
}

export function AppShell({
  user,
  spaces,
  children,
}: {
  user: User;
  spaces: Space[];
  children: ReactNode;
}) {
  return (
    <div className="grid h-svh grid-cols-1 md:grid-cols-[auto_1fr]">
      <DesktopSidebar user={user} spaces={spaces} />
      <main className="overflow-y-auto pb-16 md:pb-0">{children}</main>
      <MobileBar />
      <Toast.Group toaster={toaster}>
        {(toast) => (
          <Toast
            toast={toast}
            className={`card flex items-start gap-3 p-4 shadow-xl ${
              toast.type === "error"
                ? "preset-filled-error-500"
                : toast.type === "warning"
                  ? "preset-tonal-warning"
                  : "preset-filled-surface-200-800"
            }`}
          >
            <div className="flex-1">
              {toast.title && <Toast.Title className="font-semibold">{toast.title}</Toast.Title>}
              {toast.description && (
                <Toast.Description className="text-sm opacity-80">
                  {toast.description}
                </Toast.Description>
              )}
            </div>
            <Toast.CloseTrigger
              className="btn btn-icon btn-sm opacity-70 hover:opacity-100"
              aria-label="Dismiss"
            >
              <X className="size-4" aria-hidden="true" />
            </Toast.CloseTrigger>
          </Toast>
        )}
      </Toast.Group>
    </div>
  );
}

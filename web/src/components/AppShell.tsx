import { Link, useRouterState } from "@tanstack/react-router";
import { CircleUser, House, Layers, LayoutGrid, Plus } from "lucide-react";
import type { ReactNode } from "react";
import type { components } from "../api/schema.d.ts";
import { UserMenu } from "./UserMenu.tsx";

type User = components["schemas"]["User"];
type Space = components["schemas"]["Space"];

function DesktopSidebar({
  user,
  spaces,
  pathname,
}: {
  user: User;
  spaces: Space[];
  pathname: string;
}) {
  return (
    <aside className="bg-surface-100-900 hidden h-svh w-56 shrink-0 flex-col border-r border-surface-200-800 md:flex">
      <header className="p-3">
        <Link to="/" className="flex items-center gap-2 px-1">
          <span className="text-lg font-bold">Tend</span>
        </Link>
      </header>

      <nav className="flex flex-1 flex-col gap-1 overflow-y-auto p-2">
        <NavLink href="/" active={pathname === "/"} icon={<House className="size-5" />}>
          Home
        </NavLink>

        <div className="mt-3">
          <div className="text-surface-600-400 flex items-center justify-between px-2 py-1 text-xs font-medium uppercase tracking-wider">
            <span>Spaces</span>
            <Link
              to="/spaces/new"
              className="text-surface-600-400 hover:text-surface-900-100 transition-colors"
              aria-label="Create space"
            >
              <Plus className="size-4" />
            </Link>
          </div>
          {spaces.map((space) => (
            <NavLink
              key={space.slug}
              href={`/spaces/${space.slug}`}
              active={
                pathname === `/spaces/${space.slug}` ||
                pathname.startsWith(`/spaces/${space.slug}/`)
              }
              icon={<LayoutGrid className="size-5" />}
            >
              {space.name}
            </NavLink>
          ))}
          {spaces.length === 0 && (
            <div className="text-surface-500 px-2 py-1 text-sm">No spaces yet</div>
          )}
        </div>
      </nav>

      <footer className="border-t border-surface-200-800 p-2">
        <UserMenu user={user} />
      </footer>
    </aside>
  );
}

function MobileBar({ pathname }: { pathname: string }) {
  return (
    <nav className="bg-surface-100-900 fixed inset-x-0 bottom-0 z-40 flex items-center justify-around border-t border-surface-200-800 p-1 md:hidden">
      <MobileTab href="/" active={pathname === "/"} label="Home">
        <House className="size-5" />
      </MobileTab>
      <MobileTab href="/spaces" active={pathname.startsWith("/spaces")} label="Spaces">
        <Layers className="size-5" />
      </MobileTab>
      <MobileTab
        href="/settings"
        active={pathname.startsWith("/settings") || pathname.startsWith("/admin")}
        label="Account"
      >
        <CircleUser className="size-5" />
      </MobileTab>
    </nav>
  );
}

function NavLink({
  href,
  active,
  icon,
  children,
}: {
  href: string;
  active: boolean;
  icon: ReactNode;
  children: ReactNode;
}) {
  return (
    <Link
      to={href}
      className={`flex items-center gap-2.5 rounded-md px-2 py-1.5 text-sm font-medium transition-colors ${
        active
          ? "preset-filled-primary-500"
          : "text-surface-600-400 hover:bg-surface-200-800 hover:text-surface-900-100"
      }`}
    >
      {icon}
      <span>{children}</span>
    </Link>
  );
}

function MobileTab({
  href,
  active,
  label,
  children,
}: {
  href: string;
  active: boolean;
  label: string;
  children: ReactNode;
}) {
  return (
    <Link
      to={href}
      className={`flex flex-col items-center gap-0.5 px-3 py-1.5 text-xs transition-colors ${
        active ? "text-primary-500" : "text-surface-600-400"
      }`}
    >
      {children}
      <span>{label}</span>
    </Link>
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
  const pathname = useRouterState({ select: (s) => s.location.pathname });

  return (
    <div className="flex h-svh">
      <DesktopSidebar user={user} spaces={spaces} pathname={pathname} />
      <main className="flex-1 overflow-y-auto pb-16 md:pb-0">{children}</main>
      <MobileBar pathname={pathname} />
    </div>
  );
}

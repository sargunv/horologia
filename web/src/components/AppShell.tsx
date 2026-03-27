import { Link, useRouterState } from "@tanstack/react-router";
import type { ReactNode } from "react";
import type { components } from "../api/schema.d.ts";
import { UserMenu } from "./UserMenu.tsx";

type User = components["schemas"]["User"];
type Space = components["schemas"]["Space"];

function HomeIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 20 20"
      fill="currentColor"
      className={className}
    >
      <path
        fillRule="evenodd"
        d="M9.293 2.293a1 1 0 0 1 1.414 0l7 7A1 1 0 0 1 17 11h-1v6a1 1 0 0 1-1 1h-2a1 1 0 0 1-1-1v-3a1 1 0 0 0-1-1H9a1 1 0 0 0-1 1v3a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1v-6H3a1 1 0 0 1-.707-1.707l7-7Z"
        clipRule="evenodd"
      />
    </svg>
  );
}

function PlusIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 20 20"
      fill="currentColor"
      className={className}
    >
      <path d="M10.75 4.75a.75.75 0 0 0-1.5 0v4.5h-4.5a.75.75 0 0 0 0 1.5h4.5v4.5a.75.75 0 0 0 1.5 0v-4.5h4.5a.75.75 0 0 0 0-1.5h-4.5v-4.5Z" />
    </svg>
  );
}

function SpaceIcon({ name }: { name: string }) {
  const initial = name.charAt(0).toUpperCase();
  return (
    <span className="preset-filled-surface-200-800 inline-flex size-5 items-center justify-center rounded text-xs font-semibold">
      {initial}
    </span>
  );
}

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
        <NavLink href="/" active={pathname === "/"} icon={<HomeIcon className="size-5" />}>
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
              <PlusIcon className="size-4" />
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
              icon={<SpaceIcon name={space.name} />}
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
        <HomeIcon className="size-5" />
      </MobileTab>
      <MobileTab href="/spaces" active={pathname.startsWith("/spaces")} label="Spaces">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          className="size-5"
        >
          <path d="M5.127 3.502 5.25 3.5h9.5c.041 0 .082 0 .123.002A2.251 2.251 0 0 0 12.75 2h-5.5a2.25 2.25 0 0 0-2.123 1.502ZM1 10.25A2.25 2.25 0 0 1 3.25 8h13.5A2.25 2.25 0 0 1 19 10.25v5.5A2.25 2.25 0 0 1 16.75 18H3.25A2.25 2.25 0 0 1 1 15.75v-5.5ZM3.25 6.5c-.04 0-.082 0-.123.002A2.25 2.25 0 0 1 5.25 5h9.5c.98 0 1.814.627 2.123 1.502a3.819 3.819 0 0 0-.123-.002H3.25Z" />
        </svg>
      </MobileTab>
      <MobileTab
        href="/settings"
        active={pathname.startsWith("/settings") || pathname.startsWith("/admin")}
        label="Account"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          className="size-5"
        >
          <path d="M10 8a3 3 0 1 0 0-6 3 3 0 0 0 0 6ZM3.465 14.493a1.23 1.23 0 0 0 .41 1.412A9.957 9.957 0 0 0 10 18c2.31 0 4.438-.784 6.131-2.1.43-.333.604-.903.408-1.41a7.002 7.002 0 0 0-13.074.003Z" />
        </svg>
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

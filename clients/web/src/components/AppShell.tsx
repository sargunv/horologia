import { createLink, useRouterState } from "@tanstack/react-router";
import { CircleUser, CookingPot, Layers, LayoutGrid, ListChecks, Plus, Search } from "lucide-react";
import type { ReactNode } from "react";
import type { components } from "@horologia/client-core/schema";
import { Toaster } from "../ui/Toaster.tsx";
import { TooltipProvider } from "../ui/Tooltip.tsx";
import { GlobalSearchCombobox } from "./GlobalSearchCombobox.tsx";
import { UserMenu } from "./UserMenu.tsx";

type User = components["schemas"]["User"];
type Space = components["schemas"]["Space"];

const NavAnchor = createLink("a");

const SIDEBAR_LINK =
  "group flex items-center gap-2.5 rounded-field px-2 py-1.5 text-sm " +
  "text-base-content/70 border-l-2 border-transparent " +
  "transition-colors duration-100 hover:bg-base-100 hover:text-base-content";

const SIDEBAR_LINK_ACTIVE =
  "bg-base-100 text-base-content font-medium " +
  "border-l-primary " +
  "shadow-[inset_0_0_0_1px_var(--color-base-300)]";

const MOBILE_LINK =
  "flex flex-1 flex-col items-center justify-center gap-0.5 py-2 text-xs font-medium text-base-content/70 hover:text-base-content";

const MOBILE_LINK_ACTIVE = "text-primary";

const SECTION_LABEL =
  "px-2 pb-1.5 pt-0.5 text-3xs font-semibold uppercase tracking-caps text-base-content/70";

function DesktopSidebar({ user, spaces }: { user: User; spaces: Space[] }) {
  return (
    <aside className="hidden w-64 shrink-0 flex-col border-r border-base-300 bg-base-200 md:flex">
      <header className="px-4 pt-4 pb-2">
        <NavAnchor to="/" className="flex items-center gap-2">
          <span className="text-lg font-bold tracking-tight">Horologia</span>
        </NavAnchor>
      </header>

      <div className="px-2 pt-2 pb-1">
        <GlobalSearchCombobox />
      </div>

      <div className="flex-1 space-y-3 overflow-y-auto px-2 py-2">
        <nav aria-label="Library" className="space-y-0.5">
          <div className={SECTION_LABEL}>Library</div>
          <NavAnchor
            to="/"
            className={SIDEBAR_LINK}
            activeProps={{ className: `${SIDEBAR_LINK} ${SIDEBAR_LINK_ACTIVE}` }}
            activeOptions={{ exact: true }}
          >
            <ListChecks className="size-4" aria-hidden="true" />
            <span className="flex-1 truncate">My tasks</span>
          </NavAnchor>
          <NavAnchor
            to="/recipes"
            className={SIDEBAR_LINK}
            activeProps={{ className: `${SIDEBAR_LINK} ${SIDEBAR_LINK_ACTIVE}` }}
          >
            <CookingPot className="size-4" aria-hidden="true" />
            <span className="flex-1 truncate">Recipes</span>
          </NavAnchor>
        </nav>

        <nav aria-label="Spaces" className="space-y-0.5">
          <div className="flex items-center justify-between">
            <NavAnchor to="/spaces" className={`${SECTION_LABEL} hover:text-base-content`}>
              Spaces
            </NavAnchor>
            <NavAnchor
              to="/spaces/new"
              className="rounded-field p-1 text-base-content/60 transition-colors hover:bg-base-100 hover:text-base-content"
              aria-label="Create space"
            >
              <Plus className="size-3.5" aria-hidden="true" />
            </NavAnchor>
          </div>
          <div className="space-y-0.5">
            {spaces.map((space) => (
              <NavAnchor
                key={space.slug}
                to="/spaces/$spaceSlug"
                params={{ spaceSlug: space.slug }}
                className={SIDEBAR_LINK}
                activeProps={{
                  className: `${SIDEBAR_LINK} ${SIDEBAR_LINK_ACTIVE}`,
                }}
              >
                <LayoutGrid className="size-4" aria-hidden="true" />
                <span className="flex-1 truncate">{space.name}</span>
              </NavAnchor>
            ))}
            {spaces.length === 0 && (
              <div className="px-2 text-xs text-base-content/55">No spaces yet</div>
            )}
          </div>
        </nav>
      </div>

      <footer className="border-t border-base-300 p-3">
        <UserMenu user={user} />
      </footer>
    </aside>
  );
}

function MobileBar() {
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const itemClass = (active: boolean) => `${MOBILE_LINK} ${active ? MOBILE_LINK_ACTIVE : ""}`;
  return (
    <nav className="fixed inset-x-0 bottom-0 z-40 flex border-t border-base-300 bg-base-100 md:hidden">
      <NavAnchor
        to="/"
        className={itemClass(
          pathname === "/" || pathname.startsWith("/tasks/") || pathname === "/activity",
        )}
      >
        <ListChecks className="size-5" />
        <span>Tasks</span>
      </NavAnchor>
      <NavAnchor
        to="/library"
        className={itemClass(
          pathname.startsWith("/library") ||
            pathname.startsWith("/recipes") ||
            pathname.startsWith("/spaces"),
        )}
      >
        <Layers className="size-5" />
        <span>Library</span>
      </NavAnchor>
      <NavAnchor to="/search" className={itemClass(pathname.startsWith("/search"))}>
        <Search className="size-5" />
        <span>Search</span>
      </NavAnchor>
      <NavAnchor to="/settings" className={itemClass(pathname.startsWith("/settings"))}>
        <CircleUser className="size-5" />
        <span>Account</span>
      </NavAnchor>
    </nav>
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
    <TooltipProvider delayDuration={300}>
      <div className="flex h-svh bg-base-100 text-base-content">
        <DesktopSidebar user={user} spaces={spaces} />
        <main className="flex-1 overflow-y-auto pb-16 md:pb-0">{children}</main>
        <MobileBar />
        <Toaster />
      </div>
    </TooltipProvider>
  );
}

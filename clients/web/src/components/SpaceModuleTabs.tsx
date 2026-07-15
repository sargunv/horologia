import { CookingPot, ListChecks } from "lucide-react";
import { useRouterState } from "@tanstack/react-router";
import { AnchorLink } from "../lib/links.ts";

const TAB_CLASS =
  "-mb-px inline-flex items-center gap-1.5 border-b-2 border-transparent px-3 py-2 text-sm font-medium text-base-content/70 transition-colors hover:text-base-content";
const TAB_ACTIVE = "border-primary text-base-content";

export function SpaceModuleTabs({ spaceSlug }: { spaceSlug: string }) {
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const recipesActive = pathname.startsWith(`/spaces/${spaceSlug}/recipes`);
  return (
    <nav className="flex border-b border-base-300" aria-label="Space content">
      <AnchorLink
        to="/spaces/$spaceSlug"
        params={{ spaceSlug }}
        className={`${TAB_CLASS} ${recipesActive ? "" : TAB_ACTIVE}`}
      >
        <ListChecks className="size-4" aria-hidden="true" />
        Tasks
      </AnchorLink>
      <AnchorLink
        to="/spaces/$spaceSlug/recipes"
        params={{ spaceSlug }}
        className={`${TAB_CLASS} ${recipesActive ? TAB_ACTIVE : ""}`}
      >
        <CookingPot className="size-4" aria-hidden="true" />
        Recipes
      </AnchorLink>
    </nav>
  );
}

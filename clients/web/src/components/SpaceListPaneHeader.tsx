import { createLink } from "@tanstack/react-router";
import { Activity, Settings } from "lucide-react";
import { TooltipContent, TooltipRoot, TooltipTrigger } from "../ui/Tooltip.tsx";
import { SpaceModuleTabs } from "./SpaceModuleTabs.tsx";

const ActivityLink = createLink("a");
const SettingsLink = createLink("a");

export function SpaceListPaneHeader({ spaceSlug, name }: { spaceSlug: string; name: string }) {
  return (
    <>
      <div className="flex items-center justify-between">
        <h2 className="truncate text-lg font-semibold">{name}</h2>
        <div className="flex shrink-0 items-center gap-1">
          <TooltipRoot>
            <TooltipTrigger asChild>
              <ActivityLink
                to="/spaces/$spaceSlug/activity"
                params={{ spaceSlug }}
                className="btn btn-soft btn-square btn-sm"
                aria-label="Activity"
              >
                <Activity className="size-4" aria-hidden="true" />
              </ActivityLink>
            </TooltipTrigger>
            <TooltipContent>Activity</TooltipContent>
          </TooltipRoot>
          <TooltipRoot>
            <TooltipTrigger asChild>
              <SettingsLink
                to="/spaces/$spaceSlug/settings"
                params={{ spaceSlug }}
                className="btn btn-soft btn-square btn-sm"
                aria-label="Settings"
              >
                <Settings className="size-4" aria-hidden="true" />
              </SettingsLink>
            </TooltipTrigger>
            <TooltipContent>Settings</TooltipContent>
          </TooltipRoot>
        </div>
      </div>
      <SpaceModuleTabs spaceSlug={spaceSlug} />
    </>
  );
}

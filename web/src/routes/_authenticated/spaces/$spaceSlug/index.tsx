import { createFileRoute, createLink } from "@tanstack/react-router";
import { useSuspenseQuery } from "@tanstack/react-query";
import { Settings } from "lucide-react";
import { spaceQueryOptions } from "../../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug/")({
  component: SpacePage,
});

const SettingsLink = createLink("a");

function SpacePage() {
  const { spaceSlug } = Route.useParams();
  const { data: space } = useSuspenseQuery(spaceQueryOptions(spaceSlug));

  return (
    <div className="p-6">
      <div className="flex items-center justify-between">
        <h1 className="h3">{space.name}</h1>
        <SettingsLink
          to="/spaces/$spaceSlug/settings"
          params={{ spaceSlug }}
          className="btn preset-outlined-surface-200-800 flex items-center gap-2"
        >
          <Settings className="size-4" />
          Settings
        </SettingsLink>
      </div>
      <p className="text-surface-600-400 mt-1">Tasks will appear here.</p>
    </div>
  );
}

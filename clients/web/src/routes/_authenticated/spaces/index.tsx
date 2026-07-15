import { createLink, createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { LayoutGrid, Plus } from "lucide-react";
import { spacesQueryOptions } from "../../../lib/queries.ts";
import { Card } from "../../../ui/Card.tsx";

export const Route = createFileRoute("/_authenticated/spaces/")({
  component: SpacesPage,
});

const SpaceLink = createLink("a");

function SpacesPage() {
  const { data: spaces, isLoading } = useQuery(spacesQueryOptions);

  return (
    <div className="p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">Spaces</h1>
        <SpaceLink to="/spaces/new" className="btn btn-primary flex items-center gap-2">
          <Plus className="size-4" />
          Create space
        </SpaceLink>
      </div>

      <div className="mt-6">
        {isLoading ? (
          <div className="text-base-content/70 text-sm">Loading...</div>
        ) : spaces && spaces.length > 0 ? (
          <div className="flex flex-col gap-3">
            {spaces.map((space) => (
              <SpaceLink
                key={space.slug}
                to="/spaces/$spaceSlug"
                params={{ spaceSlug: space.slug }}
                className="rounded-box bg-base-100 border border-base-300 flex items-center gap-4 p-4 transition-colors hover:bg-base-100"
              >
                <LayoutGrid className="text-primary size-6 shrink-0" />
                <div className="min-w-0">
                  <p className="font-medium">{space.name}</p>
                  <p className="text-base-content/70 truncate text-sm">{space.slug}</p>
                </div>
              </SpaceLink>
            ))}
          </div>
        ) : (
          <Card className="flex flex-col items-center gap-3 p-12 text-center">
            <LayoutGrid className="text-base-content/40 size-12" />
            <div>
              <p className="font-medium">No spaces yet</p>
              <p className="text-base-content/70 mt-1 text-sm">
                Create your first space to start organizing what belongs together.
              </p>
            </div>
            <SpaceLink to="/spaces/new" className="btn btn-primary mt-2">
              Create space
            </SpaceLink>
          </Card>
        )}
      </div>
    </div>
  );
}

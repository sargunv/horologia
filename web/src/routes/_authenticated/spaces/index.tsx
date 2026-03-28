import { createLink, createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { LayoutGrid, Plus } from "lucide-react";
import { spacesQueryOptions } from "../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/spaces/")({
  component: SpacesPage,
});

const SpaceLink = createLink("a");

function SpacesPage() {
  const { data: spaces, isLoading } = useQuery(spacesQueryOptions);

  return (
    <div className="p-6">
      <div className="flex items-center justify-between">
        <h1 className="h3">Spaces</h1>
        <SpaceLink
          to="/spaces/new"
          className="btn preset-filled-primary-500 flex items-center gap-2"
        >
          <Plus className="size-4" />
          Create space
        </SpaceLink>
      </div>

      <div className="mt-6">
        {isLoading ? (
          <div className="text-surface-600-400 text-sm">Loading...</div>
        ) : spaces && spaces.length > 0 ? (
          <div className="flex flex-col gap-3">
            {spaces.map((space) => (
              <SpaceLink
                key={space.slug}
                to="/spaces/$spaceSlug"
                params={{ spaceSlug: space.slug }}
                className="card preset-outlined-surface-200-800 flex items-center gap-4 p-4 transition-colors hover:preset-filled-surface-100-900"
              >
                <LayoutGrid className="text-primary-500 size-6 shrink-0" />
                <div className="min-w-0">
                  <p className="font-medium">{space.name}</p>
                  {space.description && (
                    <p className="text-surface-600-400 truncate text-sm">{space.description}</p>
                  )}
                </div>
              </SpaceLink>
            ))}
          </div>
        ) : (
          <div className="card preset-outlined-surface-200-800 flex flex-col items-center gap-3 p-12 text-center">
            <LayoutGrid className="text-surface-400 size-12" />
            <div>
              <p className="font-medium">No spaces yet</p>
              <p className="text-surface-600-400 mt-1 text-sm">
                Create your first space to start organizing tasks.
              </p>
            </div>
            <SpaceLink to="/spaces/new" className="btn preset-filled-primary-500 mt-2">
              Create space
            </SpaceLink>
          </div>
        )}
      </div>
    </div>
  );
}

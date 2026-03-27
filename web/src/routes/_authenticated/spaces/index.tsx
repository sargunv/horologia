import { useQuery } from "@tanstack/react-query";
import { Link, createFileRoute } from "@tanstack/react-router";
import { spacesQueryOptions } from "../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/spaces/")({
  component: SpacesPage,
});

function SpacesPage() {
  const { data: spaces } = useQuery(spacesQueryOptions);

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="h3">Spaces</h1>
        <Link to="/spaces/new" className="btn preset-filled-primary-500">
          Create space
        </Link>
      </div>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {spaces?.map((space) => (
          <Link
            key={space.slug}
            to={`/spaces/${space.slug}`}
            className="card preset-outlined-surface-200-800 p-4 transition-colors hover:border-primary-500"
          >
            <h2 className="font-medium">{space.name}</h2>
            {space.description && (
              <p className="text-surface-600-400 mt-1 text-sm">{space.description}</p>
            )}
          </Link>
        ))}
        {spaces?.length === 0 && (
          <p className="text-surface-600-400">No spaces yet. Create one to get started.</p>
        )}
      </div>
    </div>
  );
}

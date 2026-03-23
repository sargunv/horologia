import { useSuspenseQuery } from "@tanstack/react-query";
import { Link, createFileRoute } from "@tanstack/react-router";
import { spacesQueryOptions } from "../../queries/spaces.ts";

export const Route = createFileRoute("/_auth/spaces")({
  loader: ({ context }) => context.queryClient.ensureQueryData(spacesQueryOptions()),
  component: SpaceListPage,
});

function SpaceListPage() {
  const { data } = useSuspenseQuery(spacesQueryOptions());

  return (
    <div className="space-y-4">
      <h1 className="h2">Spaces</h1>
      {data.items.length === 0 ? (
        <p className="text-surface-500">No spaces yet.</p>
      ) : (
        <div className="grid gap-3">
          {data.items.map((space) => (
            <Link
              key={space.slug}
              to="/spaces/$spaceSlug"
              params={{ spaceSlug: space.slug }}
              className="card p-4 hover:preset-tonal-primary transition-colors"
            >
              <h2 className="h4">{space.name}</h2>
              {space.description ? (
                <p className="text-sm text-surface-500">{space.description}</p>
              ) : null}
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

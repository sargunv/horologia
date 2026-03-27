import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug")({
  component: SpacePage,
});

function SpacePage() {
  const { spaceSlug } = Route.useParams();

  return (
    <div className="p-6">
      <h1 className="h3">{spaceSlug}</h1>
      <p className="text-surface-600-400 mt-1">Tasks will appear here.</p>
    </div>
  );
}

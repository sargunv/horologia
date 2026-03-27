import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/spaces/")({
  component: SpacesPage,
});

function SpacesPage() {
  return (
    <div className="p-6">
      <h1 className="h3">Spaces</h1>
      <p className="text-surface-600-400 mt-1">Space list coming soon.</p>
    </div>
  );
}

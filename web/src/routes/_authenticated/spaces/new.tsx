import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/spaces/new")({
  component: NewSpacePage,
});

function NewSpacePage() {
  return (
    <div className="p-6">
      <h1 className="h3">Create space</h1>
      <p className="text-surface-600-400 mt-1">Space creation form coming soon.</p>
    </div>
  );
}

import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug/settings")({
  component: SpaceSettingsPage,
});

function SpaceSettingsPage() {
  return (
    <div className="p-6">
      <h1 className="h3">Space Settings</h1>
      <p className="text-surface-600-400 mt-1">Space settings coming soon.</p>
    </div>
  );
}

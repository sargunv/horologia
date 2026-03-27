import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/settings")({
  component: SettingsPage,
});

function SettingsPage() {
  return (
    <div className="p-6">
      <h1 className="h3">Settings</h1>
      <p className="text-surface-600-400 mt-1">Account settings coming soon.</p>
    </div>
  );
}

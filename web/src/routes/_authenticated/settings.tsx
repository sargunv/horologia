import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { currentUserQueryOptions } from "../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/settings")({
  component: SettingsPage,
});

function SettingsPage() {
  const { data: user } = useQuery(currentUserQueryOptions);

  return (
    <div className="p-6">
      <h1 className="h3">Settings</h1>
      <div className="mt-6 max-w-lg">
        <div className="card preset-outlined-surface-200-800 p-4">
          <h2 className="text-sm font-medium">Account</h2>
          <div className="text-surface-600-400 mt-2 space-y-1 text-sm">
            <p>{user?.name}</p>
            <p>{user?.email}</p>
          </div>
        </div>
        <p className="text-surface-600-400 mt-4 text-sm">
          API tokens, profile editing, and more coming soon.
        </p>
      </div>
    </div>
  );
}

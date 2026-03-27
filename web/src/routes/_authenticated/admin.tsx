import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/admin")({
  component: AdminPage,
});

function AdminPage() {
  return (
    <div className="p-6">
      <h1 className="h3">Admin</h1>
      <p className="text-surface-600-400 mt-1">User management and system settings coming soon.</p>
    </div>
  );
}

import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/")({
  component: IndexPage,
});

function IndexPage() {
  return (
    <div className="p-8">
      <h1 className="h1">Tend</h1>
      <p className="text-surface-600-400">Task manager — coming soon.</p>
    </div>
  );
}

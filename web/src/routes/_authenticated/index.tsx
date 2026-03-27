import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/")({
  component: HomePage,
});

function HomePage() {
  return (
    <div className="p-6">
      <h1 className="h3">Home</h1>
      <p className="text-surface-600-400 mt-1">
        Your unified task dashboard across all spaces will appear here.
      </p>
    </div>
  );
}

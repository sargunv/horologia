import { createFileRoute } from "@tanstack/react-router";
import { CookingPot, Layers } from "lucide-react";
import { AnchorLink } from "../../lib/links.ts";
import { Card } from "../../ui/Card.tsx";

export const Route = createFileRoute("/_authenticated/library")({
  component: LibraryPage,
});

function LibraryPage() {
  return (
    <div className="mx-auto max-w-3xl p-4 sm:p-6">
      <h1 className="text-xl font-semibold">Library</h1>
      <div className="mt-5 grid gap-3 sm:grid-cols-2">
        <AnchorLink to="/recipes" className="block">
          <Card className="flex min-h-32 flex-col justify-between gap-4 p-4 transition-colors hover:bg-base-200">
            <CookingPot className="size-6 text-primary" aria-hidden="true" />
            <div>
              <h2 className="font-medium">Recipes</h2>
              <p className="text-sm text-base-content/60">Ingredients and instructions</p>
            </div>
          </Card>
        </AnchorLink>
        <AnchorLink to="/spaces" className="block">
          <Card className="flex min-h-32 flex-col justify-between gap-4 p-4 transition-colors hover:bg-base-200">
            <Layers className="size-6 text-primary" aria-hidden="true" />
            <div>
              <h2 className="font-medium">Spaces</h2>
              <p className="text-sm text-base-content/60">Ownership and collaboration boundaries</p>
            </div>
          </Card>
        </AnchorLink>
      </div>
    </div>
  );
}

import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug/recipes/")({
  component: () => null,
});

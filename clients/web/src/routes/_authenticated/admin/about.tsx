import { createFileRoute } from "@tanstack/react-router";
import { authConfigQueryOptions } from "../../../lib/queries.ts";
import { AboutSection } from "../../../components/admin/AboutSection.tsx";

export const Route = createFileRoute("/_authenticated/admin/about")({
  loader: ({ context: { queryClient } }) => queryClient.ensureQueryData(authConfigQueryOptions),
  component: AdminAboutPage,
});

function AdminAboutPage() {
  return <AboutSection />;
}

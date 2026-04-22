import { createFileRoute } from "@tanstack/react-router";
import { usersQueryOptions } from "../../../lib/queries.ts";
import { UsersSection } from "../../../components/admin/UsersSection.tsx";

export const Route = createFileRoute("/_authenticated/admin/users")({
  loader: ({ context: { queryClient } }) => queryClient.ensureQueryData(usersQueryOptions),
  component: AdminUsersPage,
});

function AdminUsersPage() {
  return <UsersSection />;
}

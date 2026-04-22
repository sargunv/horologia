import { useSuspenseQuery } from "@tanstack/react-query";
import { Suspense, useState } from "react";
import { currentUserQueryOptions, usersQueryOptions } from "../../lib/queries.ts";
import { ListDetailLayout } from "../ListDetailLayout.tsx";
import { CreateUserPane } from "./CreateUserPane.tsx";
import { UserDetailPane } from "./UserDetailPane.tsx";
import { UserListPane } from "./UserListPane.tsx";

export function UsersSection() {
  const { data: users } = useSuspenseQuery(usersQueryOptions);
  const { data: currentUser } = useSuspenseQuery(currentUserQueryOptions);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  function renderDetail() {
    if (selectedId === "new") {
      return (
        <CreateUserPane onBack={() => setSelectedId(null)} onCreated={(id) => setSelectedId(id)} />
      );
    }
    if (selectedId) {
      return (
        <Suspense
          fallback={<div className="text-base-content/60 p-6 text-center text-sm">Loading...</div>}
        >
          <UserDetailPane
            key={selectedId}
            userId={selectedId}
            onBack={() => setSelectedId(null)}
            onDeleted={() => setSelectedId(null)}
          />
        </Suspense>
      );
    }
    return null;
  }

  return (
    <ListDetailLayout
      list={
        <UserListPane
          users={users}
          currentUserId={currentUser.id}
          selectedId={selectedId}
          onSelect={(id) => setSelectedId(id === selectedId ? null : id)}
        />
      }
      detail={renderDetail()}
      emptyState={
        <span className="text-base-content/60 text-sm">Select a user to view details</span>
      }
    />
  );
}

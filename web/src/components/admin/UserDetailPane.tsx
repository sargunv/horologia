import { useSuspenseQuery } from "@tanstack/react-query";
import { ArrowLeft } from "lucide-react";
import { currentUserQueryOptions, userQueryOptions } from "../../lib/queries.ts";
import { UserDangerZoneCard } from "./UserDangerZoneCard.tsx";
import { UserPasswordCard } from "./UserPasswordCard.tsx";
import { UserPropertiesCard } from "./UserPropertiesCard.tsx";

export function UserDetailPane({
  userId,
  onBack,
  onDeleted,
}: {
  userId: string;
  onBack: () => void;
  onDeleted: () => void;
}) {
  const { data: user } = useSuspenseQuery(userQueryOptions(userId));
  const { data: currentUser } = useSuspenseQuery(currentUserQueryOptions);
  const isSelf = user.id === currentUser.id;

  return (
    <div className="flex flex-col gap-4">
      <button
        type="button"
        onClick={onBack}
        className="btn btn-square btn-sm btn-soft self-start lg:hidden"
        aria-label="Back to user list"
      >
        <ArrowLeft className="size-4" aria-hidden="true" />
      </button>

      <UserPropertiesCard user={user} isSelf={isSelf} />
      <UserPasswordCard user={user} />
      <UserDangerZoneCard user={user} isSelf={isSelf} onDeleted={onDeleted} />
    </div>
  );
}

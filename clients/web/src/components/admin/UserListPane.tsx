import { Plus } from "lucide-react";
import type { components } from "../../api/schema.d.ts";
import { Card } from "../../ui/Card.tsx";

type User = components["schemas"]["User"];

export function UserListPane({
  users,
  currentUserId,
  selectedId,
  onSelect,
}: {
  users: User[];
  currentUserId: string;
  selectedId: string | null;
  onSelect: (id: string) => void;
}) {
  return (
    <div className="flex flex-col gap-3">
      <Card className="divide-y divide-base-300 overflow-hidden">
        {users.map((user) => (
          <button
            key={user.id}
            type="button"
            onClick={() => onSelect(user.id)}
            aria-pressed={selectedId === user.id}
            className={`flex w-full items-center gap-3 p-3 text-left transition-colors ${
              selectedId === user.id ? "bg-primary text-primary-content" : "hover:bg-base-100"
            }`}
          >
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-1.5">
                <span className="truncate text-sm font-medium">{user.name}</span>
                {user.isOwner && (
                  <span
                    className={`rounded-box px-1.5 py-0.5 text-xs ${
                      selectedId === user.id ? "bg-white/20" : "bg-primary text-primary-content"
                    }`}
                  >
                    <span className="sr-only">Role: </span>Owner
                  </span>
                )}
                {user.id === currentUserId && <span className="text-xs opacity-60">(you)</span>}
              </div>
              <div className="truncate text-xs opacity-70">{user.email}</div>
            </div>
          </button>
        ))}
      </Card>

      <button
        type="button"
        onClick={() => onSelect("new")}
        aria-pressed={selectedId === "new"}
        className={`flex w-full items-center justify-center gap-2 rounded-box border-2 border-dashed p-3 text-sm transition-colors ${
          selectedId === "new"
            ? "border-primary text-primary"
            : "border-base-300 text-base-content/60 hover:border-base-content/40 hover:text-base-content/80"
        }`}
      >
        <Plus className="size-4" aria-hidden="true" />
        Add user
      </button>
    </div>
  );
}

import { useQueryClient } from "@tanstack/react-query";
import { CircleAlert, UserPlus, Users, X } from "lucide-react";
import { type FormEvent, useEffect, useRef, useState } from "react";
import { apiClient } from "../../api/client.ts";
import type { components } from "../../api/schema.d.ts";
import { SettingsSection } from "./SettingsSection.tsx";

type SpaceMember = components["schemas"]["SpaceMember"];
type SpaceRole = components["schemas"]["SpaceRole"];

const ROLES: [SpaceRole, string][] = [
  ["admin", "Admin"],
  ["member", "Member"],
  ["viewer", "Viewer"],
];

const ROLE_LABELS: Record<SpaceRole, string> = {
  admin: "Admin",
  member: "Member",
  viewer: "Viewer",
};

function RoleOptions() {
  return (
    <>
      {ROLES.map(([value, label]) => (
        <option key={value} value={value}>
          {label}
        </option>
      ))}
    </>
  );
}

export function MembersSection({
  spaceSlug,
  members,
  isAdmin,
  currentUserId,
}: {
  spaceSlug: string;
  members: SpaceMember[];
  isAdmin: boolean;
  currentUserId: string;
}) {
  return (
    <SettingsSection
      icon={<Users className="size-5" />}
      title="Members"
      description="Manage who has access to this space."
    >
      <div className="flex flex-col divide-y divide-surface-200-800">
        {members.map((member) => (
          <MemberRow
            key={member.userId}
            spaceSlug={spaceSlug}
            member={member}
            isAdmin={isAdmin}
            isSelf={member.userId === currentUserId}
          />
        ))}
      </div>
      {isAdmin && <AddMemberForm spaceSlug={spaceSlug} />}
    </SettingsSection>
  );
}

function MemberRow({
  spaceSlug,
  member,
  isAdmin,
  isSelf,
}: {
  spaceSlug: string;
  member: SpaceMember;
  isAdmin: boolean;
  isSelf: boolean;
}) {
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);
  const [rolePending, setRolePending] = useState(false);
  const [removePending, setRemovePending] = useState(false);
  const [confirmingRemove, setConfirmingRemove] = useState(false);
  const confirmButtonRef = useRef<HTMLButtonElement>(null);

  const pending = rolePending || removePending;

  useEffect(() => {
    if (confirmingRemove) {
      confirmButtonRef.current?.focus();
    }
  }, [confirmingRemove]);

  async function handleRoleChange(newRole: SpaceRole) {
    if (newRole === member.role) return;
    setError(null);
    setRolePending(true);
    try {
      const { error: apiError } = await apiClient.PATCH("/spaces/{spaceSlug}/members/{userId}", {
        params: { path: { spaceSlug, userId: member.userId } },
        body: { role: newRole },
      });
      if (apiError) {
        setError((apiError as { message?: string }).message ?? "Failed to update role");
        return;
      }
      await queryClient.invalidateQueries({ queryKey: ["spaces", spaceSlug, "members"] });
    } catch {
      setError("Something went wrong. Please try again.");
    } finally {
      setRolePending(false);
    }
  }

  async function handleRemove() {
    setError(null);
    setRemovePending(true);
    try {
      const { error: apiError } = await apiClient.DELETE("/spaces/{spaceSlug}/members/{userId}", {
        params: { path: { spaceSlug, userId: member.userId } },
      });
      if (apiError) {
        setError((apiError as { message?: string }).message ?? "Failed to remove member");
        return;
      }
      await queryClient.invalidateQueries({ queryKey: ["spaces", spaceSlug, "members"] });
    } catch {
      setError("Something went wrong. Please try again.");
    } finally {
      setConfirmingRemove(false);
      setRemovePending(false);
    }
  }

  return (
    <div className="flex flex-col gap-2 py-3 first:pt-0 last:pb-0">
      <div className="flex items-center gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <span className="truncate text-sm font-medium">{member.userName}</span>
            {isSelf && <span className="text-surface-500 text-xs">(you)</span>}
          </div>
          <div className="text-surface-600-400 truncate text-xs">{member.userEmail}</div>
        </div>

        {isAdmin ? (
          <div className="flex items-center gap-2 shrink-0">
            <select
              aria-label={`Role for ${member.userName}`}
              value={member.role}
              onChange={(e) => handleRoleChange(e.target.value as SpaceRole)}
              disabled={pending}
              className="select preset-outlined-surface-200-800 w-28 py-1.5 text-sm"
            >
              <RoleOptions />
            </select>

            {confirmingRemove ? (
              <div className="flex items-center gap-1">
                <button
                  ref={confirmButtonRef}
                  type="button"
                  onClick={handleRemove}
                  disabled={removePending}
                  className="btn btn-sm preset-filled-error-500 text-xs"
                >
                  {removePending ? "Removing..." : "Confirm"}
                </button>
                <button
                  type="button"
                  onClick={() => setConfirmingRemove(false)}
                  disabled={removePending}
                  className="btn-icon btn-icon-sm preset-outlined-surface-200-800"
                  aria-label="Cancel remove"
                >
                  <X className="size-3.5" aria-hidden="true" />
                </button>
              </div>
            ) : (
              <button
                type="button"
                onClick={() => {
                  setError(null);
                  setConfirmingRemove(true);
                }}
                disabled={pending}
                className="btn btn-sm preset-outlined-surface-200-800 text-xs"
              >
                Remove
              </button>
            )}
          </div>
        ) : (
          <span className="text-surface-600-400 text-sm">{ROLE_LABELS[member.role]}</span>
        )}
      </div>

      {error && (
        <div
          role="alert"
          className="preset-filled-error-500 flex items-center gap-2 rounded-base px-3 py-2 text-sm"
        >
          <CircleAlert className="size-4 shrink-0" aria-hidden="true" />
          {error}
        </div>
      )}
    </div>
  );
}

function AddMemberForm({ spaceSlug }: { spaceSlug: string }) {
  const queryClient = useQueryClient();
  const [userId, setUserId] = useState("");
  const [role, setRole] = useState<SpaceRole>("member");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setPending(true);

    try {
      const { error: apiError } = await apiClient.POST("/spaces/{spaceSlug}/members", {
        params: { path: { spaceSlug } },
        body: { userId: userId.trim(), role },
      });
      if (apiError) {
        setError((apiError as { message?: string }).message ?? "Failed to add member");
        return;
      }
      await queryClient.invalidateQueries({ queryKey: ["spaces", spaceSlug, "members"] });
      setUserId("");
      setRole("member");
    } catch {
      setError("Something went wrong. Please try again.");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="border-surface-200-800 flex flex-col gap-3 border-t pt-4">
      <h3 className="text-surface-600-400 text-sm font-medium">Add member</h3>
      <form onSubmit={handleSubmit} className="flex items-end gap-2">
        <label className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="text-surface-600-400 text-sm font-medium">User ID</span>
          <input
            type="text"
            required
            value={userId}
            onChange={(e) => setUserId(e.target.value)}
            className="input preset-outlined-surface-200-800 w-full"
            placeholder="U123"
            disabled={pending}
          />
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-surface-600-400 text-sm font-medium">Role</span>
          <select
            value={role}
            onChange={(e) => setRole(e.target.value as SpaceRole)}
            disabled={pending}
            className="select preset-outlined-surface-200-800 w-28"
          >
            <RoleOptions />
          </select>
        </label>
        <button
          type="submit"
          disabled={pending || !userId.trim()}
          className="btn preset-filled-primary-500"
        >
          {pending ? (
            "Adding..."
          ) : (
            <>
              <UserPlus className="size-4" aria-hidden="true" />
              Add
            </>
          )}
        </button>
      </form>

      {error && (
        <div
          role="alert"
          className="preset-filled-error-500 flex items-center gap-2 rounded-base px-3 py-2 text-sm"
        >
          <CircleAlert className="size-4 shrink-0" aria-hidden="true" />
          {error}
        </div>
      )}
    </div>
  );
}

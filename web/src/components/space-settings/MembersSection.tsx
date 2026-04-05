import { useMutation, useQueryClient } from "@tanstack/react-query";
import { UserPlus, Users, X } from "lucide-react";
import { type FormEvent, useEffect, useRef, useState } from "react";
import { apiClient } from "../../api/client.ts";
import type { components } from "../../api/schema.d.ts";
import { toaster } from "../../lib/toaster.ts";
import { ErrorAlert } from "./ErrorAlert.tsx";
import { SettingsSection } from "./SettingsSection.tsx";

function notifyStaleData() {
  toaster.warning({
    title: "Data may be out of date",
    description: "Refresh the page to see the latest changes.",
  });
}

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

function isSpaceRole(value: string): value is SpaceRole {
  return value in ROLE_LABELS;
}

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
  const [confirmingRemove, setConfirmingRemove] = useState(false);
  const confirmButtonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (confirmingRemove) {
      confirmButtonRef.current?.focus();
    }
  }, [confirmingRemove]);

  const roleMutation = useMutation({
    mutationFn: async (newRole: SpaceRole) => {
      const { error } = await apiClient.PATCH("/spaces/{spaceSlug}/members/{userId}", {
        params: { path: { spaceSlug, userId: member.userId } },
        body: { role: newRole },
      });
      if (error) throw new Error(error.message ?? "Failed to update role");
    },
    onSuccess: async () => {
      try {
        await queryClient.invalidateQueries({ queryKey: ["spaces", spaceSlug, "members"] });
      } catch {
        notifyStaleData();
      }
    },
  });

  const removeMutation = useMutation({
    mutationFn: async () => {
      const { error } = await apiClient.DELETE("/spaces/{spaceSlug}/members/{userId}", {
        params: { path: { spaceSlug, userId: member.userId } },
      });
      if (error) throw new Error(error.message ?? "Failed to remove member");
    },
    onSuccess: async () => {
      try {
        await queryClient.invalidateQueries({ queryKey: ["spaces", spaceSlug, "members"] });
      } catch {
        notifyStaleData();
      }
    },
    onSettled: () => {
      setConfirmingRemove(false);
    },
  });

  const pending = roleMutation.isPending || removeMutation.isPending;
  const error = roleMutation.error ?? removeMutation.error;

  function handleRoleChange(newRole: SpaceRole) {
    if (newRole === member.role) return;
    removeMutation.reset();
    roleMutation.mutate(newRole);
  }

  function handleRemove() {
    roleMutation.reset();
    removeMutation.mutate();
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
              onChange={(e) => {
                if (isSpaceRole(e.target.value)) handleRoleChange(e.target.value);
              }}
              disabled={pending}
              className="select preset-outlined-surface-200-800 w-28"
            >
              <RoleOptions />
            </select>

            {confirmingRemove ? (
              <div className="flex items-center gap-1">
                <button
                  ref={confirmButtonRef}
                  type="button"
                  onClick={handleRemove}
                  disabled={removeMutation.isPending}
                  className="btn btn-sm preset-filled-error-500 text-xs"
                >
                  {removeMutation.isPending ? "Removing..." : "Confirm"}
                </button>
                <button
                  type="button"
                  onClick={() => setConfirmingRemove(false)}
                  disabled={removeMutation.isPending}
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
                  roleMutation.reset();
                  removeMutation.reset();
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

      {error && <ErrorAlert message={error.message} />}
    </div>
  );
}

function AddMemberForm({ spaceSlug }: { spaceSlug: string }) {
  const queryClient = useQueryClient();
  const [userId, setUserId] = useState("");
  const [role, setRole] = useState<SpaceRole>("member");

  const addMutation = useMutation({
    mutationFn: async (body: { userId: string; role: SpaceRole }) => {
      const { error } = await apiClient.POST("/spaces/{spaceSlug}/members", {
        params: { path: { spaceSlug } },
        body,
      });
      if (error) throw new Error(error.message ?? "Failed to add member");
    },
    onSuccess: async () => {
      setUserId("");
      setRole("member");
      try {
        await queryClient.invalidateQueries({ queryKey: ["spaces", spaceSlug, "members"] });
      } catch {
        notifyStaleData();
      }
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    addMutation.mutate({ userId: userId.trim(), role });
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
            disabled={addMutation.isPending}
          />
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-surface-600-400 text-sm font-medium">Role</span>
          <select
            value={role}
            onChange={(e) => {
              if (isSpaceRole(e.target.value)) setRole(e.target.value);
            }}
            disabled={addMutation.isPending}
            className="select preset-outlined-surface-200-800 w-28"
          >
            <RoleOptions />
          </select>
        </label>
        <button
          type="submit"
          disabled={addMutation.isPending || !userId.trim()}
          className="btn preset-filled-primary-500"
        >
          {addMutation.isPending ? (
            "Adding..."
          ) : (
            <>
              <UserPlus className="size-4" aria-hidden="true" />
              Add
            </>
          )}
        </button>
      </form>

      {addMutation.error && <ErrorAlert message={addMutation.error.message} />}
    </div>
  );
}

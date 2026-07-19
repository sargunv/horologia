import { useMutation, useQuery } from "@tanstack/react-query";
import { Command } from "cmdk";
import { Check, UserPlus, Users, X } from "lucide-react";
import { type FormEvent, useEffect, useMemo, useRef, useState } from "react";
import type { components } from "@horologia/client-core/schema";
import { useSettingsCommands } from "../../lib/mutations.ts";
import { usersQueryOptions } from "../../lib/queries.ts";
import { ErrorAlert } from "./ErrorAlert.tsx";
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
      <div className="flex flex-col divide-y divide-base-300">
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
      {isAdmin && <AddMemberForm spaceSlug={spaceSlug} members={members} />}
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
  const commands = useSettingsCommands();
  const [confirmingRemove, setConfirmingRemove] = useState(false);
  const confirmButtonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (confirmingRemove) {
      confirmButtonRef.current?.focus();
    }
  }, [confirmingRemove]);

  const roleMutation = useMutation({
    mutationFn: (newRole: SpaceRole) =>
      commands.updateMember(spaceSlug, member.userId, { role: newRole }),
  });

  const removeMutation = useMutation({
    mutationFn: () => commands.deleteMember(spaceSlug, member.userId),
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
            {isSelf && <span className="text-xs text-base-content/60">(you)</span>}
          </div>
          <div className="truncate text-xs text-base-content/70">{member.userEmail}</div>
        </div>

        {isAdmin ? (
          <div className="flex shrink-0 items-center gap-2">
            <select
              aria-label={`Role for ${member.userName}`}
              value={member.role}
              onChange={(e) => {
                if (isSpaceRole(e.target.value)) handleRoleChange(e.target.value);
              }}
              disabled={pending}
              className="select select-sm w-28"
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
                  className="btn btn-error btn-sm"
                >
                  {removeMutation.isPending ? "Removing..." : "Confirm"}
                </button>
                <button
                  type="button"
                  onClick={() => setConfirmingRemove(false)}
                  disabled={removeMutation.isPending}
                  className="btn btn-soft btn-square btn-sm"
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
                className="btn btn-soft btn-sm"
              >
                Remove
              </button>
            )}
          </div>
        ) : (
          <span className="text-sm text-base-content/70">{ROLE_LABELS[member.role]}</span>
        )}
      </div>

      {error && <ErrorAlert message={error.message} />}
    </div>
  );
}

function AddMemberForm({ spaceSlug, members }: { spaceSlug: string; members: SpaceMember[] }) {
  const commands = useSettingsCommands();
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null);
  const [inputValue, setInputValue] = useState("");
  const [role, setRole] = useState<SpaceRole>("member");

  const {
    data: allUsers,
    isLoading: usersLoading,
    error: usersError,
  } = useQuery(usersQueryOptions);

  const availableUsers = useMemo(() => {
    const existingMemberIds = new Set(members.map((m) => m.userId));
    return (allUsers ?? []).filter((u) => !existingMemberIds.has(u.id));
  }, [allUsers, members]);

  const filteredUsers = useMemo(() => {
    const query = inputValue.toLowerCase();
    if (!query) return availableUsers;
    return availableUsers.filter(
      (u) => u.name.toLowerCase().includes(query) || u.email.toLowerCase().includes(query),
    );
  }, [availableUsers, inputValue]);

  const addMutation = useMutation({
    mutationFn: (body: { userId: string; role: SpaceRole }) =>
      commands.createMember(spaceSlug, body),
    onSuccess: () => {
      setSelectedUserId(null);
      setInputValue("");
      setRole("member");
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!selectedUserId) return;
    addMutation.mutate({ userId: selectedUserId, role });
  }

  return (
    <div className="flex flex-col gap-3 border-t border-base-300 pt-4">
      <h3 className="text-sm font-medium text-base-content/70">Add member</h3>
      <form onSubmit={handleSubmit} className="flex flex-col gap-2">
        <Command
          shouldFilter={false}
          className="min-w-0 overflow-hidden rounded-box border border-base-300"
        >
          <div className="flex flex-wrap items-center gap-2 border-b border-base-300 p-2">
            <Command.Input
              value={inputValue}
              onValueChange={(v) => {
                setInputValue(v);
                setSelectedUserId(null);
              }}
              aria-label="Search users to add"
              placeholder="Search by name or email..."
              className="min-w-40 flex-1 bg-transparent px-1 text-sm outline-none placeholder:text-base-content/50"
            />
            <div className="ml-auto flex shrink-0 items-center gap-2">
              <select
                aria-label="Role"
                value={role}
                onChange={(e) => {
                  if (isSpaceRole(e.target.value)) setRole(e.target.value);
                }}
                disabled={addMutation.isPending}
                className="select select-sm"
              >
                <RoleOptions />
              </select>
              <button
                type="submit"
                disabled={addMutation.isPending || !selectedUserId}
                className="btn btn-primary btn-sm"
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
            </div>
          </div>
          <Command.List className="max-h-60 overflow-y-auto p-1">
            {usersLoading ? (
              <div className="flex items-center gap-2 px-3 py-2 text-sm text-base-content/60">
                <span className="loading loading-spinner loading-xs" />
                Loading users...
              </div>
            ) : usersError ? (
              <div className="px-3 py-2 text-sm text-error" role="alert">
                Failed to load users
              </div>
            ) : filteredUsers.length === 0 ? (
              <Command.Empty className="px-3 py-2 text-sm text-base-content/60">
                {availableUsers.length === 0
                  ? "All users are already members"
                  : "No matching users"}
              </Command.Empty>
            ) : (
              filteredUsers.map((user) => (
                <Command.Item
                  key={user.id}
                  value={user.id}
                  onSelect={() => {
                    setSelectedUserId(user.id);
                    setInputValue(user.name);
                  }}
                  className="flex cursor-default select-none items-center gap-2 rounded-field px-2 py-1.5 text-sm outline-none data-[selected=true]:bg-base-200"
                >
                  <span className="flex-1">{user.name}</span>
                  <span className="text-xs text-base-content/60">{user.email}</span>
                  {selectedUserId === user.id && <Check className="size-3.5" aria-hidden="true" />}
                </Command.Item>
              ))
            )}
          </Command.List>
        </Command>
      </form>
      <div aria-live="polite" role="status" className="sr-only">
        {usersLoading
          ? "Loading users..."
          : usersError
            ? "Failed to load users"
            : filteredUsers.length === 0
              ? availableUsers.length === 0
                ? "All users are already members"
                : "No matching users"
              : ""}
      </div>
      {addMutation.error && <ErrorAlert message={addMutation.error.message} />}
    </div>
  );
}

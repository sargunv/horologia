import { Combobox, Portal, useListCollection } from "@skeletonlabs/skeleton-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Loader2, UserPlus, Users, X } from "lucide-react";
import { type FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { apiClient } from "../../api/client.ts";
import type { components } from "../../api/schema.d.ts";
import { usersQueryOptions } from "../../lib/queries.ts";
import { notifyStaleData } from "../../lib/toaster.ts";
import { ErrorAlert } from "./ErrorAlert.tsx";
import { SettingsSection } from "./SettingsSection.tsx";

type User = components["schemas"]["User"];
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
      } catch (err) {
        console.error("Cache invalidation failed after mutation:", err);
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
      } catch (err) {
        console.error("Cache invalidation failed after mutation:", err);
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

function AddMemberForm({ spaceSlug, members }: { spaceSlug: string; members: SpaceMember[] }) {
  const queryClient = useQueryClient();
  const [value, setValue] = useState<string[]>([]);
  const [inputValue, setInputValue] = useState("");
  const [role, setRole] = useState<SpaceRole>("member");

  const {
    data: allUsers,
    isLoading: usersLoading,
    error: usersError,
  } = useQuery(usersQueryOptions);

  const existingMemberIds = useMemo(() => new Set(members.map((m) => m.userId)), [members]);

  const availableUsers = useMemo(
    () => (allUsers ?? []).filter((u) => !existingMemberIds.has(u.id)),
    [allUsers, existingMemberIds],
  );

  const filteredUsers = useMemo(() => {
    const query = inputValue.toLowerCase();
    if (!query) return availableUsers;
    return availableUsers.filter(
      (u) => u.name.toLowerCase().includes(query) || u.email.toLowerCase().includes(query),
    );
  }, [availableUsers, inputValue]);

  const collection = useListCollection({
    items: filteredUsers,
    itemToValue: (user: User) => user.id,
    itemToString: (user: User) => user.name,
  });

  const addMutation = useMutation({
    mutationFn: async (body: { userId: string; role: SpaceRole }) => {
      const { error } = await apiClient.POST("/spaces/{spaceSlug}/members", {
        params: { path: { spaceSlug } },
        body,
      });
      if (error) throw new Error(error.message ?? "Failed to add member");
    },
    onSuccess: async () => {
      setValue([]);
      setInputValue("");
      setRole("member");
      try {
        await queryClient.invalidateQueries({ queryKey: ["spaces", spaceSlug, "members"] });
      } catch (err) {
        console.error("Cache invalidation failed after mutation:", err);
        notifyStaleData();
      }
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const userId = value[0];
    if (!userId) return;
    addMutation.mutate({ userId, role });
  }

  return (
    <div className="border-surface-200-800 flex flex-col gap-3 border-t pt-4">
      <h3 className="text-surface-600-400 text-sm font-medium">Add member</h3>
      <form onSubmit={handleSubmit} className="flex items-end gap-2">
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <Combobox
            collection={collection}
            value={value}
            onValueChange={({ value: v }) => setValue(v)}
            inputValue={inputValue}
            onInputValueChange={({ inputValue: v }) => setInputValue(v)}
            disabled={addMutation.isPending}
            openOnClick
            closeOnSelect
            placeholder="Search by name or email..."
          >
            <Combobox.Label className="text-surface-600-400 text-sm font-medium">
              User
            </Combobox.Label>
            <Combobox.Control className="input preset-outlined-surface-200-800 w-full">
              <Combobox.Input />
            </Combobox.Control>
            <Portal>
              <Combobox.Positioner>
                <Combobox.Content className="max-h-60 overflow-y-auto">
                  {usersLoading ? (
                    <div className="text-surface-500 flex items-center gap-2 px-3 py-2 text-sm">
                      <Loader2 className="size-4 animate-spin" />
                      Loading users...
                    </div>
                  ) : usersError ? (
                    <div className="text-error-500 px-3 py-2 text-sm">Failed to load users</div>
                  ) : filteredUsers.length === 0 ? (
                    <div className="text-surface-500 px-3 py-2 text-sm">
                      {availableUsers.length === 0
                        ? "All users are already members"
                        : "No matching users"}
                    </div>
                  ) : (
                    filteredUsers.map((user) => (
                      <Combobox.Item key={user.id} item={user}>
                        <Combobox.ItemText>{user.name}</Combobox.ItemText>
                        <span className="text-surface-500 ml-auto text-xs">{user.email}</span>
                        <Combobox.ItemIndicator>✓</Combobox.ItemIndicator>
                      </Combobox.Item>
                    ))
                  )}
                </Combobox.Content>
              </Combobox.Positioner>
            </Portal>
          </Combobox>
        </div>
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
          disabled={addMutation.isPending || value.length === 0}
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

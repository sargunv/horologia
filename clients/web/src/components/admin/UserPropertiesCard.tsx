import { useNavigate } from "@tanstack/react-router";
import { Calendar, Hash, Mail, Shield, User as UserIcon } from "lucide-react";
import type { components } from "../../api/schema.d.ts";
import { useUserPatch } from "../../lib/mutations.ts";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";
import { SettingsSection } from "../space-settings/SettingsSection.tsx";
import { EditableField, PropertyRow } from "../EditableField.tsx";

type User = components["schemas"]["User"];

function OwnerToggle({ user, isSelf }: { user: User; isSelf: boolean }) {
  const mutation = useUserPatch(user.id);
  const navigate = useNavigate();

  function handleChange(checked: boolean) {
    mutation.reset();
    mutation.mutate(
      { isOwner: checked },
      {
        onSuccess: async () => {
          if (isSelf && !checked) {
            await navigate({ to: "/" });
          }
        },
      },
    );
  }

  return (
    <div className="flex flex-col gap-1">
      <label className="flex items-center gap-2">
        <input
          type="checkbox"
          checked={user.isOwner}
          onChange={(e) => handleChange(e.target.checked)}
          className="checkbox"
          disabled={mutation.isPending}
        />
        <span className="text-sm">{user.isOwner ? "Yes" : "No"}</span>
      </label>
      {isSelf && !user.isOwner && (
        <span className="text-warning text-xs">You will lose admin access</span>
      )}
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </div>
  );
}

export function UserPropertiesCard({ user, isSelf }: { user: User; isSelf: boolean }) {
  return (
    <SettingsSection
      icon={<UserIcon className="size-5" aria-hidden="true" />}
      title="Properties"
      description="User account details."
    >
      <PropertyRow label="Name" icon={<UserIcon className="size-4" aria-hidden="true" />}>
        <EditableField userId={user.id} field="name" value={user.name} label="Name" />
      </PropertyRow>
      <PropertyRow label="Email" icon={<Mail className="size-4" aria-hidden="true" />}>
        <EditableField
          userId={user.id}
          field="email"
          value={user.email}
          label="Email"
          type="email"
        />
      </PropertyRow>
      <PropertyRow label="Owner" icon={<Shield className="size-4" aria-hidden="true" />}>
        <OwnerToggle user={user} isSelf={isSelf} />
      </PropertyRow>
      <PropertyRow label="ID" icon={<Hash className="size-4" aria-hidden="true" />}>
        <span className="font-mono text-sm">{user.id}</span>
      </PropertyRow>
      <PropertyRow label="Created" icon={<Calendar className="size-4" aria-hidden="true" />}>
        <span className="text-sm">
          {new Date(user.createdAt).toLocaleDateString(undefined, {
            year: "numeric",
            month: "short",
            day: "numeric",
          })}
        </span>
      </PropertyRow>
    </SettingsSection>
  );
}

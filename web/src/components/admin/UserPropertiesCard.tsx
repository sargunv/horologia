import { useNavigate } from "@tanstack/react-router";
import { Calendar, Hash, Mail, Shield, User as UserIcon } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import type { components } from "../../api/schema.d.ts";
import { useUserPatch } from "../../lib/mutations.ts";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";
import { SettingsSection } from "../space-settings/SettingsSection.tsx";

type User = components["schemas"]["User"];

function PropertyRow({
  label,
  icon,
  children,
}: {
  label: string;
  icon?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-start gap-4 border-b border-surface-200-800 py-3 last:border-b-0">
      <span className="text-surface-600-400 flex w-28 shrink-0 items-center gap-2 pt-1 text-sm">
        {icon}
        {label}
      </span>
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  );
}

function EditableField({
  userId,
  field,
  value,
  label,
  type = "text",
  maxLength = 200,
}: {
  userId: string;
  field: "name" | "email";
  value: string;
  label: string;
  type?: string;
  maxLength?: number;
}) {
  const mutation = useUserPatch(userId);
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(value);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!editing) setDraft(value);
  }, [value, editing]);

  useEffect(() => {
    if (editing) inputRef.current?.focus();
  }, [editing]);

  function save() {
    if (mutation.isPending) return;
    setEditing(false);
    const trimmed = draft.trim();
    if (trimmed && trimmed !== value) {
      mutation.reset();
      mutation.mutate({ [field]: trimmed });
    } else {
      setDraft(value);
    }
  }

  function enterEditing() {
    mutation.reset();
    setEditing(true);
  }

  if (editing) {
    return (
      <div>
        <input
          ref={inputRef}
          type={type}
          aria-label={label}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={save}
          onKeyDown={(e) => {
            if (e.key === "Enter") save();
            if (e.key === "Escape") {
              setDraft(value);
              setEditing(false);
            }
          }}
          className="w-full border-b-2 border-primary-500 bg-transparent text-sm outline-none"
          maxLength={maxLength}
          disabled={mutation.isPending}
        />
        {mutation.error && <ErrorAlert message={mutation.error.message} />}
      </div>
    );
  }

  return (
    <div>
      <span
        className="cursor-pointer rounded px-1 -mx-1 text-sm hover:bg-surface-100-900 transition-colors"
        onClick={enterEditing}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            enterEditing();
          }
        }}
        role="button"
        tabIndex={0}
        aria-label={`Edit ${label}`}
      >
        {value}
      </span>
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </div>
  );
}

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
        <span className="text-warning-500 text-xs">You will lose admin access</span>
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

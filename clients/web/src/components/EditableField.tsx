import { useEffect, useRef, useState } from "react";
import { useUserPatch } from "../lib/mutations.ts";
import { ErrorAlert } from "./space-settings/ErrorAlert.tsx";

export function EditableField({
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

  function enterEditing() {
    mutation.reset();
    setEditing(true);
    // Select text on enter so the user can start typing immediately.
    requestAnimationFrame(() => inputRef.current?.select());
  }

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

  return (
    <div>
      <input
        ref={inputRef}
        type={type}
        aria-label={editing ? label : `Edit ${label}`}
        value={editing ? draft : value}
        readOnly={!editing}
        onFocus={enterEditing}
        onBlur={save}
        onChange={(e) => setDraft(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") inputRef.current?.blur();
          if (e.key === "Escape") {
            setDraft(value);
            setEditing(false);
            inputRef.current?.blur();
          }
        }}
        className={
          editing
            ? "input w-full"
            : "w-full cursor-pointer rounded bg-transparent px-1 text-sm outline-none transition-colors hover:bg-base-100"
        }
        maxLength={maxLength}
        disabled={mutation.isPending}
      />
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </div>
  );
}

export function PropertyRow({
  label,
  icon,
  children,
}: {
  label: string;
  icon?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-start gap-4 border-b border-base-300 py-3 last:border-b-0">
      <span className="text-base-content/70 flex w-28 shrink-0 items-center gap-2 pt-1 text-sm">
        {icon}
        {label}
      </span>
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  );
}

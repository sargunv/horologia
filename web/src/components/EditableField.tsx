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
          className="input preset-outlined-surface-200-800 w-full"
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
    <div className="flex items-start gap-4 border-b border-surface-200-800 py-3 last:border-b-0">
      <span className="text-surface-600-400 flex w-28 shrink-0 items-center gap-2 pt-1 text-sm">
        {icon}
        {label}
      </span>
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  );
}

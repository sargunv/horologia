import { useMutation } from "@tanstack/react-query";
import { Check, Pencil, Plus, Tags, Trash2, X } from "lucide-react";
import { useState } from "react";
import type { components } from "@horologia/client-core/schema";
import { useSettingsCommands } from "../../lib/mutations.ts";
import {
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogRoot,
} from "../../ui/AlertDialog.tsx";
import { ErrorAlert } from "./ErrorAlert.tsx";
import { SettingsSection } from "./SettingsSection.tsx";

type Tag = components["schemas"]["Tag"];

export function TagsSection({ spaceSlug, tags }: { spaceSlug: string; tags: Tag[] }) {
  const commands = useSettingsCommands();

  const [adding, setAdding] = useState(false);
  const [newTagName, setNewTagName] = useState("");

  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);

  const createMutation = useMutation({
    mutationFn: (name: string) => commands.createTag(spaceSlug, name),
    onSuccess: () => {
      setAdding(false);
      setNewTagName("");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (tagName: string) => commands.deleteTag(spaceSlug, tagName),
    onSuccess: () => {
      setDeleteTarget(null);
    },
  });

  function handleAddStart() {
    setNewTagName("");
    createMutation.reset();
    setAdding(true);
  }

  function handleAddConfirm() {
    const trimmed = newTagName.trim();
    if (!trimmed) return;
    createMutation.mutate(trimmed);
  }

  function handleAddCancel() {
    setAdding(false);
    setNewTagName("");
    createMutation.reset();
  }

  function handleAddKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter") {
      e.preventDefault();
      handleAddConfirm();
    } else if (e.key === "Escape") {
      if (!createMutation.isPending) handleAddCancel();
    }
  }

  function handleDeleteOpenChange(open: boolean) {
    if (deleteMutation.isPending) return;
    if (!open) {
      setDeleteTarget(null);
      deleteMutation.reset();
    }
  }

  return (
    <SettingsSection
      icon={<Tags className="size-5" />}
      title="Tags"
      description="Manage tags shared across this space."
    >
      <div className="flex flex-col gap-2">
        {tags.map((tag) => (
          <TagRow
            key={tag.name}
            spaceSlug={spaceSlug}
            tag={tag}
            onDeleteRequest={setDeleteTarget}
          />
        ))}

        {(adding || createMutation.isPending) && (
          <div className="flex flex-col gap-1">
            <div className="flex items-center gap-2">
              <input
                type="text"
                value={newTagName}
                onChange={(e) => {
                  if (!createMutation.isPending) createMutation.reset();
                  setNewTagName(e.target.value);
                }}
                onKeyDown={handleAddKeyDown}
                className="input flex-1"
                placeholder="Tag name"
                maxLength={100}
                disabled={createMutation.isPending}
                autoFocus
                aria-label="New tag name"
              />
              <button
                type="button"
                onClick={handleAddConfirm}
                disabled={createMutation.isPending || !newTagName.trim()}
                className="btn btn-primary btn-square btn-sm shrink-0"
                aria-label="Save"
              >
                <Check className="size-3.5" aria-hidden="true" />
              </button>
              <button
                type="button"
                onClick={handleAddCancel}
                disabled={createMutation.isPending}
                className="btn btn-soft btn-square btn-sm shrink-0"
                aria-label="Cancel"
              >
                <X className="size-3.5" aria-hidden="true" />
              </button>
            </div>
            {createMutation.error && <ErrorAlert message={createMutation.error.message} />}
          </div>
        )}
      </div>

      {!adding && !createMutation.isPending && (
        <button type="button" onClick={handleAddStart} className="btn btn-soft btn-sm self-start">
          <Plus className="size-3.5" aria-hidden="true" />
          Add tag
        </button>
      )}

      {/* Delete confirmation dialog */}
      <AlertDialogRoot open={deleteTarget !== null} onOpenChange={handleDeleteOpenChange}>
        <AlertDialogContent className="max-w-md space-y-4">
          <AlertDialogHeader title="Delete tag" />
          <AlertDialogDescription>
            This will remove the tag <strong className="text-base-content">{deleteTarget}</strong>{" "}
            from everything that uses it. This action cannot be undone.
          </AlertDialogDescription>
          {deleteMutation.error && <ErrorAlert message={deleteMutation.error.message} />}
          <AlertDialogFooter>
            <AlertDialogCancel className="btn btn-soft">Cancel</AlertDialogCancel>
            <AlertDialogAction asChild>
              <button
                type="button"
                disabled={deleteMutation.isPending}
                onClick={(e) => {
                  e.preventDefault();
                  if (deleteTarget) deleteMutation.mutate(deleteTarget);
                }}
                className="btn btn-error"
              >
                {deleteMutation.isPending ? "Deleting..." : "Delete tag"}
              </button>
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialogRoot>
    </SettingsSection>
  );
}

function TagRow({
  spaceSlug,
  tag,
  onDeleteRequest,
}: {
  spaceSlug: string;
  tag: Tag;
  onDeleteRequest: (tagName: string) => void;
}) {
  const commands = useSettingsCommands();
  const [editing, setEditing] = useState(false);
  const [editName, setEditName] = useState(tag.name);

  const renameMutation = useMutation({
    mutationFn: ({ oldName, newName }: { oldName: string; newName: string }) =>
      commands.updateTag(spaceSlug, oldName, newName),
    onSuccess: () => {
      setEditing(false);
    },
  });

  function handleStartEdit() {
    setEditName(tag.name);
    renameMutation.reset();
    setEditing(true);
  }

  function handleCancelEdit() {
    setEditing(false);
    setEditName(tag.name);
    renameMutation.reset();
  }

  function handleConfirmEdit() {
    const trimmed = editName.trim();
    if (!trimmed || trimmed === tag.name) {
      handleCancelEdit();
      return;
    }
    renameMutation.mutate({ oldName: tag.name, newName: trimmed });
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter") {
      e.preventDefault();
      handleConfirmEdit();
    } else if (e.key === "Escape") {
      handleCancelEdit();
    }
  }

  const pending = renameMutation.isPending;

  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center gap-2">
        {editing || pending ? (
          <>
            <input
              type="text"
              value={editName}
              onChange={(e) => {
                if (!renameMutation.isPending) renameMutation.reset();
                setEditName(e.target.value);
              }}
              onKeyDown={handleKeyDown}
              className="input flex-1"
              maxLength={100}
              disabled={pending}
              autoFocus
              aria-label={`Rename tag: ${tag.name}`}
            />
            <button
              type="button"
              onClick={handleConfirmEdit}
              disabled={pending || !editName.trim()}
              className="btn btn-primary btn-square btn-sm shrink-0"
              aria-label="Save"
            >
              <Check className="size-3.5" aria-hidden="true" />
            </button>
            <button
              type="button"
              onClick={handleCancelEdit}
              disabled={pending}
              className="btn btn-soft btn-square btn-sm shrink-0"
              aria-label="Cancel"
            >
              <X className="size-3.5" aria-hidden="true" />
            </button>
          </>
        ) : (
          <>
            <button
              type="button"
              onClick={handleStartEdit}
              className="flex flex-1 items-center gap-2 truncate rounded-box px-3 py-2 text-left text-sm hover:bg-base-200"
              aria-label={`Edit ${tag.name}`}
            >
              <span className="flex-1 truncate">{tag.name}</span>
              <Pencil className="size-3.5 shrink-0 text-base-content/60" aria-hidden="true" />
            </button>
            <button
              type="button"
              onClick={() => onDeleteRequest(tag.name)}
              className="btn btn-soft btn-square btn-sm shrink-0"
              aria-label={`Delete ${tag.name}`}
            >
              <Trash2 className="size-3.5" aria-hidden="true" />
            </button>
          </>
        )}
      </div>
      {renameMutation.error && <ErrorAlert message={renameMutation.error.message} />}
    </div>
  );
}

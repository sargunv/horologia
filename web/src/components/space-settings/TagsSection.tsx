import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Dialog, Portal } from "@skeletonlabs/skeleton-react";
import { Check, Pencil, Plus, Tags, Trash2, X, XIcon } from "lucide-react";
import { useRef, useState } from "react";
import { apiClient } from "../../api/client.ts";
import type { components } from "../../api/schema.d.ts";
import { spaceTagsQueryOptions } from "../../lib/queries.ts";
import { ErrorAlert } from "./ErrorAlert.tsx";
import { SettingsSection } from "./SettingsSection.tsx";

type Tag = components["schemas"]["Tag"];

export function TagsSection({ spaceSlug, tags }: { spaceSlug: string; tags: Tag[] }) {
  const queryClient = useQueryClient();
  const queryKey = spaceTagsQueryOptions(spaceSlug).queryKey;

  const [adding, setAdding] = useState(false);
  const [newTagName, setNewTagName] = useState("");

  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const cancelRef = useRef<HTMLButtonElement>(null);

  const createMutation = useMutation({
    mutationFn: async (name: string) => {
      const { data, error } = await apiClient.POST("/spaces/{spaceSlug}/tags", {
        params: { path: { spaceSlug } },
        body: { name },
      });
      if (error) throw new Error(error.message ?? "Failed to create tag");
      return data;
    },
    onSuccess: async () => {
      setAdding(false);
      setNewTagName("");
      await queryClient.invalidateQueries({ queryKey });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (tagName: string) => {
      const { error } = await apiClient.DELETE("/spaces/{spaceSlug}/tags/{tagName}", {
        params: { path: { spaceSlug, tagName } },
      });
      if (error) throw new Error(error.message ?? "Failed to delete tag");
    },
    onSuccess: async () => {
      setDeleteTarget(null);
      await queryClient.invalidateQueries({ queryKey });
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

  function handleDeleteOpenChange(details: { open: boolean }) {
    if (deleteMutation.isPending) return;
    if (!details.open) {
      setDeleteTarget(null);
      deleteMutation.reset();
    }
  }

  const animation =
    "transition transition-discrete opacity-0 translate-y-[100px] starting:data-[state=open]:opacity-0 starting:data-[state=open]:translate-y-[100px] data-[state=open]:opacity-100 data-[state=open]:translate-y-0";

  return (
    <SettingsSection
      icon={<Tags className="size-5" />}
      title="Tags"
      description="Manage tags for organizing tasks."
    >
      <div className="flex flex-col gap-2">
        {tags.map((tag) => (
          <TagRow
            key={tag.name}
            spaceSlug={spaceSlug}
            tag={tag}
            queryKey={queryKey}
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
                className="input preset-outlined-surface-200-800 flex-1"
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
                className="btn-icon btn-icon-sm preset-filled-primary-500 shrink-0"
                aria-label="Save"
              >
                <Check className="size-3.5" aria-hidden="true" />
              </button>
              <button
                type="button"
                onClick={handleAddCancel}
                disabled={createMutation.isPending}
                className="btn-icon btn-icon-sm preset-outlined-surface-200-800 shrink-0"
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
        <button
          type="button"
          onClick={handleAddStart}
          className="btn btn-sm preset-outlined-surface-200-800 self-start text-xs"
        >
          <Plus className="size-3.5" aria-hidden="true" />
          Add tag
        </button>
      )}

      {/* Delete confirmation dialog */}
      <Dialog
        role="alertdialog"
        open={deleteTarget !== null}
        onOpenChange={handleDeleteOpenChange}
        initialFocusEl={() => cancelRef.current}
      >
        <Portal>
          <Dialog.Backdrop className="fixed inset-0 z-50 bg-surface-50-950/50" />
          <Dialog.Positioner className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <Dialog.Content
              className={`card bg-surface-100-900 w-full max-w-md space-y-4 p-6 shadow-xl ${animation}`}
            >
              <header className="flex items-start justify-between gap-2">
                <Dialog.Title className="text-lg font-bold">Delete tag</Dialog.Title>
                <Dialog.CloseTrigger
                  className="btn-icon hover:preset-tonal"
                  aria-label="Close dialog"
                >
                  <XIcon className="size-4" aria-hidden="true" />
                </Dialog.CloseTrigger>
              </header>
              <Dialog.Description className="text-surface-600-400 text-sm">
                This will remove the tag{" "}
                <strong className="text-surface-950-50">{deleteTarget}</strong> from all tasks that
                use it. This action cannot be undone.
              </Dialog.Description>
              {deleteMutation.error && <ErrorAlert message={deleteMutation.error.message} />}
              <footer className="flex justify-end gap-2">
                <Dialog.CloseTrigger
                  ref={cancelRef}
                  className="btn preset-outlined-surface-200-800"
                >
                  Cancel
                </Dialog.CloseTrigger>
                <button
                  type="button"
                  disabled={deleteMutation.isPending}
                  onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget)}
                  className="btn preset-filled-error-500"
                >
                  {deleteMutation.isPending ? "Deleting..." : "Delete tag"}
                </button>
              </footer>
            </Dialog.Content>
          </Dialog.Positioner>
        </Portal>
      </Dialog>
    </SettingsSection>
  );
}

function TagRow({
  spaceSlug,
  tag,
  queryKey,
  onDeleteRequest,
}: {
  spaceSlug: string;
  tag: Tag;
  queryKey: readonly string[];
  onDeleteRequest: (tagName: string) => void;
}) {
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState(false);
  const [editName, setEditName] = useState(tag.name);

  const renameMutation = useMutation({
    mutationFn: async ({ oldName, newName }: { oldName: string; newName: string }) => {
      const { data, error } = await apiClient.PATCH("/spaces/{spaceSlug}/tags/{tagName}", {
        params: { path: { spaceSlug, tagName: oldName } },
        body: { name: newName },
      });
      if (error) throw new Error(error.message ?? "Failed to rename tag");
      return data;
    },
    onSuccess: async () => {
      setEditing(false);
      await queryClient.invalidateQueries({ queryKey });
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
              className="input preset-outlined-surface-200-800 flex-1"
              maxLength={100}
              disabled={pending}
              autoFocus
              aria-label={`Rename tag: ${tag.name}`}
            />
            <button
              type="button"
              onClick={handleConfirmEdit}
              disabled={pending || !editName.trim()}
              className="btn-icon btn-icon-sm preset-filled-primary-500 shrink-0"
              aria-label="Save"
            >
              <Check className="size-3.5" aria-hidden="true" />
            </button>
            <button
              type="button"
              onClick={handleCancelEdit}
              disabled={pending}
              className="btn-icon btn-icon-sm preset-outlined-surface-200-800 shrink-0"
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
              className="flex flex-1 items-center gap-2 truncate rounded-base px-3 py-2 text-left text-sm hover:bg-surface-200-800"
              aria-label={`Edit ${tag.name}`}
            >
              <span className="flex-1 truncate">{tag.name}</span>
              <Pencil className="text-surface-600-400 size-3.5 shrink-0" aria-hidden="true" />
            </button>
            <button
              type="button"
              onClick={() => onDeleteRequest(tag.name)}
              className="btn-icon btn-icon-sm preset-outlined-surface-200-800 shrink-0"
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

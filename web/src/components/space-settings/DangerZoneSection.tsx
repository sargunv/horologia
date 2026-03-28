import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { Dialog, Portal } from "@skeletonlabs/skeleton-react";
import { Trash2, XIcon } from "lucide-react";
import { useRef, useState } from "react";
import { apiClient } from "../../api/client.ts";
import type { components } from "../../api/schema.d.ts";
import { ErrorAlert } from "./ErrorAlert.tsx";
import { SettingsSection } from "./SettingsSection.tsx";

type Space = components["schemas"]["Space"];

export function DangerZoneSection({ space }: { space: Pick<Space, "slug" | "name"> }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [open, setOpen] = useState(false);
  const [confirmation, setConfirmation] = useState("");
  const cancelRef = useRef<HTMLButtonElement>(null);

  const confirmed = confirmation.trim() === space.slug;

  const deleteMutation = useMutation({
    mutationFn: async () => {
      const { error } = await apiClient.DELETE("/spaces/{spaceSlug}", {
        params: { path: { spaceSlug: space.slug } },
      });
      if (error)
        throw new Error((error as { message?: string }).message ?? "Failed to delete space");
    },
    onSuccess: async () => {
      try {
        queryClient.removeQueries({ queryKey: ["spaces", space.slug] });
        await queryClient.invalidateQueries({ queryKey: ["spaces"] });
      } catch (err) {
        console.error("Failed to refresh after space deletion:", err);
      }
      void navigate({ to: "/spaces" });
    },
  });

  function handleOpenChange(details: { open: boolean }) {
    setOpen(details.open);
    if (!details.open) {
      setConfirmation("");
      deleteMutation.reset();
    }
  }

  const animation =
    "transition transition-discrete opacity-0 translate-y-[100px] starting:data-[state=open]:opacity-0 starting:data-[state=open]:translate-y-[100px] data-[state=open]:opacity-100 data-[state=open]:translate-y-0";

  return (
    <SettingsSection
      icon={<Trash2 className="size-5" />}
      title="Danger Zone"
      description="Irreversible actions for this space."
    >
      <div className="flex items-center justify-between gap-4">
        <div>
          <p className="text-sm font-medium">Delete this space</p>
          <p className="text-surface-600-400 text-sm">
            Permanently delete this space and all of its data. This cannot be undone.
          </p>
        </div>
        <Dialog
          role="alertdialog"
          open={open}
          onOpenChange={handleOpenChange}
          initialFocusEl={() => cancelRef.current}
        >
          <Dialog.Trigger className="btn preset-filled-error-500 shrink-0">
            Delete space
          </Dialog.Trigger>
          <Portal>
            <Dialog.Backdrop className="fixed inset-0 z-50 bg-surface-50-950/50" />
            <Dialog.Positioner className="fixed inset-0 z-50 flex items-center justify-center p-4">
              <Dialog.Content
                className={`card bg-surface-100-900 w-full max-w-md space-y-4 p-6 shadow-xl ${animation}`}
              >
                <header className="flex items-start justify-between gap-2">
                  <Dialog.Title className="text-lg font-bold">Delete space</Dialog.Title>
                  <Dialog.CloseTrigger
                    className="btn-icon hover:preset-tonal"
                    aria-label="Close dialog"
                  >
                    <XIcon className="size-4" aria-hidden="true" />
                  </Dialog.CloseTrigger>
                </header>
                <Dialog.Description className="text-surface-600-400 text-sm">
                  This will permanently delete{" "}
                  <strong className="text-surface-950-50">{space.name}</strong> and all of its
                  tasks, statuses, members, and activity. This action cannot be undone.
                </Dialog.Description>
                <label className="flex flex-col gap-1">
                  <span className="text-sm">
                    Type <strong className="font-mono">{space.slug}</strong> to confirm
                  </span>
                  <input
                    type="text"
                    value={confirmation}
                    onChange={(e) => setConfirmation(e.target.value)}
                    className="input preset-outlined-surface-200-800 w-full"
                    placeholder={space.slug}
                    disabled={deleteMutation.isPending}
                    autoComplete="off"
                  />
                </label>
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
                    disabled={!confirmed || deleteMutation.isPending}
                    onClick={() => deleteMutation.mutate()}
                    className="btn preset-filled-error-500"
                  >
                    {deleteMutation.isPending ? "Deleting..." : "Delete space"}
                  </button>
                </footer>
              </Dialog.Content>
            </Dialog.Positioner>
          </Portal>
        </Dialog>
      </div>
    </SettingsSection>
  );
}

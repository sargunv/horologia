import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { Dialog, Portal } from "@skeletonlabs/skeleton-react";
import { Trash2, XIcon } from "lucide-react";
import { useRef, useState } from "react";
import { apiClient } from "../../api/client.ts";
import type { components } from "../../api/schema.d.ts";
import { DIALOG_ANIMATION } from "../../lib/dialog.ts";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";
import { SettingsSection } from "../space-settings/SettingsSection.tsx";

type User = components["schemas"]["User"];

export function UserDangerZoneCard({
  user,
  isSelf,
  onDeleted,
}: {
  user: User;
  isSelf: boolean;
  onDeleted: () => void;
}) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const cancelRef = useRef<HTMLButtonElement>(null);

  const deleteMutation = useMutation({
    mutationFn: async () => {
      const { error } = await apiClient.DELETE("/users/{userId}", {
        params: { path: { userId: user.id } },
      });
      if (error) throw new Error(error.message ?? "Failed to delete user");
    },
    onSuccess: async () => {
      if (isSelf) {
        queryClient.clear();
        await navigate({ to: "/login" });
      } else {
        await queryClient.invalidateQueries({ queryKey: ["users"] });
        setOpen(false);
        onDeleted();
      }
    },
  });

  function handleOpenChange(details: { open: boolean }) {
    setOpen(details.open);
    if (!details.open) deleteMutation.reset();
  }

  return (
    <SettingsSection
      icon={<Trash2 className="size-5" aria-hidden="true" />}
      title="Danger zone"
      description="Irreversible actions for this user."
    >
      <div className="flex items-center justify-between gap-4">
        <p className="text-surface-600-400 text-sm">
          Permanently delete this user and all of their data.
        </p>
        <Dialog
          role="alertdialog"
          open={open}
          onOpenChange={handleOpenChange}
          initialFocusEl={() => cancelRef.current}
        >
          <Dialog.Trigger className="btn btn-sm preset-filled-error-500 shrink-0">
            Delete user
          </Dialog.Trigger>
          <Portal>
            <Dialog.Backdrop className="fixed inset-0 z-50 bg-surface-50-950/50" />
            <Dialog.Positioner className="fixed inset-0 z-50 flex items-center justify-center p-4">
              <Dialog.Content
                className={`card bg-surface-100-900 w-full max-w-md space-y-4 p-6 shadow-xl ${DIALOG_ANIMATION}`}
              >
                <header className="flex items-start justify-between gap-2">
                  <Dialog.Title className="text-lg font-bold">Delete user</Dialog.Title>
                  <Dialog.CloseTrigger
                    className="btn-icon hover:preset-tonal"
                    aria-label="Close dialog"
                  >
                    <XIcon className="size-4" aria-hidden="true" />
                  </Dialog.CloseTrigger>
                </header>
                <Dialog.Description className="text-surface-600-400 text-sm">
                  {isSelf ? (
                    <>
                      You are about to delete{" "}
                      <strong className="text-surface-950-50">your own account</strong>. You will be
                      signed out immediately and will not be able to log in again.
                    </>
                  ) : (
                    <>
                      Permanently delete{" "}
                      <strong className="text-surface-950-50">{user.name}</strong> ({user.email}).
                      Their tasks, memberships, and tokens will be removed. This cannot be undone.
                    </>
                  )}
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
                    onClick={() => deleteMutation.mutate()}
                    className="btn preset-filled-error-500"
                  >
                    {deleteMutation.isPending ? "Deleting..." : "Delete user"}
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

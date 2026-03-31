import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Dialog, Portal } from "@skeletonlabs/skeleton-react";
import { Key, XIcon } from "lucide-react";
import { type FormEvent, useRef, useState } from "react";
import { apiClient } from "../../api/client.ts";
import type { components } from "../../api/schema.d.ts";
import { DIALOG_ANIMATION } from "../../lib/dialog.ts";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";
import { SettingsSection } from "../space-settings/SettingsSection.tsx";

type User = components["schemas"]["User"];

function usePasswordMutation(userId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: components["schemas"]["UserUpdate"]) => {
      const { error } = await apiClient.PATCH("/users/{userId}", {
        params: { path: { userId } },
        body,
      });
      if (error) throw new Error(error.message ?? "Failed to update password");
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["users", userId] }),
        queryClient.invalidateQueries({ queryKey: ["users"] }),
      ]);
    },
  });
}

function SetPasswordDialog({
  userId,
  open,
  onOpenChange,
  label,
}: {
  userId: string;
  open: boolean;
  onOpenChange: (details: { open: boolean }) => void;
  label: string;
}) {
  const mutation = usePasswordMutation(userId);
  const cancelRef = useRef<HTMLButtonElement>(null);
  const [password, setPassword] = useState("");

  function handleOpenChange(details: { open: boolean }) {
    onOpenChange(details);
    if (!details.open) {
      setPassword("");
      mutation.reset();
    }
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    mutation.mutate({ setPassword: password }, { onSuccess: () => onOpenChange({ open: false }) });
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange} initialFocusEl={() => cancelRef.current}>
      <Portal>
        <Dialog.Backdrop className="fixed inset-0 z-50 bg-surface-50-950/50" />
        <Dialog.Positioner className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <Dialog.Content
            className={`card bg-surface-100-900 w-full max-w-sm space-y-4 p-6 shadow-xl ${DIALOG_ANIMATION}`}
          >
            <header className="flex items-start justify-between gap-2">
              <Dialog.Title className="text-lg font-bold">{label}</Dialog.Title>
              <Dialog.CloseTrigger
                className="btn-icon hover:preset-tonal"
                aria-label="Close dialog"
              >
                <XIcon className="size-4" aria-hidden="true" />
              </Dialog.CloseTrigger>
            </header>
            <form onSubmit={handleSubmit} className="flex flex-col gap-3">
              <label className="flex flex-col gap-1">
                <span className="text-surface-600-400 text-sm font-medium">New password</span>
                <input
                  type="password"
                  required
                  minLength={8}
                  maxLength={72}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="input preset-outlined-surface-200-800 w-full"
                  disabled={mutation.isPending}
                />
              </label>
              {mutation.error && <ErrorAlert message={mutation.error.message} />}
              <footer className="flex justify-end gap-2">
                <Dialog.CloseTrigger
                  ref={cancelRef}
                  className="btn preset-outlined-surface-200-800"
                >
                  Cancel
                </Dialog.CloseTrigger>
                <button
                  type="submit"
                  disabled={mutation.isPending || !password}
                  className="btn preset-filled-primary-500"
                >
                  {mutation.isPending ? "Saving..." : "Save"}
                </button>
              </footer>
            </form>
          </Dialog.Content>
        </Dialog.Positioner>
      </Portal>
    </Dialog>
  );
}

function RemovePasswordDialog({
  userId,
  open,
  onOpenChange,
}: {
  userId: string;
  open: boolean;
  onOpenChange: (details: { open: boolean }) => void;
}) {
  const mutation = usePasswordMutation(userId);
  const cancelRef = useRef<HTMLButtonElement>(null);

  function handleOpenChange(details: { open: boolean }) {
    onOpenChange(details);
    if (!details.open) mutation.reset();
  }

  return (
    <Dialog
      role="alertdialog"
      open={open}
      onOpenChange={handleOpenChange}
      initialFocusEl={() => cancelRef.current}
    >
      <Portal>
        <Dialog.Backdrop className="fixed inset-0 z-50 bg-surface-50-950/50" />
        <Dialog.Positioner className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <Dialog.Content
            className={`card bg-surface-100-900 w-full max-w-sm space-y-4 p-6 shadow-xl ${DIALOG_ANIMATION}`}
          >
            <header className="flex items-start justify-between gap-2">
              <Dialog.Title className="text-lg font-bold">Remove password</Dialog.Title>
              <Dialog.CloseTrigger
                className="btn-icon hover:preset-tonal"
                aria-label="Close dialog"
              >
                <XIcon className="size-4" aria-hidden="true" />
              </Dialog.CloseTrigger>
            </header>
            <Dialog.Description className="text-surface-600-400 text-sm">
              This user will no longer be able to sign in with a password. They will need to use
              OIDC to authenticate.
            </Dialog.Description>
            {mutation.error && <ErrorAlert message={mutation.error.message} />}
            <footer className="flex justify-end gap-2">
              <Dialog.CloseTrigger ref={cancelRef} className="btn preset-outlined-surface-200-800">
                Cancel
              </Dialog.CloseTrigger>
              <button
                type="button"
                disabled={mutation.isPending}
                onClick={() =>
                  mutation.mutate(
                    { clearPassword: true },
                    { onSuccess: () => onOpenChange({ open: false }) },
                  )
                }
                className="btn preset-filled-error-500"
              >
                {mutation.isPending ? "Removing..." : "Remove password"}
              </button>
            </footer>
          </Dialog.Content>
        </Dialog.Positioner>
      </Portal>
    </Dialog>
  );
}

export function UserPasswordCard({ user }: { user: User }) {
  const [setOpen, setSetOpen] = useState(false);
  const [removeOpen, setRemoveOpen] = useState(false);

  return (
    <SettingsSection
      icon={<Key className="size-5" aria-hidden="true" />}
      title="Password"
      description={user.hasPassword ? "Password is set." : "No password set (OIDC only)."}
    >
      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          onClick={() => setSetOpen(true)}
          className="btn btn-sm preset-outlined-surface-200-800"
        >
          {user.hasPassword ? "Change password" : "Set password"}
        </button>
        {user.hasPassword && (
          <button
            type="button"
            onClick={() => setRemoveOpen(true)}
            className="btn btn-sm preset-outlined-error-500"
          >
            Remove password
          </button>
        )}
      </div>

      <SetPasswordDialog
        userId={user.id}
        open={setOpen}
        onOpenChange={(d) => setSetOpen(d.open)}
        label={user.hasPassword ? "Change password" : "Set password"}
      />
      <RemovePasswordDialog
        userId={user.id}
        open={removeOpen}
        onOpenChange={(d) => setRemoveOpen(d.open)}
      />
    </SettingsSection>
  );
}

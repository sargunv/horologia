import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { Dialog, Portal } from "@skeletonlabs/skeleton-react";
import { Key, Mail, Trash2, User as UserIcon, XIcon } from "lucide-react";
import { type FormEvent, useRef, useState } from "react";
import { apiClient } from "../../api/client.ts";
import { EditableField, PropertyRow } from "../../components/EditableField.tsx";
import { ErrorAlert } from "../../components/space-settings/ErrorAlert.tsx";
import { SettingsSection } from "../../components/space-settings/SettingsSection.tsx";
import { DIALOG_ANIMATION } from "../../lib/dialog.ts";
import { authConfigQueryOptions, currentUserQueryOptions } from "../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/settings")({
  loader: ({ context: { queryClient } }) =>
    Promise.all([
      queryClient.ensureQueryData(currentUserQueryOptions),
      queryClient.ensureQueryData(authConfigQueryOptions),
    ]),
  component: SettingsPage,
});

function SettingsPage() {
  const { data: user } = useQuery(currentUserQueryOptions);
  const { data: authConfig } = useQuery(authConfigQueryOptions);
  if (!user) return null;

  return (
    <div className="mx-auto max-w-2xl space-y-6 p-6">
      <div>
        <h1 className="h3">Account settings</h1>
        <p className="text-surface-600-400 mt-1 text-sm">
          Manage your profile and account preferences.
        </p>
      </div>
      <ProfileCard userId={user.id} name={user.name} email={user.email} />
      {authConfig?.password.enabled && (
        <PasswordCard userId={user.id} hasPassword={user.hasPassword} />
      )}
      <DangerZoneCard userId={user.id} email={user.email} />
    </div>
  );
}

function ProfileCard({ userId, name, email }: { userId: string; name: string; email: string }) {
  return (
    <SettingsSection
      icon={<UserIcon className="size-5" aria-hidden="true" />}
      title="Profile"
      description="Your personal information."
    >
      <PropertyRow label="Name" icon={<UserIcon className="size-4" aria-hidden="true" />}>
        <EditableField userId={userId} field="name" value={name} label="Name" />
      </PropertyRow>
      <PropertyRow label="Email" icon={<Mail className="size-4" aria-hidden="true" />}>
        <EditableField userId={userId} field="email" value={email} label="Email" type="email" />
      </PropertyRow>
    </SettingsSection>
  );
}

function PasswordCard({ userId, hasPassword }: { userId: string; hasPassword: boolean }) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const cancelRef = useRef<HTMLButtonElement>(null);
  const [password, setPassword] = useState("");

  const mutation = useMutation({
    mutationFn: async (body: { setPassword: string }) => {
      const { error } = await apiClient.PATCH("/users/{userId}", {
        params: { path: { userId } },
        body,
      });
      if (error) throw new Error(error.message ?? "Failed to update password");
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["currentUser"] });
      setOpen(false);
    },
  });

  const label = hasPassword ? "Change password" : "Set password";

  function handleOpenChange(details: { open: boolean }) {
    setOpen(details.open);
    if (!details.open) {
      setPassword("");
      mutation.reset();
    }
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    mutation.reset();
    mutation.mutate({ setPassword: password });
  }

  return (
    <SettingsSection
      icon={<Key className="size-5" aria-hidden="true" />}
      title="Password"
      description={hasPassword ? "Password is set." : "No password set."}
    >
      <div>
        <Dialog
          open={open}
          onOpenChange={handleOpenChange}
          initialFocusEl={() => cancelRef.current}
        >
          <Dialog.Trigger className="btn btn-sm preset-outlined-surface-200-800">
            {label}
          </Dialog.Trigger>
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
      </div>
    </SettingsSection>
  );
}

function DangerZoneCard({ userId, email }: { userId: string; email: string }) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [confirmation, setConfirmation] = useState("");
  const cancelRef = useRef<HTMLButtonElement>(null);

  const deleteMutation = useMutation({
    mutationFn: async () => {
      const { error } = await apiClient.DELETE("/users/{userId}", {
        params: { path: { userId } },
      });
      if (error) throw new Error(error.message ?? "Failed to delete account");
    },
    onSuccess: async () => {
      queryClient.clear();
      await navigate({ to: "/login" });
    },
  });

  const confirmationMatches = confirmation.toLowerCase() === email.toLowerCase();

  function handleOpenChange(details: { open: boolean }) {
    setOpen(details.open);
    if (!details.open) {
      deleteMutation.reset();
      setConfirmation("");
    }
  }

  return (
    <SettingsSection
      icon={<Trash2 className="size-5" aria-hidden="true" />}
      title="Danger zone"
      description="Irreversible actions for your account."
    >
      <div className="flex items-center justify-between gap-4">
        <p className="text-surface-600-400 text-sm">
          Permanently delete your account and all associated data.
        </p>
        <Dialog
          role="alertdialog"
          open={open}
          onOpenChange={handleOpenChange}
          initialFocusEl={() => cancelRef.current}
        >
          <Dialog.Trigger className="btn btn-sm preset-filled-error-500 shrink-0">
            Delete account
          </Dialog.Trigger>
          <Portal>
            <Dialog.Backdrop className="fixed inset-0 z-50 bg-surface-50-950/50" />
            <Dialog.Positioner className="fixed inset-0 z-50 flex items-center justify-center p-4">
              <Dialog.Content
                className={`card bg-surface-100-900 w-full max-w-md space-y-4 p-6 shadow-xl ${DIALOG_ANIMATION}`}
              >
                <header className="flex items-start justify-between gap-2">
                  <Dialog.Title className="text-lg font-bold">Delete account</Dialog.Title>
                  <Dialog.CloseTrigger
                    className="btn-icon hover:preset-tonal"
                    aria-label="Close dialog"
                  >
                    <XIcon className="size-4" aria-hidden="true" />
                  </Dialog.CloseTrigger>
                </header>
                <Dialog.Description className="text-surface-600-400 text-sm">
                  This will permanently delete your account, including all tasks, memberships, and
                  tokens. You will be signed out immediately and will not be able to log in again.
                </Dialog.Description>
                <div>
                  <label
                    htmlFor="delete-confirmation"
                    className="text-surface-600-400 mb-1 block text-sm"
                  >
                    Type <strong className="text-surface-950-50">{email}</strong> to confirm
                  </label>
                  <input
                    id="delete-confirmation"
                    type="text"
                    value={confirmation}
                    onChange={(e) => setConfirmation(e.target.value)}
                    className="input"
                    placeholder={email}
                    autoComplete="off"
                  />
                </div>
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
                    disabled={!confirmationMatches || deleteMutation.isPending}
                    onClick={() => {
                      deleteMutation.reset();
                      deleteMutation.mutate();
                    }}
                    className="btn preset-filled-error-500"
                  >
                    {deleteMutation.isPending ? "Deleting..." : "Delete account"}
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

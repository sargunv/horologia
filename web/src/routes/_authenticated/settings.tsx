import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { Dialog, Portal } from "@skeletonlabs/skeleton-react";
import {
  Check,
  Copy,
  Key,
  KeyRound,
  Mail,
  Trash2,
  TriangleAlert,
  User as UserIcon,
  XIcon,
} from "lucide-react";
import { type FormEvent, useEffect, useRef, useState } from "react";
import { apiClient } from "../../api/client.ts";
import { EditableField, PropertyRow } from "../../components/EditableField.tsx";
import { ErrorAlert } from "../../components/space-settings/ErrorAlert.tsx";
import { SettingsSection } from "../../components/space-settings/SettingsSection.tsx";
import { DIALOG_ANIMATION } from "../../lib/dialog.ts";
import { useUserPatch } from "../../lib/mutations.ts";
import {
  authConfigQueryOptions,
  authTokensQueryOptions,
  currentUserQueryOptions,
} from "../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/settings")({
  loader: ({ context: { queryClient } }) =>
    Promise.all([
      queryClient.ensureQueryData(currentUserQueryOptions),
      queryClient.ensureQueryData(authConfigQueryOptions),
      queryClient.ensureQueryData(authTokensQueryOptions),
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
      <TokensCard />
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
  const mutation = useUserPatch(userId);
  const [open, setOpen] = useState(false);
  const passwordRef = useRef<HTMLInputElement>(null);
  const [password, setPassword] = useState("");

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
    mutation.mutate({ setPassword: password }, { onSuccess: () => setOpen(false) });
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
          initialFocusEl={() => passwordRef.current}
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
                      ref={passwordRef}
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
                    <Dialog.CloseTrigger className="btn preset-outlined-surface-200-800">
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

function TokensCard() {
  const queryClient = useQueryClient();
  const { data: tokens } = useQuery(authTokensQueryOptions);

  const apiTokens = tokens?.filter((t) => t.kind === "api") ?? [];

  const [createOpen, setCreateOpen] = useState(false);
  const [tokenName, setTokenName] = useState("");
  const [revealedToken, setRevealedToken] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const nameRef = useRef<HTMLInputElement>(null);

  const doneRef = useRef<HTMLButtonElement>(null);
  const copyTimerRef = useRef<ReturnType<typeof setTimeout>>(null);
  const tokenInputRef = useRef<HTMLInputElement>(null);

  const [revokeTarget, setRevokeTarget] = useState<{ id: string; name: string } | null>(null);
  const cancelRef = useRef<HTMLButtonElement>(null);

  const createMutation = useMutation({
    mutationFn: async (name: string) => {
      const { data, error } = await apiClient.POST("/auth/tokens", {
        body: { name },
      });
      if (error) throw new Error(error.message ?? "Failed to create token");
      return data;
    },
    onSuccess: async (data) => {
      setRevealedToken(data.token);
      setTokenName("");
      await queryClient.invalidateQueries({ queryKey: ["authTokens"] });
    },
  });

  const revokeMutation = useMutation({
    mutationFn: async (tokenId: string) => {
      const { error } = await apiClient.DELETE("/auth/tokens/{tokenId}", {
        params: { path: { tokenId } },
      });
      if (error) throw new Error(error.message ?? "Failed to revoke token");
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["authTokens"] });
      setRevokeTarget(null);
    },
  });

  useEffect(() => {
    if (revealedToken) doneRef.current?.focus();
  }, [revealedToken]);

  function handleCreateOpenChange(details: { open: boolean }) {
    setCreateOpen(details.open);
    if (!details.open) {
      setTokenName("");
      setRevealedToken(null);
      setCopied(false);
      if (copyTimerRef.current) clearTimeout(copyTimerRef.current);
      createMutation.reset();
    }
  }

  function handleCreateSubmit(e: FormEvent) {
    e.preventDefault();
    createMutation.reset();
    createMutation.mutate(tokenName);
  }

  function handleRevokeOpenChange(details: { open: boolean }) {
    if (!details.open) {
      setRevokeTarget(null);
      revokeMutation.reset();
    }
  }

  async function handleCopy() {
    if (!revealedToken) return;
    try {
      await navigator.clipboard.writeText(revealedToken);
      setCopied(true);
      if (copyTimerRef.current) clearTimeout(copyTimerRef.current);
      copyTimerRef.current = setTimeout(() => setCopied(false), 2000);
    } catch {
      tokenInputRef.current?.select();
    }
  }

  const description = apiTokens.length === 0 ? "No API tokens yet." : "Manage your API tokens.";

  return (
    <SettingsSection
      icon={<KeyRound className="size-5" aria-hidden="true" />}
      title="API tokens"
      description={description}
    >
      {apiTokens.length > 0 && (
        <div className="divide-surface-200-800 divide-y">
          {apiTokens.map((token) => (
            <div key={token.id} className="flex items-center justify-between gap-4 py-2">
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">{token.name}</p>
                <p className="text-surface-600-400 text-xs">
                  Created {new Date(token.createdAt).toLocaleDateString()}
                </p>
              </div>
              <button
                type="button"
                className="btn btn-sm preset-outlined-surface-200-800 shrink-0"
                onClick={() => setRevokeTarget({ id: token.id, name: token.name })}
                disabled={revokeMutation.isPending}
              >
                Revoke
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Create token dialog */}
      <div>
        <Dialog
          open={createOpen}
          onOpenChange={handleCreateOpenChange}
          initialFocusEl={() => nameRef.current}
        >
          <Dialog.Trigger className="btn btn-sm preset-filled-primary-500">
            Create token
          </Dialog.Trigger>
          <Portal>
            <Dialog.Backdrop className="fixed inset-0 z-50 bg-surface-50-950/50" />
            <Dialog.Positioner className="fixed inset-0 z-50 flex items-center justify-center p-4">
              <Dialog.Content
                className={`card bg-surface-100-900 w-full max-w-sm space-y-4 p-6 shadow-xl ${DIALOG_ANIMATION}`}
              >
                <header className="flex items-start justify-between gap-2">
                  <Dialog.Title className="text-lg font-bold">
                    {revealedToken ? "Token created" : "Create API token"}
                  </Dialog.Title>
                  {!revealedToken && (
                    <Dialog.CloseTrigger
                      className="btn-icon hover:preset-tonal"
                      aria-label="Close dialog"
                    >
                      <XIcon className="size-4" aria-hidden="true" />
                    </Dialog.CloseTrigger>
                  )}
                </header>

                {revealedToken ? (
                  <div className="flex flex-col gap-3">
                    <div
                      role="status"
                      className="flex items-start gap-2 rounded-base bg-warning-500/10 px-3 py-2 text-sm text-warning-600 dark:text-warning-400"
                    >
                      <TriangleAlert className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
                      This token won't be shown again. Copy it now.
                    </div>
                    <div className="flex items-center gap-2">
                      <input
                        ref={tokenInputRef}
                        type="text"
                        readOnly
                        value={revealedToken}
                        onClick={(e) => e.currentTarget.select()}
                        className="bg-surface-200-800 flex-1 rounded-base border-none px-3 py-2 font-mono text-sm"
                      />
                      <button
                        type="button"
                        className="btn-icon preset-outlined-surface-200-800 shrink-0"
                        onClick={handleCopy}
                        aria-label={copied ? "Copied" : "Copy token"}
                      >
                        {copied ? (
                          <Check className="size-4 text-success-500" aria-hidden="true" />
                        ) : (
                          <Copy className="size-4" aria-hidden="true" />
                        )}
                      </button>
                    </div>
                    <footer className="flex justify-end">
                      <Dialog.CloseTrigger ref={doneRef} className="btn preset-filled-primary-500">
                        Done
                      </Dialog.CloseTrigger>
                    </footer>
                  </div>
                ) : (
                  <form onSubmit={handleCreateSubmit} className="flex flex-col gap-3">
                    <label className="flex flex-col gap-1">
                      <span className="text-surface-600-400 text-sm font-medium">Token name</span>
                      <input
                        ref={nameRef}
                        type="text"
                        required
                        minLength={1}
                        maxLength={100}
                        value={tokenName}
                        onChange={(e) => setTokenName(e.target.value)}
                        className="input preset-outlined-surface-200-800 w-full"
                        placeholder='e.g. "Claude", "CI pipeline"'
                        disabled={createMutation.isPending}
                      />
                    </label>
                    {createMutation.error && <ErrorAlert message={createMutation.error.message} />}
                    <footer className="flex justify-end gap-2">
                      <Dialog.CloseTrigger className="btn preset-outlined-surface-200-800">
                        Cancel
                      </Dialog.CloseTrigger>
                      <button
                        type="submit"
                        disabled={createMutation.isPending || !tokenName.trim()}
                        className="btn preset-filled-primary-500"
                      >
                        {createMutation.isPending ? "Creating..." : "Create"}
                      </button>
                    </footer>
                  </form>
                )}
              </Dialog.Content>
            </Dialog.Positioner>
          </Portal>
        </Dialog>
      </div>

      {/* Revoke confirmation dialog */}
      <Dialog
        role="alertdialog"
        open={revokeTarget !== null}
        onOpenChange={handleRevokeOpenChange}
        initialFocusEl={() => cancelRef.current}
      >
        <Portal>
          <Dialog.Backdrop className="fixed inset-0 z-50 bg-surface-50-950/50" />
          <Dialog.Positioner className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <Dialog.Content
              className={`card bg-surface-100-900 w-full max-w-sm space-y-4 p-6 shadow-xl ${DIALOG_ANIMATION}`}
            >
              <header className="flex items-start justify-between gap-2">
                <Dialog.Title className="text-lg font-bold">Revoke token</Dialog.Title>
                <Dialog.CloseTrigger
                  className="btn-icon hover:preset-tonal"
                  aria-label="Close dialog"
                  disabled={revokeMutation.isPending}
                >
                  <XIcon className="size-4" aria-hidden="true" />
                </Dialog.CloseTrigger>
              </header>
              <Dialog.Description className="text-surface-600-400 text-sm">
                Are you sure you want to revoke{" "}
                <strong className="text-surface-950-50">{revokeTarget?.name}</strong>? Any
                applications using this token will lose access immediately.
              </Dialog.Description>
              {revokeMutation.error && <ErrorAlert message={revokeMutation.error.message} />}
              <footer className="flex justify-end gap-2">
                <Dialog.CloseTrigger
                  ref={cancelRef}
                  className="btn preset-outlined-surface-200-800"
                  disabled={revokeMutation.isPending}
                >
                  Cancel
                </Dialog.CloseTrigger>
                <button
                  type="button"
                  disabled={revokeMutation.isPending}
                  onClick={() => {
                    if (revokeTarget) {
                      revokeMutation.reset();
                      revokeMutation.mutate(revokeTarget.id);
                    }
                  }}
                  className="btn preset-filled-error-500"
                >
                  {revokeMutation.isPending ? "Revoking..." : "Revoke"}
                </button>
              </footer>
            </Dialog.Content>
          </Dialog.Positioner>
        </Portal>
      </Dialog>
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
                  This will permanently delete your account, including your memberships,
                  assignments, and tokens. You will be signed out immediately and will not be able
                  to log in again.
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
                    className="input preset-outlined-surface-200-800 w-full"
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

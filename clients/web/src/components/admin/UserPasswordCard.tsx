import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Key } from "lucide-react";
import { type FormEvent, useState } from "react";
import { apiClient } from "../../api/client.ts";
import type { components } from "../../api/schema.d.ts";
import { notifyStaleData } from "../../lib/toaster.ts";
import {
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogRoot,
} from "../../ui/AlertDialog.tsx";
import {
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogRoot,
} from "../../ui/Dialog.tsx";
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
      try {
        await Promise.all([
          queryClient.invalidateQueries({ queryKey: [window.location.origin, "users", userId] }),
          queryClient.invalidateQueries({ queryKey: [window.location.origin, "users"] }),
        ]);
      } catch (err) {
        console.error("Cache invalidation failed after mutation:", err);
        notifyStaleData();
      }
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
  onOpenChange: (open: boolean) => void;
  label: string;
}) {
  const mutation = usePasswordMutation(userId);
  const [password, setPassword] = useState("");

  function handleOpenChange(next: boolean) {
    onOpenChange(next);
    if (!next) {
      setPassword("");
      mutation.reset();
    }
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    mutation.mutate(
      { setPassword: password },
      // Route through handleOpenChange so password + mutation state get
      // cleared — Radix only fires onOpenChange for user-initiated close.
      { onSuccess: () => handleOpenChange(false) },
    );
  }

  return (
    <DialogRoot open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-sm space-y-4">
        <DialogHeader title={label} />
        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
          <label className="flex flex-col gap-1">
            <span className="text-sm font-medium text-base-content/70">New password</span>
            <input
              type="password"
              required
              minLength={8}
              maxLength={72}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="input w-full"
              disabled={mutation.isPending}
            />
          </label>
          {mutation.error && <ErrorAlert message={mutation.error.message} />}
          <DialogFooter>
            <DialogClose className="btn btn-soft">Cancel</DialogClose>
            <button
              type="submit"
              disabled={mutation.isPending || !password}
              className="btn btn-primary"
            >
              {mutation.isPending ? "Saving..." : "Save"}
            </button>
          </DialogFooter>
        </form>
      </DialogContent>
    </DialogRoot>
  );
}

function RemovePasswordDialog({
  userId,
  open,
  onOpenChange,
}: {
  userId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const mutation = usePasswordMutation(userId);

  function handleOpenChange(next: boolean) {
    onOpenChange(next);
    if (!next) mutation.reset();
  }

  return (
    <AlertDialogRoot open={open} onOpenChange={handleOpenChange}>
      <AlertDialogContent className="max-w-sm space-y-4">
        <AlertDialogHeader title="Remove password" />
        <AlertDialogDescription>
          This user will no longer be able to sign in with a password. They will need to use OIDC to
          authenticate.
        </AlertDialogDescription>
        {mutation.error && <ErrorAlert message={mutation.error.message} />}
        <AlertDialogFooter>
          <AlertDialogCancel className="btn btn-soft">Cancel</AlertDialogCancel>
          <AlertDialogAction asChild>
            <button
              type="button"
              disabled={mutation.isPending}
              onClick={(e) => {
                e.preventDefault();
                mutation.mutate(
                  { clearPassword: true },
                  { onSuccess: () => handleOpenChange(false) },
                );
              }}
              className="btn btn-error"
            >
              {mutation.isPending ? "Removing..." : "Remove password"}
            </button>
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialogRoot>
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
        <button type="button" onClick={() => setSetOpen(true)} className="btn btn-soft btn-sm">
          {user.hasPassword ? "Change password" : "Set password"}
        </button>
        {user.hasPassword && (
          <button
            type="button"
            onClick={() => setRemoveOpen(true)}
            className="btn btn-outline btn-error btn-sm"
          >
            Remove password
          </button>
        )}
      </div>

      <SetPasswordDialog
        userId={user.id}
        open={setOpen}
        onOpenChange={setSetOpen}
        label={user.hasPassword ? "Change password" : "Set password"}
      />
      <RemovePasswordDialog userId={user.id} open={removeOpen} onOpenChange={setRemoveOpen} />
    </SettingsSection>
  );
}

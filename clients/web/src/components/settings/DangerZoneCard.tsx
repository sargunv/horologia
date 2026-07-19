import { useMutation } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { Trash2 } from "lucide-react";
import { useState } from "react";
import { useSettingsCommands } from "../../lib/mutations.ts";
import {
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogRoot,
  AlertDialogTrigger,
} from "../../ui/AlertDialog.tsx";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";
import { SettingsSection } from "../space-settings/SettingsSection.tsx";

export function DangerZoneCard({ userId, email }: { userId: string; email: string }) {
  const commands = useSettingsCommands();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [confirmation, setConfirmation] = useState("");

  const deleteMutation = useMutation({
    mutationFn: () => commands.deleteUser(userId),
    onSuccess: async () => {
      await navigate({ to: "/login" });
    },
  });

  const confirmationMatches = confirmation.toLowerCase() === email.toLowerCase();

  function handleOpenChange(next: boolean) {
    setOpen(next);
    if (!next) {
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
        <p className="text-sm text-base-content/70">
          Permanently delete your account and all associated data.
        </p>
        <AlertDialogRoot open={open} onOpenChange={handleOpenChange}>
          <AlertDialogTrigger className="btn btn-error btn-sm shrink-0">
            Delete account
          </AlertDialogTrigger>
          <AlertDialogContent className="max-w-md space-y-4">
            <AlertDialogHeader title="Delete account" />
            <AlertDialogDescription>
              This will permanently delete your account, including your memberships, assignments,
              and tokens. You will be signed out immediately and will not be able to log in again.
            </AlertDialogDescription>
            <div>
              <label
                htmlFor="delete-confirmation"
                className="mb-1 block text-sm text-base-content/70"
              >
                Type <strong className="text-base-content">{email}</strong> to confirm
              </label>
              <input
                id="delete-confirmation"
                type="text"
                value={confirmation}
                onChange={(e) => setConfirmation(e.target.value)}
                className="input w-full"
                placeholder={email}
                autoComplete="off"
              />
            </div>
            {deleteMutation.error && <ErrorAlert message={deleteMutation.error.message} />}
            <AlertDialogFooter>
              <AlertDialogCancel className="btn btn-soft">Cancel</AlertDialogCancel>
              <AlertDialogAction asChild>
                <button
                  type="button"
                  disabled={!confirmationMatches || deleteMutation.isPending}
                  onClick={(e) => {
                    e.preventDefault();
                    deleteMutation.reset();
                    deleteMutation.mutate();
                  }}
                  className="btn btn-error"
                >
                  {deleteMutation.isPending ? "Deleting..." : "Delete account"}
                </button>
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialogRoot>
      </div>
    </SettingsSection>
  );
}

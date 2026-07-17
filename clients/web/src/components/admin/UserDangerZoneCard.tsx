import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { Trash2 } from "lucide-react";
import { useState } from "react";
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
  AlertDialogTrigger,
} from "../../ui/AlertDialog.tsx";
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
        try {
          await queryClient.invalidateQueries({ queryKey: [window.location.origin, "users"] });
        } catch (err) {
          console.error("Cache invalidation failed after mutation:", err);
          notifyStaleData();
        }
        setOpen(false);
        onDeleted();
      }
    },
  });

  function handleOpenChange(next: boolean) {
    setOpen(next);
    if (!next) deleteMutation.reset();
  }

  return (
    <SettingsSection
      icon={<Trash2 className="size-5" aria-hidden="true" />}
      title="Danger zone"
      description="Irreversible actions for this user."
    >
      <div className="flex items-center justify-between gap-4">
        <p className="text-sm text-base-content/70">
          Permanently delete this user and all of their data.
        </p>
        <AlertDialogRoot open={open} onOpenChange={handleOpenChange}>
          <AlertDialogTrigger className="btn btn-error btn-sm shrink-0">
            Delete user
          </AlertDialogTrigger>
          <AlertDialogContent className="max-w-md space-y-4">
            <AlertDialogHeader title="Delete user" />
            <AlertDialogDescription>
              {isSelf ? (
                <>
                  You are about to delete{" "}
                  <strong className="text-base-content">your own account</strong>. You will be
                  signed out immediately and will not be able to log in again.
                </>
              ) : (
                <>
                  Permanently delete <strong className="text-base-content">{user.name}</strong> (
                  {user.email}). Their tasks, memberships, and tokens will be removed. This cannot
                  be undone.
                </>
              )}
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
                    deleteMutation.mutate();
                  }}
                  className="btn btn-error"
                >
                  {deleteMutation.isPending ? "Deleting..." : "Delete user"}
                </button>
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialogRoot>
      </div>
    </SettingsSection>
  );
}

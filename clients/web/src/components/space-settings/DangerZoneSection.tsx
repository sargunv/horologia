import { useMutation } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { Trash2 } from "lucide-react";
import { useState } from "react";
import type { components } from "../../api/schema.d.ts";
import { useLibraryCommands } from "../../lib/mutations.ts";
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
import { ErrorAlert } from "./ErrorAlert.tsx";
import { SettingsSection } from "./SettingsSection.tsx";

type Space = components["schemas"]["Space"];

export function DangerZoneSection({ space }: { space: Pick<Space, "slug" | "name"> }) {
  const navigate = useNavigate();
  const commands = useLibraryCommands();

  const [open, setOpen] = useState(false);
  const [confirmation, setConfirmation] = useState("");

  const confirmed = confirmation.trim() === space.slug;

  const deleteMutation = useMutation({
    mutationFn: () => commands.deleteSpace(space.slug),
    onSuccess: async () => {
      await navigate({ to: "/spaces" });
    },
  });

  function handleOpenChange(next: boolean) {
    setOpen(next);
    if (!next) {
      setConfirmation("");
      deleteMutation.reset();
    }
  }

  return (
    <SettingsSection
      icon={<Trash2 className="size-5" />}
      title="Danger Zone"
      description="Irreversible actions for this space."
    >
      <div className="flex items-center justify-between gap-4">
        <div>
          <p className="text-sm font-medium">Delete this space</p>
          <p className="text-sm text-base-content/70">
            Permanently delete this space and all of its data. This cannot be undone.
          </p>
        </div>
        <AlertDialogRoot open={open} onOpenChange={handleOpenChange}>
          <AlertDialogTrigger className="btn btn-error shrink-0">Delete space</AlertDialogTrigger>
          <AlertDialogContent className="max-w-md space-y-4">
            <AlertDialogHeader title="Delete space" />
            <AlertDialogDescription>
              This will permanently delete{" "}
              <strong className="text-base-content">{space.name}</strong>, including its content,
              tags, members, and settings. Historical activity is retained. This action cannot be
              undone.
            </AlertDialogDescription>
            <label className="flex flex-col gap-1">
              <span className="text-sm">
                Type <strong className="font-mono">{space.slug}</strong> to confirm
              </span>
              <input
                type="text"
                value={confirmation}
                onChange={(e) => setConfirmation(e.target.value)}
                className="input w-full"
                placeholder={space.slug}
                disabled={deleteMutation.isPending}
                autoComplete="off"
              />
            </label>
            {deleteMutation.error && <ErrorAlert message={deleteMutation.error.message} />}
            <AlertDialogFooter>
              <AlertDialogCancel className="btn btn-soft">Cancel</AlertDialogCancel>
              <AlertDialogAction asChild>
                <button
                  type="button"
                  disabled={!confirmed || deleteMutation.isPending}
                  onClick={(e) => {
                    e.preventDefault();
                    deleteMutation.mutate();
                  }}
                  className="btn btn-error"
                >
                  {deleteMutation.isPending ? "Deleting..." : "Delete space"}
                </button>
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialogRoot>
      </div>
    </SettingsSection>
  );
}

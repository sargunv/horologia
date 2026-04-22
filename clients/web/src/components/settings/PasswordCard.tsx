import { Key } from "lucide-react";
import { type FormEvent, useState } from "react";
import { useUserPatch } from "../../lib/mutations.ts";
import {
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogRoot,
  DialogTrigger,
} from "../../ui/Dialog.tsx";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";
import { SettingsSection } from "../space-settings/SettingsSection.tsx";

export function PasswordCard({ userId, hasPassword }: { userId: string; hasPassword: boolean }) {
  const mutation = useUserPatch(userId);
  const [open, setOpen] = useState(false);
  const [password, setPassword] = useState("");

  const label = hasPassword ? "Change password" : "Set password";

  function handleOpenChange(next: boolean) {
    setOpen(next);
    if (!next) {
      setPassword("");
      mutation.reset();
    }
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    mutation.reset();
    mutation.mutate(
      { setPassword: password },
      // Route through handleOpenChange so password + mutation state get
      // cleared — Radix only fires onOpenChange for user-initiated close.
      { onSuccess: () => handleOpenChange(false) },
    );
  }

  return (
    <SettingsSection
      icon={<Key className="size-5" aria-hidden="true" />}
      title="Password"
      description={hasPassword ? "Password is set." : "No password set."}
    >
      <div>
        <DialogRoot open={open} onOpenChange={handleOpenChange}>
          <DialogTrigger className="btn btn-soft btn-sm">{label}</DialogTrigger>
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
                  autoFocus
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
      </div>
    </SettingsSection>
  );
}

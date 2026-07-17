import { useMutation, useQuery } from "@tanstack/react-query";
import { Check, Copy, KeyRound, TriangleAlert } from "lucide-react";
import { type FormEvent, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { useSettingsCommands } from "../../lib/mutations.ts";
import { authTokensQueryOptions } from "../../lib/queries.ts";
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
  DialogTrigger,
} from "../../ui/Dialog.tsx";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";
import { SettingsSection } from "../space-settings/SettingsSection.tsx";

export function TokensCard() {
  const commands = useSettingsCommands();
  const { data: tokens } = useQuery(authTokensQueryOptions);

  const apiTokens = tokens?.filter((t) => t.kind === "api") ?? [];

  const [createOpen, setCreateOpen] = useState(false);
  const [tokenName, setTokenName] = useState("");
  const [revealedToken, setRevealedToken] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const copyTimerRef = useRef<ReturnType<typeof setTimeout>>(null);
  const tokenInputRef = useRef<HTMLInputElement>(null);

  const [revokeTarget, setRevokeTarget] = useState<{
    id: string;
    name: string;
  } | null>(null);

  const createMutation = useMutation({
    mutationFn: (name: string) => commands.createToken(name),
    onSuccess: (data) => {
      setRevealedToken(data.token);
      setTokenName("");
    },
  });

  const revokeMutation = useMutation({
    mutationFn: (tokenId: string) => commands.revokeToken(tokenId),
    onSuccess: () => {
      setRevokeTarget(null);
    },
  });

  useEffect(() => {
    return () => {
      if (copyTimerRef.current) clearTimeout(copyTimerRef.current);
    };
  }, []);

  function handleCreateOpenChange(next: boolean) {
    setCreateOpen(next);
    if (!next) {
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

  function handleRevokeOpenChange(next: boolean) {
    if (!next) {
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
      // Fallback: select the token so the user can Cmd/Ctrl+C it manually,
      // and surface the failure via toast so the silent retry is noticed.
      tokenInputRef.current?.select();
      toast.error("Couldn't copy automatically — select + copy manually");
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
        <div className="divide-y divide-base-300">
          {apiTokens.map((token) => (
            <div key={token.id} className="flex items-center justify-between gap-4 py-2">
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">{token.name}</p>
                <p className="text-xs text-base-content/70">
                  Created {new Date(token.createdAt).toLocaleDateString()}
                </p>
              </div>
              <button
                type="button"
                className="btn btn-soft btn-sm shrink-0"
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
        <DialogRoot open={createOpen} onOpenChange={handleCreateOpenChange}>
          <DialogTrigger className="btn btn-primary btn-sm">Create token</DialogTrigger>
          <DialogContent
            className="max-w-sm space-y-4"
            onInteractOutside={(event) => {
              if (revealedToken !== null) event.preventDefault();
            }}
            onEscapeKeyDown={(event) => {
              if (revealedToken !== null) event.preventDefault();
            }}
          >
            <DialogHeader
              title={revealedToken ? "Token created" : "Create API token"}
              closeButton={!revealedToken}
            />

            {revealedToken ? (
              <div className="flex flex-col gap-3">
                <div role="status" className="alert alert-warning alert-soft text-sm">
                  <TriangleAlert className="size-4 shrink-0" aria-hidden="true" />
                  This token won't be shown again. Copy it now.
                </div>
                <div className="flex items-center gap-2">
                  <input
                    ref={tokenInputRef}
                    type="text"
                    readOnly
                    value={revealedToken}
                    onClick={(e) => e.currentTarget.select()}
                    className="flex-1 rounded-field border border-base-300 bg-base-200 px-3 py-2 font-mono text-sm"
                  />
                  <button
                    type="button"
                    className="btn btn-soft btn-square shrink-0"
                    onClick={handleCopy}
                    aria-label={copied ? "Copied" : "Copy token"}
                  >
                    {copied ? (
                      <Check className="size-4 text-success" aria-hidden="true" />
                    ) : (
                      <Copy className="size-4" aria-hidden="true" />
                    )}
                  </button>
                </div>
                <DialogFooter>
                  <DialogClose className="btn btn-primary">Done</DialogClose>
                </DialogFooter>
              </div>
            ) : (
              <form onSubmit={handleCreateSubmit} className="flex flex-col gap-3">
                <label className="flex flex-col gap-1">
                  <span className="text-sm font-medium text-base-content/70">Token name</span>
                  <input
                    type="text"
                    required
                    minLength={1}
                    maxLength={100}
                    value={tokenName}
                    onChange={(e) => setTokenName(e.target.value)}
                    className="input w-full"
                    placeholder='e.g. "Claude", "CI pipeline"'
                    disabled={createMutation.isPending}
                    autoFocus
                  />
                </label>
                {createMutation.error && <ErrorAlert message={createMutation.error.message} />}
                <DialogFooter>
                  <DialogClose className="btn btn-soft">Cancel</DialogClose>
                  <button
                    type="submit"
                    disabled={createMutation.isPending || !tokenName.trim()}
                    className="btn btn-primary"
                  >
                    {createMutation.isPending ? "Creating..." : "Create"}
                  </button>
                </DialogFooter>
              </form>
            )}
          </DialogContent>
        </DialogRoot>
      </div>

      {/* Revoke confirmation dialog */}
      <AlertDialogRoot open={revokeTarget !== null} onOpenChange={handleRevokeOpenChange}>
        <AlertDialogContent className="max-w-sm space-y-4">
          <AlertDialogHeader title="Revoke token" />
          <AlertDialogDescription>
            Are you sure you want to revoke{" "}
            <strong className="text-base-content">{revokeTarget?.name}</strong>? Any applications
            using this token will lose access immediately.
          </AlertDialogDescription>
          {revokeMutation.error && <ErrorAlert message={revokeMutation.error.message} />}
          <AlertDialogFooter>
            <AlertDialogCancel className="btn btn-soft" disabled={revokeMutation.isPending}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction asChild>
              <button
                type="button"
                disabled={revokeMutation.isPending}
                onClick={(e) => {
                  e.preventDefault();
                  if (revokeTarget) {
                    revokeMutation.reset();
                    revokeMutation.mutate(revokeTarget.id);
                  }
                }}
                className="btn btn-error"
              >
                {revokeMutation.isPending ? "Revoking..." : "Revoke"}
              </button>
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialogRoot>
    </SettingsSection>
  );
}

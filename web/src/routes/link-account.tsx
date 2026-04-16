import { useMutation, useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { CircleAlert } from "lucide-react";
import { type FormEvent, useEffect, useState } from "react";
import { appClient, getApiErrorMessage } from "../api/client.ts";
import type { components } from "../api/schema.d.ts";
import { navigateToTarget } from "../lib/navigation.ts";
import { linkPendingQueryOptions } from "../lib/queries.ts";

export const Route = createFileRoute("/link-account")({
  component: LinkAccountPage,
});

function LinkAccountPage() {
  const navigate = useNavigate();
  const { data: pending, isPending: isLoading, isError } = useQuery(linkPendingQueryOptions);

  const [password, setPassword] = useState("");

  const linkMutation = useMutation({
    mutationFn: async ({
      password,
    }: {
      password: string;
    }): Promise<components["schemas"]["AuthLinkResponse"]> => {
      const { data, error } = await appClient.POST("/app/auth/link", {
        body: { password },
      });
      if (error) {
        throw new Error(getApiErrorMessage(error, "Failed to link account"));
      }
      if (isAuthLinkResponse(data)) {
        return data;
      }
      throw new Error("Failed to link account");
    },
    onSuccess: (data) => {
      navigateToTarget(data.redirectTo || "/", navigate);
    },
  });

  useEffect(() => {
    if (!isLoading && (pending === null || isError)) {
      void navigate({ to: "/login" });
    }
  }, [isError, isLoading, navigate, pending]);

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    linkMutation.mutate({ password });
  }

  if (!isLoading && (pending === null || isError)) {
    return null;
  }

  return (
    <div className="flex min-h-svh items-center justify-center p-4">
      <div className="flex w-full max-w-sm flex-col gap-4">
        <div className="flex flex-col items-center gap-1">
          <h1 className="h1">Horologia</h1>
          <p className="text-surface-600-400 text-sm">Confirm account linking</p>
        </div>

        <div className="card preset-outlined-surface-200-800 flex flex-col gap-6 p-6">
          {isLoading ? (
            <div className="flex items-center justify-center py-4">
              <span className="text-surface-500 text-sm">Loading...</span>
            </div>
          ) : (
            <>
              <p className="text-sm">
                An account with the email <strong>{pending?.email}</strong> already exists. Enter
                your password to link your login method to this account.
              </p>

              <form onSubmit={handleSubmit} className="flex flex-col gap-4">
                <label className="flex flex-col gap-1">
                  <span className="text-surface-600-400 text-sm font-medium">Password</span>
                  <input
                    type="password"
                    required
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="input preset-outlined-surface-200-800 w-full"
                    placeholder="Enter your password"
                    autoComplete="current-password"
                    disabled={linkMutation.isPending}
                  />
                </label>

                {linkMutation.error && (
                  <div className="preset-filled-error-500 flex items-center gap-2 rounded-base px-3 py-2 text-sm">
                    <CircleAlert className="size-4 shrink-0" />
                    {linkMutation.error.message}
                  </div>
                )}

                <button
                  type="submit"
                  disabled={linkMutation.isPending}
                  className="btn preset-filled-primary-500 w-full"
                >
                  {linkMutation.isPending ? "Linking..." : "Link account"}
                </button>
              </form>

              <button
                type="button"
                onClick={() => void navigate({ to: "/login" })}
                className="btn preset-outlined-surface-200-800 w-full"
                disabled={linkMutation.isPending}
              >
                Cancel
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function isAuthLinkResponse(value: unknown): value is components["schemas"]["AuthLinkResponse"] {
  return (
    typeof value === "object" &&
    value !== null &&
    "redirectTo" in value &&
    typeof value.redirectTo === "string"
  );
}

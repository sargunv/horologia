import { useMutation, useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { CircleAlert } from "lucide-react";
import { type FormEvent, useEffect, useState } from "react";
import { apiClient, getApiErrorMessage } from "../api/client.ts";
import { authConfigQueryOptions } from "../lib/queries.ts";
import { buildOIDCLoginURL, navigateToTarget } from "../lib/redirects.ts";

interface LoginSearch {
  redirect?: string;
  noredirect?: boolean;
}

export const Route = createFileRoute("/login")({
  validateSearch: (search: Record<string, unknown>): LoginSearch => {
    const r = typeof search["redirect"] === "string" ? search["redirect"] : undefined;
    const redirect = r && r.startsWith("/") && !r.startsWith("//") ? r : undefined;
    const noredirect =
      search["noredirect"] === true ||
      search["noredirect"] === "true" ||
      search["noredirect"] === "";
    return {
      ...(redirect !== undefined ? { redirect } : {}),
      ...(noredirect ? { noredirect } : {}),
    };
  },
  component: LoginPage,
});

function LoginPage() {
  const navigate = useNavigate();
  const { redirect, noredirect } = Route.useSearch();
  const { data: authConfig, isPending: isAuthConfigPending } = useQuery(authConfigQueryOptions);

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  const loginMutation = useMutation({
    mutationFn: async ({ email, password }: { email: string; password: string }) => {
      const { error } = await apiClient.POST("/auth/login", {
        body: { email, password },
      });
      if (error) throw new Error(getApiErrorMessage(error, "Invalid email or password"));
    },
    onSuccess: () => {
      navigateToTarget(redirect ?? "/", navigate);
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    loginMutation.mutate({ email, password });
  }

  function handleOIDCLogin() {
    window.location.href = buildOIDCLoginURL(redirect);
  }

  useEffect(() => {
    if (
      authConfig &&
      !noredirect &&
      authConfig.oidc.enabled &&
      authConfig.oidc.autoRedirect &&
      !authConfig.password.enabled
    ) {
      window.location.href = buildOIDCLoginURL(redirect);
    }
  }, [authConfig, noredirect, redirect]);

  return (
    <div className="flex min-h-svh items-center justify-center p-4">
      <div className="flex w-full max-w-sm flex-col gap-4">
        <div className="flex flex-col items-center gap-1">
          <h1 className="h1">Horologia</h1>
          <p className="text-surface-600-400 text-sm">Sign in to your account</p>
        </div>

        <div className="card preset-outlined-surface-200-800 flex flex-col gap-6 p-6">
          {isAuthConfigPending ? (
            <div className="flex items-center justify-center py-4">
              <span className="text-surface-500 text-sm">Loading...</span>
            </div>
          ) : !authConfig?.password.enabled && !authConfig?.oidc.enabled ? (
            <div className="preset-filled-error-500 flex items-center gap-2 rounded-base px-3 py-2 text-sm">
              <CircleAlert className="size-4 shrink-0" />
              No authentication methods available
            </div>
          ) : null}

          {authConfig?.password.enabled && (
            <form onSubmit={handleSubmit} className="flex flex-col gap-4">
              <label className="flex flex-col gap-1">
                <span className="text-surface-600-400 text-sm font-medium">Email</span>
                <input
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="input preset-outlined-surface-200-800 w-full"
                  placeholder="you@example.com"
                  autoComplete="email"
                  disabled={loginMutation.isPending}
                />
              </label>

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
                  disabled={loginMutation.isPending}
                />
              </label>

              {loginMutation.error && (
                <div className="preset-filled-error-500 flex items-center gap-2 rounded-base px-3 py-2 text-sm">
                  <CircleAlert className="size-4 shrink-0" />
                  {loginMutation.error.message}
                </div>
              )}

              <button
                type="submit"
                disabled={loginMutation.isPending}
                className="btn preset-filled-primary-500 w-full"
              >
                {loginMutation.isPending ? "Signing in..." : "Sign in"}
              </button>
            </form>
          )}

          {authConfig?.password.enabled && authConfig?.oidc.enabled && (
            <div className="flex items-center gap-3">
              <hr className="border-surface-200-800 flex-1" />
              <span className="text-surface-500 text-xs uppercase tracking-wider">or</span>
              <hr className="border-surface-200-800 flex-1" />
            </div>
          )}

          {authConfig?.oidc.enabled && (
            <button
              type="button"
              onClick={handleOIDCLogin}
              className="btn preset-outlined-surface-200-800 w-full"
            >
              Sign in with {authConfig.oidc.label}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

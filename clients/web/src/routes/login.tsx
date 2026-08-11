import { useMutation, useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { CircleAlert } from "lucide-react";
import { type FormEvent, useEffect, useState, type ReactNode } from "react";
import { getApiErrorMessage } from "@horologia/client-core/api";

import { appClient } from "../api/client.ts";
import { useTheme } from "../lib/theme.tsx";
import { authConfigQueryOptions } from "../lib/queries.ts";
import { navigateToTarget } from "../lib/navigation.ts";
import { BotanicalPlate } from "../ui/BotanicalPlate.tsx";
import { Card } from "../ui/Card.tsx";
import { CatalogLabel } from "../ui/CatalogLabel.tsx";

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
  const { resolvedTheme } = useTheme();
  const isFlorilegium = resolvedTheme === "florilegium";
  const { data: authConfig, isPending: isAuthConfigPending } = useQuery(authConfigQueryOptions);

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  const loginMutation = useMutation({
    mutationFn: async ({ email, password }: { email: string; password: string }) => {
      const { error } = await appClient.POST("/app/auth/login", {
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

  const authBody = (
    <>
      {isAuthConfigPending ? (
        <div className="flex items-center justify-center py-4">
          <span className="text-base-content/60 text-sm">Loading...</span>
        </div>
      ) : !authConfig?.password.enabled && !authConfig?.oidc.enabled ? (
        <div role="alert" className="alert alert-error alert-soft text-sm">
          <CircleAlert className="size-4 shrink-0" />
          No authentication methods available
        </div>
      ) : null}

      {authConfig?.password.enabled && (
        <form onSubmit={handleSubmit} className="flex flex-col gap-5">
          <label className="flex flex-col gap-1.5">
            {isFlorilegium ? (
              <CatalogLabel>Email</CatalogLabel>
            ) : (
              <span className="text-base-content/70 text-sm font-medium">Email</span>
            )}
            <input
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className={`input w-full ${isFlorilegium ? "input-ledger" : ""}`}
              placeholder="you@example.com"
              autoComplete="email"
              disabled={loginMutation.isPending}
            />
          </label>

          <label className="flex flex-col gap-1.5">
            {isFlorilegium ? (
              <CatalogLabel>Password</CatalogLabel>
            ) : (
              <span className="text-base-content/70 text-sm font-medium">Password</span>
            )}
            <input
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={`input w-full ${isFlorilegium ? "input-ledger" : ""}`}
              placeholder="Enter your password"
              autoComplete="current-password"
              disabled={loginMutation.isPending}
            />
          </label>

          {loginMutation.error && (
            <div role="alert" className="alert alert-error alert-soft text-sm">
              <CircleAlert className="size-4 shrink-0" />
              {loginMutation.error.message}
            </div>
          )}

          <button
            type="submit"
            disabled={loginMutation.isPending}
            className="btn btn-primary w-full"
          >
            {loginMutation.isPending ? "Signing in..." : "Sign in"}
          </button>
        </form>
      )}

      {authConfig?.password.enabled && authConfig?.oidc.enabled && (
        <div className="flex items-center gap-3">
          <hr className="border-base-300 flex-1" />
          {isFlorilegium ? (
            <CatalogLabel>or</CatalogLabel>
          ) : (
            <span className="text-base-content/60 text-xs uppercase tracking-wider">or</span>
          )}
          <hr className="border-base-300 flex-1" />
        </div>
      )}

      {authConfig?.oidc.enabled && (
        <button type="button" onClick={handleOIDCLogin} className="btn btn-soft w-full">
          Sign in with {authConfig.oidc.label}
        </button>
      )}
    </>
  );

  if (isFlorilegium) {
    return (
      <FlorilegiumGate>
        <div className="florilegium-gate-hero flex flex-col items-center gap-5 text-center">
          <BotanicalPlate className="florilegium-gate-ornament w-36" />
          <div className="flex flex-col items-center gap-2">
            <CatalogLabel>A household florilegium</CatalogLabel>
            <h1 className="florilegium-gate-title text-5xl font-bold tracking-tight md:text-6xl">
              Horologia
            </h1>
            <p className="max-w-xs text-sm leading-relaxed text-base-content/70">
              Tasks, recipes, and the quiet work of keeping house — filed and ready.
            </p>
          </div>
        </div>
        <Card className="specimen-sheet florilegium-gate-card flex flex-col gap-6 p-6 md:p-8">
          {authBody}
        </Card>
      </FlorilegiumGate>
    );
  }

  return (
    <div className="flex min-h-svh items-center justify-center p-4">
      <div className="flex w-full max-w-sm flex-col gap-5">
        <div className="flex flex-col items-center gap-1.5 text-center">
          <h1 className="text-4xl font-bold tracking-tight">Horologia</h1>
          <p className="text-base-content/70 text-sm">Sign in to your account</p>
        </div>
        <Card className="flex flex-col gap-6 p-6">{authBody}</Card>
      </div>
    </div>
  );
}

function FlorilegiumGate({ children }: { children: ReactNode }) {
  return (
    <div className="florilegium-gate flex min-h-svh items-center justify-center p-4 md:p-8">
      <div className="florilegium-gate-plate flex w-full max-w-md flex-col gap-8">{children}</div>
    </div>
  );
}

function buildOIDCLoginURL(redirect?: string): string {
  const params = new URLSearchParams();
  if (redirect) {
    params.set("redirect", redirect);
  }
  const query = params.toString();
  return `/app/auth/oidc${query ? `?${query}` : ""}`;
}

import { useMutation } from "@tanstack/react-query";
import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router";
import { type FormEvent, useState } from "react";
import { ErrorDisplay } from "../components/ui/error-display.tsx";
import { queryClient } from "../lib/query-client.ts";
import { meQueryOptions, webLogin } from "../queries/auth.ts";

export const Route = createFileRoute("/login")({
  validateSearch: (search: Record<string, unknown>) => ({
    redirect: (search["redirect"] as string) ?? undefined,
  }),
  beforeLoad: async ({ context }) => {
    try {
      await context.queryClient.ensureQueryData(meQueryOptions());
      throw redirect({ to: "/" });
    } catch (e: unknown) {
      if (e !== null && typeof e === "object" && "to" in e) throw e;
    }
  },
  component: LoginPage,
});

function LoginPage() {
  const navigate = useNavigate();
  const search = Route.useSearch();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  const login = useMutation({
    mutationFn: () => webLogin(email, password),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["me"] });
      const destination = search.redirect?.startsWith("/") ? search.redirect : "/";
      void navigate({ to: destination });
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    login.mutate();
  }

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="card w-full max-w-sm p-6">
        <h1 className="h3 mb-6 text-center">Sign in to Tend</h1>
        <form onSubmit={handleSubmit} className="space-y-4">
          <label className="label">
            <span className="label-text">Email</span>
            <input
              type="email"
              className="input"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              autoFocus
            />
          </label>
          <label className="label">
            <span className="label-text">Password</span>
            <input
              type="password"
              className="input"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </label>
          {login.error ? <ErrorDisplay error={login.error} /> : null}
          <button
            type="submit"
            className="btn preset-filled-primary-500 w-full"
            disabled={login.isPending}
          >
            {login.isPending ? "Signing in\u2026" : "Sign in"}
          </button>
        </form>
        <div className="mt-4 flex items-center gap-3">
          <hr className="flex-1 border-surface-300 dark:border-surface-700" />
          <span className="text-xs text-surface-500">or</span>
          <hr className="flex-1 border-surface-300 dark:border-surface-700" />
        </div>
        <a href="/api/auth/oidc" className="btn preset-tonal-surface w-full mt-4">
          Sign in with OIDC
        </a>
      </div>
    </div>
  );
}

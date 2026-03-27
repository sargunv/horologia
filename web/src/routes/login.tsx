import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { type FormEvent, useState } from "react";
import { authConfigQueryOptions } from "../lib/queries.ts";

interface LoginSearch {
  redirect: string | undefined;
}

export const Route = createFileRoute("/login")({
  validateSearch: (search: Record<string, unknown>): LoginSearch => {
    const r = typeof search["redirect"] === "string" ? search["redirect"] : undefined;
    return { redirect: r && r.startsWith("/") && !r.startsWith("//") ? r : undefined };
  },
  component: LoginPage,
});

function LoginPage() {
  const navigate = useNavigate();
  const { redirect } = Route.useSearch();
  const { data: authConfig } = useQuery(authConfigQueryOptions);

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setPending(true);

    try {
      const res = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ email, password }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => null);
        setError(body?.message ?? "Invalid email or password");
        return;
      }
      void navigate({ to: redirect ?? "/" });
    } catch {
      setError("Something went wrong. Please try again.");
    } finally {
      setPending(false);
    }
  }

  function handleOIDCLogin() {
    const params = new URLSearchParams();
    if (redirect) params.set("redirect", redirect);
    const query = params.toString();
    window.location.href = `/api/auth/oidc${query ? `?${query}` : ""}`;
  }

  return (
    <div className="flex min-h-svh items-center justify-center p-4">
      <div className="flex w-full max-w-sm flex-col gap-4">
        <div className="flex flex-col items-center gap-1">
          <h1 className="h1">Tend</h1>
          <p className="text-surface-600-400 text-sm">Sign in to your account</p>
        </div>

        <div className="card preset-outlined-surface-200-800 flex flex-col gap-6 p-6">
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
                disabled={pending}
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
                disabled={pending}
              />
            </label>

            {error && (
              <div className="preset-filled-error-500 flex items-center gap-2 rounded-base px-3 py-2 text-sm">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 20 20"
                  fill="currentColor"
                  className="size-4 shrink-0"
                >
                  <path
                    fillRule="evenodd"
                    d="M18 10a8 8 0 1 1-16 0 8 8 0 0 1 16 0Zm-8-5a.75.75 0 0 1 .75.75v4.5a.75.75 0 0 1-1.5 0v-4.5A.75.75 0 0 1 10 5Zm0 10a1 1 0 1 0 0-2 1 1 0 0 0 0 2Z"
                    clipRule="evenodd"
                  />
                </svg>
                {error}
              </div>
            )}

            <button
              type="submit"
              disabled={pending}
              className="btn preset-filled-primary-500 w-full"
            >
              {pending ? "Signing in..." : "Sign in"}
            </button>
          </form>

          {authConfig?.oidc.enabled && (
            <>
              <div className="flex items-center gap-3">
                <hr className="border-surface-200-800 flex-1" />
                <span className="text-surface-500 text-xs uppercase tracking-wider">or</span>
                <hr className="border-surface-200-800 flex-1" />
              </div>

              <button
                type="button"
                onClick={handleOIDCLogin}
                disabled={pending}
                className="btn preset-outlined-surface-200-800 w-full"
              >
                Sign in with {authConfig.oidc.label}
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

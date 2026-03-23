import { Link, useNavigate } from "@tanstack/react-router";
import type { ReactNode } from "react";
import { queryClient } from "../../lib/query-client.ts";
import { webLogout } from "../../queries/auth.ts";

interface AppShellProps {
  user: { id: string; email: string; name: string };
  children: ReactNode;
}

export function AppShell({ user, children }: AppShellProps) {
  const navigate = useNavigate();

  async function handleLogout() {
    await webLogout();
    queryClient.clear();
    void navigate({ to: "/login" });
  }

  return (
    <div className="flex min-h-screen flex-col">
      <header className="flex items-center justify-between border-b border-surface-200 dark:border-surface-800 px-6 py-3">
        <div className="flex items-center gap-6">
          <Link to="/" className="text-lg font-bold">
            Tend
          </Link>
          <nav className="flex items-center gap-4 text-sm">
            <Link to="/spaces" className="hover:underline">
              Spaces
            </Link>
          </nav>
        </div>
        <div className="flex items-center gap-4">
          <span className="text-sm text-surface-500">{user.name}</span>
          <button
            type="button"
            onClick={() => void handleLogout()}
            className="btn btn-sm preset-tonal-surface"
          >
            Sign out
          </button>
        </div>
      </header>
      <main className="flex-1 p-6">
        <div className="mx-auto max-w-4xl">{children}</div>
      </main>
    </div>
  );
}

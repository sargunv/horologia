import { useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { Menu, Portal } from "@skeletonlabs/skeleton-react";
import type { components } from "../api/schema.d.ts";

type User = components["schemas"]["User"];

export function UserMenu({ user }: { user: User }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  async function handleLogout() {
    try {
      await fetch("/api/auth/logout", {
        method: "POST",
        credentials: "include",
      });
    } catch {
      // Proceed with local logout regardless of network failure
    }
    queryClient.clear();
    void navigate({ to: "/login" });
  }

  return (
    <Menu>
      <Menu.Trigger className="flex w-full items-center gap-2.5 rounded-md px-2 py-2 outline-none cursor-pointer hover:bg-surface-200-800 transition-colors">
        <div className="flex-1 min-w-0 text-left">
          <div className="text-sm font-medium truncate">{user.name}</div>
          <div className="text-surface-600-400 text-xs truncate">{user.email}</div>
        </div>
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          className="text-surface-500 size-4 shrink-0"
        >
          <path
            fillRule="evenodd"
            d="M10 3a.75.75 0 0 1 .55.24l3.25 3.5a.75.75 0 1 1-1.1 1.02L10 4.852 7.3 7.76a.75.75 0 0 1-1.1-1.02l3.25-3.5A.75.75 0 0 1 10 3Zm-3.76 9.2a.75.75 0 0 1 1.06.04l2.7 2.908 2.7-2.908a.75.75 0 1 1 1.1 1.02l-3.25 3.5a.75.75 0 0 1-1.1 0l-3.25-3.5a.75.75 0 0 1 .04-1.06Z"
            clipRule="evenodd"
          />
        </svg>
      </Menu.Trigger>
      <Portal>
        <Menu.Positioner>
          <Menu.Content>
            <Menu.Item value="settings">
              <Link to="/settings">
                <Menu.ItemText>Settings</Menu.ItemText>
              </Link>
            </Menu.Item>
            {user.isOwner && (
              <Menu.Item value="admin">
                <Link to="/admin">
                  <Menu.ItemText>Admin</Menu.ItemText>
                </Link>
              </Menu.Item>
            )}
            <Menu.Separator />
            <Menu.Item value="logout" onClick={handleLogout}>
              <Menu.ItemText>Sign out</Menu.ItemText>
            </Menu.Item>
          </Menu.Content>
        </Menu.Positioner>
      </Portal>
    </Menu>
  );
}

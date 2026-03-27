import { useQueryClient } from "@tanstack/react-query";
import { createLink, useNavigate } from "@tanstack/react-router";
import { ChevronsUpDown } from "lucide-react";
import { Menu, Portal } from "@skeletonlabs/skeleton-react";
import type { components } from "../api/schema.d.ts";

const MenuItemLink = createLink(Menu.Item);

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
      <Menu.Trigger className="btn preset-tonal-surface w-full gap-2.5">
        <div className="flex-1 min-w-0 text-left">
          <div className="text-sm font-medium truncate">{user.name}</div>
          <div className="text-surface-600-400 text-xs truncate">{user.email}</div>
        </div>
        <ChevronsUpDown className="text-surface-500 size-4 shrink-0" />
      </Menu.Trigger>
      <Portal>
        <Menu.Positioner>
          <Menu.Content>
            <MenuItemLink to="/settings" value="settings">
              <Menu.ItemText>Settings</Menu.ItemText>
            </MenuItemLink>
            {user.isOwner && (
              <MenuItemLink to="/admin" value="admin">
                <Menu.ItemText>Admin</Menu.ItemText>
              </MenuItemLink>
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

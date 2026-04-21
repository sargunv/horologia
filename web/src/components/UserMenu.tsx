import { useQueryClient } from "@tanstack/react-query";
import { createLink, useNavigate } from "@tanstack/react-router";
import { ChevronsUpDown } from "lucide-react";
import { appClient } from "../api/client.ts";
import type { components } from "../api/schema.d.ts";
import {
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRoot,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../ui/DropdownMenu.tsx";

const MenuItemLink = createLink(DropdownMenuItem);

type User = components["schemas"]["User"];

export function UserMenu({ user }: { user: User }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  async function handleLogout() {
    try {
      await appClient.POST("/app/auth/logout");
    } catch {
      // Proceed with local logout regardless of network failure
    }
    queryClient.clear();
    void navigate({ to: "/login" });
  }

  return (
    <DropdownMenuRoot>
      <DropdownMenuTrigger className="btn btn-soft w-full justify-start gap-2.5">
        <div className="min-w-0 flex-1 text-left">
          <div className="truncate text-sm font-medium">{user.name}</div>
          <div className="truncate text-xs text-base-content/60">{user.email}</div>
        </div>
        <ChevronsUpDown className="size-4 shrink-0 text-base-content/60" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-56">
        <MenuItemLink to="/settings">Settings</MenuItemLink>
        {user.isOwner && <MenuItemLink to="/admin">Admin</MenuItemLink>}
        <DropdownMenuSeparator />
        <DropdownMenuItem onSelect={handleLogout}>Sign out</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenuRoot>
  );
}

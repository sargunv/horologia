import { queryOptions } from "@tanstack/react-query";
import { apiClient } from "../api/client.ts";

export function meQueryOptions() {
  return queryOptions({
    queryKey: ["me"],
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/users/me");
      if (error) throw error;
      return data;
    },
  });
}

async function apiFetch(path: string, init?: RequestInit): Promise<Response> {
  const res = await fetch(`/api${path}`, { credentials: "include", ...init });
  if (res.status === 401) {
    window.dispatchEvent(new CustomEvent("tend:unauthorized"));
  }
  return res;
}

export async function webLogin(email: string, password: string) {
  const res = await apiFetch("/auth/web-login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) {
    const body = (await res.json()) as { message: string };
    throw new Error(body.message);
  }
  return (await res.json()) as { user: { id: string; email: string; name: string } };
}

export async function webLogout() {
  await apiFetch("/auth/web-logout", { method: "POST" });
}

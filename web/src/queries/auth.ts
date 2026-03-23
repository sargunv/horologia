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

export async function webLogin(email: string, password: string) {
  const res = await fetch("/api/auth/web-login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) {
    const body = (await res.json()) as { message: string };
    throw new Error(body.message);
  }
  return (await res.json()) as { user: { id: string; email: string; name: string } };
}

export async function webLogout() {
  await fetch("/api/auth/web-logout", {
    method: "POST",
    credentials: "include",
  });
}

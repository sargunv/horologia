import createClient from "openapi-fetch";
import type { paths } from "./schema.d.ts";

export const apiClient = createClient<paths>({
  baseUrl: "/api",
  credentials: "include",
});

export const appClient = createClient<paths>({
  baseUrl: "",
  credentials: "include",
});

for (const client of [apiClient, appClient]) {
  client.use({
    onResponse({ response }) {
      if (response.status === 401) {
        window.dispatchEvent(new CustomEvent("horologia:unauthorized"));
      }
      return response;
    },
  });
}

export function getApiErrorMessage(error: unknown, fallback: string): string {
  if (
    typeof error === "object" &&
    error !== null &&
    "message" in error &&
    typeof error.message === "string" &&
    error.message.length > 0
  ) {
    return error.message;
  }
  return fallback;
}

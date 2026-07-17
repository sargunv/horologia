import createClient, { type Client } from "openapi-fetch";

import type { paths } from "./schema.d.ts";

export type HorologiaClient = Client<paths>;

export interface ClientOptions {
  baseUrl: string;
  getAccessToken?: () => Promise<string | null>;
  onUnauthorized?: () => void;
  credentials?: RequestCredentials;
  fetch?: typeof globalThis.fetch;
}

export function createHorologiaClient({
  baseUrl,
  getAccessToken,
  onUnauthorized,
  credentials,
  fetch,
}: ClientOptions): HorologiaClient {
  const client = createClient<paths>({
    baseUrl,
    ...(credentials ? { credentials } : {}),
    ...(fetch ? { fetch } : {}),
  });

  if (getAccessToken) {
    client.use({
      async onRequest({ request }) {
        const accessToken = await getAccessToken();
        if (accessToken) request.headers.set("Authorization", `Bearer ${accessToken}`);
        return request;
      },
    });
  }

  if (onUnauthorized) {
    client.use({
      onResponse({ response }) {
        if (response.status === 401) onUnauthorized();
        return response;
      },
    });
  }

  return client;
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

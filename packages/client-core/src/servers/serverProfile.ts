export interface ServerProfile {
  id: string;
  baseUrl: string;
  displayName: string;
  lastUsedAt: string;
}

export function normalizeServerUrl(input: string): string {
  const trimmed = input.trim();
  if (!trimmed) throw new Error("Enter a server URL");

  const candidate = /^[a-z][a-z\d+.-]*:\/\//iu.test(trimmed) ? trimmed : `https://${trimmed}`;
  const url = new URL(candidate);
  if (url.protocol !== "https:" && url.protocol !== "http:") {
    throw new Error("Server URL must use HTTP or HTTPS");
  }
  if (url.username || url.password) throw new Error("Server URL must not include credentials");
  url.hash = "";
  url.search = "";
  url.pathname = url.pathname.replace(/\/+$/u, "") || "/";
  return url.toString().replace(/\/$/u, "");
}

export function createServerProfile(input: {
  id: string;
  baseUrl: string;
  displayName?: string;
  now: string;
}): ServerProfile {
  const baseUrl = normalizeServerUrl(input.baseUrl);
  const url = new URL(baseUrl);
  return {
    id: input.id,
    baseUrl,
    displayName: input.displayName?.trim() || url.host,
    lastUsedAt: input.now,
  };
}

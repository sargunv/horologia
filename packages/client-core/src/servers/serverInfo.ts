import type { HorologiaClient } from "../api/client";

export const SUPPORTED_API_VERSION = 1;
export const REQUIRED_SERVER_CAPABILITIES = ["oauth-2.1-pkce"] as const;

export interface CompatibleServerInfo {
  apiVersion: 1;
  capabilities: string[];
}

export async function fetchCompatibleServerInfo(
  client: HorologiaClient,
): Promise<CompatibleServerInfo> {
  const { data, error, response } = await client.GET("/server-info");
  if (!response.ok || error || !data) {
    throw new Error("Horologia could not reach that server");
  }
  if (data.apiVersion !== SUPPORTED_API_VERSION) {
    throw new Error(`This app does not support server API version ${String(data.apiVersion)}`);
  }
  const missing = REQUIRED_SERVER_CAPABILITIES.filter(
    (capability) => !data.capabilities.includes(capability),
  );
  if (missing.length > 0) {
    throw new Error(`Server is missing required capability: ${missing.join(", ")}`);
  }
  return data;
}

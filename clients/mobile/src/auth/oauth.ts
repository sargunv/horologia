import type { ServerProfile } from "@horologia/client-core/servers";
import {
  AuthRequest,
  exchangeCodeAsync,
  makeRedirectUri,
  type TokenResponse,
} from "expo-auth-session";
import * as SecureStore from "expo-secure-store";
import * as WebBrowser from "expo-web-browser";

WebBrowser.maybeCompleteAuthSession();

export const MOBILE_OAUTH_SCOPES = [
  "profile:read",
  "recipes:read",
  "spaces:read",
  "tasks:read",
] as const;

const redirectUri = makeRedirectUri({
  scheme: "horologia",
  path: "oauth/callback",
});
const pendingAuthorizationKeyPrefix = "oauth.pending";

type OAuthServerProfile = Pick<ServerProfile, "id" | "baseUrl">;

type PendingAuthorization = {
  codeVerifier: string;
  serverId: string;
  serverBaseUrl: string;
  state: string;
};

function pendingAuthorizationKey(serverId: string): string {
  return `${pendingAuthorizationKeyPrefix}.${serverId}`;
}

function resolveServerUrl(serverBaseUrl: string, path: string): string {
  return new URL(path, `${serverBaseUrl.replace(/\/+$/, "")}/`).toString();
}

export async function authorizeMobile(profile: OAuthServerProfile): Promise<TokenResponse> {
  const discovery = createDiscovery(profile.baseUrl);
  const request = new AuthRequest({
    clientId: "horologia-mobile",
    redirectUri,
    scopes: [...MOBILE_OAUTH_SCOPES],
    usePKCE: true,
    extraParams: {
      resource: resolveServerUrl(profile.baseUrl, "api"),
    },
  });
  const authorizationUrl = await request.makeAuthUrlAsync(discovery);
  if (!request.codeVerifier) {
    throw new Error("Authorization request did not create a PKCE verifier");
  }
  const storageKey = pendingAuthorizationKey(profile.id);
  await SecureStore.setItemAsync(
    storageKey,
    JSON.stringify({
      codeVerifier: request.codeVerifier,
      serverId: profile.id,
      serverBaseUrl: profile.baseUrl,
      state: request.state,
    } satisfies PendingAuthorization),
  );

  let result: Awaited<ReturnType<AuthRequest["promptAsync"]>>;
  try {
    result = await request.promptAsync(discovery, { url: authorizationUrl });
  } catch (error) {
    await SecureStore.deleteItemAsync(storageKey);
    throw error;
  }
  if (result.type !== "success") {
    await SecureStore.deleteItemAsync(storageKey);
    throw new Error(
      result.type === "dismiss"
        ? "Sign-in was cancelled before it finished."
        : `Authorization did not complete: ${result.type}`,
    );
  }

  const code = result.params["code"];
  if (!code) {
    await SecureStore.deleteItemAsync(storageKey);
    throw new Error("Authorization response did not include a usable code");
  }

  return completeMobileAuthorization(profile.id, code, result.params["state"]);
}

export async function completeMobileAuthorization(
  serverId: string,
  code: string,
  returnedState: string | undefined,
): Promise<TokenResponse> {
  const storageKey = pendingAuthorizationKey(serverId);
  const serialized = await SecureStore.getItemAsync(storageKey);
  if (!serialized) {
    throw new Error("The authorization request is no longer available for this server");
  }
  await SecureStore.deleteItemAsync(storageKey);
  const pending: unknown = JSON.parse(serialized);
  if (!isPendingAuthorization(pending)) {
    throw new Error("The pending authorization request is invalid");
  }
  if (pending.serverId !== serverId) {
    throw new Error("The authorization request does not belong to the selected server");
  }
  if (!returnedState || returnedState !== pending.state) {
    throw new Error("Authorization state did not match the pending request");
  }

  const discovery = createDiscovery(pending.serverBaseUrl);
  const response = await exchangeCodeAsync(
    {
      clientId: "horologia-mobile",
      code,
      redirectUri,
      extraParams: {
        code_verifier: pending.codeVerifier,
      },
    },
    discovery,
  );
  return response;
}

export async function cancelMobileAuthorization(serverId: string): Promise<void> {
  try {
    WebBrowser.dismissAuthSession();
  } catch {
    // Android custom tabs cannot always be dismissed programmatically.
  }
  await SecureStore.deleteItemAsync(pendingAuthorizationKey(serverId));
}

export async function revokeMobileToken(
  serverBaseUrl: string,
  token: string,
  tokenTypeHint: "access_token" | "refresh_token",
): Promise<void> {
  const response = await fetch(resolveServerUrl(serverBaseUrl, "oauth/revoke"), {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      client_id: "horologia-mobile",
      token,
      token_type_hint: tokenTypeHint,
    }).toString(),
  });
  if (!response.ok) throw new Error("Token revocation failed");
}

function createDiscovery(serverBaseUrl: string) {
  return {
    authorizationEndpoint: resolveServerUrl(serverBaseUrl, "oauth/authorize"),
    tokenEndpoint: resolveServerUrl(serverBaseUrl, "oauth/token"),
    revocationEndpoint: resolveServerUrl(serverBaseUrl, "oauth/revoke"),
  };
}

function isPendingAuthorization(value: unknown): value is PendingAuthorization {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  return (
    "codeVerifier" in value &&
    typeof value.codeVerifier === "string" &&
    "serverId" in value &&
    typeof value.serverId === "string" &&
    "serverBaseUrl" in value &&
    typeof value.serverBaseUrl === "string" &&
    "state" in value &&
    typeof value.state === "string"
  );
}

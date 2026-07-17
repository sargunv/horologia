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
  "activity:read",
  "profile:read",
  "recipes:read",
  "recipes:write",
  "spaces:read",
  "spaces:write",
  "tags:read",
  "tags:write",
  "tasks:read",
  "tasks:write",
  "users:read",
  "users:write",
] as const;

const redirectUri = makeRedirectUri({
  scheme: "horologia",
  path: "oauth/callback",
});
const pendingAuthorizationKey = "oauth.pending";

type PendingAuthorization = {
  codeVerifier: string;
  serverBaseUrl: string;
  state: string;
};

function resolveServerUrl(serverBaseUrl: string, path: string): string {
  return new URL(path, `${serverBaseUrl.replace(/\/+$/, "")}/`).toString();
}

export async function authorizeMobile(serverBaseUrl: string): Promise<TokenResponse> {
  const discovery = createDiscovery(serverBaseUrl);
  const request = new AuthRequest({
    clientId: "horologia-mobile",
    redirectUri,
    scopes: [...MOBILE_OAUTH_SCOPES],
    usePKCE: true,
    extraParams: {
      resource: resolveServerUrl(serverBaseUrl, "api"),
    },
  });
  const authorizationUrl = await request.makeAuthUrlAsync(discovery);
  if (!request.codeVerifier) {
    throw new Error("Authorization request did not create a PKCE verifier");
  }
  await SecureStore.setItemAsync(
    pendingAuthorizationKey,
    JSON.stringify({
      codeVerifier: request.codeVerifier,
      serverBaseUrl,
      state: request.state,
    } satisfies PendingAuthorization),
  );

  const result = await request.promptAsync(discovery, { url: authorizationUrl });
  if (result.type !== "success") {
    throw new Error(`Authorization did not complete: ${result.type}`);
  }

  const code = result.params["code"];
  if (!code) {
    throw new Error("Authorization response did not include a usable code");
  }

  return completeMobileAuthorization(code, result.params["state"]);
}

export async function completeMobileAuthorization(
  code: string,
  returnedState: string | undefined,
): Promise<TokenResponse> {
  const serialized = await SecureStore.getItemAsync(pendingAuthorizationKey);
  if (!serialized) {
    throw new Error("The authorization request is no longer available");
  }
  const pending: unknown = JSON.parse(serialized);
  if (!isPendingAuthorization(pending)) {
    throw new Error("The pending authorization request is invalid");
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
  await SecureStore.deleteItemAsync(pendingAuthorizationKey);
  return response;
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
    "serverBaseUrl" in value &&
    typeof value.serverBaseUrl === "string" &&
    "state" in value &&
    typeof value.state === "string"
  );
}

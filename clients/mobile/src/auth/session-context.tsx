import {
  createHorologiaClient,
  createRefreshCoordinator,
  fetchCompatibleServerInfo,
  normalizeServerUrl,
  type HorologiaClient,
  type OAuthCredentials,
  type ServerProfile,
} from "@horologia/client-core";
import type { TokenResponse } from "expo-auth-session";
import {
  createContext,
  type PropsWithChildren,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import {
  authorizeMobile,
  cancelMobileAuthorization,
  MOBILE_OAUTH_SCOPES,
  revokeMobileToken,
} from "@/auth/oauth";
import {
  attachActiveAccount,
  clearCachedMyTasks,
  detachActiveAccount,
  loadActiveAccount,
  saveActiveServer,
} from "@/persistence/database";
import {
  deleteCredentials,
  getCredentials,
  setCredentials,
  setCredentialsWhileCurrent,
} from "@/persistence/credentials";
import { clearWidgetSnapshot } from "@/widgets/publishWidgetSnapshot";

type SessionSnapshot =
  | {
      status: "signed-in";
      detail: null;
      profile: ServerProfile;
      accountId: string;
      client: HorologiaClient;
    }
  | {
      status:
        | "restoring"
        | "signed-out"
        | "connecting"
        | "authorizing"
        | "signing-in"
        | "signing-out"
        | "error";
      detail: string | null;
      profile: ServerProfile | null;
      accountId: null;
      client: null;
    };

interface SessionActions {
  connect(serverUrl: string): Promise<void>;
  signIn(): Promise<void>;
  cancelAuthorization(): Promise<void>;
  signOut(): Promise<void>;
  recover(): Promise<void>;
}

type SessionValue = SessionSnapshot & SessionActions;

const SessionContext = createContext<SessionValue | null>(null);

export function SessionProvider({ children }: PropsWithChildren) {
  const [snapshot, setSnapshot] = useState<SessionSnapshot>({
    status: "restoring",
    detail: null,
    profile: null,
    accountId: null,
    client: null,
  });
  const sessionGeneration = useRef(0);
  const authorizationGeneration = useRef<number | null>(null);

  async function expireSession(
    key: { serverId: string; accountId: string },
    generation: number,
    profile: ServerProfile,
    detail: string,
  ) {
    if (sessionGeneration.current !== generation) return;
    authorizationGeneration.current = null;
    const cleanupGeneration = generation + 1;
    sessionGeneration.current = cleanupGeneration;
    setSnapshot({
      status: "signing-out",
      detail,
      profile,
      accountId: null,
      client: null,
    });
    const cleanup = await Promise.allSettled([
      deleteCredentials(key),
      detachActiveAccount(key.serverId, key.accountId),
      clearCachedMyTasks(key.serverId, key.accountId),
      clearWidgetSnapshot(),
    ]);
    if (sessionGeneration.current !== cleanupGeneration) return;
    const cleanupFailed = cleanup.some((result) => result.status === "rejected");
    setSnapshot(
      signedOut(profile, cleanupFailed ? `${detail} Secure local cleanup also failed.` : detail),
    );
  }

  async function installSession(
    nextProfile: ServerProfile,
    nextAccountId: string,
    initial: OAuthCredentials,
    generation: number,
  ): Promise<boolean> {
    if (sessionGeneration.current !== generation) return false;
    const key = { serverId: nextProfile.id, accountId: nextAccountId };
    const coordinator = createRefreshCoordinator(initial, {
      async refresh(current) {
        try {
          return await refreshCredentials(nextProfile.baseUrl, current.refreshToken);
        } catch (error) {
          await expireSession(
            key,
            generation,
            nextProfile,
            message(error, "Your session expired. Sign in again."),
          );
          throw error;
        }
      },
      async persist(next) {
        const persisted = await setCredentialsWhileCurrent(
          key,
          next,
          () => sessionGeneration.current === generation,
        );
        if (persisted) return;
        await revokeCredentials(nextProfile.baseUrl, next);
        throw new Error("The session changed while credentials were refreshing");
      },
    });
    try {
      await coordinator.getAccessToken();
    } catch (error) {
      if (sessionGeneration.current !== generation) return false;
      throw error;
    }
    if (sessionGeneration.current !== generation) return false;
    const nextClient = createHorologiaClient({
      baseUrl: apiBaseUrl(nextProfile.baseUrl),
      getAccessToken: () => coordinator.getAccessToken(),
      onUnauthorized: () => {
        void expireSession(key, generation, nextProfile, "Your session expired. Sign in again.");
      },
    });
    setSnapshot({
      status: "signed-in",
      detail: null,
      profile: nextProfile,
      accountId: nextAccountId,
      client: nextClient,
    });
    return true;
  }

  async function restore() {
    const generation = sessionGeneration.current + 1;
    sessionGeneration.current = generation;
    authorizationGeneration.current = null;
    let restoringProfile: ServerProfile | null = null;
    setSnapshot({
      status: "restoring",
      detail: null,
      profile: null,
      accountId: null,
      client: null,
    });
    try {
      const active = await loadActiveAccount();
      if (sessionGeneration.current !== generation) return;
      restoringProfile = active?.profile ?? null;
      if (!active?.accountId) {
        setSnapshot(signedOut(restoringProfile));
        return;
      }
      const credentials = await getCredentials({
        serverId: active.profile.id,
        accountId: active.accountId,
      });
      if (sessionGeneration.current !== generation) return;
      if (!credentials) {
        const cleanup = await Promise.allSettled([
          detachActiveAccount(active.profile.id, active.accountId),
          clearCachedMyTasks(active.profile.id, active.accountId),
          clearWidgetSnapshot(),
        ]);
        if (sessionGeneration.current !== generation) return;
        const detail = cleanup.some((result) => result.status === "rejected")
          ? "Your secure session is no longer available, and some saved data could not be cleared. Sign in again."
          : "Your secure session is no longer available. Sign in again.";
        setSnapshot(signedOut(active.profile, detail));
        return;
      }
      await installSession(active.profile, active.accountId, credentials, generation);
    } catch (error) {
      if (sessionGeneration.current !== generation) return;
      setSnapshot({
        status: "error",
        detail: message(error, "Could not restore the session"),
        profile: restoringProfile,
        accountId: null,
        client: null,
      });
    }
  }

  useEffect(() => {
    void restore();
  }, []);

  const value = useMemo<SessionValue>(
    () => ({
      ...snapshot,
      async connect(serverUrl) {
        if (authorizationGeneration.current !== null) return;
        const generation = sessionGeneration.current + 1;
        sessionGeneration.current = generation;
        authorizationGeneration.current = null;
        const previousProfile = snapshot.profile;
        setSnapshot({
          status: "connecting",
          detail: "Checking server compatibility…",
          profile: previousProfile,
          accountId: null,
          client: null,
        });
        try {
          const normalizedBaseUrl = normalizeServerUrl(serverUrl);
          const publicClient = createHorologiaClient({ baseUrl: apiBaseUrl(normalizedBaseUrl) });
          await fetchCompatibleServerInfo(publicClient);
          if (sessionGeneration.current !== generation) return;
          const candidate = await saveActiveServer(normalizedBaseUrl);
          if (sessionGeneration.current === generation) setSnapshot(signedOut(candidate));
        } catch (error) {
          if (sessionGeneration.current !== generation) return;
          setSnapshot({
            status: "error",
            detail: message(error, "Could not connect to that server"),
            profile: previousProfile,
            accountId: null,
            client: null,
          });
        }
      },
      async signIn() {
        if (authorizationGeneration.current !== null) return;
        const profile = snapshot.profile;
        if (!profile) return;
        const profileBaseUrl = profile.baseUrl;
        const generation = sessionGeneration.current + 1;
        sessionGeneration.current = generation;
        authorizationGeneration.current = generation;
        let tokenResponse: TokenResponse | null = null;
        let credentialKey: { serverId: string; accountId: string } | null = null;
        let credentialsStored = false;
        let accountAttached = false;

        async function cleanupAbandonedSignIn() {
          const cleanup: Promise<unknown>[] = [];
          if (credentialsStored && credentialKey) cleanup.push(deleteCredentials(credentialKey));
          if (accountAttached && credentialKey) {
            cleanup.push(detachActiveAccount(credentialKey.serverId, credentialKey.accountId));
          }
          if (tokenResponse) cleanup.push(revokeTokenResponse(profileBaseUrl, tokenResponse));
          await Promise.allSettled(cleanup);
        }

        setSnapshot({
          status: "authorizing",
          detail: "Complete sign-in in the secure browser…",
          profile,
          accountId: null,
          client: null,
        });
        try {
          tokenResponse = await authorizeMobile(profile);
          if (authorizationGeneration.current === generation) {
            authorizationGeneration.current = null;
          }
          if (sessionGeneration.current !== generation) {
            await cleanupAbandonedSignIn();
            return;
          }
          setSnapshot({
            status: "signing-in",
            detail: "Verifying your account and securing the session…",
            profile,
            accountId: null,
            client: null,
          });
          const initial = credentialsFromTokenResponse(tokenResponse);
          const bootstrapClient = createHorologiaClient({
            baseUrl: apiBaseUrl(profile.baseUrl),
            getAccessToken: async () => initial.accessToken,
          });
          const { data, error, response: meResponse } = await bootstrapClient.GET("/users/me");
          if (!meResponse.ok || error || !data) throw new Error("Could not load your account");
          if (sessionGeneration.current !== generation) {
            await cleanupAbandonedSignIn();
            return;
          }
          credentialKey = { serverId: profile.id, accountId: data.id };
          await setCredentials(credentialKey, initial);
          credentialsStored = true;
          if (sessionGeneration.current !== generation) {
            await cleanupAbandonedSignIn();
            return;
          }
          await attachActiveAccount(profile.id, data.id);
          accountAttached = true;
          if (sessionGeneration.current !== generation) {
            await cleanupAbandonedSignIn();
            return;
          }
          const installed = await installSession(profile, data.id, initial, generation);
          if (!installed && sessionGeneration.current !== generation) {
            await cleanupAbandonedSignIn();
          }
        } catch (error) {
          if (authorizationGeneration.current === generation) {
            authorizationGeneration.current = null;
          }
          await cleanupAbandonedSignIn();
          if (sessionGeneration.current !== generation) return;
          setSnapshot({
            status: "error",
            detail: message(error, "Sign-in failed"),
            profile,
            accountId: null,
            client: null,
          });
        }
      },
      async cancelAuthorization() {
        const generation = authorizationGeneration.current;
        if (generation === null || sessionGeneration.current !== generation) return;
        sessionGeneration.current = generation + 1;
        const profile = snapshot.profile;
        setSnapshot(signedOut(profile, "Sign-in was cancelled."));
        if (!profile) return;
        try {
          await cancelMobileAuthorization(profile.id);
        } catch {
          if (sessionGeneration.current === generation + 1) {
            setSnapshot(
              signedOut(
                profile,
                "Sign-in was cancelled, but its temporary state could not be cleared.",
              ),
            );
          }
        }
      },
      async signOut() {
        const generation = sessionGeneration.current + 1;
        sessionGeneration.current = generation;
        authorizationGeneration.current = null;
        const currentProfile = snapshot.profile;
        const currentAccountId = snapshot.accountId;
        setSnapshot({
          status: "signing-out",
          detail: "Removing secure credentials and saved data…",
          profile: currentProfile,
          accountId: null,
          client: null,
        });
        const localCleanup: Promise<unknown>[] = [clearWidgetSnapshot()];
        let credentialReadFailed = false;
        if (currentProfile && currentAccountId) {
          const key = { serverId: currentProfile.id, accountId: currentAccountId };
          let credentials: OAuthCredentials | null = null;
          try {
            credentials = await getCredentials(key);
          } catch {
            credentialReadFailed = true;
          }
          if (credentials) void revokeCredentials(currentProfile.baseUrl, credentials);
          localCleanup.push(
            detachActiveAccount(currentProfile.id, currentAccountId),
            deleteCredentials(key),
            clearCachedMyTasks(currentProfile.id, currentAccountId),
          );
        }
        const results = await Promise.allSettled(localCleanup);
        if (sessionGeneration.current !== generation) return;
        const cleanupFailed =
          credentialReadFailed || results.some((result) => result.status === "rejected");
        setSnapshot(
          signedOut(
            currentProfile,
            cleanupFailed ? "Signed out, but some local data could not be cleared." : null,
          ),
        );
      },
      recover: restore,
    }),
    [snapshot],
  );

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

function signedOut(profile: ServerProfile | null, detail: string | null = null): SessionSnapshot {
  return { status: "signed-out", detail, profile, accountId: null, client: null };
}

export function useSession(): SessionValue {
  const value = useContext(SessionContext);
  if (!value) throw new Error("useSession must be used within SessionProvider");
  return value;
}

function apiBaseUrl(serverBaseUrl: string): string {
  return new URL("api/", `${serverBaseUrl.replace(/\/+$/u, "")}/`).toString();
}

async function revokeTokenResponse(serverBaseUrl: string, response: TokenResponse): Promise<void> {
  const revocations = [revokeMobileToken(serverBaseUrl, response.accessToken, "access_token")];
  if (response.refreshToken) {
    revocations.push(revokeMobileToken(serverBaseUrl, response.refreshToken, "refresh_token"));
  }
  await Promise.allSettled(revocations);
}

async function revokeCredentials(
  serverBaseUrl: string,
  credentials: OAuthCredentials,
): Promise<void> {
  await Promise.allSettled([
    revokeMobileToken(serverBaseUrl, credentials.accessToken, "access_token"),
    revokeMobileToken(serverBaseUrl, credentials.refreshToken, "refresh_token"),
  ]);
}

function credentialsFromTokenResponse(response: TokenResponse): OAuthCredentials {
  if (!response.refreshToken) throw new Error("Server did not issue a refresh token");
  const scope = response.scope ?? "";
  assertRequiredScopes(scope);
  return {
    accessToken: response.accessToken,
    refreshToken: response.refreshToken,
    expiresAt: new Date(Date.now() + (response.expiresIn ?? 900) * 1000).toISOString(),
    scope,
  };
}

function assertRequiredScopes(scope: string): void {
  const granted = new Set(scope.split(/\s+/u).filter(Boolean));
  const missing = MOBILE_OAUTH_SCOPES.filter((required) => !granted.has(required));
  if (missing.length > 0) {
    throw new Error(`The server did not grant the required access: ${missing.join(", ")}`);
  }
}

async function refreshCredentials(
  baseUrl: string,
  refreshToken: string,
): Promise<OAuthCredentials> {
  const response = await fetch(new URL("oauth/token", `${baseUrl.replace(/\/+$/u, "")}/`), {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      grant_type: "refresh_token",
      client_id: "horologia-mobile",
      refresh_token: refreshToken,
    }).toString(),
  });
  if (!response.ok) throw new Error("Your session expired. Sign in again.");
  const body: unknown = await response.json();
  if (!isTokenResponseBody(body)) throw new Error("Server returned an invalid refresh response");
  assertRequiredScopes(body.scope);
  return {
    accessToken: body.access_token,
    refreshToken: body.refresh_token,
    expiresAt: new Date(Date.now() + body.expires_in * 1000).toISOString(),
    scope: body.scope,
  };
}

function isTokenResponseBody(value: unknown): value is {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  scope: string;
} {
  return (
    typeof value === "object" &&
    value !== null &&
    "access_token" in value &&
    typeof value.access_token === "string" &&
    "refresh_token" in value &&
    typeof value.refresh_token === "string" &&
    "expires_in" in value &&
    typeof value.expires_in === "number" &&
    "scope" in value &&
    typeof value.scope === "string"
  );
}

function message(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback;
}

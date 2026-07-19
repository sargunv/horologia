import {
  createHorologiaClient,
  createRefreshCoordinator,
  fetchCompatibleServerInfo,
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

import { authorizeMobile, revokeMobileToken } from "@/auth/oauth";
import {
  attachActiveAccount,
  clearCachedMyTasks,
  clearActiveAccount,
  loadActiveAccount,
  saveActiveServer,
} from "@/persistence/database";
import { deleteCredentials, getCredentials, setCredentials } from "@/persistence/credentials";
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
      status: "restoring" | "signed-out" | "authorizing" | "error";
      detail: string | null;
      profile: ServerProfile | null;
      accountId: null;
      client: null;
    };

interface SessionActions {
  connect(serverUrl: string): Promise<void>;
  signIn(): Promise<void>;
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

  async function expireSession(
    key: { serverId: string; accountId: string },
    generation: number,
    profile: ServerProfile,
    detail: string,
  ) {
    if (sessionGeneration.current !== generation) return;
    sessionGeneration.current += 1;
    setSnapshot(signedOut(profile, detail));
    try {
      await deleteCredentials(key);
    } catch {
      setSnapshot((current) =>
        current.status === "signed-out" && current.profile?.id === profile.id
          ? signedOut(profile, `${detail} Secure credential cleanup also failed.`)
          : current,
      );
    }
  }

  async function installSession(
    nextProfile: ServerProfile,
    nextAccountId: string,
    initial: OAuthCredentials,
  ) {
    const key = { serverId: nextProfile.id, accountId: nextAccountId };
    const generation = sessionGeneration.current + 1;
    sessionGeneration.current = generation;
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
      persist: (next) => setCredentials(key, next),
    });
    try {
      await coordinator.getAccessToken();
    } catch {
      return;
    }
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
  }

  async function restore() {
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
      restoringProfile = active?.profile ?? null;
      if (!active?.accountId) {
        setSnapshot(signedOut(restoringProfile));
        return;
      }
      const credentials = await getCredentials({
        serverId: active.profile.id,
        accountId: active.accountId,
      });
      if (!credentials) {
        setSnapshot(
          signedOut(active.profile, "Your secure session is no longer available. Sign in again."),
        );
        return;
      }
      await installSession(active.profile, active.accountId, credentials);
    } catch (error) {
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
        setSnapshot({
          status: "authorizing",
          detail: "Checking server compatibility…",
          profile: snapshot.profile,
          accountId: null,
          client: null,
        });
        try {
          const candidate = await saveActiveServer(serverUrl);
          const publicClient = createHorologiaClient({ baseUrl: apiBaseUrl(candidate.baseUrl) });
          await fetchCompatibleServerInfo(publicClient);
          setSnapshot(signedOut(candidate));
        } catch (error) {
          await clearActiveAccount();
          setSnapshot({
            status: "error",
            detail: message(error, "Could not connect to that server"),
            profile: null,
            accountId: null,
            client: null,
          });
        }
      },
      async signIn() {
        const profile = snapshot.profile;
        if (!profile) return;
        setSnapshot({
          status: "authorizing",
          detail: "Complete sign-in in the secure browser…",
          profile,
          accountId: null,
          client: null,
        });
        try {
          const response = await authorizeMobile(profile.baseUrl);
          const initial = credentialsFromTokenResponse(response);
          const bootstrapClient = createHorologiaClient({
            baseUrl: apiBaseUrl(profile.baseUrl),
            getAccessToken: async () => initial.accessToken,
          });
          const { data, error, response: meResponse } = await bootstrapClient.GET("/users/me");
          if (!meResponse.ok || error || !data) throw new Error("Could not load your account");
          const key = { serverId: profile.id, accountId: data.id };
          await setCredentials(key, initial);
          await attachActiveAccount(profile.id, data.id);
          await installSession(profile, data.id, initial);
        } catch (error) {
          setSnapshot({
            status: "error",
            detail: message(error, "Sign-in failed"),
            profile,
            accountId: null,
            client: null,
          });
        }
      },
      async signOut() {
        const generation = sessionGeneration.current + 1;
        sessionGeneration.current = generation;
        const currentProfile = snapshot.profile;
        const currentAccountId = snapshot.accountId;
        setSnapshot(signedOut(null));
        const localCleanup: Promise<unknown>[] = [clearActiveAccount(), clearWidgetSnapshot()];
        let credentialReadFailed = false;
        if (currentProfile && currentAccountId) {
          const key = { serverId: currentProfile.id, accountId: currentAccountId };
          let credentials: OAuthCredentials | null = null;
          try {
            credentials = await getCredentials(key);
          } catch {
            credentialReadFailed = true;
          }
          if (credentials) {
            void Promise.allSettled([
              revokeMobileToken(currentProfile.baseUrl, credentials.accessToken, "access_token"),
              revokeMobileToken(currentProfile.baseUrl, credentials.refreshToken, "refresh_token"),
            ]);
          }
          localCleanup.push(
            deleteCredentials(key),
            clearCachedMyTasks(currentProfile.id, currentAccountId),
          );
        }
        const results = await Promise.allSettled(localCleanup);
        if (credentialReadFailed || results.some((result) => result.status === "rejected")) {
          setSnapshot((current) =>
            sessionGeneration.current === generation && current.status === "signed-out"
              ? signedOut(null, "Signed out, but some local data could not be cleared.")
              : current,
          );
        }
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

function credentialsFromTokenResponse(response: TokenResponse): OAuthCredentials {
  if (!response.refreshToken) throw new Error("Server did not issue a refresh token");
  return {
    accessToken: response.accessToken,
    refreshToken: response.refreshToken,
    expiresAt: new Date(Date.now() + (response.expiresIn ?? 900) * 1000).toISOString(),
    scope: response.scope ?? "",
  };
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

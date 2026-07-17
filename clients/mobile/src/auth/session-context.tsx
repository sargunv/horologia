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
  useState,
} from "react";

import { authorizeMobile, revokeMobileToken } from "@/auth/oauth";
import {
  attachActiveAccount,
  clearActiveAccount,
  loadActiveAccount,
  saveActiveServer,
} from "@/persistence/database";
import { deleteCredentials, getCredentials, setCredentials } from "@/persistence/credentials";
import { clearWidgetSnapshot } from "@/widgets/publishWidgetSnapshot";

type SessionStatus = "restoring" | "signed-out" | "authorizing" | "signed-in" | "error";

interface SessionValue {
  status: SessionStatus;
  detail: string | null;
  profile: ServerProfile | null;
  accountId: string | null;
  client: HorologiaClient | null;
  connect(serverUrl: string): Promise<void>;
  signIn(): Promise<void>;
  signOut(): Promise<void>;
  recover(): Promise<void>;
}

const SessionContext = createContext<SessionValue | null>(null);

export function SessionProvider({ children }: PropsWithChildren) {
  const [status, setStatus] = useState<SessionStatus>("restoring");
  const [detail, setDetail] = useState<string | null>(null);
  const [profile, setProfile] = useState<ServerProfile | null>(null);
  const [accountId, setAccountId] = useState<string | null>(null);
  const [client, setClient] = useState<HorologiaClient | null>(null);

  async function installSession(
    nextProfile: ServerProfile,
    nextAccountId: string,
    initial: OAuthCredentials,
  ) {
    const key = { serverId: nextProfile.id, accountId: nextAccountId };
    const coordinator = createRefreshCoordinator(initial, {
      refresh: (current) => refreshCredentials(nextProfile.baseUrl, current.refreshToken),
      persist: (next) => setCredentials(key, next),
    });
    const nextClient = createHorologiaClient({
      baseUrl: apiBaseUrl(nextProfile.baseUrl),
      getAccessToken: () => coordinator.getAccessToken(),
      onUnauthorized: () => setStatus("error"),
    });
    setProfile(nextProfile);
    setAccountId(nextAccountId);
    setClient(nextClient);
    setStatus("signed-in");
    setDetail(null);
  }

  async function restore() {
    setStatus("restoring");
    try {
      const active = await loadActiveAccount();
      if (!active?.accountId) {
        setProfile(active?.profile ?? null);
        setStatus("signed-out");
        return;
      }
      const credentials = await getCredentials({
        serverId: active.profile.id,
        accountId: active.accountId,
      });
      if (!credentials) {
        setProfile(active.profile);
        setStatus("signed-out");
        setDetail("Your secure session is no longer available. Sign in again.");
        return;
      }
      await installSession(active.profile, active.accountId, credentials);
    } catch (error) {
      setStatus("error");
      setDetail(message(error, "Could not restore the session"));
    }
  }

  useEffect(() => {
    void restore();
  }, []);

  const value = useMemo<SessionValue>(
    () => ({
      status,
      detail,
      profile,
      accountId,
      client,
      async connect(serverUrl) {
        setStatus("authorizing");
        setDetail("Checking server compatibility…");
        try {
          const candidate = await saveActiveServer(serverUrl);
          const publicClient = createHorologiaClient({ baseUrl: apiBaseUrl(candidate.baseUrl) });
          await fetchCompatibleServerInfo(publicClient);
          setProfile(candidate);
          setStatus("signed-out");
          setDetail(null);
        } catch (error) {
          await clearActiveAccount();
          setProfile(null);
          setStatus("error");
          setDetail(message(error, "Could not connect to that server"));
        }
      },
      async signIn() {
        if (!profile) return;
        setStatus("authorizing");
        setDetail("Complete sign-in in the secure browser…");
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
          setStatus("error");
          setDetail(message(error, "Sign-in failed"));
        }
      },
      async signOut() {
        const currentProfile = profile;
        const currentAccountId = accountId;
        if (currentProfile && currentAccountId) {
          const key = { serverId: currentProfile.id, accountId: currentAccountId };
          const credentials = await getCredentials(key);
          if (credentials) {
            await Promise.allSettled([
              revokeMobileToken(currentProfile.baseUrl, credentials.accessToken, "access_token"),
              revokeMobileToken(currentProfile.baseUrl, credentials.refreshToken, "refresh_token"),
            ]);
          }
          await deleteCredentials(key);
        }
        await Promise.all([clearActiveAccount(), clearWidgetSnapshot()]);
        setClient(null);
        setAccountId(null);
        setProfile(null);
        setStatus("signed-out");
        setDetail(null);
      },
      recover: restore,
    }),
    [status, detail, profile, accountId, client],
  );

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
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

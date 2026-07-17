export interface OAuthCredentials {
  accessToken: string;
  refreshToken: string;
  expiresAt: string;
  scope: string;
}

export interface CredentialKey {
  serverId: string;
  accountId: string;
}

export interface CredentialStore {
  get(key: CredentialKey): Promise<OAuthCredentials | null>;
  set(key: CredentialKey, credentials: OAuthCredentials): Promise<void>;
  delete(key: CredentialKey): Promise<void>;
}

export interface ServerProfileStore<TProfile> {
  getActive(): Promise<TProfile | null>;
  setActive(profile: TProfile): Promise<void>;
  clearActive(): Promise<void>;
}

export interface RefreshCoordinatorOptions {
  now?: () => number;
  skewMs?: number;
  refresh: (credentials: OAuthCredentials) => Promise<OAuthCredentials>;
  persist: (credentials: OAuthCredentials) => Promise<void>;
}

/** Coordinates rotating refresh tokens so concurrent API calls refresh exactly once. */
export function createRefreshCoordinator(
  initial: OAuthCredentials,
  options: RefreshCoordinatorOptions,
) {
  let credentials = initial;
  let pending: Promise<OAuthCredentials> | null = null;
  const now = options.now ?? Date.now;
  const skewMs = options.skewMs ?? 30_000;

  async function refresh(): Promise<OAuthCredentials> {
    if (pending) return pending;
    pending = performRefresh();
    try {
      return await pending;
    } finally {
      pending = null;
    }
  }

  async function performRefresh(): Promise<OAuthCredentials> {
    const next = await options.refresh(credentials);
    await options.persist(next);
    credentials = next;
    return next;
  }

  return {
    async getAccessToken(): Promise<string> {
      if (Date.parse(credentials.expiresAt) - skewMs <= now()) {
        return (await refresh()).accessToken;
      }
      return credentials.accessToken;
    },
    refresh,
    current(): OAuthCredentials {
      return credentials;
    },
  };
}

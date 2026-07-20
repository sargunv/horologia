import { beforeEach, describe, expect, it, vi } from "vitest";

import { cancelMobileAuthorization, completeMobileAuthorization } from "./oauth";

const mocks = vi.hoisted(() => ({
  dismissAuthSession: vi.fn<() => void>(),
  exchangeCodeAsync: vi.fn<(...args: unknown[]) => Promise<unknown>>(),
  items: new Map<string, string>(),
}));

vi.mock("expo-auth-session", () => ({
  AuthRequest: class {},
  exchangeCodeAsync: mocks.exchangeCodeAsync,
  makeRedirectUri: () => "horologia://oauth/callback",
}));

vi.mock("expo-secure-store", () => ({
  deleteItemAsync: vi.fn<(key: string) => Promise<void>>(async (key) => {
    mocks.items.delete(key);
  }),
  getItemAsync: vi.fn<(key: string) => Promise<string | null>>(
    async (key) => mocks.items.get(key) ?? null,
  ),
  setItemAsync: vi.fn<(key: string, value: string) => Promise<void>>(async (key, value) => {
    mocks.items.set(key, value);
  }),
}));

vi.mock("expo-web-browser", () => ({
  dismissAuthSession: mocks.dismissAuthSession,
  maybeCompleteAuthSession: vi.fn<() => void>(),
}));

const serverId = "server-a";
const pendingKey = `oauth.pending.${serverId}`;

beforeEach(() => {
  mocks.dismissAuthSession.mockReset();
  mocks.exchangeCodeAsync.mockReset();
  mocks.items.clear();
});

describe("completeMobileAuthorization", () => {
  it("exchanges a matching one-time authorization against its bound server", async () => {
    mocks.items.set(
      pendingKey,
      JSON.stringify({
        codeVerifier: "verifier-a",
        serverId,
        serverBaseUrl: "https://a.example/horologia",
        state: "state-a",
      }),
    );
    mocks.exchangeCodeAsync.mockResolvedValue({ accessToken: "access-token" });

    await expect(completeMobileAuthorization(serverId, "code-a", "state-a")).resolves.toEqual({
      accessToken: "access-token",
    });
    expect(mocks.exchangeCodeAsync).toHaveBeenCalledWith(
      {
        clientId: "horologia-mobile",
        code: "code-a",
        redirectUri: "horologia://oauth/callback",
        extraParams: { code_verifier: "verifier-a" },
      },
      {
        authorizationEndpoint: "https://a.example/horologia/oauth/authorize",
        tokenEndpoint: "https://a.example/horologia/oauth/token",
        revocationEndpoint: "https://a.example/horologia/oauth/revoke",
      },
    );
    expect(mocks.items.has(pendingKey)).toBe(false);
  });

  it("refuses pending state that belongs to a different server profile", async () => {
    mocks.items.set(
      pendingKey,
      JSON.stringify({
        codeVerifier: "verifier-b",
        serverId: "server-b",
        serverBaseUrl: "https://b.example",
        state: "state-b",
      }),
    );

    await expect(completeMobileAuthorization(serverId, "code-b", "state-b")).rejects.toThrow(
      "does not belong to the selected server",
    );
    expect(mocks.exchangeCodeAsync).not.toHaveBeenCalled();
    expect(mocks.items.has(pendingKey)).toBe(false);
  });

  it("refuses a callback state mismatch without exchanging the code", async () => {
    mocks.items.set(
      pendingKey,
      JSON.stringify({
        codeVerifier: "verifier-a",
        serverId,
        serverBaseUrl: "https://a.example",
        state: "expected-state",
      }),
    );

    await expect(completeMobileAuthorization(serverId, "code-a", "other-state")).rejects.toThrow(
      "Authorization state did not match",
    );
    expect(mocks.exchangeCodeAsync).not.toHaveBeenCalled();
    expect(mocks.items.has(pendingKey)).toBe(false);
  });

  it("does not consume another server's pending authorization", async () => {
    const otherKey = "oauth.pending.server-b";
    mocks.items.set(
      otherKey,
      JSON.stringify({
        codeVerifier: "verifier-b",
        serverId: "server-b",
        serverBaseUrl: "https://b.example",
        state: "state-b",
      }),
    );

    await expect(completeMobileAuthorization(serverId, "code-a", "state-b")).rejects.toThrow(
      "no longer available for this server",
    );
    expect(mocks.items.has(otherKey)).toBe(true);
    expect(mocks.exchangeCodeAsync).not.toHaveBeenCalled();
  });
});

describe("cancelMobileAuthorization", () => {
  it("dismisses the browser and clears only the selected server's pending request", async () => {
    const otherKey = "oauth.pending.server-b";
    mocks.items.set(pendingKey, "pending-a");
    mocks.items.set(otherKey, "pending-b");

    await cancelMobileAuthorization(serverId);

    expect(mocks.dismissAuthSession).toHaveBeenCalledOnce();
    expect(mocks.items.has(pendingKey)).toBe(false);
    expect(mocks.items.get(otherKey)).toBe("pending-b");
  });

  it("clears pending state when the platform cannot dismiss the browser", async () => {
    mocks.items.set(pendingKey, "pending-a");
    mocks.dismissAuthSession.mockImplementation(() => {
      throw new Error("Browser dismissal is unavailable");
    });

    await expect(cancelMobileAuthorization(serverId)).resolves.toBeUndefined();
    expect(mocks.items.has(pendingKey)).toBe(false);
  });
});

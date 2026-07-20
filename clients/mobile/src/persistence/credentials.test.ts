import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  deleteCredentials,
  getCredentials,
  setCredentials,
  setCredentialsWhileCurrent,
} from "./credentials";

const mocks = vi.hoisted(() => ({
  deleteItemAsync: vi.fn<(key: string) => Promise<void>>(),
  getItemAsync: vi.fn<(key: string) => Promise<string | null>>(),
  setItemAsync:
    vi.fn<(key: string, value: string, options: { keychainAccessible: string }) => Promise<void>>(),
}));

vi.mock("expo-secure-store", () => ({
  WHEN_UNLOCKED_THIS_DEVICE_ONLY: "device-only",
  deleteItemAsync: mocks.deleteItemAsync,
  getItemAsync: mocks.getItemAsync,
  setItemAsync: mocks.setItemAsync,
}));

beforeEach(() => {
  mocks.deleteItemAsync.mockReset();
  mocks.getItemAsync.mockReset();
  mocks.setItemAsync.mockReset();
});

describe("scoped credential persistence", () => {
  it("stores and deletes credentials under both server and account identity", async () => {
    const credentials = {
      accessToken: "access",
      refreshToken: "refresh",
      expiresAt: "2026-07-19T12:00:00Z",
      scope: "tasks:read",
    };

    await setCredentials({ serverId: "server-a", accountId: "account-1" }, credentials);
    await deleteCredentials({ serverId: "server-a", accountId: "account-2" });

    expect(mocks.setItemAsync).toHaveBeenCalledWith(
      "oauth.server-a.account-1",
      JSON.stringify(credentials),
      { keychainAccessible: "device-only" },
    );
    expect(mocks.deleteItemAsync).toHaveBeenCalledWith("oauth.server-a.account-2");
  });

  it("reads only the requested scope and removes invalid data from that scope", async () => {
    mocks.getItemAsync.mockResolvedValue(JSON.stringify({ accessToken: "incomplete" }));

    await expect(
      getCredentials({ serverId: "server-b", accountId: "account-4" }),
    ).resolves.toBeNull();

    expect(mocks.getItemAsync).toHaveBeenCalledWith("oauth.server-b.account-4");
    expect(mocks.deleteItemAsync).toHaveBeenCalledWith("oauth.server-b.account-4");
  });

  it("does not persist refreshed credentials after their session becomes stale", async () => {
    let finishWrite: (() => void) | undefined;
    mocks.setItemAsync.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          finishWrite = resolve;
        }),
    );
    let current = true;
    const key = { serverId: "server-a", accountId: "account-1" };
    const credentials = {
      accessToken: "rotated-access",
      refreshToken: "rotated-refresh",
      expiresAt: "2026-07-19T13:00:00Z",
      scope: "tasks:read",
    };

    const persistence = setCredentialsWhileCurrent(key, credentials, () => current);
    current = false;
    finishWrite?.();

    await expect(persistence).resolves.toBe(false);
    expect(mocks.deleteItemAsync).toHaveBeenCalledWith("oauth.server-a.account-1");
  });

  it("does not start a credential write for an inactive session", async () => {
    await expect(
      setCredentialsWhileCurrent(
        { serverId: "server-b", accountId: "account-2" },
        {
          accessToken: "access",
          refreshToken: "refresh",
          expiresAt: "2026-07-19T13:00:00Z",
          scope: "tasks:read",
        },
        () => false,
      ),
    ).resolves.toBe(false);

    expect(mocks.setItemAsync).not.toHaveBeenCalled();
    expect(mocks.deleteItemAsync).not.toHaveBeenCalled();
  });
});

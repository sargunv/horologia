import { describe, expect, it, vi } from "vitest";

import { createRefreshCoordinator, type OAuthCredentials } from "./session";

const expired: OAuthCredentials = {
  accessToken: "old-access",
  refreshToken: "old-refresh",
  expiresAt: "2026-01-01T00:00:00.000Z",
  scope: "tasks:read",
};

describe("createRefreshCoordinator", () => {
  it("shares one rotating refresh between concurrent callers", async () => {
    const next = {
      ...expired,
      accessToken: "new-access",
      refreshToken: "new-refresh",
      expiresAt: "2026-01-02T00:00:00.000Z",
    };
    const refresh = vi.fn<() => Promise<OAuthCredentials>>(async () => next);
    const persist = vi.fn<(credentials: OAuthCredentials) => Promise<void>>(async () => undefined);
    const coordinator = createRefreshCoordinator(expired, {
      now: () => Date.parse("2026-01-01T12:00:00.000Z"),
      refresh,
      persist,
    });

    await expect(
      Promise.all([coordinator.getAccessToken(), coordinator.getAccessToken()]),
    ).resolves.toEqual(["new-access", "new-access"]);
    expect(refresh).toHaveBeenCalledOnce();
    expect(persist).toHaveBeenCalledWith(next);
  });

  it("keeps credentials scoped by caller-provided server and account keys", async () => {
    const values = new Map<string, OAuthCredentials>();
    const set = (serverId: string, accountId: string, value: OAuthCredentials) =>
      values.set(`${serverId}:${accountId}`, value);
    set("server-a", "account", expired);
    set("server-b", "account", { ...expired, accessToken: "other" });
    expect(values.get("server-a:account")?.accessToken).toBe("old-access");
    expect(values.get("server-b:account")?.accessToken).toBe("other");
  });
});

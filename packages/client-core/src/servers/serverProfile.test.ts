import { describe, expect, it } from "vitest";

import { createServerProfile, normalizeServerUrl } from "./serverProfile.ts";

describe("normalizeServerUrl", () => {
  it("normalizes a host into a stable HTTPS base URL", () => {
    expect(normalizeServerUrl(" example.com/horologia/ ")).toBe("https://example.com/horologia");
  });

  it("preserves development HTTP and strips URL-only state", () => {
    expect(normalizeServerUrl("http://localhost:8080/?from=app#setup")).toBe(
      "http://localhost:8080",
    );
  });

  it("rejects credentials and unsupported schemes", () => {
    expect(() => normalizeServerUrl("https://user:password@example.com")).toThrow(
      "must not include credentials",
    );
    expect(() => normalizeServerUrl("file:///tmp/horologia")).toThrow("HTTP or HTTPS");
  });
});

describe("createServerProfile", () => {
  it("keeps identity independent from the mutable URL", () => {
    expect(
      createServerProfile({
        id: "server-1",
        baseUrl: "horologia.example",
        now: "2026-07-17T00:00:00.000Z",
      }),
    ).toEqual({
      id: "server-1",
      baseUrl: "https://horologia.example",
      displayName: "horologia.example",
      lastUsedAt: "2026-07-17T00:00:00.000Z",
    });
  });
});

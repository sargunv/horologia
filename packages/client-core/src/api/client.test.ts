import { describe, expect, it, vi } from "vitest";

import { createHorologiaClient } from "./client.ts";

describe("createHorologiaClient", () => {
  it("observes unauthorized responses without replacing them", async () => {
    const onUnauthorized = vi.fn<() => void>();
    const fetchResponse = vi.fn<typeof globalThis.fetch>(async () =>
      Response.json({ message: "unauthorized" }, { status: 401 }),
    );
    const client = createHorologiaClient({
      baseUrl: "https://home.example.test/api/",
      fetch: fetchResponse,
      onUnauthorized,
    });

    const result = await client.GET("/users/me");

    expect(result.response.status).toBe(401);
    expect(onUnauthorized).toHaveBeenCalledOnce();
  });
});

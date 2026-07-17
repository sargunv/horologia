import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

import { createHorologiaClient } from "../api/client.ts";
import { createSettingsCommands } from "./settings.ts";

describe("settings commands", () => {
  it("updates only the active server's account caches", async () => {
    const fetchResponse = vi.fn<typeof globalThis.fetch>(async () => Response.json(user));
    const queryClient = new QueryClient();
    const own = ["server-a", "currentUser"];
    const foreign = ["server-b", "currentUser"];
    queryClient.setQueryData(own, user);
    queryClient.setQueryData(foreign, user);
    const commands = createSettingsCommands({
      serverId: "server-a",
      apiClient: createHorologiaClient({
        baseUrl: "https://home.example.test/api/",
        fetch: fetchResponse,
      }),
      queryClient,
    });

    await expect(commands.updateUser("user-1", { name: "Ada" })).resolves.toEqual(user);
    expect(queryClient.getQueryState(own)?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(foreign)?.isInvalidated).toBe(false);
  });
});

const user = {
  id: "user-1",
  email: "ada@example.test",
  name: "Ada",
  isOwner: false,
  hasPassword: true,
  appearanceMode: "system" as const,
  appearanceLightTheme: "light",
  appearanceDarkTheme: "dark",
  createdAt: "2026-07-17T00:00:00Z",
  updatedAt: "2026-07-17T00:00:00Z",
};

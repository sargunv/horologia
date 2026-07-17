import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

import { createHorologiaClient } from "../api/client.ts";
import { createLibraryCommands } from "./library.ts";

describe("library commands", () => {
  it("updates recipe caches without crossing server namespaces", async () => {
    const fetchResponse = vi.fn<typeof globalThis.fetch>(async () => Response.json(recipe));
    const queryClient = new QueryClient();
    const own = ["server-a", "recipes", "list", "all"];
    const foreign = ["server-b", "recipes", "list", "all"];
    queryClient.setQueryData(own, []);
    queryClient.setQueryData(foreign, []);
    const commands = createLibraryCommands({
      serverId: "server-a",
      apiClient: createHorologiaClient({
        baseUrl: "https://home.example.test/api/",
        fetch: fetchResponse,
      }),
      queryClient,
    });

    await expect(commands.updateRecipe("home", "recipe-1", { name: "Toast" })).resolves.toEqual(
      recipe,
    );
    expect(queryClient.getQueryState(own)?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(foreign)?.isInvalidated).toBe(false);
  });
});

const recipe = {
  id: "recipe-1",
  spaceSlug: "home",
  name: "Toast",
  description: "",
  yield: null,
  prepMinutes: null,
  cookMinutes: null,
  tags: [],
  ingredientSections: [],
  instructionSections: [],
  createdAt: "2026-07-17T00:00:00Z",
  updatedAt: "2026-07-17T00:00:00Z",
};

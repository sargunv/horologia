import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

import { createHorologiaClient } from "../api/client.ts";
import { createTaskCommands } from "./tasks.ts";

describe("task commands", () => {
  it("creates through the API and invalidates only the active server", async () => {
    const fetchResponse = vi.fn<typeof globalThis.fetch>(async () =>
      Response.json(task, { status: 201 }),
    );
    const queryClient = new QueryClient();
    const ownList = ["server-a", "spaces", "home", "tasks", "list"];
    const foreignList = ["server-b", "spaces", "home", "tasks", "list"];
    queryClient.setQueryData(ownList, []);
    queryClient.setQueryData(foreignList, []);

    const commands = createTaskCommands({
      serverId: "server-a",
      apiClient: createHorologiaClient({
        baseUrl: "https://home.example.test/api/",
        fetch: fetchResponse,
      }),
      queryClient,
    });

    await expect(commands.create("home", { title: "Water herbs" })).resolves.toEqual(task);
    expect(fetchResponse).toHaveBeenCalledOnce();
    expect(queryClient.getQueryState(ownList)?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(foreignList)?.isInvalidated).toBe(false);
  });

  it("surfaces API errors without invalidating accepted data", async () => {
    const fetchResponse = vi.fn<typeof globalThis.fetch>(async () =>
      Response.json({ code: "invalid", message: "Title is required" }, { status: 400 }),
    );
    const queryClient = new QueryClient();
    const list = ["server-a", "spaces", "home", "tasks", "list"];
    queryClient.setQueryData(list, []);
    const commands = createTaskCommands({
      serverId: "server-a",
      apiClient: createHorologiaClient({
        baseUrl: "https://home.example.test/api/",
        fetch: fetchResponse,
      }),
      queryClient,
    });

    await expect(commands.create("home", { title: "" })).rejects.toThrow("Title is required");
    expect(queryClient.getQueryState(list)?.isInvalidated).toBe(false);
  });
});

const task = {
  id: "task-1",
  spaceSlug: "home",
  title: "Water herbs",
  description: "",
  status: "todo",
  effort: null,
  priority: null,
  recurrenceType: "one_off",
  recurrenceRule: null,
  lastCompletedAt: null,
  assigneeIds: [],
  rotationPool: [],
  tags: [],
  relations: [],
  due: null,
  overdueActionRule: null,
  createdAt: "2026-07-17T00:00:00Z",
  updatedAt: "2026-07-17T00:00:00Z",
};

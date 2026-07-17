import { describe, expect, it } from "vitest";

import { createWidgetSnapshotV1, projectMyTasksWidgetSnapshot } from "./widgetSnapshot.ts";

describe("createWidgetSnapshotV1", () => {
  it("creates a server- and account-scoped display snapshot", () => {
    expect(
      createWidgetSnapshotV1({
        serverId: "server-1",
        accountId: "account-1",
        generatedAt: "2026-07-17T00:00:00.000Z",
        tasks: [
          {
            id: "task-1",
            spaceSlug: "home",
            title: "Water the herbs",
            due: null,
            status: "open",
          },
        ],
      }),
    ).toEqual({
      version: 1,
      serverId: "server-1",
      accountId: "account-1",
      generatedAt: "2026-07-17T00:00:00.000Z",
      tasks: [
        {
          id: "task-1",
          spaceSlug: "home",
          title: "Water the herbs",
          due: null,
          status: "open",
        },
      ],
    });
  });
});

describe("projectMyTasksWidgetSnapshot", () => {
  it("keeps ordered display fields and strips full task data", () => {
    expect(
      projectMyTasksWidgetSnapshot({
        serverId: "server-a",
        accountId: "account-a",
        generatedAt: "2026-07-17T00:00:00.000Z",
        limit: 1,
        tasks: [
          {
            id: "first",
            spaceSlug: "home",
            title: "First",
            due: { at: "2026-07-18" },
            status: "open",
          },
          { id: "second", spaceSlug: "home", title: "Second", due: null, status: "open" },
        ],
      }),
    ).toEqual({
      version: 1,
      serverId: "server-a",
      accountId: "account-a",
      generatedAt: "2026-07-17T00:00:00.000Z",
      tasks: [
        { id: "first", spaceSlug: "home", title: "First", due: "2026-07-18", status: "open" },
      ],
    });
  });
});

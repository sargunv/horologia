import { describe, expect, it } from "vitest";

import { createWidgetSnapshotV1 } from "./widgetSnapshot.ts";

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

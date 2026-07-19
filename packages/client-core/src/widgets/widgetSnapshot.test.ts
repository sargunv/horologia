import { describe, expect, it } from "vitest";

import { projectMyTasksWidgetSnapshot } from "./widgetSnapshot.ts";

describe("projectMyTasksWidgetSnapshot", () => {
  it("keeps ordered display fields and strips full task data", () => {
    expect(
      projectMyTasksWidgetSnapshot({
        serverId: "server-a",
        accountId: "account-a",
        generatedAt: "2026-07-17T00:00:00.000Z",
        hasMore: true,
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
      taskCount: 2,
      hasMore: true,
      tasks: [
        { id: "first", spaceSlug: "home", title: "First", due: "2026-07-18", status: "open" },
      ],
    });
  });
});

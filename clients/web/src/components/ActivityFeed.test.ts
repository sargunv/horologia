import { describe, expect, it } from "vitest";
import type { components } from "@horologia/client-core/schema";
import { groupCompactActivityEntries } from "./ActivityFeed.tsx";

type ActivityLogEntry = components["schemas"]["ActivityLogEntry"];

function update(
  id: string,
  createdAt: string,
  from: string | null,
  to: string | null,
  overrides: Partial<ActivityLogEntry> = {},
): ActivityLogEntry {
  return {
    id,
    spaceSlug: "test",
    actorId: "actor-1",
    tokenId: null,
    tokenName: null,
    entityType: "task",
    entityId: "T1",
    action: "updated",
    details: [{ field: "description", from, to }],
    createdAt,
    ...overrides,
  };
}

describe("groupCompactActivityEntries", () => {
  it("groups adjacent updates and retains the complete before/after range", () => {
    const groups = groupCompactActivityEntries([
      update("newest", "2026-07-11T12:04:00Z", "second", "third"),
      update("middle", "2026-07-11T12:02:00Z", "first", "second"),
      update("oldest", "2026-07-11T12:00:00Z", null, "first"),
    ]);

    expect(groups).toHaveLength(1);
    expect(groups[0]!.count).toBe(3);
    expect(groups[0]!.entry.id).toBe("newest");
    expect(groups[0]!.entry.details).toEqual([{ field: "description", from: null, to: "third" }]);
  });

  it("groups consecutive updates to different fields into one editing session", () => {
    const groups = groupCompactActivityEntries([
      update("description", "2026-07-11T12:02:00Z", "old", "new"),
      update("priority", "2026-07-11T12:01:00Z", "low", "high", {
        details: [{ field: "priority", from: "low", to: "high" }],
      }),
      update("status", "2026-07-11T12:00:00Z", "todo", "done", {
        details: [{ field: "status", from: "todo", to: "done" }],
      }),
    ]);

    expect(groups).toHaveLength(1);
    expect(groups[0]!.entry.details).toEqual([
      { field: "description", from: "old", to: "new" },
      { field: "priority", from: "low", to: "high" },
      { field: "status", from: "todo", to: "done" },
    ]);
  });

  it("groups repeated opaque recipe collection updates", () => {
    const details = [{ field: "ingredients", from: null, to: "updated" }];
    const groups = groupCompactActivityEntries([
      update("newest", "2026-07-11T12:02:00Z", null, "updated", {
        entityType: "recipe",
        entityId: "R1",
        details,
      }),
      update("oldest", "2026-07-11T12:00:00Z", null, "updated", {
        entityType: "recipe",
        entityId: "R1",
        details,
      }),
    ]);

    expect(groups).toHaveLength(1);
    expect(groups[0]!.count).toBe(2);
    expect(groups[0]!.entry.details).toEqual(details);
  });

  it("does not group across actors, discontinuous values, or the time window", () => {
    const groups = groupCompactActivityEntries([
      update("newest", "2026-07-11T12:20:00Z", "third", "fourth"),
      update("too-old", "2026-07-11T12:10:00Z", "second", "third"),
      update("discontinuous", "2026-07-11T12:09:00Z", "first", "unrelated"),
      update("other-actor", "2026-07-11T12:08:00Z", null, "first", { actorId: "actor-2" }),
    ]);

    expect(groups).toHaveLength(4);
  });
});

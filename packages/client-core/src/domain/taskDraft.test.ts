import { describe, expect, it } from "vitest";

import { taskCreateFromDraft, taskDraftFromTask, taskUpdateFromDraft } from "./taskDraft.ts";

describe("portable task editor state", () => {
  it("normalizes a complete advanced task update", () => {
    const draft = taskDraftFromTask(undefined, "America/Los_Angeles");
    Object.assign(draft, {
      title: "  Water herbs  ",
      status: "todo",
      effort: "small",
      priority: "high",
      assigneeIds: "user-1, user-2, user-1",
      tags: "garden, kitchen",
      dueDate: "2026-07-18",
      recurrenceType: "fixed_accumulating",
      recurrenceRule: "FREQ=WEEKLY;BYDAY=SA",
      overdueAction: "set_status",
      overdueAfter: "2",
      overdueStatus: "missed",
    });

    expect(taskUpdateFromDraft(draft)).toMatchObject({
      title: "Water herbs",
      effort: "small",
      priority: "high",
      assigneeIds: ["user-1", "user-2"],
      tags: ["garden", "kitchen"],
      due: { at: "2026-07-18", timezone: "America/Los_Angeles" },
      recurrenceType: "fixed_accumulating",
      recurrenceRule: "FREQ=WEEKLY;BYDAY=SA",
      overdueActionRule: { action: "set_status", after: 2, status: "missed" },
    });
  });

  it("omits nullable level fields from creation and clears them on update", () => {
    const draft = taskDraftFromTask();
    draft.title = "One off";
    expect(taskCreateFromDraft(draft)).not.toHaveProperty("effort");
    expect(taskCreateFromDraft(draft)).not.toHaveProperty("priority");
    expect(taskUpdateFromDraft(draft)).toMatchObject({
      effort: null,
      priority: null,
      recurrenceRule: null,
    });
  });
});

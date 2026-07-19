import { describe, expect, it } from "vitest";

import { namedItemsValidationError, taskStatusesValidationError } from "./taskSettings.ts";

describe("namedItemsValidationError", () => {
  it("rejects duplicate normalized names", () => {
    expect(namedItemsValidationError([{ name: "High" }, { name: " high " }], "Level")).toBe(
      "Level names must be unique.",
    );
  });
});

describe("taskStatusesValidationError", () => {
  it("requires one initial and at least one completion status", () => {
    expect(
      taskStatusesValidationError([
        { name: "Todo", category: "initial" },
        { name: "Queued", category: "initial" },
        { name: "Doing", category: "intermediate" },
      ]),
    ).toBe("There must be exactly one initial status.");
    expect(
      taskStatusesValidationError([
        { name: "Todo", category: "initial" },
        { name: "Doing", category: "intermediate" },
      ]),
    ).toBe("There must be at least one completion status.");
  });
});

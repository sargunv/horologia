import { describe, expect, it } from "vitest";

import { parseLevels, parseStatuses } from "./ordered-input";

describe("ordered settings input", () => {
  it("parses trimmed statuses in display order", () => {
    expect(parseStatuses(" Todo | initial | circle \n\nDone|completion|check ")).toEqual([
      { name: "Todo", category: "initial", icon: "circle" },
      { name: "Done", category: "completion", icon: "check" },
    ]);
  });

  it("rejects a status without a supported category", () => {
    expect(() => parseStatuses("Waiting | someday")).toThrow("Waiting needs a valid category.");
  });

  it("parses levels and clears an omitted icon", () => {
    expect(parseLevels("Low\nHigh | flame")).toEqual([
      { name: "Low", icon: "" },
      { name: "High", icon: "flame" },
    ]);
  });
});

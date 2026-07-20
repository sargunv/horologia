import { describe, expect, it, vi } from "vitest";

import {
  classifyWindow,
  EXPANDED_WINDOW_MIN_WIDTH,
  getListDetailGeometry,
  MEDIUM_WINDOW_MIN_WIDTH,
} from "./window-class";

vi.mock("react-native", () => ({
  useWindowDimensions: vi.fn<() => { width: number }>(() => ({ width: 320 })),
}));

describe("classifyWindow", () => {
  it("classifies the current window width at the adaptive breakpoints", () => {
    expect(classifyWindow(MEDIUM_WINDOW_MIN_WIDTH - 1)).toBe("compact");
    expect(classifyWindow(MEDIUM_WINDOW_MIN_WIDTH)).toBe("medium");
    expect(classifyWindow(EXPANDED_WINDOW_MIN_WIDTH - 1)).toBe("medium");
    expect(classifyWindow(EXPANDED_WINDOW_MIN_WIDTH)).toBe("expanded");
  });

  it("rejects widths that cannot describe a window", () => {
    expect(() => classifyWindow(0)).toThrow("positive finite number");
    expect(() => classifyWindow(Number.NaN)).toThrow("positive finite number");
  });
});

describe("getListDetailGeometry", () => {
  it("uses one pane below expanded width", () => {
    expect(getListDetailGeometry(700)).toEqual({
      windowClass: "medium",
      presentation: "single-pane",
      listPaneWidth: 700,
      detailPaneWidth: 700,
    });
  });

  it("derives bounded list and remaining detail widths from the current width", () => {
    expect(getListDetailGeometry(1_000)).toEqual({
      windowClass: "expanded",
      presentation: "list-detail",
      listPaneWidth: 380,
      detailPaneWidth: 620,
    });
    expect(getListDetailGeometry(1_600)).toEqual({
      windowClass: "expanded",
      presentation: "list-detail",
      listPaneWidth: 440,
      detailPaneWidth: 1_160,
    });
  });

  it("supports feature-specific pane constraints", () => {
    expect(
      getListDetailGeometry(1_000, {
        minimumListPaneWidth: 280,
        maximumListPaneWidth: 360,
        preferredListFraction: 0.3,
      }),
    ).toEqual({
      windowClass: "expanded",
      presentation: "list-detail",
      listPaneWidth: 300,
      detailPaneWidth: 700,
    });
  });
});

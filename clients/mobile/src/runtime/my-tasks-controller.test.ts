import { describe, expect, it } from "vitest";

import { selectMyTasksSource } from "./my-tasks-controller";

describe("selectMyTasksSource", () => {
  it("uses accepted network pages instead of an older cache", () => {
    expect(
      selectMyTasksSource({
        networkPages: [
          { items: ["network-1"], nextCursor: "page-2" },
          { items: ["network-2"], nextCursor: null },
        ],
        cache: { tasks: ["cached"], hasMore: true },
        queryHasNextPage: false,
      }),
    ).toEqual({
      tasks: ["network-1", "network-2"],
      source: "network",
      networkNextCursor: null,
      canLoadMore: false,
      cachedHasMore: false,
    });
  });

  it("uses cached tasks while the network has no accepted response", () => {
    expect(
      selectMyTasksSource({
        networkPages: undefined,
        cache: { tasks: ["cached-1", "cached-2"], hasMore: false },
        queryHasNextPage: undefined,
      }),
    ).toEqual({
      tasks: ["cached-1", "cached-2"],
      source: "cache",
      networkNextCursor: null,
      canLoadMore: false,
      cachedHasMore: false,
    });
  });

  it("reports cached hasMore without fabricating a network pagination cursor", () => {
    const selection = selectMyTasksSource({
      networkPages: undefined,
      cache: { tasks: ["cached"], hasMore: true },
      queryHasNextPage: true,
    });

    expect(selection.cachedHasMore).toBe(true);
    expect(selection.networkNextCursor).toBeNull();
    expect(selection.canLoadMore).toBe(false);
  });

  it("loads more only when network data and its real cursor agree", () => {
    expect(
      selectMyTasksSource({
        networkPages: [{ items: ["network"], nextCursor: "page-2" }],
        cache: null,
        queryHasNextPage: true,
      }).canLoadMore,
    ).toBe(true);
    expect(
      selectMyTasksSource({
        networkPages: [{ items: ["network"], nextCursor: null }],
        cache: null,
        queryHasNextPage: true,
      }).canLoadMore,
    ).toBe(false);
  });
});

import { describe, expect, it } from "vitest";

import { createSearchResults, searchResultRoute } from "./search-controller";

const scope = { serverId: "server-a", accountId: "account-1" };

describe("search result presentation", () => {
  it("maps task and recipe API results into canonical presentation items", () => {
    expect(
      createSearchResults({
        tasks: [{ id: "task-1", spaceSlug: "home", title: "Change filter", status: "Next" }],
        recipes: [
          {
            id: "recipe-1",
            spaceSlug: "kitchen",
            name: "Soup",
            yield: null,
            prepMinutes: 10,
            cookMinutes: 30,
            tags: ["weeknight", "vegetarian", "winter"],
            updatedAt: "2026-07-19T12:00:00Z",
          },
          {
            id: "recipe-2",
            spaceSlug: "kitchen",
            name: "Toast",
            yield: null,
            prepMinutes: null,
            cookMinutes: null,
            tags: [],
            updatedAt: "2026-07-19T12:00:00Z",
          },
        ],
      }),
    ).toEqual([
      {
        kind: "task",
        id: "task-1",
        spaceSlug: "home",
        title: "Change filter",
        meta: "Next",
      },
      {
        kind: "recipe",
        id: "recipe-1",
        spaceSlug: "kitchen",
        title: "Soup",
        meta: "weeknight · vegetarian",
      },
      {
        kind: "recipe",
        id: "recipe-2",
        spaceSlug: "kitchen",
        title: "Toast",
        meta: "Recipe",
      },
    ]);
  });
});

describe("search result routing", () => {
  it("routes tasks through My Tasks and recipes through the global recipe route", () => {
    expect(
      searchResultRoute(scope, {
        kind: "task",
        id: "task-1",
        spaceSlug: "home",
        title: "Task",
        meta: "Next",
      }),
    ).toEqual({
      pathname: "/[serverId]/[accountId]/tasks/[spaceSlug]/[taskId]",
      params: { ...scope, spaceSlug: "home", taskId: "task-1" },
    });
    expect(
      searchResultRoute(scope, {
        kind: "recipe",
        id: "recipe-1",
        spaceSlug: "kitchen",
        title: "Soup",
        meta: "Recipe",
      }),
    ).toEqual({
      pathname: "/[serverId]/[accountId]/library/recipes/[spaceSlug]/[recipeId]",
      params: { ...scope, spaceSlug: "kitchen", recipeId: "recipe-1" },
    });
  });
});

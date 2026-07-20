import { describe, expect, it } from "vitest";

import {
  parseRouteScope,
  type RouteScope,
  routes,
  routeScopeKey,
  routeScopesMatch,
} from "./routes";

const scope: RouteScope = { serverId: "server/a", accountId: "account:b" };

describe("scoped route builders", () => {
  it("keeps server and account scope on every authenticated destination", () => {
    const destinations = [
      routes.tasks(scope),
      routes.library(scope),
      routes.search(scope),
      routes.taskDetail(scope, "home", "task-1"),
      routes.librarySpaces(scope),
      routes.librarySpace(scope, "home"),
      routes.spaceTasks(scope, "home"),
      routes.spaceTaskDetail(scope, "home", "task-1"),
      routes.recipes(scope),
      routes.spaceRecipes(scope, "home"),
      routes.recipeDetail(scope, "home", "recipe-1"),
      routes.spaceRecipeDetail(scope, "home", "recipe-1"),
      routes.account(scope),
    ];

    for (const destination of destinations) {
      expect(destination.params).toMatchObject(scope);
      expect(destination.pathname).toContain("/[serverId]/[accountId]/");
    }
  });

  it("places resource identifiers on their canonical routes", () => {
    expect(routes.taskDetail(scope, "kitchen", "task-7")).toEqual({
      pathname: "/[serverId]/[accountId]/tasks/[spaceSlug]/[taskId]",
      params: { ...scope, spaceSlug: "kitchen", taskId: "task-7" },
    });
    expect(routes.recipeDetail(scope, "kitchen", "recipe-3")).toEqual({
      pathname: "/[serverId]/[accountId]/library/recipes/[spaceSlug]/[recipeId]",
      params: { ...scope, spaceSlug: "kitchen", recipeId: "recipe-3" },
    });
    expect(routes.spaceRecipeDetail(scope, "kitchen", "recipe-3")).toEqual({
      pathname: "/[serverId]/[accountId]/library/spaces/[spaceSlug]/recipes/[recipeId]",
      params: { ...scope, spaceSlug: "kitchen", recipeId: "recipe-3" },
    });
  });
});

describe("route scope identity", () => {
  it("requires an exact server and account match", () => {
    expect(routeScopesMatch(scope, { ...scope })).toBe(true);
    expect(routeScopesMatch(scope, { ...scope, serverId: "server-b" })).toBe(false);
    expect(routeScopesMatch(scope, { ...scope, accountId: "account-b" })).toBe(false);
  });

  it("creates unambiguous keys even when identifiers contain separators", () => {
    expect(routeScopeKey(scope)).not.toBe(
      routeScopeKey({ serverId: "server", accountId: "a/account:b" }),
    );
  });

  it("accepts only one non-empty value for each route parameter", () => {
    expect(parseRouteScope(scope)).toEqual(scope);
    expect(parseRouteScope({ serverId: [scope.serverId], accountId: scope.accountId })).toBeNull();
    expect(parseRouteScope({ serverId: scope.serverId, accountId: "" })).toBeNull();
    expect(parseRouteScope({ serverId: scope.serverId })).toBeNull();
  });
});

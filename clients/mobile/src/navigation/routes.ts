export interface RouteScope {
  serverId: string;
  accountId: string;
}

export type ScopedRoute<Pathname extends string, Params extends object> = {
  pathname: Pathname;
  params: Params;
};

interface ScopedTabParams extends RouteScope {
  [key: string]: string | number | (string | number)[] | null | undefined;
}

interface TaskDetailParams extends ScopedTabParams {
  spaceSlug: string;
  taskId: string;
}

interface SpaceParams extends ScopedTabParams {
  spaceSlug: string;
}

interface RecipeDetailParams extends SpaceParams {
  recipeId: string;
}

function scopedTabParams(scope: RouteScope): ScopedTabParams {
  return { serverId: scope.serverId, accountId: scope.accountId };
}

export const routes = {
  root(): "/" {
    return "/";
  },

  oauthCallback(): "/oauth/callback" {
    return "/oauth/callback";
  },

  tasks(scope: RouteScope): ScopedRoute<"/[serverId]/[accountId]/tasks", ScopedTabParams> {
    return {
      pathname: "/[serverId]/[accountId]/tasks",
      params: scopedTabParams(scope),
    };
  },

  library(scope: RouteScope): ScopedRoute<"/[serverId]/[accountId]/library", ScopedTabParams> {
    return {
      pathname: "/[serverId]/[accountId]/library",
      params: scopedTabParams(scope),
    };
  },

  search(scope: RouteScope): ScopedRoute<"/[serverId]/[accountId]/search", ScopedTabParams> {
    return {
      pathname: "/[serverId]/[accountId]/search",
      params: scopedTabParams(scope),
    };
  },

  taskDetail(
    scope: RouteScope,
    spaceSlug: string,
    taskId: string,
  ): ScopedRoute<"/[serverId]/[accountId]/tasks/[spaceSlug]/[taskId]", TaskDetailParams> {
    return {
      pathname: "/[serverId]/[accountId]/tasks/[spaceSlug]/[taskId]",
      params: { ...scopedTabParams(scope), spaceSlug, taskId },
    };
  },

  librarySpaces(
    scope: RouteScope,
  ): ScopedRoute<"/[serverId]/[accountId]/library/spaces", ScopedTabParams> {
    return {
      pathname: "/[serverId]/[accountId]/library/spaces",
      params: scopedTabParams(scope),
    };
  },

  librarySpace(
    scope: RouteScope,
    spaceSlug: string,
  ): ScopedRoute<"/[serverId]/[accountId]/library/spaces/[spaceSlug]", SpaceParams> {
    return {
      pathname: "/[serverId]/[accountId]/library/spaces/[spaceSlug]",
      params: { ...scopedTabParams(scope), spaceSlug },
    };
  },

  spaceTasks(
    scope: RouteScope,
    spaceSlug: string,
  ): ScopedRoute<"/[serverId]/[accountId]/library/spaces/[spaceSlug]/tasks", SpaceParams> {
    return {
      pathname: "/[serverId]/[accountId]/library/spaces/[spaceSlug]/tasks",
      params: { ...scopedTabParams(scope), spaceSlug },
    };
  },

  spaceTaskDetail(
    scope: RouteScope,
    spaceSlug: string,
    taskId: string,
  ): ScopedRoute<
    "/[serverId]/[accountId]/library/spaces/[spaceSlug]/tasks/[taskId]",
    TaskDetailParams
  > {
    return {
      pathname: "/[serverId]/[accountId]/library/spaces/[spaceSlug]/tasks/[taskId]",
      params: { ...scopedTabParams(scope), spaceSlug, taskId },
    };
  },

  recipes(
    scope: RouteScope,
  ): ScopedRoute<"/[serverId]/[accountId]/library/recipes", ScopedTabParams> {
    return {
      pathname: "/[serverId]/[accountId]/library/recipes",
      params: scopedTabParams(scope),
    };
  },

  spaceRecipes(
    scope: RouteScope,
    spaceSlug: string,
  ): ScopedRoute<"/[serverId]/[accountId]/library/spaces/[spaceSlug]/recipes", SpaceParams> {
    return {
      pathname: "/[serverId]/[accountId]/library/spaces/[spaceSlug]/recipes",
      params: { ...scopedTabParams(scope), spaceSlug },
    };
  },

  recipeDetail(
    scope: RouteScope,
    spaceSlug: string,
    recipeId: string,
  ): ScopedRoute<
    "/[serverId]/[accountId]/library/recipes/[spaceSlug]/[recipeId]",
    RecipeDetailParams
  > {
    return {
      pathname: "/[serverId]/[accountId]/library/recipes/[spaceSlug]/[recipeId]",
      params: { ...scopedTabParams(scope), spaceSlug, recipeId },
    };
  },

  spaceRecipeDetail(
    scope: RouteScope,
    spaceSlug: string,
    recipeId: string,
  ): ScopedRoute<
    "/[serverId]/[accountId]/library/spaces/[spaceSlug]/recipes/[recipeId]",
    RecipeDetailParams
  > {
    return {
      pathname: "/[serverId]/[accountId]/library/spaces/[spaceSlug]/recipes/[recipeId]",
      params: { ...scopedTabParams(scope), spaceSlug, recipeId },
    };
  },

  account(scope: RouteScope): ScopedRoute<"/[serverId]/[accountId]/account", ScopedTabParams> {
    return {
      pathname: "/[serverId]/[accountId]/account",
      params: scopedTabParams(scope),
    };
  },
};

export function routeScopeKey(scope: RouteScope): string {
  return JSON.stringify([scope.serverId, scope.accountId]);
}

export function routeScopesMatch(left: RouteScope, right: RouteScope): boolean {
  return left.serverId === right.serverId && left.accountId === right.accountId;
}

export function parseRouteScope(params: {
  serverId?: string | string[];
  accountId?: string | string[];
}): RouteScope | null {
  const serverId = singleRouteParam(params.serverId);
  const accountId = singleRouteParam(params.accountId);
  return serverId && accountId ? { serverId, accountId } : null;
}

function singleRouteParam(value: string | string[] | undefined): string | null {
  return typeof value === "string" && value.length > 0 ? value : null;
}

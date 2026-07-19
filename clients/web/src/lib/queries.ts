import { createAppQueries, createQueries } from "@horologia/client-core/queries";

import { apiClient, appClient } from "../api/client.ts";

export type { AuthConfig, LinkPendingInfo } from "@horologia/client-core/queries";

const queries = {
  ...createAppQueries({ serverId: window.location.origin, appClient }),
  ...createQueries({ serverId: window.location.origin, apiClient }),
};

export const {
  authConfigQueryOptions,
  linkPendingQueryOptions,
  currentUserQueryOptions,
  usersQueryOptions,
  userQueryOptions,
  spacesQueryOptions,
  spaceQueryOptions,
  spaceMembersQueryOptions,
  spaceTaskStatusesQueryOptions,
  spaceEffortLevelsQueryOptions,
  spacePriorityLevelsQueryOptions,
  spaceTagsQueryOptions,
  spaceTasksInfiniteQueryOptions,
  authTokensQueryOptions,
  spaceTaskQueryOptions,
  recipesInfiniteQueryOptions,
  recipeQueryOptions,
  recipeActivityInfiniteQueryOptions,
  recipeSearchQueryOptions,
  userTasksInfiniteQueryOptions,
  taskActivityInfiniteQueryOptions,
  taskSearchQueryOptions,
  spaceActivityInfiniteQueryOptions,
  userActivityInfiniteQueryOptions,
} = queries;

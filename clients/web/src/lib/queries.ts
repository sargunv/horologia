import { createQueries } from "@horologia/client-core/queries";

import { apiClient, appClient } from "../api/client.ts";

export type { AuthConfig, LinkPendingInfo } from "@horologia/client-core/queries";

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
} = createQueries({
  serverId: window.location.origin,
  apiClient,
  appClient,
});

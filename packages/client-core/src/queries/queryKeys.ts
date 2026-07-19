export function createQueryKeys(serverId: string) {
  const spaces = [serverId, "spaces"] as const;
  const users = [serverId, "users"] as const;
  const recipes = [serverId, "recipes"] as const;
  const tasks = [serverId, "tasks"] as const;

  return {
    authConfig: [serverId, "authConfig"] as const,
    linkPending: [serverId, "linkPending"] as const,
    currentUser: [serverId, "currentUser"] as const,
    serverInfo: [serverId, "serverInfo"] as const,
    users,
    user: (userId: string) => [...users, userId] as const,
    userTasks: (userId: string) => [...users, userId, "tasks", "list"] as const,
    userActivity: (userId: string) => [...users, userId, "activity"] as const,
    isUserTaskList: (queryKey: readonly unknown[]) =>
      queryKey[0] === serverId &&
      queryKey[1] === "users" &&
      queryKey[3] === "tasks" &&
      queryKey[4] === "list",
    spaces,
    space: (spaceSlug: string) => [...spaces, spaceSlug] as const,
    spaceMembers: (spaceSlug: string) => [...spaces, spaceSlug, "members"] as const,
    spaceTaskStatuses: (spaceSlug: string) => [...spaces, spaceSlug, "taskStatuses"] as const,
    spaceEffortLevels: (spaceSlug: string) => [...spaces, spaceSlug, "effortLevels"] as const,
    spacePriorityLevels: (spaceSlug: string) => [...spaces, spaceSlug, "priorityLevels"] as const,
    spaceTags: (spaceSlug: string) => [...spaces, spaceSlug, "tags"] as const,
    spaceTasks: (spaceSlug: string) => [...spaces, spaceSlug, "tasks", "list"] as const,
    spaceTask: (spaceSlug: string, taskId: string) =>
      [...spaces, spaceSlug, "tasks", taskId] as const,
    taskActivity: (spaceSlug: string, taskId: string) =>
      [...spaces, spaceSlug, "tasks", taskId, "activity"] as const,
    spaceActivity: (spaceSlug: string) => [...spaces, spaceSlug, "activity"] as const,
    authTokens: [serverId, "authTokens"] as const,
    recipes,
    recipeLists: [...recipes, "list"] as const,
    recipeList: (spaceSlug?: string) => [...recipes, "list", spaceSlug ?? "all"] as const,
    recipe: (spaceSlug: string, recipeId: string) =>
      [...spaces, spaceSlug, "recipes", recipeId] as const,
    recipeActivity: (spaceSlug: string, recipeId: string) =>
      [...spaces, spaceSlug, "recipes", recipeId, "activity"] as const,
    recipeSearches: [...recipes, "search"] as const,
    recipeSearch: (query: string, spaceSlug: string | undefined, limit: number) =>
      [...recipes, "search", query, spaceSlug ?? null, limit] as const,
    taskSearches: [...tasks, "search"] as const,
    taskSearch: (
      query: string,
      spaceSlug: string | undefined,
      excludeTaskId: string | undefined,
      limit: number,
    ) => [...tasks, "search", query, spaceSlug ?? null, excludeTaskId ?? null, limit] as const,
  };
}

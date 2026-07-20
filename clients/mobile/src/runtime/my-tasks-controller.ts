export interface MyTasksPage<Item> {
  items: Item[];
  nextCursor: string | null;
}

export interface MyTasksCache<Item> {
  tasks: Item[];
  hasMore: boolean;
}

export interface MyTasksSelection<Item> {
  tasks: Item[];
  source: "network" | "cache";
  networkNextCursor: string | null;
  canLoadMore: boolean;
  cachedHasMore: boolean;
}

export function selectMyTasksSource<Item>({
  networkPages,
  cache,
  queryHasNextPage,
}: {
  networkPages: MyTasksPage<Item>[] | undefined;
  cache: MyTasksCache<Item> | null;
  queryHasNextPage: boolean | undefined;
}): MyTasksSelection<Item> {
  if (networkPages) {
    const networkNextCursor = networkPages.at(-1)?.nextCursor ?? null;
    return {
      tasks: networkPages.flatMap((page) => page.items),
      source: "network",
      networkNextCursor,
      canLoadMore: networkNextCursor !== null && queryHasNextPage === true,
      cachedHasMore: false,
    };
  }

  return {
    tasks: cache?.tasks ?? [],
    source: "cache",
    networkNextCursor: null,
    canLoadMore: false,
    cachedHasMore: cache?.hasMore ?? false,
  };
}

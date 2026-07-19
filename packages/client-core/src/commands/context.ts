import type { QueryClient } from "@tanstack/react-query";

import type { HorologiaClient } from "../api/client";

export interface CommandContext {
  serverId: string;
  apiClient: HorologiaClient;
  queryClient: QueryClient;
  onCacheError?(error: unknown): void;
}

export async function synchronizeCache(
  context: CommandContext,
  operation: () => Promise<unknown>,
): Promise<void> {
  try {
    await operation();
  } catch (error) {
    context.onCacheError?.(error);
  }
}

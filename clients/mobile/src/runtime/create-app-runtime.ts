import type { HorologiaClient } from "@horologia/client-core/api";
import { createLibraryCommands } from "@horologia/client-core/commands/library";
import { createSettingsCommands } from "@horologia/client-core/commands/settings";
import { createTaskCommands } from "@horologia/client-core/commands/tasks";
import { createQueries } from "@horologia/client-core/queries";
import { createQueryClient } from "@horologia/client-core/runtime";

import type { RouteScope } from "@/navigation/routes";

export interface AppRuntime {
  scope: RouteScope;
  queryClient: ReturnType<typeof createQueryClient>;
  queries: ReturnType<typeof createQueries>;
  taskCommands: ReturnType<typeof createTaskCommands>;
  libraryCommands: ReturnType<typeof createLibraryCommands>;
  settingsCommands: ReturnType<typeof createSettingsCommands>;
  isActive(): boolean;
  dispose(): void;
}

export interface CreateAppRuntimeOptions {
  scope: RouteScope;
  apiClient: HorologiaClient;
}

export function createAppRuntime({ scope, apiClient }: CreateAppRuntimeOptions): AppRuntime {
  const queryClient = createQueryClient();
  const commandContext = {
    serverId: scope.serverId,
    apiClient,
    queryClient,
  };
  let active = true;
  return {
    scope,
    queryClient,
    queries: createQueries({ serverId: scope.serverId, apiClient }),
    taskCommands: createTaskCommands(commandContext),
    libraryCommands: createLibraryCommands(commandContext),
    settingsCommands: createSettingsCommands(commandContext),
    isActive: () => active,
    dispose() {
      if (!active) return;
      active = false;
      queryClient.clear();
    },
  };
}

export function disposeAppRuntime(runtime: AppRuntime): void {
  runtime.dispose();
}

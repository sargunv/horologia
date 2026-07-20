import { createHorologiaClient } from "@horologia/client-core";
import { describe, expect, it } from "vitest";

import { createAppRuntime, disposeAppRuntime } from "./create-app-runtime";

const apiClient = createHorologiaClient({ baseUrl: "https://example.test/api/" });

describe("scoped app runtime", () => {
  it("creates isolated runtime and query-client identities for each scope", () => {
    const first = createAppRuntime({
      scope: { serverId: "server-a", accountId: "account-1" },
      apiClient,
    });
    const second = createAppRuntime({
      scope: { serverId: "server-b", accountId: "account-1" },
      apiClient,
    });
    const third = createAppRuntime({
      scope: { serverId: "server-a", accountId: "account-2" },
      apiClient,
    });

    expect(first).not.toBe(second);
    expect(first.queryClient).not.toBe(second.queryClient);
    expect(first.queryClient).not.toBe(third.queryClient);
    expect(first.queries.userTasksInfiniteQueryOptions(first.scope.accountId).queryKey).not.toEqual(
      second.queries.userTasksInfiniteQueryOptions(second.scope.accountId).queryKey,
    );
    expect(first.queries.userTasksInfiniteQueryOptions(first.scope.accountId).queryKey).not.toEqual(
      third.queries.userTasksInfiniteQueryOptions(third.scope.accountId).queryKey,
    );
  });

  it("clears only the disposed scope's query cache", () => {
    const first = createAppRuntime({
      scope: { serverId: "server-a", accountId: "account-1" },
      apiClient,
    });
    const second = createAppRuntime({
      scope: { serverId: "server-b", accountId: "account-1" },
      apiClient,
    });
    first.queryClient.setQueryData(["probe"], "first");
    second.queryClient.setQueryData(["probe"], "second");

    expect(first.isActive()).toBe(true);
    disposeAppRuntime(first);

    expect(first.isActive()).toBe(false);
    expect(first.queryClient.getQueryData(["probe"])).toBeUndefined();
    expect(second.queryClient.getQueryData(["probe"])).toBe("second");
  });
});

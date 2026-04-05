import { MutationCache, QueryClient } from "@tanstack/react-query";
import { toaster } from "./toaster.ts";

export const queryClient = new QueryClient({
  mutationCache: new MutationCache({
    onError: (error) => {
      const message =
        typeof error === "object" && error !== null && "message" in error
          ? String(error.message)
          : "An unexpected error occurred.";
      toaster.error({ title: "Action failed", description: message });
    },
  }),
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry(failureCount, error) {
        // openapi-fetch throws the error body object (e.g. { code, message }),
        // not a Response. Don't retry client errors.
        if (typeof error === "object" && error !== null && "code" in error) return false;
        return failureCount < 3;
      },
    },
  },
});

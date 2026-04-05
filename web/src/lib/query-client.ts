import { MutationCache, QueryClient } from "@tanstack/react-query";
import { toaster } from "./toaster.ts";

// Two-layer error notification model:
//   Tier 1 – mutation failure (API error): handled globally here → "Action failed" error toast.
//   Tier 2 – post-success cache invalidation failure: handled per-mutation via notifyStaleData()
//             → "Data may be out of date" warning toast.
// These are mutually exclusive in the normal happy path.
// To suppress the global tier-1 toast for a specific mutation, pass
// `meta: { suppressGlobalError: true }` to useMutation.
export const queryClient = new QueryClient({
  mutationCache: new MutationCache({
    onError: (error, _vars, _ctx, mutation) => {
      if (mutation.meta?.["suppressGlobalError"]) return;
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

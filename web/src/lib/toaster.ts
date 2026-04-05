import { createToaster } from "@skeletonlabs/skeleton-react";

// Module-level singleton — safe for CSR-only use. createToaster creates a
// reactive store; no DOM or React context is required at creation time.
export const toaster = createToaster({
  placement: "bottom-end",
  // 4.5rem bottom offset clears the mobile nav bar (pb-16 = 4rem) on small screens.
  offsets: { bottom: "4.5rem", right: "1rem", left: "1rem", top: "1rem" },
  removeDelay: 250,
  // Default display durations (ms)
  duration: 5000,
});

/**
 * Show a transient warning when query-cache invalidation fails after a
 * successful mutation, leaving the UI potentially stale.
 *
 * This is the "tier 2" error path. The "tier 1" path (mutation failure itself)
 * is handled globally in MutationCache.onError in query-client.ts. The two
 * are mutually exclusive in the normal happy path.
 */
export function notifyStaleData() {
  toaster.warning({
    title: "Data may be out of date",
    description: "Refresh the page to see the latest changes.",
  });
}

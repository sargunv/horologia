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
 * Mutation failures themselves are handled per-component via
 * `mutation.error` rendering (e.g., ErrorAlert). This function covers
 * the narrower case where the mutation succeeds but cache invalidation
 * throws.
 */
export function notifyStaleData() {
  toaster.warning({
    title: "Data may be out of date",
    description: "Refresh the page to see the latest changes.",
  });
}

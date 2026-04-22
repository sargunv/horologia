import { toast } from "sonner";

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
  toast.warning("Data may be out of date", {
    description: "Refresh the page to see the latest changes.",
  });
}

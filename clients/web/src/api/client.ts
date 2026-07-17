import { createHorologiaClient, getApiErrorMessage } from "@horologia/client-core/api";

export const apiClient = createHorologiaClient({
  baseUrl: "/api",
  credentials: "include",
  onUnauthorized() {
    window.dispatchEvent(new CustomEvent("horologia:unauthorized"));
  },
});

export const appClient = createHorologiaClient({
  baseUrl: "",
  credentials: "include",
  onUnauthorized() {
    window.dispatchEvent(new CustomEvent("horologia:unauthorized"));
  },
});

export { getApiErrorMessage };

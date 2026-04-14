import createClient from "openapi-fetch";
import type { paths } from "./schema.d.ts";

export const apiClient = createClient<paths>({
  baseUrl: "/api",
  credentials: "include",
});

apiClient.use({
  onResponse({ response }) {
    if (response.status === 401) {
      window.dispatchEvent(new CustomEvent("horologia:unauthorized"));
    }
    return response;
  },
});

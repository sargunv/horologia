import { QueryClient } from "@tanstack/react-query";

export function createQueryClient(): QueryClient {
  return new QueryClient({
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
}

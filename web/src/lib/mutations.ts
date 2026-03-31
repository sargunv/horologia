import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "../api/client.ts";
import type { components } from "../api/schema.d.ts";

export function useUserPatch(userId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: components["schemas"]["UserUpdate"]) => {
      const { data, error } = await apiClient.PATCH("/users/{userId}", {
        params: { path: { userId } },
        body,
      });
      if (error) throw new Error(error.message ?? "Failed to update user");
      return data;
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["users", userId] }),
        queryClient.invalidateQueries({ queryKey: ["users"] }),
        queryClient.invalidateQueries({ queryKey: ["currentUser"] }),
      ]);
    },
  });
}

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, UserPlus } from "lucide-react";
import { type FormEvent, useState } from "react";
import { apiClient } from "../../api/client.ts";
import type { components } from "@horologia/client-core/schema";
import { notifyStaleData } from "../../lib/toaster.ts";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";
import { SettingsSection } from "../space-settings/SettingsSection.tsx";

type UserCreate = components["schemas"]["UserCreate"];

export function CreateUserPane({
  onBack,
  onCreated,
}: {
  onBack: () => void;
  onCreated: (userId: string) => void;
}) {
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [isOwner, setIsOwner] = useState(false);

  const createMutation = useMutation({
    mutationFn: async () => {
      const body: UserCreate = { name, email };
      if (password) body.password = password;
      if (isOwner) body.isOwner = true;

      const { data, error } = await apiClient.POST("/users", { body });
      if (error) throw new Error(error.message ?? "Failed to create user");
      return data;
    },
    onSuccess: async (data) => {
      try {
        await queryClient.invalidateQueries({ queryKey: [window.location.origin, "users"] });
      } catch (err) {
        console.error("Cache invalidation failed after mutation:", err);
        notifyStaleData();
      }
      onCreated(data.id);
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    createMutation.mutate();
  }

  return (
    <div className="flex flex-col gap-4">
      <button
        type="button"
        onClick={onBack}
        className="btn btn-square btn-sm btn-soft self-start lg:hidden"
        aria-label="Back to user list"
      >
        <ArrowLeft className="size-4" aria-hidden="true" />
      </button>

      <SettingsSection
        icon={<UserPlus className="size-5" aria-hidden="true" />}
        title="Create user"
        description="Add a new user to this instance."
      >
        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
          <label className="flex flex-col gap-1">
            <span className="text-base-content/70 text-sm font-medium">Name</span>
            <input
              type="text"
              required
              maxLength={200}
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="input w-full"
              placeholder="Alice Johnson"
              disabled={createMutation.isPending}
            />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-base-content/70 text-sm font-medium">Email</span>
            <input
              type="email"
              required
              maxLength={200}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="input w-full"
              placeholder="alice@example.com"
              disabled={createMutation.isPending}
            />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-base-content/70 text-sm font-medium">
              Password <span className="text-base-content/60">(optional)</span>
            </span>
            <input
              type="password"
              minLength={8}
              maxLength={72}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="input w-full"
              placeholder="Leave blank for OIDC-only"
              disabled={createMutation.isPending}
            />
          </label>
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={isOwner}
              onChange={(e) => setIsOwner(e.target.checked)}
              className="checkbox"
              disabled={createMutation.isPending}
            />
            <span className="text-sm">Owner</span>
          </label>

          {createMutation.error && <ErrorAlert message={createMutation.error.message} />}

          <div className="flex justify-end">
            <button
              type="submit"
              disabled={createMutation.isPending || !name.trim() || !email.trim()}
              className="btn btn-primary"
            >
              {createMutation.isPending ? "Creating..." : "Create user"}
            </button>
          </div>
        </form>
      </SettingsSection>
    </div>
  );
}

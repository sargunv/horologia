import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, UserPlus } from "lucide-react";
import { type FormEvent, useState } from "react";
import { apiClient } from "../../api/client.ts";
import type { components } from "../../api/schema.d.ts";
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
      await queryClient.invalidateQueries({ queryKey: ["users"] });
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
        className="btn-icon btn-icon-sm preset-outlined-surface-200-800 self-start lg:hidden"
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
            <span className="text-surface-600-400 text-sm font-medium">Name</span>
            <input
              type="text"
              required
              maxLength={200}
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="input preset-outlined-surface-200-800 w-full"
              placeholder="Alice Johnson"
              disabled={createMutation.isPending}
            />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-surface-600-400 text-sm font-medium">Email</span>
            <input
              type="email"
              required
              maxLength={200}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="input preset-outlined-surface-200-800 w-full"
              placeholder="alice@example.com"
              disabled={createMutation.isPending}
            />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-surface-600-400 text-sm font-medium">
              Password <span className="text-surface-500">(optional)</span>
            </span>
            <input
              type="password"
              minLength={8}
              maxLength={72}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="input preset-outlined-surface-200-800 w-full"
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
              className="btn preset-filled-primary-500"
            >
              {createMutation.isPending ? "Creating..." : "Create user"}
            </button>
          </div>
        </form>
      </SettingsSection>
    </div>
  );
}

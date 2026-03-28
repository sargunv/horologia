import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { SquareKanban } from "lucide-react";
import { type FormEvent, useState } from "react";
import { apiClient } from "../../api/client.ts";
import type { components } from "../../api/schema.d.ts";
import { ErrorAlert } from "./ErrorAlert.tsx";
import { SettingsSection } from "./SettingsSection.tsx";

type Space = components["schemas"]["Space"];

export function GeneralSettingsSection({
  space,
}: {
  space: Pick<Space, "slug" | "name" | "description">;
}) {
  return (
    <SettingsSection
      icon={<SquareKanban className="size-5" />}
      title="General"
      description="Name, slug, and description for this space."
    >
      <GeneralSettingsForm space={space} />
    </SettingsSection>
  );
}

function GeneralSettingsForm({ space }: { space: Pick<Space, "slug" | "name" | "description"> }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [name, setName] = useState(space.name);
  const [slug, setSlug] = useState(space.slug);
  const [description, setDescription] = useState(space.description);

  const hasChanges =
    name !== space.name || slug !== space.slug || description !== space.description;

  const updateMutation = useMutation({
    mutationFn: async (body: { name?: string; slug?: string; description?: string }) => {
      const { data, error } = await apiClient.PATCH("/spaces/{spaceSlug}", {
        params: { path: { spaceSlug: space.slug } },
        body,
      });
      if (error)
        throw new Error((error as { message?: string }).message ?? "Failed to update space");
      return data;
    },
    onSuccess: async (data) => {
      try {
        if (data.slug !== space.slug) {
          queryClient.removeQueries({ queryKey: ["spaces", space.slug] });
        }
        await queryClient.invalidateQueries({ queryKey: ["spaces"] });
      } catch (err) {
        console.error("Failed to refresh after space update:", err);
      }
      if (data.slug !== space.slug) {
        void navigate({ to: "/spaces/$spaceSlug/settings", params: { spaceSlug: data.slug } });
      }
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const body: { name?: string; slug?: string; description?: string } = {};
    if (name !== space.name) body.name = name;
    if (slug !== space.slug) body.slug = slug;
    if (description !== space.description) body.description = description;
    updateMutation.mutate(body);
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <label className="flex flex-col gap-1">
        <span className="text-surface-600-400 text-sm font-medium">Name</span>
        <input
          type="text"
          required
          value={name}
          onChange={(e) => {
            updateMutation.reset();
            setName(e.target.value);
          }}
          className="input preset-outlined-surface-200-800 w-full"
          placeholder="My Project"
          maxLength={200}
          disabled={updateMutation.isPending}
        />
      </label>

      <label className="flex flex-col gap-1">
        <span className="text-surface-600-400 text-sm font-medium">Slug</span>
        <input
          type="text"
          required
          value={slug}
          onChange={(e) => {
            updateMutation.reset();
            setSlug(e.target.value);
          }}
          className="input preset-outlined-surface-200-800 w-full"
          placeholder="my-project"
          maxLength={100}
          disabled={updateMutation.isPending}
        />
        <span className="text-surface-500 text-xs">
          Used in URLs. Changing this will update all links to this space.
        </span>
      </label>

      <label className="flex flex-col gap-1">
        <span className="text-surface-600-400 text-sm font-medium">
          Description <span className="text-surface-500 font-normal">(optional)</span>
        </span>
        <textarea
          value={description}
          onChange={(e) => {
            updateMutation.reset();
            setDescription(e.target.value);
          }}
          className="textarea preset-outlined-surface-200-800 w-full resize-none"
          placeholder="What is this space for?"
          rows={3}
          maxLength={1000}
          disabled={updateMutation.isPending}
        />
      </label>

      {updateMutation.error && <ErrorAlert message={updateMutation.error.message} />}

      <div className="flex justify-end">
        <button
          type="submit"
          disabled={updateMutation.isPending || !hasChanges}
          className="btn preset-filled-primary-500"
        >
          {updateMutation.isPending ? "Saving..." : "Save changes"}
        </button>
      </div>
    </form>
  );
}

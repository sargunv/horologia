import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { CircleAlert, SquareKanban } from "lucide-react";
import { type FormEvent, useState } from "react";
import { apiClient } from "../../api/client.ts";
import { SettingsSection } from "./SettingsSection.tsx";

export function GeneralSettingsSection({
  space,
}: {
  space: { slug: string; name: string; description: string };
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

function GeneralSettingsForm({
  space,
}: {
  space: { slug: string; name: string; description: string };
}) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [name, setName] = useState(space.name);
  const [slug, setSlug] = useState(space.slug);
  const [description, setDescription] = useState(space.description);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  const hasChanges =
    name !== space.name || slug !== space.slug || description !== space.description;

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setPending(true);

    const body: { name?: string; slug?: string; description?: string } = {};
    if (name !== space.name) body.name = name;
    if (slug !== space.slug) body.slug = slug;
    if (description !== space.description) body.description = description;

    try {
      const { data, error: apiError } = await apiClient.PATCH("/spaces/{spaceSlug}", {
        params: { path: { spaceSlug: space.slug } },
        body,
      });
      if (apiError) {
        setError((apiError as { message?: string }).message ?? "Failed to update space");
        return;
      }
      if (data.slug !== space.slug) {
        queryClient.removeQueries({ queryKey: ["spaces", space.slug] });
      }
      await queryClient.invalidateQueries({ queryKey: ["spaces"] });
      if (data.slug !== space.slug) {
        void navigate({ to: "/spaces/$spaceSlug/settings", params: { spaceSlug: data.slug } });
      }
    } catch {
      setError("Something went wrong. Please try again.");
    } finally {
      setPending(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <label className="flex flex-col gap-1">
        <span className="text-surface-600-400 text-sm font-medium">Name</span>
        <input
          type="text"
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="input preset-outlined-surface-200-800 w-full"
          placeholder="My Project"
          maxLength={200}
          disabled={pending}
        />
      </label>

      <label className="flex flex-col gap-1">
        <span className="text-surface-600-400 text-sm font-medium">Slug</span>
        <input
          type="text"
          required
          value={slug}
          onChange={(e) => setSlug(e.target.value)}
          className="input preset-outlined-surface-200-800 w-full"
          placeholder="my-project"
          maxLength={100}
          disabled={pending}
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
          onChange={(e) => setDescription(e.target.value)}
          className="input preset-outlined-surface-200-800 w-full resize-none"
          placeholder="What is this space for?"
          rows={3}
          maxLength={1000}
          disabled={pending}
        />
      </label>

      {error && (
        <div className="preset-filled-error-500 flex items-center gap-2 rounded-base px-3 py-2 text-sm">
          <CircleAlert className="size-4 shrink-0" />
          {error}
        </div>
      )}

      <div className="flex justify-end">
        <button
          type="submit"
          disabled={pending || !hasChanges}
          className="btn preset-filled-primary-500"
        >
          {pending ? "Saving..." : "Save changes"}
        </button>
      </div>
    </form>
  );
}

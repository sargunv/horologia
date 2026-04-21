import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, createLink, useNavigate } from "@tanstack/react-router";
import { CircleAlert } from "lucide-react";
import { type FormEvent, useState } from "react";
import { apiClient } from "../../../api/client.ts";
import { notifyStaleData } from "../../../lib/toaster.ts";
import { Card } from "../../../ui/Card.tsx";

export const Route = createFileRoute("/_authenticated/spaces/new")({
  component: NewSpacePage,
});

const CancelLink = createLink("a");

// Keep in sync with backend: api/src/spaces.tsp (SpaceCreate/SpaceUpdate slug @pattern)
function toSlug(name: string): string {
  return name
    .toLocaleLowerCase()
    .replace(/[^\p{L}0-9]+/gu, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 100);
}

function NewSpacePage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [slugEdited, setSlugEdited] = useState(false);
  const [description, setDescription] = useState("");

  const createMutation = useMutation({
    mutationFn: async (body: { name: string; slug: string; description?: string }) => {
      const { data, error } = await apiClient.POST("/spaces", { body });
      if (error) throw new Error(error.message ?? "Failed to create space");
      return data;
    },
    onSuccess: async (data) => {
      try {
        await queryClient.invalidateQueries({ queryKey: ["spaces"] });
      } catch (err) {
        console.error("Cache invalidation failed after mutation:", err);
        notifyStaleData();
      }
      void navigate({ to: "/spaces/$spaceSlug", params: { spaceSlug: data.slug } });
    },
  });

  function handleNameChange(value: string) {
    setName(value);
    if (!slugEdited) {
      setSlug(toSlug(value));
    }
  }

  function handleSlugChange(value: string) {
    setSlug(value);
    setSlugEdited(true);
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    createMutation.mutate({ name, slug, ...(description ? { description } : {}) });
  }

  return (
    <div className="mx-auto max-w-3xl p-6">
      <h1 className="text-xl font-semibold">Create space</h1>
      <p className="text-base-content/70 mt-1">
        Spaces are where you organize tasks around a project or team.
      </p>

      <Card className="mt-6 flex flex-col gap-6 p-6">
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <label className="flex flex-col gap-1">
            <span className="text-base-content/70 text-sm font-medium">Name</span>
            <input
              type="text"
              required
              value={name}
              onChange={(e) => handleNameChange(e.target.value)}
              className="input w-full"
              placeholder="My Project"
              maxLength={200}
              disabled={createMutation.isPending}
            />
          </label>

          <label className="flex flex-col gap-1">
            <span className="text-base-content/70 text-sm font-medium">Slug</span>
            <input
              type="text"
              required
              value={slug}
              onChange={(e) => handleSlugChange(e.target.value)}
              className="input w-full"
              placeholder="my-project"
              maxLength={100}
              disabled={createMutation.isPending}
            />
            <span className="text-base-content/60 text-xs">
              Used in URLs. Auto-derived from name, but you can customize it.
            </span>
          </label>

          <label className="flex flex-col gap-1">
            <span className="text-base-content/70 text-sm font-medium">
              Description <span className="text-base-content/60 font-normal">(optional)</span>
            </span>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="textarea w-full"
              placeholder="What is this space for?"
              rows={3}
              maxLength={1000}
              disabled={createMutation.isPending}
            />
          </label>

          {createMutation.error && (
            <div role="alert" className="alert alert-error alert-soft text-sm">
              <CircleAlert className="size-4 shrink-0" />
              {createMutation.error.message}
            </div>
          )}

          <div className="flex gap-3">
            <button
              type="submit"
              disabled={createMutation.isPending}
              className="btn btn-primary flex-1"
            >
              {createMutation.isPending ? "Creating..." : "Create space"}
            </button>
            <CancelLink to="/spaces" className="btn btn-soft">
              Cancel
            </CancelLink>
          </div>
        </form>
      </Card>
    </div>
  );
}

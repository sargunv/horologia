import { createLibraryCommands } from "@horologia/client-core/commands/library";
import { slugifySpaceName } from "@horologia/client-core/domain/space-slug";
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

function NewSpacePage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [slugEdited, setSlugEdited] = useState(false);
  const [description, setDescription] = useState("");
  const commands = createLibraryCommands({
    serverId: window.location.origin,
    apiClient,
    queryClient,
    onCacheError(error) {
      console.error("Cache invalidation failed after mutation:", error);
      notifyStaleData();
    },
  });

  const createMutation = useMutation({
    mutationFn: (body: { name: string; slug: string; description?: string }) =>
      commands.createSpace(body),
    onSuccess: (data) => {
      void navigate({ to: "/spaces/$spaceSlug", params: { spaceSlug: data.slug } });
    },
  });

  function handleNameChange(value: string) {
    setName(value);
    if (!slugEdited) {
      setSlug(slugifySpaceName(value));
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

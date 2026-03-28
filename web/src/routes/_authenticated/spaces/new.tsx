import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute, createLink, useNavigate } from "@tanstack/react-router";
import { CircleAlert } from "lucide-react";
import { type FormEvent, useState } from "react";
import { apiClient } from "../../../api/client.ts";

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
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

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

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setPending(true);

    try {
      const { data, error: apiError } = await apiClient.POST("/spaces", {
        body: { name, slug, ...(description ? { description } : {}) },
      });
      if (apiError) {
        setError((apiError as { message?: string }).message ?? "Failed to create space");
        return;
      }
      await queryClient.invalidateQueries({ queryKey: ["spaces"] });
      void navigate({ to: "/spaces/$spaceSlug", params: { spaceSlug: data.slug } });
    } catch {
      setError("Something went wrong. Please try again.");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="p-6">
      <div className="max-w-lg">
        <h1 className="h3">Create space</h1>
        <p className="text-surface-600-400 mt-1">
          Spaces are where you organize tasks around a project or team.
        </p>

        <div className="card preset-outlined-surface-200-800 mt-6 flex flex-col gap-6 p-6">
          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <label className="flex flex-col gap-1">
              <span className="text-surface-600-400 text-sm font-medium">Name</span>
              <input
                type="text"
                required
                value={name}
                onChange={(e) => handleNameChange(e.target.value)}
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
                onChange={(e) => handleSlugChange(e.target.value)}
                className="input preset-outlined-surface-200-800 w-full"
                placeholder="my-project"
                maxLength={100}
                disabled={pending}
              />
              <span className="text-surface-500 text-xs">
                Used in URLs. Auto-derived from name, but you can customize it.
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

            <div className="flex gap-3">
              <button
                type="submit"
                disabled={pending}
                className="btn preset-filled-primary-500 flex-1"
              >
                {pending ? "Creating..." : "Create space"}
              </button>
              <CancelLink to="/spaces" className="btn preset-outlined-surface-200-800">
                Cancel
              </CancelLink>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}

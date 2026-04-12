import { useMutation, useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute, createLink, useNavigate } from "@tanstack/react-router";
import { ArrowLeft, ChevronRight } from "lucide-react";
import { type FormEvent, useState } from "react";
import { apiClient } from "../../../../../api/client.ts";
import { ErrorAlert } from "../../../../../components/space-settings/ErrorAlert.tsx";
import { spaceQueryOptions } from "../../../../../lib/queries.ts";
import { notifyStaleData } from "../../../../../lib/toaster.ts";

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug/tasks/new")({
  component: CreateTaskPage,
});

const BackLink = createLink("a");
const BreadcrumbLink = createLink("a");

function CreateTaskPage() {
  const { spaceSlug } = Route.useParams();
  const { data: space } = useSuspenseQuery(spaceQueryOptions(spaceSlug));
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [title, setTitle] = useState("");

  const createMutation = useMutation({
    mutationFn: async (body: { title: string }) => {
      const { data, error } = await apiClient.POST("/spaces/{spaceSlug}/tasks", {
        params: { path: { spaceSlug } },
        body,
      });
      if (error) throw new Error(error.message ?? "Failed to create task");
      return data;
    },
    onSuccess: async (data) => {
      try {
        await queryClient.invalidateQueries({
          queryKey: ["spaces", spaceSlug, "tasks", "list"],
        });
      } catch (err) {
        console.error("Cache invalidation failed after mutation:", err);
        notifyStaleData();
      }
      await navigate({
        to: "/spaces/$spaceSlug/tasks/$taskId",
        params: { spaceSlug, taskId: data.id },
      });
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmed = title.trim();
    if (!trimmed || createMutation.isPending) return;
    createMutation.mutate({ title: trimmed });
  }

  return (
    <div className="space-y-4">
      <BackLink
        to="/spaces/$spaceSlug"
        params={{ spaceSlug }}
        className="text-surface-600-400 hover:text-surface-950-50 inline-flex items-center gap-1 text-sm transition-colors lg:hidden"
      >
        <ArrowLeft className="size-4" aria-hidden="true" />
        Back
      </BackLink>

      <ol className="flex items-center gap-1 text-sm">
        <li>
          <BreadcrumbLink
            to="/spaces/$spaceSlug"
            params={{ spaceSlug }}
            className="text-surface-600-400 truncate hover:underline"
          >
            {space.name}
          </BreadcrumbLink>
        </li>
        <li className="text-surface-500" aria-hidden="true">
          <ChevronRight className="size-3" />
        </li>
        <li className="shrink-0">New task</li>
      </ol>

      <h2 className="h4">Create task</h2>

      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <label className="flex flex-col gap-1">
          <span className="text-surface-600-400 text-sm font-medium">Title</span>
          <input
            type="text"
            required
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="input preset-outlined-surface-200-800 w-full"
            placeholder="What needs to be done?"
            maxLength={500}
            autoFocus
            disabled={createMutation.isPending}
          />
        </label>

        <div role="alert" aria-live="assertive">
          {createMutation.error && <ErrorAlert message={createMutation.error.message} />}
        </div>

        <div className="flex gap-3">
          <button
            type="submit"
            disabled={createMutation.isPending || !title.trim()}
            className="btn preset-filled-primary-500"
          >
            {createMutation.isPending ? "Creating..." : "Create task"}
          </button>
        </div>
      </form>
    </div>
  );
}

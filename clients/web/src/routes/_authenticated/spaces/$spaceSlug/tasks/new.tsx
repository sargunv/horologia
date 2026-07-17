import { createTaskCommands } from "@horologia/client-core/commands/tasks";
import { useMutation, useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute, createLink, useNavigate } from "@tanstack/react-router";
import { ArrowLeft, ChevronRight } from "lucide-react";
import { type FormEvent, useState } from "react";
import { apiClient } from "../../../../../api/client.ts";
import {
  DetailPaneHeader,
  DETAIL_PANE_TITLE_CLASS,
} from "../../../../../components/DetailPaneHeader.tsx";
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
  const commands = createTaskCommands({
    serverId: window.location.origin,
    apiClient,
    queryClient,
    onCacheError(error) {
      console.error("Cache invalidation failed after mutation:", error);
      notifyStaleData();
    },
  });

  const createMutation = useMutation({
    mutationFn: (body: { title: string }) => commands.create(spaceSlug, body),
    onSuccess: async (data) => {
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
      <DetailPaneHeader
        backLink={
          <BackLink
            to="/spaces/$spaceSlug"
            params={{ spaceSlug }}
            className="text-base-content/70 hover:text-base-content inline-flex items-center gap-1 text-sm transition-colors lg:hidden"
          >
            <ArrowLeft className="size-4" aria-hidden="true" />
            Back to {space.name}
          </BackLink>
        }
        breadcrumb={
          <ol className="flex min-w-0 items-center gap-1 text-sm">
            <li>
              <BreadcrumbLink
                to="/spaces/$spaceSlug"
                params={{ spaceSlug }}
                className="text-base-content/70 truncate hover:underline"
              >
                {space.name}
              </BreadcrumbLink>
            </li>
            <li className="text-base-content/60" aria-hidden="true">
              <ChevronRight className="size-3" />
            </li>
            <li className="shrink-0" aria-current="page">
              New task
            </li>
          </ol>
        }
        title={<h1 className={DETAIL_PANE_TITLE_CLASS}>Create task</h1>}
      />

      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <label className="flex flex-col gap-1">
          <span className="text-base-content/70 text-sm font-medium">Title</span>
          <input
            type="text"
            required
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="input w-full"
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
            className="btn btn-primary"
          >
            {createMutation.isPending ? "Creating..." : "Create task"}
          </button>
        </div>
      </form>
    </div>
  );
}

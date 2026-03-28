import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute, createLink } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { spaceQueryOptions, spaceTaskQueryOptions } from "../../../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug/tasks/$taskId")({
  loader: ({ context: { queryClient }, params: { spaceSlug, taskId } }) =>
    queryClient.ensureQueryData(spaceTaskQueryOptions(spaceSlug, taskId)),
  component: TaskDetailPage,
});

const BackLink = createLink("a");

function TaskDetailPage() {
  const { spaceSlug, taskId } = Route.useParams();
  const { data: space } = useSuspenseQuery(spaceQueryOptions(spaceSlug));
  const { data: task } = useSuspenseQuery(spaceTaskQueryOptions(spaceSlug, taskId));

  return (
    <div className="mx-auto max-w-3xl p-6">
      <BackLink
        to="/spaces/$spaceSlug"
        params={{ spaceSlug }}
        className="text-surface-600-400 hover:text-surface-950-50 mb-4 inline-flex items-center gap-1 text-sm transition-colors"
      >
        <ArrowLeft className="size-4" />
        Back to {space.name}
      </BackLink>

      <h1 className="h3">{task.title}</h1>
      <p className="text-surface-600-400 mt-1">Task details coming soon.</p>
    </div>
  );
}

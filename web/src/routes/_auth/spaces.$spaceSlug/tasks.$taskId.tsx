import { useSuspenseQuery } from "@tanstack/react-query";
import { Link, createFileRoute } from "@tanstack/react-router";
import { taskQueryOptions } from "../../../queries/tasks.ts";

export const Route = createFileRoute("/_auth/spaces/$spaceSlug/tasks/$taskId")({
  loader: ({ context, params }) =>
    context.queryClient.ensureQueryData(taskQueryOptions(params.taskId)),
  component: TaskDetailPage,
});

function TaskDetailPage() {
  const { spaceSlug, taskId } = Route.useParams();
  const { data: task } = useSuspenseQuery(taskQueryOptions(taskId));

  const categoryColor =
    task.status.category === "completion"
      ? "preset-filled-success-500"
      : task.status.category === "initial"
        ? "preset-filled-surface-500"
        : "preset-filled-primary-500";

  return (
    <div className="space-y-6">
      <div>
        <div className="flex gap-2 text-sm text-surface-500">
          <Link to="/spaces" className="hover:underline">
            Spaces
          </Link>
          <span>/</span>
          <Link to="/spaces/$spaceSlug" params={{ spaceSlug }} className="hover:underline">
            {spaceSlug}
          </Link>
        </div>
        <div className="mt-2 flex items-center gap-3">
          <span className="font-mono text-sm text-surface-500">{task.id}</span>
          <h1 className="h2">{task.title}</h1>
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="card p-4 space-y-3">
          <h2 className="font-semibold text-sm text-surface-500 uppercase tracking-wide">
            Details
          </h2>
          <dl className="space-y-2 text-sm">
            <div>
              <dt className="text-surface-500">Status</dt>
              <dd>
                <span className={`badge ${categoryColor}`}>{task.status.name}</span>
              </dd>
            </div>
            <div>
              <dt className="text-surface-500">Due date</dt>
              <dd>{task.dueDate ?? "None"}</dd>
            </div>
            <div>
              <dt className="text-surface-500">Assignees</dt>
              <dd>{task.assigneeIds.length > 0 ? task.assigneeIds.join(", ") : "Unassigned"}</dd>
            </div>
          </dl>
        </div>
      </div>

      {task.description ? (
        <div className="card p-4">
          <h2 className="font-semibold text-sm text-surface-500 uppercase tracking-wide mb-2">
            Description
          </h2>
          <div className="prose dark:prose-invert max-w-none whitespace-pre-wrap">
            {task.description}
          </div>
        </div>
      ) : null}
    </div>
  );
}

import { useSuspenseQuery } from "@tanstack/react-query";
import { Link, createFileRoute } from "@tanstack/react-router";
import { spaceQueryOptions } from "../../../../queries/spaces.ts";
import { taskQueryOptions } from "../../../../queries/tasks.ts";

export const Route = createFileRoute("/_auth/spaces/$spaceSlug/tasks/$taskId")({
  loader: ({ context, params }) =>
    Promise.all([
      context.queryClient.ensureQueryData(spaceQueryOptions(params.spaceSlug)),
      context.queryClient.ensureQueryData(taskQueryOptions(params.spaceSlug, params.taskId)),
    ]),
  component: TaskDetailPage,
});

function TaskDetailPage() {
  const { spaceSlug } = Route.useParams();
  const { data: space } = useSuspenseQuery(spaceQueryOptions(spaceSlug));
  const { taskId } = Route.useParams();
  const { data: task } = useSuspenseQuery(taskQueryOptions(spaceSlug, taskId));

  return (
    <div className="space-y-6">
      <div>
        <nav className="flex gap-2 text-sm text-surface-500">
          <Link to="/spaces" className="hover:underline">
            Spaces
          </Link>
          <span>/</span>
          <Link to="/spaces/$spaceSlug" params={{ spaceSlug }} className="hover:underline">
            {space.name}
          </Link>
        </nav>
        <div className="mt-2 flex items-center gap-3">
          <span className="font-mono text-sm text-surface-500">{task.id}</span>
          <h1 className="h2">{task.title}</h1>
        </div>
      </div>

      <div className="card p-4 space-y-3">
        <h2 className="h5">Details</h2>
        <dl className="space-y-2 text-sm">
          <div className="flex gap-4">
            <dt className="text-surface-500 w-24">Status</dt>
            <dd>
              <span className="badge">{task.status}</span>
            </dd>
          </div>
          <div className="flex gap-4">
            <dt className="text-surface-500 w-24">Due date</dt>
            <dd>{task.dueDate ?? "None"}</dd>
          </div>
          <div className="flex gap-4">
            <dt className="text-surface-500 w-24">Assignees</dt>
            <dd>{task.assigneeIds.length > 0 ? task.assigneeIds.join(", ") : "Unassigned"}</dd>
          </div>
        </dl>
      </div>

      {task.description ? (
        <div className="card p-4">
          <h2 className="h5 mb-2">Description</h2>
          <div className="whitespace-pre-wrap text-sm">{task.description}</div>
        </div>
      ) : null}
    </div>
  );
}

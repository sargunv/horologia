import { useSuspenseQuery } from "@tanstack/react-query";
import { Link, createFileRoute } from "@tanstack/react-router";
import { ErrorDisplay } from "../../../../components/ui/error-display.tsx";
import { spaceQueryOptions } from "../../../../queries/spaces.ts";
import { spaceTasksQueryOptions } from "../../../../queries/tasks.ts";

export const Route = createFileRoute("/_auth/spaces/$spaceSlug/")({
  loader: ({ context, params }) =>
    Promise.all([
      context.queryClient.ensureQueryData(spaceQueryOptions(params.spaceSlug)),
      context.queryClient.ensureQueryData(spaceTasksQueryOptions(params.spaceSlug)),
    ]),
  errorComponent: ({ error }) => <ErrorDisplay error={error} />,
  pendingComponent: () => <p className="text-surface-500">Loading...</p>,
  component: TaskListPage,
});

function TaskListPage() {
  const { spaceSlug } = Route.useParams();
  const { data: space } = useSuspenseQuery(spaceQueryOptions(spaceSlug));
  const { data: tasks } = useSuspenseQuery(spaceTasksQueryOptions(spaceSlug));

  return (
    <div className="space-y-4">
      <div>
        <Link to="/spaces" className="text-sm text-surface-500 hover:underline">
          &larr; Spaces
        </Link>
        <h1 className="h2">{space.name}</h1>
        {space.description ? <p className="text-surface-500">{space.description}</p> : null}
      </div>

      {tasks.items.length === 0 ? (
        <p className="text-surface-500">No tasks yet.</p>
      ) : (
        <div className="table-wrap">
          <table className="table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Title</th>
                <th>Status</th>
                <th>Effort</th>
                <th>Priority</th>
                <th>Due</th>
              </tr>
            </thead>
            <tbody>
              {tasks.items.map((task) => (
                <tr key={task.id}>
                  <td className="font-mono text-sm text-surface-500">{task.id}</td>
                  <td>
                    <Link
                      to="/spaces/$spaceSlug/tasks/$taskId"
                      params={{ spaceSlug, taskId: task.id }}
                      className="hover:underline"
                    >
                      {task.title}
                    </Link>
                  </td>
                  <td>
                    <span className="badge">{task.status}</span>
                  </td>
                  <td>
                    {task.effort ? (
                      <span className="badge">{task.effort}</span>
                    ) : (
                      <span className="text-surface-500">&mdash;</span>
                    )}
                  </td>
                  <td>
                    {task.priority ? (
                      <span className="badge">{task.priority}</span>
                    ) : (
                      <span className="text-surface-500">&mdash;</span>
                    )}
                  </td>
                  <td>
                    {task.due ? (
                      <span className="text-sm">{task.due.at}</span>
                    ) : (
                      <span className="text-surface-500">&mdash;</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {tasks.nextCursor ? (
        <p className="text-sm text-surface-500">There are more tasks not shown on this page.</p>
      ) : null}
    </div>
  );
}

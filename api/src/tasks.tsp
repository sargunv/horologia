import "@typespec/http";

using Http;

namespace Tend;

model Task {
  id: string;
  title: string;
  description: string;
  status: string;
  assigneeIds: string[];
  tags: string[];
  dueDate: plainDate | null;
  createdAt: utcDateTime;
  updatedAt: utcDateTime;
}

model TaskCreate {
  title: string;
  description?: string;
  status?: string;
  @maxItems(100) assigneeIds?: string[];
  @maxItems(100) tags?: string[];
  dueDate?: plainDate | null;
}

model TaskUpdate {
  title?: string;
  description?: string;
  status?: string;
  @maxItems(100) assigneeIds?: string[];
  @maxItems(100) tags?: string[];
  dueDate?: plainDate | null;
}

model TaskListParams {
  ...PageParams;
}

model TaskPage is Page<Task>;

@route("/spaces/{spaceSlug}/tasks")
@tag("Tasks")
interface SpaceTasks {
  @get list(@path spaceSlug: string, ...TaskListParams): TaskPage | ApiError;

  @post create(@path spaceSlug: string, @body body: TaskCreate): {
    @statusCode statusCode: 201;
    @body body: Task;
  } | ApiError;
}

@route("/tasks")
@tag("Tasks")
interface Tasks {
  @get read(@path taskId: string): Task | ApiError;

  @patch update(@path taskId: string, @body body: TaskUpdate): Task | ApiError;

  @delete delete(@path taskId: string): void | ApiError;
}

import "@typespec/http";

using Http;

namespace Tend;

enum StatusCategory {
  initial,
  intermediate,
  completion,
}

model TaskStatus {
  name: string;
  category: StatusCategory;
}

model Task {
  id: string;
  title: string;
  description: string;
  status: TaskStatus;
  dueDate: plainDate | null;
  createdAt: utcDateTime;
  updatedAt: utcDateTime;
}

model TaskCreate {
  title: string;
  description?: string;
  statusName?: string;
  dueDate?: plainDate | null;
}

model TaskUpdate {
  title?: string;
  description?: string;
  statusName?: string;
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

import "@typespec/http";

using Http;

namespace Tend;

enum TaskStatusCategory {
  initial,
  intermediate,
  completion,
}

model TaskStatus {
  name: string;
  category: TaskStatusCategory;
  position: int64;
}

model TaskStatusList {
  items: TaskStatus[];
}

model TaskStatusInput {
  @minLength(1) @maxLength(100) name: string;
  category: TaskStatusCategory;
}

model TaskStatusReplace {
  items: TaskStatusInput[];
}

@route("/spaces/{spaceSlug}/task-statuses")
@tag("Task Statuses")
interface SpaceTaskStatuses {
  @get list(@path spaceSlug: string): TaskStatusList | ApiError;

  @put replace(
    @path spaceSlug: string,
    @body body: TaskStatusReplace,
  ): TaskStatusList | ApiError;
}

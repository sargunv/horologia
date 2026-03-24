import "@typespec/http";

using Http;

namespace Tend;

model TaskPriorityLevel {
  name: string;
  position: int64;
}

model TaskPriorityLevelList {
  items: TaskPriorityLevel[];
}

@route("/spaces/{spaceSlug}/task-priority-levels")
@tag("Task Priority Levels")
interface SpaceTaskPriorityLevels {
  @get list(@path spaceSlug: string): TaskPriorityLevelList | ApiError;
}

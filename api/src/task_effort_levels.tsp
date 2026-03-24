import "@typespec/http";

using Http;

namespace Tend;

model TaskEffortLevel {
  name: string;
  position: int64;
}

model TaskEffortLevelList {
  items: TaskEffortLevel[];
}

@route("/spaces/{spaceSlug}/task-effort-levels")
@tag("Task Effort Levels")
interface SpaceTaskEffortLevels {
  @get list(@path spaceSlug: string): TaskEffortLevelList | ApiError;
}

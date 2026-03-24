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

model TaskEffortLevelInput {
  @minLength(1) @maxLength(100) name: string;
}

model TaskEffortLevelReplace {
  items: TaskEffortLevelInput[];
}

@route("/spaces/{spaceSlug}/task-effort-levels")
@tag("Task Effort Levels")
interface SpaceTaskEffortLevels {
  @get list(@path spaceSlug: string): TaskEffortLevelList | ApiError;

  @put replace(
    @path spaceSlug: string,
    @body body: TaskEffortLevelReplace,
  ): TaskEffortLevelList | ApiError;
}

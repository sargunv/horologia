import "@typespec/http";
import "@tend/typespec-mcp-go";

using Http;
using MCP;

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
  @mcpTool("effort_level_list", "List task effort levels in a space.")
  @get
  list(
    /** Slug of the space. */
    @path spaceSlug: string,
  ): TaskEffortLevelList | ApiError;

  @put replace(
    @path spaceSlug: string,
    @body body: TaskEffortLevelReplace,
  ): TaskEffortLevelList | ApiError;
}

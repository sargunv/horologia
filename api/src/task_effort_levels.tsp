import "@typespec/http";
import "@tend/typespec-mcp-go";

using Http;
using MCP;

namespace Tend;

model TaskEffortLevel {
  name: string;
  position: int64;

  /** Lucide icon name in kebab-case (e.g. "gauge", "flame"). Empty string if no icon is set. */
  icon: string;
}

model TaskEffortLevelList {
  items: TaskEffortLevel[];
}

model TaskEffortLevelInput {
  @minLength(1) @maxLength(100) name: string;

  /** Lucide icon name in kebab-case (e.g. "gauge", "flame"). Omit or set to empty string to clear. */
  @maxLength(50) icon?: string;
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

  @mcpTool("effort_level_replace", "Replace all task effort levels in a space.")
  @put
  replace(
    /** Slug of the space. */
    @path spaceSlug: string,

    @body body: TaskEffortLevelReplace,
  ): TaskEffortLevelList | ApiError;
}

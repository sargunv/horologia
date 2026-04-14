import "@typespec/http";
import "@horologia/typespec-mcp-go";

using Http;
using MCP;

namespace Horologia;

model TaskPriorityLevel {
  name: string;
  position: int64;

  /** Lucide icon name in kebab-case (e.g. "signal-high", "flag"). Empty string if no icon is set. */
  icon: string;
}

model TaskPriorityLevelList {
  items: TaskPriorityLevel[];
}

model TaskPriorityLevelInput {
  @minLength(1) @maxLength(100) name: string;

  /** Lucide icon name in kebab-case (e.g. "signal-high", "flag"). Omit or set to empty string to clear. */
  @maxLength(50) icon?: string;
}

model TaskPriorityLevelReplace {
  items: TaskPriorityLevelInput[];
}

@route("/spaces/{spaceSlug}/task-priority-levels")
@tag("Task Priority Levels")
interface SpaceTaskPriorityLevels {
  @mcpTool("priority_level_list", "List task priority levels in a space.")
  @get
  list(
    /** Slug of the space. */
    @path spaceSlug: string,
  ): TaskPriorityLevelList | ApiError;

  @mcpTool(
    "priority_level_replace",
    "Replace all task priority levels in a space."
  )
  @put
  replace(
    /** Slug of the space. */
    @path spaceSlug: string,

    @body body: TaskPriorityLevelReplace,
  ): TaskPriorityLevelList | ApiError;
}

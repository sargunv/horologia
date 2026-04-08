import "@typespec/http";
import "@tend/typespec-mcp-go";

using Http;
using MCP;

namespace Tend;

model TaskPriorityLevel {
  name: string;
  position: int64;
}

model TaskPriorityLevelList {
  items: TaskPriorityLevel[];
}

model TaskPriorityLevelInput {
  @minLength(1) @maxLength(100) name: string;
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

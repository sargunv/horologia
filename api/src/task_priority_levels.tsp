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
  list(@path spaceSlug: string): TaskPriorityLevelList | ApiError;

  @put replace(
    @path spaceSlug: string,
    @body body: TaskPriorityLevelReplace,
  ): TaskPriorityLevelList | ApiError;
}

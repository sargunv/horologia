import "@typespec/http";
import "@tend/typespec-mcp-go";

using Http;
using MCP;

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
  @mcpTool("status_list", "List task statuses in a space.")
  @get
  list(
    /** Slug of the space. */
    @path spaceSlug: string,
  ): TaskStatusList | ApiError;

  @mcpTool("status_replace", "Replace all task statuses in a space.")
  @put
  replace(
    /** Slug of the space. */
    @path spaceSlug: string,

    @body body: TaskStatusReplace,
  ): TaskStatusList | ApiError;
}

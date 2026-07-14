import "@typespec/http";
import "@horologia/typespec-mcp-go";

using Http;
using MCP;

namespace Horologia;

enum ActivityEntityType {
  task,
  recipe,
  space,
  member,
  tag,
  status,
  effort_level,
  priority_level,
  relation,
}

enum ActivityAction {
  created,
  updated,
  deleted,
}

model ActivityDetail {
  field: string;
  from: string | null;
  to: string | null;
}

model ActivityLogEntry {
  id: string;
  spaceSlug: string;
  actorId: string | null;
  tokenId: string | null;
  tokenName: string | null;
  entityType: ActivityEntityType;
  entityId: string;
  action: ActivityAction;
  details: ActivityDetail[];
  createdAt: utcDateTime;
}

model ActivityLogPage is Page<ActivityLogEntry>;

@route("/spaces/{spaceSlug}/tasks/{taskId}/activity")
@tag("Activity")
interface SpaceTaskActivity {
  @mcpTool(
    "task_activity_list",
    "List activity log entries for a specific task."
  )
  @get
  list(
    /** Slug of the space. */
    @path spaceSlug: string,

    /** Task ID. */
    @path taskId: string,

    ...PageParams,
  ): ActivityLogPage | ApiError;
}

@route("/spaces/{spaceSlug}/recipes/{recipeId}/activity")
@tag("Activity")
interface SpaceRecipeActivity {
  @mcpTool(
    "recipe_activity_list",
    "List activity log entries for a specific recipe."
  )
  @get
  list(@path spaceSlug: string, @path recipeId: string, ...PageParams):
    | ActivityLogPage
    | ApiError;
}

@route("/spaces/{spaceSlug}/activity")
@tag("Activity")
interface SpaceActivity {
  @mcpTool("space_activity_list", "List activity log entries for a space.")
  @get
  list(
    /** Slug of the space. */
    @path spaceSlug: string,

    ...PageParams,
  ): ActivityLogPage | ApiError;
}

@route("/users/{userId}/activity")
@tag("Activity")
interface UserActivity {
  @mcpTool("user_activity_list", "List activity log entries for a user.")
  @get
  list(
    /** User ID. */
    @path userId: string,

    ...PageParams,
  ): ActivityLogPage | ApiError;
}

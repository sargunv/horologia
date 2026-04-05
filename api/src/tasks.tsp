import "@typespec/http";
import "@tend/typespec-mcp-go";

using Http;
using MCP;

namespace Tend;

enum TaskRecurrenceType {
  one_off,
  completion_based,
  fixed_non_accumulating,
  fixed_accumulating,
  on_dependency,
}

enum TaskOverdueAction {
  advance_recurrence,
  set_status,
  clear_due_date,
}

model TaskOverdueActionRule {
  /** Grace period in days after the due date before the action fires.
      null means act immediately when the task becomes overdue. */
  after: int32 | null;

  action: TaskOverdueAction;

  /** Required when action is set_status; the status name to transition to. */
  @minLength(1) @maxLength(100) status?: string;
}

enum TaskRelationKind {
  parent_of,
  child_of,
  blocks,
  blocked_by,
  relates_to,
  duplicates,
  triggers,
  triggered_by,
  spawns,
  spawned_by,
}

model TaskRelation {
  kind: TaskRelationKind;
  taskId: string;
  createdAt: utcDateTime;
}

model TaskRelationCreate {
  kind: TaskRelationKind;
  taskId: string;
}

model TaskDue {
  at: plainDate;
  @minLength(1) @maxLength(100) timezone: string;
}

model Task {
  id: string;
  spaceSlug: string;
  title: string;
  description: string;
  status: string;
  effort: string | null;
  priority: string | null;
  recurrenceType: TaskRecurrenceType;
  recurrenceRule: string | null;
  lastCompletedAt: utcDateTime | null;
  assigneeIds: string[];
  rotationPool: string[];
  tags: string[];
  relations: TaskRelation[];
  due: TaskDue | null;
  overdueActionRule: TaskOverdueActionRule | null;
  createdAt: utcDateTime;
  updatedAt: utcDateTime;
}

model TaskCreate {
  @minLength(1) @maxLength(500) title: string;
  @maxLength(10000) description?: string;
  status?: string;
  @minLength(1) @maxLength(100) effort?: string;
  @minLength(1) @maxLength(100) priority?: string;
  recurrenceType?: TaskRecurrenceType;
  @minLength(1) @maxLength(500) recurrenceRule?: string;
  @maxItems(100) assigneeIds?: string[];
  @maxItems(100) rotationPool?: string[];
  @maxItems(100) tags?: string[];
  due?: TaskDue | null;
  overdueActionRule?: TaskOverdueActionRule | null;
}

model TaskUpdate {
  @minLength(1) @maxLength(500) title?: string;
  @maxLength(10000) description?: string;
  status?: string;
  @minLength(1) @maxLength(100) effort?: string | null;
  @minLength(1) @maxLength(100) priority?: string | null;
  recurrenceType?: TaskRecurrenceType;
  @minLength(1) @maxLength(500) recurrenceRule?: string | null;
  @maxItems(100) assigneeIds?: string[];
  @maxItems(100) rotationPool?: string[];
  @maxItems(100) tags?: string[];
  due?: TaskDue | null;
  overdueActionRule?: TaskOverdueActionRule | null;
}

model TaskPage is Page<Task>;

@route("/spaces/{spaceSlug}/tasks")
@tag("Tasks")
interface SpaceTasks {
  @mcpTool(
    "task_list",
    "List tasks in a space, with optional cursor pagination. Returns task IDs needed by task_get and task_update."
  )
  @get
  list(@path spaceSlug: string, ...PageParams): TaskPage | ApiError;

  @mcpTool(
    "task_create",
    "Create a new task in a space. Settable: title, description, status, effort, priority. Complex fields (due date, recurrence, assignees, tags) require the web UI."
  )
  @post
  create(@path spaceSlug: string, @body body: TaskCreate): {
    @statusCode statusCode: 201;
    @body body: Task;
  } | ApiError;

  @mcpTool("task_get", "Get a task by ID. Obtain task IDs from task_list.")
  @get
  read(@path spaceSlug: string, @path taskId: string): Task | ApiError;

  @mcpTool(
    "task_update",
    "Update a task's title, description, status, effort, or priority. Obtain task IDs from task_list. Complex fields (due date, recurrence, assignees, tags) require the web UI."
  )
  @patch
  update(
    @path spaceSlug: string,
    @path taskId: string,
    @body body: TaskUpdate,
  ): Task | ApiError;

  @delete delete(
    @path spaceSlug: string,
    @path taskId: string,
  ): void | ApiError;
}

@route("/spaces/{spaceSlug}/tasks/{taskId}/relations")
@tag("Tasks")
interface SpaceTaskRelations {
  @post create(
    @path spaceSlug: string,
    @path taskId: string,
    @body body: TaskRelationCreate,
  ): {
    @statusCode statusCode: 201;
    @body body: TaskRelation;
  } | ApiError;

  @delete delete(
    @path spaceSlug: string,
    @path taskId: string,
    @path kind: TaskRelationKind,
    @path relatedTaskId: string,
  ): void | ApiError;
}

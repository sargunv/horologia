import "@typespec/http";
import "@horologia/typespec-mcp-go";

using Http;
using MCP;

namespace Horologia;

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
  relatedTaskId: string;
  createdAt: utcDateTime;
}

model TaskRelationCreate {
  /** Relation kind (e.g. parent_of, child_of, blocks, blocked_by, relates_to, duplicates, triggers, triggered_by). */
  kind: TaskRelationKind;

  /** Task ID of the related task. */
  relatedTaskId: string;
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
  /** Task title (1–500 chars). */
  @minLength(1) @maxLength(500) title: string;

  /** Task description (max 10000 chars). */
  @maxLength(10000) description?: string;

  /** Initial status name. Defaults to the space's first status. */
  status?: string;

  /** Effort level name. */
  @minLength(1) @maxLength(100) effort?: string;

  /** Priority level name. */
  @minLength(1) @maxLength(100) priority?: string;

  /** Recurrence type (e.g. one_off, completion_based, fixed_accumulating). */
  recurrenceType?: TaskRecurrenceType;

  /** Recurrence rule (RRULE string, max 500 chars). */
  @minLength(1) @maxLength(500) recurrenceRule?: string;

  @maxItems(100) assigneeIds?: string[];
  @maxItems(100) rotationPool?: string[];
  @maxItems(100) tags?: string[];
  due?: TaskDue | null;
  overdueActionRule?: TaskOverdueActionRule | null;
}

model TaskUpdate {
  /** New title. */
  @minLength(1) @maxLength(500) title?: string;

  /** New description. */
  @maxLength(10000) description?: string;

  /** New status name. */
  status?: string;

  /** New effort level name. */
  @minLength(1) @maxLength(100) effort?: string | null;

  /** New priority level name. */
  @minLength(1) @maxLength(100) priority?: string | null;

  /** New recurrence type. */
  recurrenceType?: TaskRecurrenceType;

  /** New recurrence rule (RRULE string, max 500 chars). */
  @minLength(1) @maxLength(500) recurrenceRule?: string | null;

  @maxItems(100) assigneeIds?: string[];
  @maxItems(100) rotationPool?: string[];
  @maxItems(100) tags?: string[];
  due?: TaskDue | null;
  overdueActionRule?: TaskOverdueActionRule | null;
}

model TaskPage is Page<Task>;

model TaskSearchResult {
  id: string;
  spaceSlug: string;
  title: string;
  status: string;
}

model TaskSearchResultList {
  items: TaskSearchResult[];
}

@route("/spaces/{spaceSlug}/tasks")
@tag("Tasks")
interface SpaceTasks {
  @mcpTool(
    "task_list",
    "List tasks in a space, with optional cursor pagination. Returns task IDs needed by task_get and task_update."
  )
  @get
  list(
    /** Slug of the space. */
    @path spaceSlug: string,

    ...PageParams,
  ): TaskPage | ApiError;

  @mcpTool("task_create", "Create a new task in a space.")
  @post
  create(
    /** Slug of the space. */
    @path spaceSlug: string,

    @body body: TaskCreate,
  ):
    | {
        @statusCode statusCode: 201;
        @body body: Task;
      }
    | ApiError;

  @mcpTool("task_get", "Get a task by ID. Obtain task IDs from task_list.")
  @get
  read(
    /** Slug of the space. */
    @path spaceSlug: string,

    /** Task ID. */
    @path taskId: string,
  ): Task | ApiError;

  @mcpTool(
    "task_update",
    "Update an existing task. Obtain task IDs from task_list."
  )
  @patch
  update(
    /** Slug of the space. */
    @path spaceSlug: string,

    /** Task ID. */
    @path taskId: string,

    @body body: TaskUpdate,
  ): Task | ApiError;

  @mcpTool("task_delete", "Delete a task. Obtain task IDs from task_list.")
  @delete
  delete(
    /** Slug of the space. */
    @path spaceSlug: string,

    /** Task ID. */
    @path taskId: string,
  ): void | ApiError;
}

@route("/tasks")
@tag("Tasks")
interface Tasks {
  @mcpTool(
    "task_search",
    "Search tasks visible to the current user across all spaces."
  )
  @get
  @route("/search")
  search(
    /** Search query. */
    @query @minLength(1) q: string,

    /** Optional space slug to restrict results to a single space. */
    @query spaceSlug?: string,

    /** Optional task ID to exclude from results. */
    @query excludeTaskId?: string,

    /** Maximum number of items to return (1–100). */
    @query limit?: int32,
  ): TaskSearchResultList | ApiError;
}

@route("/spaces/{spaceSlug}/tasks/{taskId}/relations")
@tag("Tasks")
interface SpaceTaskRelations {
  @mcpTool(
    "relation_create",
    "Create a relation between two tasks. Obtain task IDs from task_list."
  )
  @post
  create(
    /** Slug of the space. */
    @path spaceSlug: string,

    /** Task ID of the source task. */
    @path taskId: string,

    @body body: TaskRelationCreate,
  ):
    | {
        @statusCode statusCode: 201;
        @body body: TaskRelation;
      }
    | ApiError;

  @mcpTool(
    "relation_delete",
    "Delete a relation between two tasks. Obtain task IDs from task_list."
  )
  @delete
  delete(
    /** Slug of the space. */
    @path spaceSlug: string,

    /** Task ID of the source task. */
    @path taskId: string,

    /** Relation kind. */
    @path kind: TaskRelationKind,

    /** Task ID of the related task. */
    @path relatedTaskId: string,
  ): void | ApiError;
}

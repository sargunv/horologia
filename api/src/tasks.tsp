import "@typespec/http";

using Http;

namespace Tend;

enum TaskRelationKind {
  parent_of,
  child_of,
  blocks,
  blocked_by,
  relates_to,
  duplicates,
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

model Task {
  id: string;
  title: string;
  description: string;
  status: string;
  effort: string | null;
  priority: string | null;
  assigneeIds: string[];
  tags: string[];
  relations: TaskRelation[];
  dueDate: plainDate | null;
  createdAt: utcDateTime;
  updatedAt: utcDateTime;
}

model TaskCreate {
  @minLength(1) @maxLength(500) title: string;
  @maxLength(10000) description?: string;
  status?: string;
  @minLength(1) @maxLength(100) effort?: string;
  @minLength(1) @maxLength(100) priority?: string;
  @maxItems(100) assigneeIds?: string[];
  @maxItems(100) tags?: string[];
  dueDate?: plainDate | null;
}

model TaskUpdate {
  @minLength(1) @maxLength(500) title?: string;
  @maxLength(10000) description?: string;
  status?: string;
  @minLength(1) @maxLength(100) effort?: string | null;
  @minLength(1) @maxLength(100) priority?: string | null;
  @maxItems(100) assigneeIds?: string[];
  @maxItems(100) tags?: string[];
  dueDate?: plainDate | null;
}

model TaskPage is Page<Task>;

@route("/spaces/{spaceSlug}/tasks")
@tag("Tasks")
interface SpaceTasks {
  @get list(@path spaceSlug: string, ...PageParams): TaskPage | ApiError;

  @post create(@path spaceSlug: string, @body body: TaskCreate): {
    @statusCode statusCode: 201;
    @body body: Task;
  } | ApiError;

  @get read(@path spaceSlug: string, @path taskId: string): Task | ApiError;

  @patch update(
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

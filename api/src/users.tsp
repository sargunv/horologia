import "@typespec/http";
import "@horologia/typespec-mcp-go";

using Http;
using MCP;

namespace Horologia;

model User {
  id: string;
  email: string;
  name: string;
  isOwner: boolean;
  hasPassword: boolean;
  createdAt: utcDateTime;
  updatedAt: utcDateTime;
}

model UserCreate {
  @minLength(1) @maxLength(200) name: string;
  @minLength(1) @maxLength(200) email: string;
  isOwner?: boolean;
  @minLength(8) @maxLength(72) password?: string;
}

model UserUpdate {
  @minLength(1) @maxLength(200) name?: string;
  @minLength(1) @maxLength(200) email?: string;
  isOwner?: boolean;

  // Mutually exclusive: set a new password or clear the existing one.
  @minLength(8) @maxLength(72) setPassword?: string;

  clearPassword?: boolean;
}

model UserList {
  items: User[];
}

@route("/users")
@tag("Users")
interface Users {
  @mcpTool("user_list", "List all users. Requires owner privileges.")
  @get
  list(): UserList | ApiError;

  @mcpTool("user_me", "Get the currently authenticated user.")
  @get
  @route("/me")
  me(): User | ApiError;

  @mcpTool("user_get", "Get a user by ID.")
  @get
  get(
    /** User ID. */
    @path userId: string,
  ): User | ApiError;

  @mcpTool(
    "user_create",
    "Create a new user with name, email, and optional password."
  )
  @post
  create(@body body: UserCreate):
    | {
        @statusCode statusCode: 201;
        @body body: User;
      }
    | ApiError;

  @mcpTool(
    "user_update",
    "Update a user's name, email, owner status, or password."
  )
  @patch
  update(
    /** User ID. */
    @path userId: string,

    @body body: UserUpdate,
  ): User | ApiError;

  @mcpTool("user_delete", "Delete a user by ID.")
  @delete
  delete(
    /** User ID. */
    @path userId: string,
  ): void | ApiError;
}

@route("/users/{userId}/tasks")
@tag("Users")
interface UserTasks {
  @mcpTool("user_task_list", "List tasks assigned to a user across all spaces.")
  @get
  list(
    /** User ID. */
    @path userId: string,

    ...PageParams,
  ): TaskPage | ApiError;
}

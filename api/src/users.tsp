import "@typespec/http";

using Http;

namespace Tend;

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
  @get list(): UserList | ApiError;
  @get @route("/me") me(): User | ApiError;
  @get get(@path userId: string): User | ApiError;

  @post create(@body body: UserCreate): {
    @statusCode statusCode: 201;
    @body body: User;
  } | ApiError;

  @patch update(@path userId: string, @body body: UserUpdate): User | ApiError;

  @delete delete(@path userId: string): void | ApiError;
}

@route("/users/{userId}/tasks")
@tag("Users")
interface UserTasks {
  @get list(@path userId: string, ...PageParams): TaskPage | ApiError;
}

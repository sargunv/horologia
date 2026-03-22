import "@typespec/http";

using Http;

namespace Tend;

model User {
  id: string;
  email: string;
  name: string;
  isOwner: boolean;
  createdAt: utcDateTime;
  updatedAt: utcDateTime;
}

@route("/users")
@tag("Users")
interface Users {
  @get
  @route("/me")
  me(): User | ApiError;
}

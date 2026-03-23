import "@typespec/http";

import "./src/lib/common.tsp";
import "./src/auth.tsp";
import "./src/users.tsp";
import "./src/spaces.tsp";
import "./src/members.tsp";
import "./src/tags.tsp";
import "./src/tasks.tsp";

using Http;

@service(#{ title: "Tend API" })
@useAuth(BearerAuth)
namespace Tend;

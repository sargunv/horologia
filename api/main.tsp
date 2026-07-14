import "@typespec/http";

import "./src/lib/common.tsp";
import "./src/auth.tsp";
import "./src/users.tsp";
import "./src/spaces.tsp";
import "./src/members.tsp";
import "./src/tags.tsp";
import "./src/tasks.tsp";
import "./src/recipes.tsp";
import "./src/task_statuses.tsp";
import "./src/task_effort_levels.tsp";
import "./src/task_priority_levels.tsp";
import "./src/activity_log.tsp";

using Http;

@service(#{ title: "Horologia API" })
@server("/api", "Same-origin API")
@useAuth(BearerAuth)
namespace Horologia;

import "@typespec/http";

using Http;

namespace Tend;

enum ActivityEntityType {
  task,
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
  @get list(
    @path spaceSlug: string,
    @path taskId: string,
    ...PageParams,
  ): ActivityLogPage | ApiError;
}

@route("/spaces/{spaceSlug}/activity")
@tag("Activity")
interface SpaceActivity {
  @get list(@path spaceSlug: string, ...PageParams): ActivityLogPage | ApiError;
}

@route("/users/{userId}/activity")
@tag("Activity")
interface UserActivity {
  @get list(@path userId: string, ...PageParams): ActivityLogPage | ApiError;
}

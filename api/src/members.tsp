import "@typespec/http";

using Http;

namespace Tend;

enum SpaceRole {
  admin,
  member,
  viewer,
}

model SpaceMember {
  userId: string;
  userName: string;
  userEmail: string;
  role: SpaceRole;
  createdAt: utcDateTime;
}

model SpaceMemberCreate {
  userId: string;
  role: SpaceRole;
}

model SpaceMemberUpdate {
  role: SpaceRole;
}

model SpaceMemberPage is Page<SpaceMember>;

@route("/spaces/{spaceSlug}/members")
@tag("SpaceMembers")
interface SpaceMembers {
  @get list(@path spaceSlug: string, ...PageParams): SpaceMemberPage | ApiError;

  @post create(@path spaceSlug: string, @body body: SpaceMemberCreate): {
    @statusCode statusCode: 201;
    @body body: SpaceMember;
  } | ApiError;

  @patch update(
    @path spaceSlug: string,
    @path userId: string,
    @body body: SpaceMemberUpdate,
  ): SpaceMember | ApiError;

  @delete delete(
    @path spaceSlug: string,
    @path userId: string,
  ): void | ApiError;
}

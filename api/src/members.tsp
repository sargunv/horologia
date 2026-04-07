import "@typespec/http";
import "@tend/typespec-mcp-go";

using Http;
using MCP;

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

model SpaceMemberList {
  items: SpaceMember[];
}

@route("/spaces/{spaceSlug}/members")
@tag("SpaceMembers")
interface SpaceMembers {
  @mcpTool("member_list", "List members of a space.")
  @get
  list(@path spaceSlug: string): SpaceMemberList | ApiError;

  @mcpTool(
    "member_create",
    "Add a user to a space with a role (admin, member, or viewer)."
  )
  @post
  create(@path spaceSlug: string, @body body: SpaceMemberCreate): {
    @statusCode statusCode: 201;
    @body body: SpaceMember;
  } | ApiError;

  @mcpTool(
    "member_update",
    "Change a member's role in a space. Obtain user IDs from member_list."
  )
  @patch
  update(
    @path spaceSlug: string,
    @path userId: string,
    @body body: SpaceMemberUpdate,
  ): SpaceMember | ApiError;

  @mcpTool(
    "member_delete",
    "Remove a member from a space. Obtain user IDs from member_list."
  )
  @delete
  delete(@path spaceSlug: string, @path userId: string): void | ApiError;
}

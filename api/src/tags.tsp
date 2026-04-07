import "@typespec/http";
import "@tend/typespec-mcp-go";

using Http;
using MCP;

namespace Tend;

model Tag {
  name: string;
  createdAt: utcDateTime;
}

model TagCreate {
  @minLength(1) @maxLength(100) name: string;
}

model TagUpdate is TagCreate;

model TagList {
  items: Tag[];
}

@route("/spaces/{spaceSlug}/tags")
@tag("Tags")
interface SpaceTags {
  @mcpTool("tag_list", "List all tags in a space.")
  @get
  list(
    /** Slug of the space. */
    @path spaceSlug: string,
  ): TagList | ApiError;

  @mcpTool("tag_create", "Create a tag in a space.")
  @post
  create(
    /** Slug of the space. */
    @path spaceSlug: string,

    @body body: TagCreate,
  ): {
    @statusCode statusCode: 201;
    @body body: Tag;
  } | ApiError;

  @mcpTool("tag_update", "Rename a tag. Obtain tag names from tag_list.")
  @patch
  update(
    /** Slug of the space. */
    @path spaceSlug: string,

    /** Current tag name. */
    @path tagName: string,

    @body body: TagUpdate,
  ): Tag | ApiError;

  @mcpTool(
    "tag_delete",
    "Delete a tag from a space. Obtain tag names from tag_list."
  )
  @delete
  delete(
    /** Slug of the space. */
    @path spaceSlug: string,

    /** Tag name to delete. */
    @path tagName: string,
  ): void | ApiError;
}

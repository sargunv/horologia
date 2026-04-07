import "@typespec/http";
import "@tend/typespec-mcp-go";

using Http;
using MCP;

namespace Tend;

model Space {
  slug: string;
  name: string;
  description: string;
  createdAt: utcDateTime;
  updatedAt: utcDateTime;
}

model SpaceCreate {
  // Keep in sync with frontend: web/src/routes/_authenticated/spaces/new.tsx (toSlug)
  @minLength(1)
  @maxLength(100)
  @pattern("^[\\p{L}0-9]+(?:-[\\p{L}0-9]+)*$")
  slug: string;

  @minLength(1) @maxLength(200) name: string;
  @maxLength(1000) description?: string;
}

model SpaceUpdate {
  // Keep in sync with frontend: web/src/routes/_authenticated/spaces/new.tsx (toSlug)
  @minLength(1)
  @maxLength(100)
  @pattern("^[\\p{L}0-9]+(?:-[\\p{L}0-9]+)*$")
  slug?: string;

  @minLength(1) @maxLength(200) name?: string;
  @maxLength(1000) description?: string;
}

model SpaceList {
  items: Space[];
}

@route("/spaces")
@tag("Spaces")
interface Spaces {
  @mcpTool("space_list", "List all spaces the authenticated user can access.")
  @get
  list(): SpaceList | ApiError;

  @mcpTool(
    "space_create",
    "Create a new space with a slug, name, and optional description."
  )
  @post
  create(@body body: SpaceCreate): {
    @statusCode statusCode: 201;
    @body body: Space;
  } | ApiError;

  @mcpTool("space_get", "Get a space by slug.")
  @get
  read(
    /** Slug of the space. */
    @path spaceSlug: string,
  ): Space | ApiError;

  @mcpTool("space_update", "Update a space's slug, name, or description.")
  @patch
  update(
    /** Slug of the space. */
    @path spaceSlug: string,

    @body body: SpaceUpdate,
  ): Space | ApiError;

  @mcpTool("space_delete", "Delete a space by slug.")
  @delete
  delete(
    /** Slug of the space. */
    @path spaceSlug: string,
  ): void | ApiError;
}

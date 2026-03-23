import "@typespec/http";

using Http;

namespace Tend;

model Space {
  slug: string;
  name: string;
  description: string;
  createdAt: utcDateTime;
  updatedAt: utcDateTime;
}

model SpaceCreate {
  @minLength(1) @maxLength(100) slug: string;
  @minLength(1) @maxLength(200) name: string;
  @maxLength(1000) description?: string;
}

model SpaceUpdate {
  @minLength(1) @maxLength(200) name?: string;
  @maxLength(1000) description?: string;
}

model SpacePage is Page<Space>;

@route("/spaces")
@tag("Spaces")
interface Spaces {
  @get list(...PageParams): SpacePage | ApiError;

  @post create(@body body: SpaceCreate): {
    @statusCode statusCode: 201;
    @body body: Space;
  } | ApiError;

  @get read(@path spaceSlug: string): Space | ApiError;

  @patch update(
    @path spaceSlug: string,
    @body body: SpaceUpdate,
  ): Space | ApiError;

  @delete delete(@path spaceSlug: string): void | ApiError;
}

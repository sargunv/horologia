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
  @get list(): SpaceList | ApiError;

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

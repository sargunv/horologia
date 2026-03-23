import "@typespec/http";

using Http;

namespace Tend;

model Tag {
  name: string;
  createdAt: utcDateTime;
}

model TagCreate {
  @maxLength(100) name: string;
}

model TagUpdate is TagCreate;

model TagPage is Page<Tag>;

@route("/spaces/{spaceSlug}/tags")
@tag("Tags")
interface SpaceTags {
  @get list(@path spaceSlug: string, ...PageParams): TagPage | ApiError;

  @post create(@path spaceSlug: string, @body body: TagCreate): {
    @statusCode statusCode: 201;
    @body body: Tag;
  } | ApiError;

  @patch update(
    @path spaceSlug: string,
    @path tagName: string,
    @body body: TagUpdate,
  ): Tag | ApiError;

  @delete delete(
    @path spaceSlug: string,
    @path tagName: string,
  ): void | ApiError;
}

import "@typespec/http";

using Http;

namespace Tend;

@error
model ApiError {
  code: string;
  message: string;
}

model Page<T> {
  items: T[];
  nextCursor: string | null;
}

model PageParams {
  /** Pagination cursor from a previous response. */
  @query cursor?: string;

  /** Maximum number of items to return (1–100). */
  @query limit?: int32;
}

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
  @query cursor?: string;
  @query limit?: int32;
}

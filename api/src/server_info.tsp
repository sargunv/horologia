import "@typespec/http";

using Http;

namespace Horologia;

enum ServerCapability {
  oauth21Pkce: "oauth-2.1-pkce",
}

model ServerInfoResponse {
  apiVersion: 1;
  capabilities: ServerCapability[];
}

@route("/server-info")
@tag("Server")
@useAuth(NoAuth)
interface ServerInfo {
  /** Read the API compatibility version and optional client capabilities. */
  @get
  get(): ServerInfoResponse | ApiError;
}

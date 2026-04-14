import "@typespec/http";
import "@horologia/typespec-mcp-go";

using Http;
using MCP;

namespace Horologia;

enum AuthTokenKind {
  session,
  api,
}

model AuthToken {
  id: string;
  name: string;
  kind: AuthTokenKind;
  createdAt: utcDateTime;
}

model AuthTokenCreate {
  @minLength(1) @maxLength(100) name: string;
}

model AuthTokenCreateResponse {
  token: string;
  authToken: AuthToken;
}

model AuthTokenList {
  items: AuthToken[];
}

@route("/auth")
@tag("Auth")
interface Auth {
  @mcpTool("auth_token_list", "List all API tokens for the authenticated user.")
  @get
  @route("/tokens")
  listTokens(): AuthTokenList | ApiError;

  @mcpTool("auth_token_create", "Create a new API token with a name.")
  @post
  @route("/tokens")
  createToken(@body body: AuthTokenCreate): {
    @statusCode statusCode: 201;
    @body body: AuthTokenCreateResponse;
  } | ApiError;

  @mcpTool("auth_token_delete", "Delete an API token by ID.")
  @delete
  @route("/tokens/{tokenId}")
  deleteToken(
    /** Token ID. */
    @path tokenId: string,
  ): void | ApiError;
}

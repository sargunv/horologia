import "@typespec/http";

using Http;

namespace Tend;

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
  @get
  @route("/tokens")
  listTokens(): AuthTokenList | ApiError;

  @post
  @route("/tokens")
  createToken(@body body: AuthTokenCreate): {
    @statusCode statusCode: 201;
    @body body: AuthTokenCreateResponse;
  } | ApiError;

  @delete
  @route("/tokens/{tokenId}")
  deleteToken(@path tokenId: string): void | ApiError;
}

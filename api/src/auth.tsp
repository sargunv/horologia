import "@typespec/http";

using Http;

namespace Tend;

model LoginRequest {
  email: string;
  password: string;
}

model LoginResponse {
  token: string;
  user: User;
}

model AuthToken {
  id: string;
  name: string;
  kind: "session" | "api";
  createdAt: utcDateTime;
}

model AuthTokenCreate {
  name: string;
}

model AuthTokenCreateResponse {
  token: string;
  authToken: AuthToken;
}

model AuthTokenPage is Page<AuthToken>;

@route("/auth")
@tag("Auth")
interface Auth {
  @useAuth(NoAuth)
  @post
  @route("/login")
  login(@body body: LoginRequest): LoginResponse | ApiError;

  @get
  @route("/tokens")
  listTokens(...PageParams): AuthTokenPage | ApiError;

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

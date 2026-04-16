import "@typespec/http";
import "@typespec/openapi";
import "@horologia/typespec-mcp-go";

using Http;
using MCP;
using OpenAPI;

namespace Horologia;

model AuthConfigOIDC {
  enabled: boolean;
  label: string;
  autoRedirect: boolean;
}

model AuthConfigPassword {
  enabled: boolean;
}

model AuthConfig {
  oidc: AuthConfigOIDC;
  password: AuthConfigPassword;
}

model AuthLoginRequest {
  @minLength(1) email: string;
  @minLength(1) password: string;
}

model AuthLoginResponse {
  user: User;
}

model AuthLinkPendingResponse {
  email: string;
  name: string;
}

model AuthLinkRequest {
  @minLength(1) password: string;
}

model AuthLinkResponse {
  redirectTo: string;
}

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

/**
 * Public authentication endpoints used by the first-party web app.
 *
 * These routes do not require bearer auth. Successful login issues the
 * `horologia_session` cookie, which same-origin browser requests can use for
 * authenticated API calls.
 */
@route("/app/auth")
@tag("Auth")
@useAuth(NoAuth)
interface WebAuth {
  /**
   * Read public authentication configuration used by the web UI before login.
   */
  @extension("x-internal", true)
  @get
  @route("/config")
  config(): AuthConfig | ApiError;

  /**
   * Log in with email and password.
   *
   * Returns the authenticated user and sets the `horologia_session` cookie on
   * success. This endpoint is only available when password auth is enabled.
   */
  @extension("x-internal", true)
  @post
  @route("/login")
  login(@body body: AuthLoginRequest): {
    @statusCode statusCode: 200;
    @header("Set-Cookie") setCookie: string;
    @body body: AuthLoginResponse;
  } | ApiError;

  /**
   * Clear the current browser session.
   */
  @extension("x-internal", true)
  @post
  @route("/logout")
  logout(): {
    @statusCode statusCode: 204;
    @header("Set-Cookie") setCookie: string;
  } | ApiError;

  /**
   * Read the pending OIDC account-link request created during the OIDC callback.
   *
   * This endpoint relies on the temporary `horologia_oidc_link` cookie and is
   * only available when OIDC link consent is enabled.
   */
  @extension("x-internal", true)
  @get
  @route("/link/pending")
  linkPending(): AuthLinkPendingResponse | ApiError;

  /**
   * Confirm OIDC account linking using the existing account password.
   *
   * This endpoint relies on the temporary `horologia_oidc_link` cookie issued
   * during the OIDC callback. On success it creates the normal
   * `horologia_session` cookie and clears the pending link cookie. This
   * endpoint is only available when OIDC link consent is enabled.
   */
  @extension("x-internal", true)
  @post
  @route("/link")
  link(@body body: AuthLinkRequest): {
    @statusCode statusCode: 200;
    @header("Set-Cookie") setCookie: string[];
    @body body: AuthLinkResponse;
  } | ApiError;
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

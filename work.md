# OAuth / Delegated Access Work Plan

This document tracks the implementation plan for:

- `SV-45` OAuth 2.1 authorization flow for Tend clients
- `SV-46` Enforce delegated access for OAuth-issued API tokens
- `SV-120` CLI login, logout, and credential management

## Goals

- Let browser-capable external clients authenticate to Tend without sharing passwords or long-lived
  personal API tokens.
- Keep Tend's existing human authentication modes:
  - password login
  - upstream OIDC login
- Add Tend's own OAuth authorization server for external clients.
- Treat MCP as another API transport, not as a special permission domain. MCP access tokens must
  carry the same delegated scopes as REST access tokens, and MCP tools must enforce the same
  underlying permissions.

## Architectural Direction

### Authentication layers

- Human authentication remains:
  - web password login
  - web OIDC login via upstream provider
- Client authorization becomes:
  - OAuth 2.1 authorization code + PKCE for browser-based public clients
  - refresh token rotation for long-lived CLI sessions

### Token model

- Keep opaque DB-backed tokens for v1.
- Do not introduce JWT access tokens yet.
- Extend the current auth token system so the server can validate:
  - session tokens
  - personal API tokens
  - OAuth access tokens
  - OAuth refresh tokens

### Delegated access model

- Existing session tokens and personal API tokens remain full-trust first-party credentials.
- OAuth-issued access tokens are delegated credentials and must be scope-limited.
- Authorization is the intersection of:
  - the user's real permissions
  - the scopes granted to the OAuth client

### Scope model

Initial scope set should align to API capabilities rather than transport:

- `profile:read`
- `spaces:read`
- `spaces:write`
- `tasks:read`
- `tasks:write`
- `tags:read`
- `tags:write`
- `activity:read`
- `users:read`
- `users:write`
- `admin:read`
- `admin:write`

Notes:

- MCP should request the same scopes needed by the tools it exposes.
- There should be no `mcp` scope.
- REST and MCP should share the same delegated-access enforcement rules.

## Delivery Order

### 1. Shared auth foundation

- Expand auth token persistence to represent delegated OAuth credentials.
- Add shared token lookup/validation code used by:
  - ogen bearer auth for REST
  - MCP transport auth
- Extend request auth context with delegated metadata:
  - token kind
  - client id
  - granted scopes
  - audience/resource

Status:

- Completed.
- Implemented delegated token metadata in `auth_tokens`.
- Added shared bearer-token authentication used by both REST and MCP.
- Preserved existing personal token UX by excluding OAuth-issued tokens from the current
  token-management endpoints.

### 2. OAuth server primitives

- Add OAuth persistence for:
  - registered clients
  - authorization codes
  - refresh tokens
  - consent grants
- Implement:
  - `GET /oauth/authorize`
  - `POST /oauth/token`
  - `POST /oauth/revoke`
- Require PKCE `S256` for public/browser-based clients.
- Reuse existing Tend session auth for the user login step.

Status:

- In progress, with the core server flow implemented.
- Added OAuth client, authorization code, and consent grant persistence.
- Added `GET/POST /oauth/authorize`, `POST /oauth/token`, and `POST /oauth/revoke`.
- Implemented authorization code exchange, refresh token rotation, and loopback redirect handling
  for the seeded `tend-cli` client.

### 3. Discovery and protected resource metadata

- Publish authorization server metadata.
- Publish protected resource metadata for the API and MCP resources.
- Return OAuth-aware `401` challenges from protected endpoints when appropriate.
- Include enough metadata for interoperable remote MCP clients.

Status:

- Completed for the initial server implementation.
- Added authorization server metadata plus protected resource metadata for both `/api` and `/mcp`.
- Added OAuth-aware `WWW-Authenticate` challenges to REST and MCP unauthorized responses.

### 4. Delegated access enforcement

- Define operation-to-scope requirements for REST endpoints.
- Define tool-to-operation or tool-to-scope enforcement for MCP tools.
- Keep current full-access behavior for:
  - web session cookies
  - personal API tokens
- Enforce scope-limited behavior only for OAuth-issued tokens.

Status:

- In progress, with shared handler-level enforcement implemented.
- Shared handler-level scope checks now apply to both REST and MCP.
- Delegated OAuth tokens are blocked from personal API token management.

### 5. CLI browser login and local credential management

- Implement `tend auth login` via loopback callback + PKCE.
- Store refresh tokens securely in OS keychain.
- Keep `TEND_TOKEN` as an env override for automation and PAT use.
- Implement:
  - `tend auth login`
  - `tend auth logout`
  - `tend auth status`
- Add automatic access-token refresh in the CLI runtime.

Status:

- In progress.
- Added keychain-backed OAuth credential storage and CLI-side refresh handling.
- Implemented `tend auth login` and `tend auth logout`.
- `tend auth status` now reports keychain-backed credentials through the normal token-source path.
- Browser login still needs end-to-end manual verification against a live server.

## Implementation Slices

### Slice A: token and auth plumbing

- Add schema support for delegated token metadata.
- Refactor token loading into shared auth code.
- Update REST and MCP auth paths to use the same token validation logic.

Status:

- Completed.

### Slice B: OAuth metadata and protocol endpoints

- Add issuer/resource metadata endpoints.
- Add authorization endpoint, token endpoint, revoke endpoint.
- Add consent flow and state handling.

Status:

- Completed for the server MVP.

### Slice C: scope enforcement

- Add centralized scope checks in the REST auth path.
- Apply the same scope semantics to MCP tools.

Status:

- In progress, with shared handler-level enforcement implemented.

### Slice D: CLI integration

- Add browser handoff, callback server, token exchange, refresh, and keychain storage.

Status:

- In progress.

## Open Design Decisions

- Client registration:
  - start with a pre-registered first-party CLI client
  - support remote MCP client interop with standards-compliant client metadata
  - defer broader dynamic registration unless a target integration requires it
- Consent UX:
  - persist grants per user + client + scope set
  - re-prompt when requested scopes expand
- Audience/resource boundaries:
  - likely separate protected resource metadata for `/api` and `/mcp`
  - scopes remain shared across transports

## Automated Test Checklist

### Backend

- [x] Token lookup accepts valid session token, PAT, and OAuth access token.
- [x] Token lookup rejects revoked, expired, malformed, and unknown tokens.
- [x] REST bearer auth preserves current behavior for session tokens and PATs.
- [x] MCP bearer auth preserves current behavior for session tokens and PATs.
- [x] OAuth metadata endpoints return valid issuer/resource metadata.
- [x] `/oauth/authorize` rejects invalid client IDs, redirect URIs, scopes, and PKCE inputs.
- [x] `/oauth/authorize` requires an authenticated Tend session before consent.
- [x] Consent acceptance issues an authorization code bound to user, client, redirect URI, PKCE
      challenge, and scopes.
- [x] `/oauth/token` exchanges a valid code exactly once.
- [x] `/oauth/token` rejects wrong verifier, invalid client, wrong redirect URI, reused code, and
      expired code.
- [x] Refresh token rotation invalidates prior refresh tokens after successful use.
- [x] `/oauth/revoke` revokes access and refresh tokens idempotently.
- [x] OAuth-issued access tokens carry delegated scope metadata.
- [x] REST endpoints return `403` for underscoped OAuth tokens.
- [x] MCP tool calls fail for underscoped OAuth tokens with clear error messages.
- [x] Owner-only endpoints remain owner-gated even when scopes are present.
- [x] Space membership restrictions still apply under delegated tokens.
- [x] Full-trust PAT and session flows are not accidentally scope-limited.

### CLI

- [x] Config resolution still works when only `TEND_SERVER` is set.
- [x] `TEND_TOKEN` overrides keychain-backed OAuth credentials.
- [x] Login stores credentials without writing bearer tokens to plaintext config.
- [x] Logout clears stored OAuth credentials.
- [x] Expired access token triggers refresh and retries the request once.
- [x] Refresh failure surfaces a clean re-login error.

## Manual Test Checklist

### Browser / Web

- [x] Password login still works end-to-end.
- [ ] Upstream OIDC login still works end-to-end.
- [x] Existing SPA authenticated flows still work with session cookies.
- [x] OAuth authorization request redirects unauthenticated users to login.
- [x] Consent screen shows client identity and requested scopes clearly.
- [ ] Consent deny path returns a standards-compliant OAuth error to the client.
- [x] Consent approve path returns to the correct client redirect URI.
- [x] Protected resource metadata endpoints are reachable from the browser and by direct HTTP
      inspection.

### MCP

- [x] A remote MCP client can discover the protected resource metadata for Tend.
- [x] An unauthenticated MCP request receives an OAuth-aware challenge.
- [ ] A scoped MCP token can call only tools whose backing API permissions are granted.
- [ ] A token missing `tasks:write` can still read tasks but cannot mutate them via MCP.

### CLI

- [ ] `tend auth login` opens the browser and completes against a local Tend server.
- [x] Browser-launch failure falls back to a copy/paste URL flow.
- [ ] `tend auth status` shows authenticated server and resolved identity.
- [ ] `tend auth logout` removes local credentials and future commands fail cleanly.
- [ ] CLI commands continue to work across access-token expiry via refresh.

### Playwright walkthroughs

- [ ] Record a full browser walkthrough of the OAuth consent flow.
- [ ] Record a walkthrough showing a delegated token denied for insufficient scope.
- [ ] Record a walkthrough showing existing password/OIDC web login still works after the OAuth
      changes.

## Current Execution Order

1. Record this plan.
2. Implement shared token/auth foundations.
3. Add OAuth metadata and protocol endpoints.
4. Add delegated scope enforcement.
5. Add CLI browser login and credential storage.
6. Run targeted automated tests and Playwright walkthroughs.

## Progress Log

### Completed

- Recorded the implementation plan and test checklists in this file.
- Added delegated OAuth token metadata and token kinds to the auth token schema.
- Centralized bearer-token authentication for REST and MCP.
- Added regression tests covering OAuth access-token acceptance and existing token-list isolation.
- Added OAuth core schema for clients, authorization codes, and consent grants.
- Added authorization server metadata and protected resource discovery for `/api` and `/mcp`.
- Added browser authorization, code exchange, refresh rotation, and revoke endpoints.
- Added shared delegated-scope enforcement in the handler layer for REST and MCP.
- Added negative-path OAuth protocol coverage for invalid authorize and token exchange inputs.
- Added CLI runtime coverage for keychain resolution, env-token precedence, refresh retry, and clean
  re-login errors.
- Fixed top-level routing so root OAuth, auth, and protected-resource metadata routes are reachable
  outside the `/api` prefix.

### In Progress

- Manually verify CLI browser login and refresh against a live Tend server.

### Not Started

- Playwright walkthrough capture for OAuth and delegated-access UX.

## Live Verification Notes

- Live dev server verification now succeeds for:
  - `/.well-known/oauth-authorization-server`
  - `/mcp/.well-known/oauth-protected-resource`
  - `/oauth/authorize` redirecting unauthenticated users to `/login`
- Playwright verification succeeded for:
  - password login at `http://localhost:5173/login`
  - consent page rendering for the CLI client
  - consent approval redirecting to the loopback callback with `code` and `state`
- CLI verification found one remaining environment-sensitive blocker:
  - `tend auth login` successfully reaches the loopback callback and the server issues
    `oauth_access` + `oauth_refresh` tokens
  - the CLI then hangs before completing, and no credentials are persisted locally
  - live inspection strongly suggests the hang is in OS keychain persistence rather than in the
    OAuth exchange itself

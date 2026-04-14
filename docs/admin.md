# Administration

This page covers the first-owner bootstrap path and the main operational tasks after deployment.

## Initial owner

For a fresh deployment, the simplest approach is to bootstrap the first owner during server startup:

- `HOROLOGIA_INIT_OWNER_EMAIL`
- `HOROLOGIA_INIT_OWNER_NAME`
- `HOROLOGIA_INIT_OWNER_PASSWORD`

The server creates that owner only if no owner exists yet. After the first successful startup, you
can remove those variables from the deployment configuration.

For OIDC-only deployments, set:

- `HOROLOGIA_PASSWORD_AUTH_ENABLED=false`
- `HOROLOGIA_INIT_OWNER_EMAIL`
- `HOROLOGIA_INIT_OWNER_NAME`

In OIDC-only mode, do not set `HOROLOGIA_INIT_OWNER_PASSWORD`.

## Creating an owner later

For password-based deployments, you can create another owner with the server command:

```bash
horologia-server create-admin \
  --email admin@example.com \
  --name "Admin" \
  --password 'change-me-now'
```

That command connects to the configured database and applies pending migrations before creating the
account.

## Password policy checks

Password-based owner creation uses the same password validation path as the application. By default,
Horologia enables Have I Been Pwned password checks. In environments without outbound internet
access, set `HOROLOGIA_HIBP_ENABLED=false`.

## Runtime responsibilities

- The server process serves the SPA, API, and MCP endpoint.
- Database schema migrations run automatically on startup.
- Health checks are available at `/healthz`.
- Logs can be emitted as structured JSON by setting `HOROLOGIA_LOG_FORMAT=json`.

## Recommended day-1 checklist

1. Start the service against PostgreSQL.
2. Sign in with the bootstrap owner.
3. Confirm the public URL is correct and cookies work through the reverse proxy.
4. Remove bootstrap secrets from the deployment after the first successful sign-in.
5. If using OIDC, complete a full login flow and verify the configured redirect URL.

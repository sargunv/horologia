# Deployment

Horologia is deployed as a single server process backed by PostgreSQL. The server runs database
migrations automatically on startup, serves the web UI, exposes the HTTP API, and hosts the MCP
endpoint.

## Published artifacts

The release workflow publishes:

- A container image to `ghcr.io/<owner>/<repo>`
- CLI archives to GitHub Releases
- Server archives to GitHub Releases

The container image is tagged with both the release version and `latest`.

## Requirements

- PostgreSQL 17 or compatible
- A reverse proxy or load balancer for TLS termination
- A stable public URL for the web app and OAuth/OIDC callbacks

## Required environment

At minimum, set:

- `HOROLOGIA_DB` — PostgreSQL connection string

Common production settings:

- `HOROLOGIA_ADDR` — listen address, defaults to `:8080`
- `HOROLOGIA_PUBLIC_URL` — external base URL such as `https://tasks.example.com`
- `HOROLOGIA_SECURE_COOKIES` — should remain `true` in production
- `HOROLOGIA_LOG_FORMAT` — `text` or `json`
- `HOROLOGIA_LOG_LEVEL` — defaults to `info`

Optional bootstrap settings for the first owner account:

- `HOROLOGIA_INIT_OWNER_EMAIL`
- `HOROLOGIA_INIT_OWNER_NAME`
- `HOROLOGIA_INIT_OWNER_PASSWORD`

Set all three together for password-based bootstrap. The server only uses them when no owner exists
yet, so they are safe to remove after the first successful startup.

## Docker example

```yaml
services:
  postgres:
    image: postgres:17
    restart: unless-stopped
    environment:
      POSTGRES_DB: horologia
      POSTGRES_USER: horologia
      POSTGRES_PASSWORD: change-me
    volumes:
      - postgres-data:/var/lib/postgresql/data

  horologia:
    image: ghcr.io/<owner>/<repo>:v0.YYYYMMDD.x
    restart: unless-stopped
    depends_on:
      - postgres
    ports:
      - "8080:8080"
    environment:
      HOROLOGIA_DB: postgres://horologia:change-me@postgres:5432/horologia?sslmode=disable
      HOROLOGIA_PUBLIC_URL: https://tasks.example.com
      HOROLOGIA_SECURE_COOKIES: "true"
      HOROLOGIA_INIT_OWNER_EMAIL: admin@example.com
      HOROLOGIA_INIT_OWNER_NAME: Admin
      HOROLOGIA_INIT_OWNER_PASSWORD: change-me-now

volumes:
  postgres-data:
```

## Container behavior

- The image entrypoint is `horologia-server`
- The default command is `serve`
- The image health check runs `horologia-server healthcheck`
- Startup runs pending database migrations before serving traffic

## Upgrades

To upgrade:

1. Pull a newer image tag.
2. Restart the service with the same environment and database.
3. Watch logs until startup completes and the health check passes.

Because migrations run at startup, a failed migration blocks the new process before it begins
serving requests.

## OIDC deployments

To enable OIDC, set:

- `HOROLOGIA_OIDC_ISSUER`
- `HOROLOGIA_OIDC_CLIENT_ID`
- `HOROLOGIA_OIDC_CLIENT_SECRET`
- `HOROLOGIA_OIDC_REDIRECT_URL`

Optional OIDC settings:

- `HOROLOGIA_OIDC_LABEL`
- `HOROLOGIA_OIDC_AUTO_REDIRECT`
- `HOROLOGIA_OIDC_LINK_CONSENT`

If you want an OIDC-only deployment, set `HOROLOGIA_PASSWORD_AUTH_ENABLED=false`. In that mode, do
not set `HOROLOGIA_INIT_OWNER_PASSWORD`.

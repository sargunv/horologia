# Configuration

All configuration uses environment variables with the `HOROLOGIA_` prefix.

## Core

| Variable                   | Default    | Description                                                     |
| -------------------------- | ---------- | --------------------------------------------------------------- |
| `HOROLOGIA_DB`             | (required) | PostgreSQL connection URI                                       |
| `HOROLOGIA_ADDR`           | `:8080`    | Listen address (`host:port`). Omit host to bind all interfaces. |
| `HOROLOGIA_PUBLIC_URL`     | (none)     | External URL for redirects                                      |
| `HOROLOGIA_LOG_LEVEL`      | `info`     | `debug`, `info`, `warn`, or `error`                             |
| `HOROLOGIA_LOG_FORMAT`     | `text`     | `text` or `json`                                                |
| `HOROLOGIA_SECURE_COOKIES` | `true`     | Set `false` for non-HTTPS (dev only)                            |

## OIDC

OIDC stays disabled unless you set `HOROLOGIA_OIDC_ISSUER`. Requires `HOROLOGIA_PUBLIC_URL`.

| Variable                       | Default | Description                                                                                         |
| ------------------------------ | ------- | --------------------------------------------------------------------------------------------------- |
| `HOROLOGIA_OIDC_ISSUER`        | (none)  | OIDC issuer URL (e.g. `https://auth.example.com`), not the discovery endpoint                       |
| `HOROLOGIA_OIDC_CLIENT_ID`     | (none)  | OAuth client ID                                                                                     |
| `HOROLOGIA_OIDC_CLIENT_SECRET` | (none)  | OAuth client secret                                                                                 |
| `HOROLOGIA_OIDC_LABEL`         | `OIDC`  | Button label on login page                                                                          |
| `HOROLOGIA_OIDC_AUTO_REDIRECT` | `false` | Auto-redirect to OIDC provider. Requires `HOROLOGIA_PASSWORD_AUTH_ENABLED=false`.                   |
| `HOROLOGIA_OIDC_AUTO_REGISTER` | `false` | Automatically create a local user for an OIDC identity whose email does not already exist.          |
| `HOROLOGIA_OIDC_LINK_CONSENT`  | `true`  | Require user consent before linking OIDC to an existing account. When `false`, links automatically. |

Register `https://your-domain/auth/oidc/callback` as the authorized redirect URI in your OIDC
provider.

## Auth

| Variable                          | Default | Description                                                          |
| --------------------------------- | ------- | -------------------------------------------------------------------- |
| `HOROLOGIA_PASSWORD_AUTH_ENABLED` | `true`  | Enable password login. Cannot disable unless OIDC is configured.     |
| `HOROLOGIA_HIBP_ENABLED`          | `true`  | Reject breached passwords on creation and change (Have I Been Pwned) |

## Initial user with owner (admin) permissions

These apply on first start only. The server ignores them if any user exists.

| Variable                        | Description                                        |
| ------------------------------- | -------------------------------------------------- |
| `HOROLOGIA_INIT_OWNER_EMAIL`    | Owner email                                        |
| `HOROLOGIA_INIT_OWNER_NAME`     | Owner display name                                 |
| `HOROLOGIA_INIT_OWNER_PASSWORD` | Owner password (omit if password auth is disabled) |

# Getting Started

The server is available as a Docker image (`ghcr.io/sargunv/horologia:latest`) or as a standalone
binary from [GitHub Releases](https://github.com/sargunv/horologia/releases). Both embed the web UI
and require PostgreSQL. Migrations run automatically on startup.

Sample `docker-compose.yaml`:

```yaml
services:
  postgres:
    image: postgres:17
    restart: unless-stopped
    environment:
      POSTGRES_DB: horologia
      POSTGRES_PASSWORD: changeme
    volumes:
      - pgdata:/var/lib/postgresql/data

  horologia:
    image: ghcr.io/sargunv/horologia:latest
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      HOROLOGIA_DB: "postgres://postgres:changeme@postgres:5432/horologia?sslmode=disable"
      HOROLOGIA_PUBLIC_URL: "https://horologia.example.com"
      HOROLOGIA_INIT_OWNER_EMAIL: "admin@example.com"
      HOROLOGIA_INIT_OWNER_NAME: "Admin"
      HOROLOGIA_INIT_OWNER_PASSWORD: "changethis"
    ports:
      - "8080:8080"

volumes:
  pgdata:
```

The Dockerfile configures a built-in health check. The `HOROLOGIA_INIT_OWNER_*` variables apply only
on first start, when no users exist yet. Subsequent starts ignore them -- feel free to remove them.

To manage migrations manually:

```sh
horologia-server migrate up
horologia-server migrate status
```

## Next steps

Once the server is running, log in at your `HOROLOGIA_PUBLIC_URL` with the owner credentials you
configured above. From there you can create spaces, invite users, and start managing tasks.

- Set up the [CLI](cli.md) for command-line access
- Connect an [MCP](mcp.md) client for AI assistant integration
- Configure [OIDC](configuration.md#oidc) if you want single sign-on
- See [Configuration](configuration.md) for all environment variables

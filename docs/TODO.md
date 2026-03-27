# Tend v0.1 — TODO

## Web App Roadmap

- [x] Browser automation for agent feedback loop (agent-browser)
- [x] Login page
- [x] App shell (layout, nav, auth guard, logout)
- [ ] Space list and create
- [ ] Task list view within a space
- [ ] Task create and edit
- [ ] Task detail view (relations, activity)
- [ ] Tags management
- [ ] Space settings (members, statuses, effort levels, priority levels)
- [ ] Activity log (space-level and user-level)
- [ ] API token management

## Views & UX

1. **CLI feature parity with backend**
2. **MCP feature parity with API**
3. **Web feature parity with backend and CLI**
4. **Markdown rendering** — render descriptions as Markdown in web UI

## Testing

1. **OIDC integration tests** — use `github.com/oauth2-proxy/mockoidc` to test the full OIDC
   callback flow (email_verified rejection, auto-linking, user creation)

## Security

1. **OIDC auto-linking by email** — require user consent before linking OIDC identity to existing
   account
2. **Invalidate sessions on password change** — revoke all tokens for the user

## Infrastructure

1. **CLI** - goreleaser
2. **Dockerfile** — single container deployment

# Tend v0.1 — TODO

## Web App Roadmap

- [x] Browser automation for agent feedback loop (agent-browser)
- [ ] Login page
- [ ] App shell (layout, nav, auth guard, logout)
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

## Security

1. **OIDC auto-linking by email** — require user consent before linking OIDC identity to existing
   account
2. **Check `email_verified` on OIDC claims** — reject unverified emails from OIDC providers
3. **Login CSRF** — validate `Content-Type: application/json` on `POST /auth/web-login`
4. **Rate limiting on password login** — prevent brute-force attacks
5. **Invalidate sessions on password change** — revoke all tokens for the user

## Infrastructure

1. **CLI** - goreleaser
2. **Dockerfile** — single container deployment

# CLAUDE.md

## Project

Tend is a self-hosted task manager. See `docs/BRIEF.md` for the full product brief.

## Roadmap

Progress is tracked in Linear (project: Tend).

## Development

This project uses `mise` for tooling and task orchestration. Run `mise tasks` to see all available
tasks. Key commands:

- `mise run dev` — start local dev environment (Tilt)
- `mise run generate` — run all code generation (run after changing TypeSpec or route files)
- `mise run check` — run all linting/checks (hk)
- `mise run test` — run all tests across all packages
- `mise run fix` — auto-fix linting issues
- `mise run ci` — run the full suite: generate → fix → test

Package-scoped tasks use a `//` prefix, e.g. `mise run //server:generate`,
`mise run //server:build`, `mise run //server:test`.

## Packages

- ./api - TypeSpec definition for API.
- ./server - Golang backend service and API implementation.
- ./web - React SPA served by the backend, built with Skeleton (React) design system.
- ./cli - TODO: Golang CLI client

## Conventions

- Never use `context.Background()` when a context is available from a caller (e.g. `cmd.Context()`,
  function parameter). Thread contexts through from the top.

## Web App Conventions

- Use [Skeleton (React)](https://www.skeleton.dev/llms-react.txt) as the design system. Prefer
  Skeleton components over custom implementations wherever possible.
- Lean on Skeleton's built-in theming for all styling. Do not hand-roll colors, typography, or
  spacing tokens.
- Our concerns are layout, functionality, and correct use of Skeleton components — not bespoke
  styling.
- Use `/frontend-design` when building UI to ensure high quality.
- Use `createLink()` from TanStack Router to wrap Skeleton components for client-side navigation. Do
  not use Skeleton's `element` render prop for router integration — it's a power-user escape hatch
  that doesn't forward children and is not needed for normal usage.
- Read Skeleton's component docs before hand-rolling UI. Check whether a Skeleton component already
  exists for the pattern (e.g. Navigation, AppBar) rather than building it from raw HTML + Tailwind.

## Browser Automation

Use `playwright-cli` for web automation. Run `playwright-cli --help` for available commands.

### Capturing UI evidence (headless agents)

When running as a headless autonomous agent (e.g. via Stokowski), capture a walkthrough video after
implementing UI changes so the human reviewer can see the result:

1. Start the dev environment: `mise run dev` (wait for all services to be healthy)
2. Open the browser: `playwright-cli open http://localhost:$WEB_PORT`
3. Log in: navigate to login, fill credentials (`admin@localhost` / `password`)
4. Start recording: `playwright-cli video-start`
5. Walk through the implemented feature — navigate to relevant pages, interact with new UI
6. Stop recording: `playwright-cli video-stop --filename=walkthrough.webm`
7. Upload the video to the PR: `gh pr comment <number> --body "## UI Walkthrough" --edit-last` or
   attach via:
   `gh api repos/{owner}/{repo}/issues/{number}/comments -f body="![walkthrough](video-url)"`

For quick screenshots instead of video:

- `playwright-cli screenshot` — captures the current page
- `playwright-cli screenshot <ref>` — captures a specific element

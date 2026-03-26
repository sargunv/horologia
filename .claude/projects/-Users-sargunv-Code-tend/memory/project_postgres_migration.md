---
name: PostgreSQL migration decisions
description: Decisions made for the SQLite → PostgreSQL migration (pre-v0.1), including test strategy
type: project
---

Migrating from SQLite to PostgreSQL before v0.1. Key decisions:

- **Test strategy**: Use `fergusstrange/embedded-postgres` for tests instead of testcontainers or
  shared DB. **Why:** No Docker dependency keeps CI simple (just `go test ./...`, no
  Docker-in-Docker or service containers). Closest ergonomics to current in-memory SQLite approach.
  **How to apply:** Start embedded PG once per package in `TestMain`, create fresh database per test
  for isolation. Refactor `setupTestServer` in `testhelpers_test.go` accordingly.

- **Migration squash**: Squash all 10 SQLite migrations into one PG-native initial migration (no
  user data to migrate).

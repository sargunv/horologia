# Clear Bugs Review

## Critical

### `SpaceTasksRead` and `SpaceTasksList` open transactions that are never committed

Both read-only handlers open a transaction with `h.Pool.Begin(ctx)` and defer a rollback, but there
is no `tx.Commit(ctx)` call in either. They are read-only operations, so this does not cause data
corruption, but pgx holds the connection open until the deferred rollback fires at function return.
With any meaningful request concurrency, this will silently exhaust the pool and cause all
subsequent operations to block waiting for a free connection. Under load this manifests as request
timeouts across the entire server.

```go
// SpaceTasksList (lines 258-284):
tx, err := h.Pool.Begin(ctx)
defer func() { _ = tx.Rollback(ctx) }()
q := dbgen.New(tx)
rows, err := q.ListTasksBySpace(...)
return &apigen.TaskPage{Items: items, NextCursor: nextCursor}, nil
// tx.Commit is never called
```

**Fix:** Remove the transaction entirely — both handlers are purely read-only and do not require
snapshot isolation. Use `dbgen.New(h.Pool)` directly.

**Files:** `internal/api/task_handlers.go` lines 258-284, 295-303

---

## Important

### Session cookies have no `Secure` flag

The `tend_session` cookie is set as `HttpOnly` and `SameSite: Lax`, but the `Secure` flag is never
set. Without `Secure: true`, the browser will send the session token over plain HTTP connections,
exposing it to passive network eavesdroppers. There is no mechanism to conditionally set the flag
when TLS is in use.

**Fix:** Accept a `TEND_SECURE_COOKIES` env var and conditionally set `Secure: true`.

**Files:** `internal/api/web.go` lines 138-146, 149-156

---

### OIDC account-linking race condition

The OIDC callback handler performs a two-step lookup (by OIDC subject, then by email) and links
accounts — all outside a database transaction. A concurrent callback could race past the initial
check and both reach the "link by email" branch. More significantly, any OIDC provider the server
trusts that asserts an email owned by an existing password-based account gains full access without
consent.

**Fix:** Wrap the entire lookup-and-create/link sequence in a serializable transaction. Consider
whether auto-linking by email is appropriate security posture.

**Files:** `internal/api/oidc.go` lines 122-168

---

### `writeJSONError` produces malformed responses on write failure

After `w.WriteHeader(status)` has been called, calling `http.Error(w, ...)` in the error branch
attempts to set a new `Content-Type` header and call `WriteHeader` again. Both calls are silently
ignored, but the plain-text error string is still appended to the response body, resulting in a body
with partial JSON followed by a plain-text string.

**Fix:** Replace the fallback with a simple log call. The write error can only occur due to a broken
connection.

**Files:** `internal/api/web.go` lines 176-185

---

## Minor Observations

- `time.Now()` is called multiple times within the same request, so `created_at` and `updated_at`
  could differ by a few microseconds within a single operation.
- `requireSpaceRole` returns `pgx.ErrNoRows` (mapped to 404) when a non-member queries a space —
  intentional to avoid confirming space existence to non-members.

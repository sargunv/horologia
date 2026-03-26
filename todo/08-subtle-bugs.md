# Subtle Bugs Review

## Critical

### TOCTOU race in `ensureNotLastAdmin`

`ensureNotLastAdmin` counts admins using `SELECT COUNT(*)` inside a transaction at the default
`READ COMMITTED` isolation level. Two concurrent requests to demote two different admins in a
2-admin space will both see count=2, both pass the check, and both execute the update — leaving the
space with zero admins.

**Fix:** Lock the relevant rows before reading the count:

```sql
SELECT COUNT(*) FROM space_members
WHERE space_slug = $1 AND role = 'admin' FOR UPDATE
```

Or use a `SERIALIZABLE` transaction for the admin-removal path.

This affects both `SpaceMembersDelete` and `SpaceMembersUpdate`.

**Files:** `internal/api/member_handlers.go` lines 197-216, 98-141

---

## Important

### `allOverdueOccurrences` trims the oldest missed occurrences permanently

```go
occurrences := rr.Between(dtstart, until, false)
if len(occurrences) > maxMissedOccurrences {
    occurrences = occurrences[len(occurrences)-maxMissedOccurrences:]
}
```

When a task has more than 365 missed occurrences, the code keeps the last 365 (most recent). The
dropped earlier occurrences are silently lost forever. On the next cron tick, the freshly spawned
`fixed_accumulating` task will have a future due date and won't be overdue, so those initial missed
occurrences will never be recovered.

Whether "trim oldest" or "trim newest" is correct depends on product intent, but this is a
potentially surprising data loss path.

**Files:** `internal/taskengine/spawn.go` lines 195-200

---

### Completion + `assignee_ids` in same PATCH silently discards rotation advance

In `SpaceTasksUpdate`, the flow is:

1. `HandleCompletionTransition` -> calls `ApplyPoolRotation` which sets the next rotated assignee
2. `applyTaskCollections` -> if `req.AssigneeIds` is non-nil, `setTaskAssignees` deletes the
   rotation-computed assignee and replaces it with whatever the user specified

The rotation result is silently discarded when the user happens to also set `assignee_ids` in the
same PATCH request.

**Files:** `internal/api/task_handlers.go` lines 357-394

---

### `context.Background()` in shutdown path

`cmd.Context()` is available. Using `context.Background()` means the shutdown context is not linked
to the command's lifecycle.

**Fix:** `context.WithTimeout(cmd.Context(), 30*time.Second)`

**Files:** `cmd/server/main.go` line 178

---

### Overdue cron uses `time.Now()` without `.UTC()` for `pgtype.Date`

`pgtype.Date` stores only the date part. When constructing it with `time.Now()`, the date is taken
from the server's local timezone. If the server runs in a non-UTC timezone, the "today" date passed
to the query could be a day ahead or behind UTC midnight, causing tasks to be processed early or
late.

**Fix:** Always use `pgtype.Date{Time: now.UTC(), Valid: true}`.

**Files:** `internal/taskengine/overdue.go` line 22

---

## Low

### `time.Now()` called independently per-collection inside a transaction

`setTaskAssignees`, `setTaskRotationPool`, and `setTaskTags` each call `time.Now()` independently,
so `created_at` timestamps within the same transaction will be slightly different. While not a
correctness bug, it will cause confusion in audits.

**Fix:** Pass a single `now time.Time` parameter to these functions.

**Files:** `internal/api/task_handlers.go` lines 516, 542, 588

---

## Verified Clean

- No goroutine leaks: `RunAccumulatingCron` exits cleanly on `ctx.Done()`.
- No integer overflow in pagination: `limit` is `int32`, `cursorID` is `int64`.
- No map iteration order assumptions found.
- No slice aliasing bugs.
- Token timing-safe comparison is correctly implemented with the sentinel hash.
- Unicode/tag folding uses `golang.org/x/text/cases.Fold` correctly.
- `DueDate` timezone round-trip through `NewDueDate`/`DecomposeDueDate` is correct.
- `rrule.After(after, false)` correctly excludes the boundary.
- `defer tx.Rollback(ctx)` correctly ignores the error from rollback-after-commit.

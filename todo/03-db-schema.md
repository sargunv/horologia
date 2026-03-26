# DB Schema Review

## Notable

### Missing partial index on `task_relations` for trigger lookups

The `ListTriggerTargets` query filters on `source_task_id`, `space_slug`, and `kind = 'triggers'`. A
partial index `WHERE kind = 'triggers'` on `(source_task_id)` would be more efficient for this
high-frequency path (called on every task completion that might trigger dependents).

**Files:** `00001_initial.sql` line 170

---

### Slug-based keyset pagination on `spaces` is fragile

`spaces.sql` uses `WHERE slug > $1 ORDER BY slug ASC`. Keyset pagination on a text slug means sort
order is lexicographic, not insertion order. Inserting a space with a slug that sorts earlier than
the cursor will never appear in a subsequent page. Cursors based on a stable surrogate integer ID
(as used for tasks, tags, users, and tokens) avoid this class of problem.

**Files:** `queries/spaces.sql` lines 9-13

---

## Observations (not issues)

- `tags` table correctly stores both `name` and `name_folded` with case-insensitive deduplication.
- `EnsureTag` upsert preserves the original display name on conflict — correct behavior.
- `created_at` and `updated_at` are application-provided rather than `DEFAULT NOW()` — consistent
  throughout but means clock skew between application servers could cause inconsistent timestamps.
- `auth_tokens.name` defaults to `''` (empty string) rather than `NULL`, conflating "unnamed" with
  "explicitly empty."
- `DeleteTaskRelation` includes `space_slug` in WHERE as a redundant ownership check — deliberate
  and correct safety measure.

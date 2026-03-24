# Batch B: Typed Enums

Replace raw string literals with `apigen` typed constants throughout internal code. This is
cross-cutting but mechanically straightforward.

## 1. `TaskRecurrenceType` — use `apigen` constants internally

Currently cast to `string` immediately at `task_handlers.go:191` and `:345`. All switches in
`recurrence.go`, `spawn.go`, `cron.go`, `task_handlers.go` use bare literals like `"one_off"`,
`"completion_based"`, `"fixed_accumulating"`.

- Change signatures of `validateRecurrence`, `computeNextDueAt`, `spawnTaskFromTemplate`,
  `applyCompletionTriggers` to accept `apigen.TaskRecurrenceType`
- Replace all string literal comparisons with `apigen.TaskRecurrenceType*` constants
- Only call `string(...)` when constructing `dbgen.CreateTaskParams` / `dbgen.UpdateTaskParams`
- Add `default` error returns to every switch (some currently have no default or panic)

## 2. `SpaceRole` — use `apigen` constants in access control

`requireSpaceRole` in `space_access.go` takes `roles ...string` and callers pass `"member"`,
`"admin"`, `"viewer"`.

- Change to `roles ...apigen.SpaceRole`
- Use `apigen.SpaceRoleAdmin`, `SpaceRoleMember`, `SpaceRoleViewer` at call sites
- Convert to string only when comparing against DB values

## 3. `TaskStatusCategory` — use `apigen` constants in category checks

`"initial"` and `"completion"` literals in `recurrence.go` (lines 155, 200) and `task_handlers.go`
(lines 377, 378, 380).

- Replace with `apigen.TaskStatusCategoryInitial` / `apigen.TaskStatusCategoryCompletion`

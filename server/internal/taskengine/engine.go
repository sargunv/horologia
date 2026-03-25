package taskengine

// Engine contains configuration needed by task business logic
// (spawning, recurrence). HTTP handlers hold an *Engine and
// delegate business-rule work to it. All methods accept a
// *dbgen.Queries so they operate within the caller's transaction.
type Engine struct {
	CopyOnSpawnKinds map[string]bool
}

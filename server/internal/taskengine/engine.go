package taskengine

import (
	"database/sql"
	"log/slog"
)

// Engine contains the dependencies needed by task business logic
// (cron jobs, spawning, recurrence). HTTP handlers hold an *Engine
// and delegate business-rule work to it.
type Engine struct {
	DB               *sql.DB
	Log              *slog.Logger
	CopyOnSpawnKinds map[string]bool
}

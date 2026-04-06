package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sargunv/tend/server/internal/webui"
)

// MountRoot composes the top-level HTTP handler:
//   - /healthz returns health status (pings the database connection pool)
//   - /api/* routes to the API handler (with /api prefix stripped)
//   - /mcp routes to the MCP Streamable HTTP handler (if non-nil)
//   - /* routes to the embedded SPA (static files + index.html fallback)
func MountRoot(apiHandler http.Handler, mcpHandler http.Handler, pool *pgxpool.Pool, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler(pool, log))
	mux.Handle("/api/", http.StripPrefix("/api", apiHandler))
	if mcpHandler != nil {
		mux.Handle("/mcp", mcpHandler)
	}
	mux.Handle("/", webui.Handler())
	return mux
}

func healthHandler(pool *pgxpool.Pool, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		w.Header().Set("Content-Type", "application/json")

		if err := pool.Ping(ctx); err != nil {
			log.ErrorContext(r.Context(), "health check: database ping failed", "error", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "error",
				"error":  "database unavailable",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	}
}

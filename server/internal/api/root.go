package api

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sargunv/tend/server/internal/webui"
)

// MountRoot composes the top-level HTTP handler:
//   - /healthz returns health status (with DB readiness check)
//   - /api/* routes to the API handler (with /api prefix stripped)
//   - /* routes to the embedded SPA (static files + index.html fallback)
func MountRoot(apiHandler http.Handler, pool *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler(pool))
	mux.Handle("/api/", http.StripPrefix("/api", apiHandler))
	mux.Handle("/", webui.Handler())
	return mux
}

func healthHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := pool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	}
}

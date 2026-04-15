package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sargunv/horologia/server/internal/apidocs"
	"github.com/sargunv/horologia/server/internal/webui"
)

// MountRoot composes the top-level HTTP handler:
//   - /healthz returns health status (pings the database connection pool)
//   - /api/* routes to the API handler (with /api prefix stripped)
//   - /auth/*, /oauth/*, /.well-known/*, and /mcp/.well-known/* route to the
//     auth/OAuth stack without a prefix rewrite
//   - /openapi.yaml serves the generated OpenAPI document
//   - /docs serves the Scalar API reference UI
//   - /mcp routes to the MCP Streamable HTTP handler (if non-nil)
//   - /* routes to the embedded SPA (static files + index.html fallback)
func MountRoot(apiHandler http.Handler, mcpHandler http.Handler, pool *pgxpool.Pool, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler(pool, log))
	mux.HandleFunc("GET /openapi.yaml", apidocs.OpenAPIHandler)
	mux.HandleFunc("GET /docs", apidocs.ScalarHandler)
	mux.Handle("/api/", http.StripPrefix("/api", apiHandler))
	mux.Handle("/auth/", apiHandler)
	mux.Handle("/oauth/", apiHandler)
	mux.Handle("/.well-known/", apiHandler)
	mux.Handle("/mcp/.well-known/", apiHandler)
	if mcpHandler != nil {
		mux.Handle("/mcp", mcpHandler)
	}
	mux.Handle("/", webui.Handler())
	return internalCORSMiddleware(mux)
}

func internalCORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isInternalAPIPath(r.URL.Path) && !sameOriginRequest(r) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isInternalAPIPath(path string) bool {
	if normalized, ok := strings.CutPrefix(path, "/api"); ok {
		path = normalized
	}

	switch path {
	case "/auth/config", "/auth/login", "/auth/logout", "/auth/link", "/auth/link/pending":
		return true
	default:
		return false
	}
}

func sameOriginRequest(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return u.Host == r.Host && u.Scheme == requestScheme(r)
}

func requestScheme(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		if scheme, _, _ := strings.Cut(forwarded, ","); scheme != "" {
			return strings.TrimSpace(scheme)
		}
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
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

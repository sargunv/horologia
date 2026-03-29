package api

import (
	"net/http"

	"github.com/sargunv/tend/server/internal/webui"
)

// MountRoot composes the top-level HTTP handler:
//   - /api/* routes to the API handler (with /api prefix stripped)
//   - /* routes to the embedded SPA (static files + index.html fallback)
func MountRoot(apiHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", apiHandler))
	mux.Handle("/", webui.Handler())
	return mux
}

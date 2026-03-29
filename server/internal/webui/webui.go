// Package webui exposes the embedded React SPA for serving by the HTTP layer.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var distEmbed embed.FS

// FS is the embedded web dist, rooted at the dist directory itself.
// When the SPA has not been embedded (e.g. local dev), this contains only .gitkeep.
var FS = func() fs.FS {
	sub, err := fs.Sub(distEmbed, "dist")
	if err != nil {
		panic("webui: " + err.Error())
	}
	return sub
}()

// Handler returns an http.Handler that serves static files from the embedded
// SPA, falling back to index.html for any path that has no matching file.
// This enables client-side routing.
func Handler() http.Handler {
	fileServer := http.FileServerFS(FS)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to stat the requested path as a regular file.
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "."
		}

		if info, err := fs.Stat(FS, p); err == nil && !info.IsDir() {
			// Hashed assets (Vite output like main-abc123.js) are immutable.
			// Everything else (index.html) must be revalidated every request.
			if isHashedAsset(p) {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "no-cache")
			}
			fileServer.ServeHTTP(w, r)
			return
		}

		// No matching file — serve index.html for SPA client-side routing.
		w.Header().Set("Cache-Control", "no-cache")
		r = r.Clone(r.Context())
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

// isHashedAsset returns true if the file is inside the assets/ directory.
// Vite places all content-hashed output (JS, CSS, fonts) under assets/.
func isHashedAsset(p string) bool {
	dir, _ := path.Split(p)
	return strings.HasPrefix(dir, "assets/")
}

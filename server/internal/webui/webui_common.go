// Package webui exposes the React SPA for serving by the HTTP layer.
package webui

import (
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

var (
	distFS           fs.FS
	hasEmbeddedIndex bool
)

// Handler returns an http.Handler that serves static files from the embedded
// SPA, falling back to index.html for any path that has no matching file.
// This enables client-side routing.
func Handler() http.Handler {
	fileServer := http.FileServerFS(distFS)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !hasEmbeddedIndex {
			if target := devServerURL(r); target != "" {
				http.Redirect(w, r, target, http.StatusTemporaryRedirect)
				return
			}
		}

		// Try to stat the requested path as a regular file.
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "."
		}

		if info, err := fs.Stat(distFS, p); err == nil && !info.IsDir() {
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

func devServerURL(r *http.Request) string {
	webPort := strings.TrimSpace(os.Getenv("WEB_PORT"))
	if webPort == "" {
		webPort = "5173"
	}

	host := r.Host
	if host == "" {
		return ""
	}

	if parsedHost, parsedPort, err := net.SplitHostPort(host); err == nil {
		_ = parsedPort
		host = net.JoinHostPort(parsedHost, webPort)
	} else if strings.Contains(err.Error(), "missing port in address") {
		host = net.JoinHostPort(host, webPort)
	} else {
		return ""
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	target := &url.URL{
		Scheme:   scheme,
		Host:     host,
		Path:     r.URL.Path,
		RawPath:  r.URL.RawPath,
		RawQuery: r.URL.RawQuery,
		Fragment: r.URL.Fragment,
	}
	return target.String()
}

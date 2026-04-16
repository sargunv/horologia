package api

import (
	"net/http"
	"strings"
)

func isOAuthOrWellKnownPath(path string) bool {
	return strings.HasPrefix(path, "/oauth/") ||
		strings.HasPrefix(path, "/.well-known/") ||
		strings.HasPrefix(path, "/mcp/.well-known/")
}

func isPublicBrowserAuthHelperPath(path string) bool {
	return path == "/auth/config" ||
		path == "/auth/login" ||
		path == "/auth/logout" ||
		path == "/auth/link" ||
		path == "/auth/link/pending"
}

func isInternalAPIPath(path string, method string) bool {
	if normalized, ok := strings.CutPrefix(path, "/api"); ok {
		path = normalized
	}

	if isPublicBrowserAuthHelperPath(path) {
		return true
	}

	return path == "/oauth/authorize" && method == http.MethodPost
}

func shouldBridgeSessionAuth(path string) bool {
	return !isPublicBrowserAuthHelperPath(path) &&
		!strings.HasPrefix(path, "/auth/oidc/") &&
		!isOAuthOrWellKnownPath(path)
}

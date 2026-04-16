package api

import (
	"net/http"
	"strings"
)

func isBrowserAuthPath(path string) bool {
	return strings.HasPrefix(path, "/auth/")
}

func isOAuthOrWellKnownPath(path string) bool {
	return strings.HasPrefix(path, "/oauth/") ||
		strings.HasPrefix(path, "/.well-known/") ||
		strings.HasPrefix(path, "/mcp/.well-known/")
}

func isInternalAPIPath(path string, method string) bool {
	if normalized, ok := strings.CutPrefix(path, "/api"); ok {
		path = normalized
	}

	if isBrowserAuthPath(path) {
		switch path {
		case "/auth/config", "/auth/login", "/auth/logout", "/auth/link", "/auth/link/pending":
			return true
		}
	}

	return path == "/oauth/authorize" && method == http.MethodPost
}

func shouldBridgeSessionAuth(path string) bool {
	return !isBrowserAuthPath(path) && !isOAuthOrWellKnownPath(path)
}

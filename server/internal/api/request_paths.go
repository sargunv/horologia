package api

import (
	"strings"
)

func isOAuthOrWellKnownPath(path string) bool {
	return strings.HasPrefix(path, "/oauth/") ||
		strings.HasPrefix(path, "/api/.well-known/") ||
		strings.HasPrefix(path, "/.well-known/") ||
		strings.HasPrefix(path, "/mcp/.well-known/")
}

func isAppPath(path string) bool {
	return strings.HasPrefix(path, "/app/")
}

func isInternalAPIPath(path string, _ string) bool {
	return isAppPath(path)
}

func shouldBridgeSessionAuth(path string) bool {
	return !isAppPath(path) && !isOAuthOrWellKnownPath(path)
}

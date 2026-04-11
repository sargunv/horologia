package auth

import (
	"fmt"
	"net/http"
	"strings"
)

func SetOAuthChallengeHeader(w http.ResponseWriter, r *http.Request) {
	resourceMetadata := oauthChallengeResourceMetadataURL(r)
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="Tend", resource_metadata="%s"`, resourceMetadata))
}

// RequestPublicBaseURL derives the public origin for the current request.
func RequestPublicBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = forwarded
	}
	return scheme + "://" + r.Host
}

func oauthChallengeResourceMetadataURL(r *http.Request) string {
	base := RequestPublicBaseURL(r)
	if strings.HasPrefix(r.URL.Path, "/mcp") {
		return base + "/mcp/.well-known/oauth-protected-resource"
	}
	return base + "/api/.well-known/oauth-protected-resource"
}

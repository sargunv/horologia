package auth

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

func SetOAuthChallengeHeader(w http.ResponseWriter, r *http.Request) {
	resourceMetadata := oauthChallengeResourceMetadataURL(r)
	if resourceMetadata == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="Tend"`)
		return
	}
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="Tend", resource_metadata="%s"`, resourceMetadata))
}

// RequestPublicBaseURL derives the public origin for the current request.
// It only falls back to request-derived origins for localhost/loopback hosts.
// Production deployments should set HOROLOGIA_PUBLIC_URL explicitly.
func RequestPublicBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	host := strings.TrimSpace(r.Host)
	if host == "" {
		return ""
	}
	hostname := host
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		hostname = parsedHost
	} else {
		hostname = strings.TrimPrefix(hostname, "[")
		hostname = strings.TrimSuffix(hostname, "]")
	}
	if !isTrustedDynamicPublicHost(hostname) {
		return ""
	}
	return scheme + "://" + host
}

func oauthChallengeResourceMetadataURL(r *http.Request) string {
	base := RequestPublicBaseURL(r)
	if base == "" {
		return ""
	}
	if strings.HasPrefix(r.URL.Path, "/mcp") {
		return base + "/mcp/.well-known/oauth-protected-resource"
	}
	return base + "/api/.well-known/oauth-protected-resource"
}

func isTrustedDynamicPublicHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

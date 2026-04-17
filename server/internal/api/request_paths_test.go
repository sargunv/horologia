package api

import "testing"

func TestIsInternalAPIPathUsesAppPrefix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path string
		want bool
	}{
		{path: "/app/auth/config", want: true},
		{path: "/app/oauth/consent", want: true},
		{path: "/api/users/me", want: false},
		{path: "/oauth/authorize", want: false},
	}

	for _, tt := range tests {
		if got := isInternalAPIPath(tt.path, "GET"); got != tt.want {
			t.Fatalf("isInternalAPIPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestShouldBridgeSessionAuthOnlyForAPIPrefix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path string
		want bool
	}{
		{path: "/api/users/me", want: true},
		{path: "/api/.well-known/oauth-protected-resource", want: false},
		{path: "/app/auth/config", want: false},
		{path: "/oauth/authorize", want: false},
	}

	for _, tt := range tests {
		if got := shouldBridgeSessionAuth(tt.path); got != tt.want {
			t.Fatalf("shouldBridgeSessionAuth(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

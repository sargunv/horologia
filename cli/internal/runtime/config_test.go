package runtime

import (
	"strings"
	"testing"
)

func TestResolveConfigWithOnlyServerEnv(t *testing.T) {
	setConfigHome(t)
	withCredentialStore(t, newMemoryCredentialStore())
	setEnvValue(t, envServer, stringPtr("http://example.com/api/"))
	setEnvValue(t, envToken, nil)

	cfg, err := ResolveConfig(ResolveInput{})
	if err != nil {
		t.Fatalf("resolve config: %v", err)
	}

	if got, want := cfg.ServerString(), "http://example.com"; got != want {
		t.Fatalf("server = %q, want %q", got, want)
	}
	if got, want := cfg.ServerSource, ValueSourceEnv; got != want {
		t.Fatalf("server source = %q, want %q", got, want)
	}
	if cfg.HasToken() {
		t.Fatalf("token should not be configured")
	}
	if got, want := cfg.TokenSource, ValueSourceUnset; got != want {
		t.Fatalf("token source = %q, want %q", got, want)
	}
}

func TestResolveConfigUsesKeychainWhenEnvTokenMissing(t *testing.T) {
	setConfigHome(t)
	store := newMemoryCredentialStore()
	withCredentialStore(t, store)
	setEnvValue(t, envServer, stringPtr("http://example.com"))
	setEnvValue(t, envToken, nil)

	if err := SaveOAuthCredentials("http://example.com", OAuthCredentials{
		ClientID:     "horologia-cli",
		AccessToken:  "keychain-access-token",
		RefreshToken: "keychain-refresh-token",
		TokenType:    "Bearer",
		Scope:        "profile:read",
	}); err != nil {
		t.Fatalf("save oauth credentials: %v", err)
	}

	cfg, err := ResolveConfig(ResolveInput{})
	if err != nil {
		t.Fatalf("resolve config: %v", err)
	}

	if got, want := cfg.Token, "keychain-access-token"; got != want {
		t.Fatalf("token = %q, want %q", got, want)
	}
	if got, want := cfg.TokenSource, ValueSourceKeychain; got != want {
		t.Fatalf("token source = %q, want %q", got, want)
	}
	if cfg.OAuth == nil || cfg.OAuth.RefreshToken != "keychain-refresh-token" {
		t.Fatalf("expected loaded oauth credentials, got %#v", cfg.OAuth)
	}
}

func TestResolveConfigEnvTokenOverridesKeychain(t *testing.T) {
	setConfigHome(t)
	store := newMemoryCredentialStore()
	withCredentialStore(t, store)
	setEnvValue(t, envServer, stringPtr("http://example.com"))
	setEnvValue(t, envToken, stringPtr("env-access-token"))

	if err := SaveOAuthCredentials("http://example.com", OAuthCredentials{
		ClientID:     "horologia-cli",
		AccessToken:  "keychain-access-token",
		RefreshToken: "keychain-refresh-token",
		TokenType:    "Bearer",
		Scope:        "profile:read",
	}); err != nil {
		t.Fatalf("save oauth credentials: %v", err)
	}

	cfg, err := ResolveConfig(ResolveInput{})
	if err != nil {
		t.Fatalf("resolve config: %v", err)
	}

	if got, want := cfg.Token, "env-access-token"; got != want {
		t.Fatalf("token = %q, want %q", got, want)
	}
	if got, want := cfg.TokenSource, ValueSourceEnv; got != want {
		t.Fatalf("token source = %q, want %q", got, want)
	}
	if cfg.OAuth != nil {
		t.Fatalf("keychain oauth credentials should not load when env token is set: %#v", cfg.OAuth)
	}
}

func TestResolveConfigIgnoresUnavailableKeychain(t *testing.T) {
	setConfigHome(t)
	withCredentialStore(t, newUnavailableCredentialStore())
	setEnvValue(t, envServer, stringPtr("http://example.com"))
	setEnvValue(t, envToken, nil)

	cfg, err := ResolveConfig(ResolveInput{})
	if err != nil {
		t.Fatalf("resolve config: %v", err)
	}

	if !cfg.HasServer() {
		t.Fatalf("server should be configured")
	}
	if cfg.HasToken() {
		t.Fatalf("token should not be configured")
	}
	if got, want := cfg.TokenSource, ValueSourceUnset; got != want {
		t.Fatalf("token source = %q, want %q", got, want)
	}
	if cfg.OAuth != nil {
		t.Fatalf("oauth credentials should not load, got %#v", cfg.OAuth)
	}
}

func TestResolveConfigReturnsOtherKeychainErrors(t *testing.T) {
	setConfigHome(t)
	withCredentialStore(t, newErrorCredentialStore(nil))
	setEnvValue(t, envServer, stringPtr("http://example.com"))
	setEnvValue(t, envToken, nil)

	_, err := ResolveConfig(ResolveInput{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "load keychain credentials") {
		t.Fatalf("unexpected error: %v", err)
	}
}

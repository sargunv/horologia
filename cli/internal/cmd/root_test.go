package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/sargunv/tend/cli/internal/runtime"
)

func TestConfigCommandUsesEnv(t *testing.T) {
	setEnvValue(t, "TEND_SERVER", stringPtr("http://example.com/api/"))
	setEnvValue(t, "TEND_TOKEN", stringPtr("tokentest1234"))

	stdout, _, err := executeRoot(t, "--json", "config")
	if err != nil {
		t.Fatalf("execute config: %v", err)
	}

	var out configOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode output: %v\noutput=%s", err, stdout)
	}

	if got, want := out.Server.Value, "http://example.com"; got != want {
		t.Fatalf("server value = %q, want %q", got, want)
	}
	if got, want := out.Server.Source, runtime.ValueSourceEnv; got != want {
		t.Fatalf("server source = %q, want %q", got, want)
	}
	if got, want := out.APIBase.Value, "http://example.com/api"; got != want {
		t.Fatalf("api base = %q, want %q", got, want)
	}
	if !out.Token.Configured {
		t.Fatalf("token should be configured")
	}
	if got, want := out.Token.Preview, "toke...1234"; got != want {
		t.Fatalf("token preview = %q, want %q", got, want)
	}
	if got, want := out.Token.Source, runtime.ValueSourceEnv; got != want {
		t.Fatalf("token source = %q, want %q", got, want)
	}
}

func TestConfigFlagOverridesEnv(t *testing.T) {
	setEnvValue(t, "TEND_SERVER", stringPtr("http://env.example.com"))
	setEnvValue(t, "TEND_TOKEN", stringPtr("envtoken1234"))

	stdout, _, err := executeRoot(t, "--json", "config")
	if err != nil {
		t.Fatalf("execute config: %v", err)
	}

	var out configOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode output: %v\noutput=%s", err, stdout)
	}

	if got, want := out.Server.Value, "http://env.example.com"; got != want {
		t.Fatalf("server value = %q, want %q", got, want)
	}
	if got, want := out.Server.Source, runtime.ValueSourceEnv; got != want {
		t.Fatalf("server source = %q, want %q", got, want)
	}
	if got, want := out.Token.Preview, "envt...1234"; got != want {
		t.Fatalf("token preview = %q, want %q", got, want)
	}
	if got, want := out.Token.Source, runtime.ValueSourceEnv; got != want {
		t.Fatalf("token source = %q, want %q", got, want)
	}
}

func TestHelpDoesNotExposeServerOrTokenFlags(t *testing.T) {
	stdout, _, err := executeRoot(t, "--help")
	if err != nil {
		t.Fatalf("execute help: %v", err)
	}

	if strings.Contains(stdout, "--server") {
		t.Fatalf("help unexpectedly contains --server flag")
	}
	if strings.Contains(stdout, "--token") {
		t.Fatalf("help unexpectedly contains --token flag")
	}
	if strings.Contains(stdout, "--timeout") {
		t.Fatalf("help unexpectedly contains --timeout flag")
	}
}

func TestPingWithoutTokenSkipsAuth(t *testing.T) {
	sawWhoami := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			writeJSON(t, w, http.StatusOK, map[string]string{"status": "ok"})
		case "/api/users/me":
			sawWhoami = true
			t.Fatalf("unexpected whoami request without a token")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "TEND_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "TEND_TOKEN", nil)

	stdout, _, err := executeRoot(t, "--json", "ping")
	if err != nil {
		t.Fatalf("execute ping: %v", err)
	}

	var out pingOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode output: %v\noutput=%s", err, stdout)
	}

	if !out.Health.OK {
		t.Fatalf("health should be ok")
	}
	if !out.Auth.Skipped {
		t.Fatalf("auth should be skipped")
	}
	if out.Auth.Reason != "token not configured" {
		t.Fatalf("unexpected skip reason %q", out.Auth.Reason)
	}
	if sawWhoami {
		t.Fatalf("whoami endpoint should not be called")
	}
}

func TestWhoamiUsesTokenAndNormalizesAPIBase(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/users/me":
			if got, want := r.Header.Get("Authorization"), "Bearer abc123"; got != want {
				t.Fatalf("authorization = %q, want %q", got, want)
			}
			writeJSON(t, w, http.StatusOK, runtime.User{
				ID:          "U1",
				Email:       "admin@localhost",
				Name:        "Admin",
				IsOwner:     true,
				HasPassword: true,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "TEND_SERVER", stringPtr(srv.URL+"/api"))
	setEnvValue(t, "TEND_TOKEN", stringPtr("abc123"))

	stdout, _, err := executeRoot(t, "--json", "whoami")
	if err != nil {
		t.Fatalf("execute whoami: %v", err)
	}

	var user runtime.User
	if err := json.Unmarshal([]byte(stdout), &user); err != nil {
		t.Fatalf("decode output: %v\noutput=%s", err, stdout)
	}

	if got, want := user.ID, "U1"; got != want {
		t.Fatalf("user id = %q, want %q", got, want)
	}
	if got, want := user.Email, "admin@localhost"; got != want {
		t.Fatalf("email = %q, want %q", got, want)
	}
	if !user.IsOwner {
		t.Fatalf("expected owner user")
	}
}

func TestWhoamiRequiresToken(t *testing.T) {
	setEnvValue(t, "TEND_SERVER", stringPtr("http://example.com"))
	setEnvValue(t, "TEND_TOKEN", nil)

	_, _, err := executeRoot(t, "whoami")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "token is not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func executeRoot(t *testing.T, args ...string) (stdout string, stderr string, err error) {
	t.Helper()

	var outBuf strings.Builder
	var errBuf strings.Builder
	root := newRootCmd(commandOptions{
		stdout: &outBuf,
		stderr: &errBuf,
	})
	root.SetArgs(args)

	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

func setEnvValue(t *testing.T, key string, value *string) {
	t.Helper()

	oldValue, hadValue := os.LookupEnv(key)
	t.Cleanup(func() {
		if hadValue {
			if err := os.Setenv(key, oldValue); err != nil {
				t.Fatalf("restore env %s: %v", key, err)
			}
			return
		}
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset env %s: %v", key, err)
		}
	})

	if value == nil {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset env %s: %v", key, err)
		}
		return
	}
	if err := os.Setenv(key, *value); err != nil {
		t.Fatalf("set env %s: %v", key, err)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode json response: %v", err)
	}
}

func stringPtr(v string) *string {
	return &v
}

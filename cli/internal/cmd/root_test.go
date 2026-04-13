package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	configcmd "github.com/sargunv/horologia/cli/internal/cmd/config"
	foundationcmd "github.com/sargunv/horologia/cli/internal/cmd/foundation"
	"github.com/sargunv/horologia/cli/internal/runtime"
)

func TestConfigCommandUsesEnv(t *testing.T) {
	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr("http://example.com/api/"))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("tokentest1234"))

	stdout, _, err := executeRoot(t, "--json", "config", "show")
	if err != nil {
		t.Fatalf("execute config: %v", err)
	}

	var out configcmd.Output
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
	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr("http://env.example.com"))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("envtoken1234"))

	stdout, _, err := executeRoot(t, "--json", "config", "show")
	if err != nil {
		t.Fatalf("execute config: %v", err)
	}

	var out configcmd.Output
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

func TestConfigCommandUsesPersistedServer(t *testing.T) {
	setConfigHome(t)
	setEnvValue(t, "HOROLOGIA_SERVER", nil)
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("tokentest1234"))

	stdout, _, err := executeRoot(t, "--json", "config", "set", "server", "http://file.example.com/api/")
	if err != nil {
		t.Fatalf("execute config set server: %v", err)
	}

	var setOut struct {
		Path   string `json:"path"`
		Server string `json:"server"`
	}
	if err := json.Unmarshal([]byte(stdout), &setOut); err != nil {
		t.Fatalf("decode set output: %v\noutput=%s", err, stdout)
	}
	if got, want := setOut.Server, "http://file.example.com"; got != want {
		t.Fatalf("persisted server = %q, want %q", got, want)
	}

	data, err := os.ReadFile(setOut.Path)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}
	if strings.Contains(string(data), "tokentest1234") {
		t.Fatalf("config file should not contain the token")
	}

	setEnvValue(t, "HOROLOGIA_TOKEN", nil)
	stdout, _, err = executeRoot(t, "--json", "config", "show")
	if err != nil {
		t.Fatalf("execute config show: %v", err)
	}

	var out configcmd.Output
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode output: %v\noutput=%s", err, stdout)
	}

	if got, want := out.Server.Value, "http://file.example.com"; got != want {
		t.Fatalf("server value = %q, want %q", got, want)
	}
	if got, want := out.Server.Source, runtime.ValueSourceFile; got != want {
		t.Fatalf("server source = %q, want %q", got, want)
	}
	if out.Token.Configured {
		t.Fatalf("token should not be configured")
	}
}

func TestConfigEnvOverridesPersistedServer(t *testing.T) {
	setConfigHome(t)
	setEnvValue(t, "HOROLOGIA_SERVER", nil)
	setEnvValue(t, "HOROLOGIA_TOKEN", nil)

	if _, _, err := executeRoot(t, "config", "set", "server", "http://file.example.com"); err != nil {
		t.Fatalf("execute config set server: %v", err)
	}

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr("http://env.example.com"))

	stdout, _, err := executeRoot(t, "--json", "config", "show")
	if err != nil {
		t.Fatalf("execute config show: %v", err)
	}

	var out configcmd.Output
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode output: %v\noutput=%s", err, stdout)
	}

	if got, want := out.Server.Value, "http://env.example.com"; got != want {
		t.Fatalf("server value = %q, want %q", got, want)
	}
	if got, want := out.Server.Source, runtime.ValueSourceEnv; got != want {
		t.Fatalf("server source = %q, want %q", got, want)
	}
}

func TestConfigPathAndUnsetServer(t *testing.T) {
	setConfigHome(t)
	setEnvValue(t, "HOROLOGIA_SERVER", nil)
	setEnvValue(t, "HOROLOGIA_TOKEN", nil)

	expectedPath, err := runtime.ConfigPath()
	if err != nil {
		t.Fatalf("resolve config path: %v", err)
	}

	stdout, _, err := executeRoot(t, "--json", "config", "path")
	if err != nil {
		t.Fatalf("execute config path: %v", err)
	}

	var pathOut struct {
		Path   string `json:"path"`
		Exists bool   `json:"exists"`
	}
	if err := json.Unmarshal([]byte(stdout), &pathOut); err != nil {
		t.Fatalf("decode path output: %v\noutput=%s", err, stdout)
	}
	if got, want := pathOut.Path, expectedPath; got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
	if pathOut.Exists {
		t.Fatalf("config path should not exist yet")
	}

	if _, _, err := executeRoot(t, "config", "set", "server", "http://file.example.com"); err != nil {
		t.Fatalf("execute config set server: %v", err)
	}
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("stat config file: %v", err)
	}

	stdout, _, err = executeRoot(t, "--json", "config", "unset", "server")
	if err != nil {
		t.Fatalf("execute config unset server: %v", err)
	}

	var unsetOut struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(stdout), &unsetOut); err != nil {
		t.Fatalf("decode unset output: %v\noutput=%s", err, stdout)
	}
	if got, want := unsetOut.Path, expectedPath; got != want {
		t.Fatalf("unset path = %q, want %q", got, want)
	}
	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Fatalf("config file should be removed, stat err=%v", err)
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

func TestStatusWithoutTokenSkipsAuth(t *testing.T) {
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

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", nil)

	stdout, _, err := executeRoot(t, "--json", "status")
	if err != nil {
		t.Fatalf("execute status: %v", err)
	}

	var out foundationcmd.Output
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

func TestUserMeUsesTokenAndNormalizesAPIBase(t *testing.T) {
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

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL+"/api"))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	stdout, _, err := executeRoot(t, "--json", "user", "me")
	if err != nil {
		t.Fatalf("execute user me: %v", err)
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

func TestUserMeRequiresToken(t *testing.T) {
	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr("http://example.com"))
	setEnvValue(t, "HOROLOGIA_TOKEN", nil)

	_, _, err := executeRoot(t, "user", "me")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "token is not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthStatusWithoutServerSkipsIdentity(t *testing.T) {
	setEnvValue(t, "HOROLOGIA_SERVER", nil)
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123456789"))

	stdout, _, err := executeRoot(t, "--json", "auth", "status")
	if err != nil {
		t.Fatalf("execute auth status: %v", err)
	}

	var out struct {
		Server struct {
			Configured bool `json:"configured"`
		} `json:"server"`
		Token struct {
			Configured bool   `json:"configured"`
			Preview    string `json:"preview"`
		} `json:"token"`
		Identity struct {
			Skipped bool   `json:"skipped"`
			Reason  string `json:"reason"`
		} `json:"identity"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode output: %v\noutput=%s", err, stdout)
	}

	if out.Server.Configured {
		t.Fatalf("server should not be configured")
	}
	if !out.Token.Configured {
		t.Fatalf("token should be configured")
	}
	if got, want := out.Token.Preview, "abc1...6789"; got != want {
		t.Fatalf("token preview = %q, want %q", got, want)
	}
	if !out.Identity.Skipped {
		t.Fatalf("identity should be skipped")
	}
	if got, want := out.Identity.Reason, "server not configured"; got != want {
		t.Fatalf("identity reason = %q, want %q", got, want)
	}
}

func TestAuthStatusWithServerChecksIdentity(t *testing.T) {
	sawWhoami := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/users/me":
			sawWhoami = true
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

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	stdout, _, err := executeRoot(t, "--json", "auth", "status")
	if err != nil {
		t.Fatalf("execute auth status: %v", err)
	}

	var out struct {
		Identity struct {
			Checked bool `json:"checked"`
			OK      bool `json:"ok"`
			User    struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"identity"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode output: %v\noutput=%s", err, stdout)
	}

	if !sawWhoami {
		t.Fatalf("expected users/me request")
	}
	if !out.Identity.Checked || !out.Identity.OK {
		t.Fatalf("identity should be checked and ok")
	}
	if got, want := out.Identity.User.ID, "U1"; got != want {
		t.Fatalf("identity user id = %q, want %q", got, want)
	}
}

func TestSpaceListUsesAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"items": []map[string]any{
					{
						"slug":        "home",
						"name":        "Home",
						"description": "Household tasks",
						"createdAt":   "2026-04-11T12:00:00Z",
						"updatedAt":   "2026-04-11T12:00:00Z",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	stdout, _, err := executeRoot(t, "--json", "space", "list")
	if err != nil {
		t.Fatalf("execute space list: %v", err)
	}

	var out struct {
		Items []struct {
			Slug string `json:"slug"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode output: %v\noutput=%s", err, stdout)
	}
	if got, want := out.Items[0].Slug, "home"; got != want {
		t.Fatalf("slug = %q, want %q", got, want)
	}
}

func TestSpaceCreateSendsBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces":
			if got, want := r.Method, http.MethodPost; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if got, want := body["slug"], "home"; got != want {
				t.Fatalf("slug = %#v, want %#v", got, want)
			}
			if got, want := body["name"], "Home"; got != want {
				t.Fatalf("name = %#v, want %#v", got, want)
			}
			if got, want := body["description"], "Household tasks"; got != want {
				t.Fatalf("description = %#v, want %#v", got, want)
			}

			writeJSON(t, w, http.StatusCreated, map[string]any{
				"slug":        "home",
				"name":        "Home",
				"description": "Household tasks",
				"createdAt":   "2026-04-11T12:00:00Z",
				"updatedAt":   "2026-04-11T12:00:00Z",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "space", "create", "--slug", "home", "--name", "Home", "--description", "Household tasks")
	if err != nil {
		t.Fatalf("execute space create: %v", err)
	}
}

func TestSpaceUpdateSendsOptionalFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home":
			if got, want := r.Method, http.MethodPatch; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if got, want := body["slug"], "household"; got != want {
				t.Fatalf("slug = %#v, want %#v", got, want)
			}
			if got, want := body["name"], "Household"; got != want {
				t.Fatalf("name = %#v, want %#v", got, want)
			}
			if got, want := body["description"], "Updated"; got != want {
				t.Fatalf("description = %#v, want %#v", got, want)
			}

			writeJSON(t, w, http.StatusOK, map[string]any{
				"slug":        "household",
				"name":        "Household",
				"description": "Updated",
				"createdAt":   "2026-04-11T12:00:00Z",
				"updatedAt":   "2026-04-11T13:00:00Z",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "space", "update", "home", "--slug", "household", "--name", "Household", "--description", "Updated")
	if err != nil {
		t.Fatalf("execute space update: %v", err)
	}
}

func TestSpaceActivityPassesPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/activity":
			if got, want := r.URL.Query().Get("cursor"), "next-1"; got != want {
				t.Fatalf("cursor = %q, want %q", got, want)
			}
			if got, want := r.URL.Query().Get("limit"), "10"; got != want {
				t.Fatalf("limit = %q, want %q", got, want)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"items":      []map[string]any{},
				"nextCursor": "next-2",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	stdout, _, err := executeRoot(t, "--json", "space", "activity", "home", "--cursor", "next-1", "--limit", "10")
	if err != nil {
		t.Fatalf("execute space activity: %v", err)
	}

	var out struct {
		NextCursor string `json:"nextCursor"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode output: %v\noutput=%s", err, stdout)
	}
	if got, want := out.NextCursor, "next-2"; got != want {
		t.Fatalf("next cursor = %q, want %q", got, want)
	}
}

func TestSpaceMemberListUsesAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/members":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"items": []map[string]any{
					{
						"userId":    "U1",
						"userName":  "Admin",
						"userEmail": "admin@localhost",
						"role":      "admin",
						"createdAt": "2026-04-11T12:00:00Z",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	stdout, _, err := executeRoot(t, "--json", "space", "member", "list", "home")
	if err != nil {
		t.Fatalf("execute space member list: %v", err)
	}

	var out struct {
		Items []struct {
			UserID string `json:"userId"`
			Role   string `json:"role"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode output: %v\noutput=%s", err, stdout)
	}
	if got, want := out.Items[0].UserID, "U1"; got != want {
		t.Fatalf("user id = %q, want %q", got, want)
	}
	if got, want := out.Items[0].Role, "admin"; got != want {
		t.Fatalf("role = %q, want %q", got, want)
	}
}

func TestSpaceMemberAddSendsBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/members":
			if got, want := r.Method, http.MethodPost; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}

			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if got, want := body["userId"], "U2"; got != want {
				t.Fatalf("userId = %#v, want %#v", got, want)
			}
			if got, want := body["role"], "viewer"; got != want {
				t.Fatalf("role = %#v, want %#v", got, want)
			}

			writeJSON(t, w, http.StatusCreated, map[string]any{
				"userId":    "U2",
				"userName":  "Bob",
				"userEmail": "bob@example.com",
				"role":      "viewer",
				"createdAt": "2026-04-11T12:00:00Z",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "space", "member", "add", "home", "U2", "--role", "viewer")
	if err != nil {
		t.Fatalf("execute space member add: %v", err)
	}
}

func TestSpaceMemberSetRoleUsesPatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/members/U2":
			if got, want := r.Method, http.MethodPatch; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}

			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if got, want := body["role"], "member"; got != want {
				t.Fatalf("role = %#v, want %#v", got, want)
			}

			writeJSON(t, w, http.StatusOK, map[string]any{
				"userId":    "U2",
				"userName":  "Bob",
				"userEmail": "bob@example.com",
				"role":      "member",
				"createdAt": "2026-04-11T12:00:00Z",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "space", "member", "set-role", "home", "U2", "member")
	if err != nil {
		t.Fatalf("execute space member set-role: %v", err)
	}
}

func TestSpaceMemberRemoveUsesDelete(t *testing.T) {
	sawDelete := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/members/U2":
			sawDelete = true
			if got, want := r.Method, http.MethodDelete; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "space", "member", "remove", "home", "U2")
	if err != nil {
		t.Fatalf("execute space member remove: %v", err)
	}
	if !sawDelete {
		t.Fatalf("expected delete request")
	}
}

func TestSpaceTagListUsesAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/tags":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"items": []map[string]any{
					{
						"name":      "bug",
						"createdAt": "2026-04-11T12:00:00Z",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	stdout, _, err := executeRoot(t, "--json", "space", "tag", "list", "home")
	if err != nil {
		t.Fatalf("execute space tag list: %v", err)
	}

	var out struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode output: %v\noutput=%s", err, stdout)
	}
	if got, want := out.Items[0].Name, "bug"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
}

func TestSpaceTagCreateSendsBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/tags":
			if got, want := r.Method, http.MethodPost; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}

			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if got, want := body["name"], "bug"; got != want {
				t.Fatalf("name = %#v, want %#v", got, want)
			}

			writeJSON(t, w, http.StatusCreated, map[string]any{
				"name":      "bug",
				"createdAt": "2026-04-11T12:00:00Z",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "space", "tag", "create", "home", "--name", "bug")
	if err != nil {
		t.Fatalf("execute space tag create: %v", err)
	}
}

func TestSpaceTagRenameUsesPatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/tags/bug":
			if got, want := r.Method, http.MethodPatch; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}

			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if got, want := body["name"], "urgent"; got != want {
				t.Fatalf("name = %#v, want %#v", got, want)
			}

			writeJSON(t, w, http.StatusOK, map[string]any{
				"name":      "urgent",
				"createdAt": "2026-04-11T12:00:00Z",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "space", "tag", "rename", "home", "bug", "urgent")
	if err != nil {
		t.Fatalf("execute space tag rename: %v", err)
	}
}

func TestSpaceTagDeleteUsesDelete(t *testing.T) {
	sawDelete := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/tags/bug":
			sawDelete = true
			if got, want := r.Method, http.MethodDelete; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "space", "tag", "delete", "home", "bug")
	if err != nil {
		t.Fatalf("execute space tag delete: %v", err)
	}
	if !sawDelete {
		t.Fatalf("expected delete request")
	}
}

func TestSpaceStatusReplaceSendsOrderedCategories(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/task-statuses":
			if got, want := r.Method, http.MethodPut; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}

			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			items, ok := body["items"].([]any)
			if !ok {
				t.Fatalf("items missing or wrong type: %#v", body["items"])
			}
			if got, want := len(items), 4; got != want {
				t.Fatalf("item count = %d, want %d", got, want)
			}

			first := items[0].(map[string]any)
			second := items[1].(map[string]any)
			third := items[2].(map[string]any)
			fourth := items[3].(map[string]any)
			if got, want := first["name"], "todo"; got != want {
				t.Fatalf("first name = %#v, want %#v", got, want)
			}
			if got, want := first["category"], "initial"; got != want {
				t.Fatalf("first category = %#v, want %#v", got, want)
			}
			if got, want := second["category"], "intermediate"; got != want {
				t.Fatalf("second category = %#v, want %#v", got, want)
			}
			if got, want := third["category"], "intermediate"; got != want {
				t.Fatalf("third category = %#v, want %#v", got, want)
			}
			if got, want := fourth["category"], "completion"; got != want {
				t.Fatalf("fourth category = %#v, want %#v", got, want)
			}

			writeJSON(t, w, http.StatusOK, map[string]any{
				"items": []map[string]any{
					{"name": "todo", "category": "initial", "position": 1, "icon": ""},
					{"name": "doing", "category": "intermediate", "position": 2, "icon": ""},
					{"name": "blocked", "category": "intermediate", "position": 3, "icon": ""},
					{"name": "done", "category": "completion", "position": 4, "icon": ""},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(
		t,
		"space", "status", "replace", "home",
		"--initial", "todo",
		"--intermediate", "doing",
		"--intermediate", "blocked",
		"--completion", "done",
	)
	if err != nil {
		t.Fatalf("execute space status replace: %v", err)
	}
}

func TestSpaceStatusReplaceValidatesCompletionLocally(t *testing.T) {
	setEnvValue(t, "HOROLOGIA_SERVER", nil)
	setEnvValue(t, "HOROLOGIA_TOKEN", nil)

	_, _, err := executeRoot(t, "space", "status", "replace", "home", "--initial", "todo")
	if err == nil {
		t.Fatalf("expected error")
	}
	if got, want := err.Error(), "at least one completion status is required"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestSpaceEffortReplaceSendsOrderedLevels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/task-effort-levels":
			if got, want := r.Method, http.MethodPut; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}

			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			items := body["items"].([]any)
			if got, want := items[0].(map[string]any)["name"], "small"; got != want {
				t.Fatalf("first name = %#v, want %#v", got, want)
			}
			if got, want := items[1].(map[string]any)["name"], "medium"; got != want {
				t.Fatalf("second name = %#v, want %#v", got, want)
			}

			writeJSON(t, w, http.StatusOK, map[string]any{
				"items": []map[string]any{
					{"name": "small", "position": 1, "icon": ""},
					{"name": "medium", "position": 2, "icon": ""},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "space", "effort", "replace", "home", "--name", "small", "--name", "medium")
	if err != nil {
		t.Fatalf("execute space effort replace: %v", err)
	}
}

func TestSpaceEffortReplaceAllowsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/task-effort-levels":
			if got, want := r.Method, http.MethodPut; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}

			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			items, ok := body["items"].([]any)
			if !ok {
				t.Fatalf("items missing or wrong type: %#v", body["items"])
			}
			if len(items) != 0 {
				t.Fatalf("item count = %d, want 0", len(items))
			}

			writeJSON(t, w, http.StatusOK, map[string]any{
				"items": []map[string]any{},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "space", "effort", "replace", "home")
	if err != nil {
		t.Fatalf("execute space effort replace: %v", err)
	}
}

func TestSpacePriorityReplaceSendsOrderedLevels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/task-priority-levels":
			if got, want := r.Method, http.MethodPut; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}

			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			items := body["items"].([]any)
			if got, want := items[0].(map[string]any)["name"], "low"; got != want {
				t.Fatalf("first name = %#v, want %#v", got, want)
			}
			if got, want := items[1].(map[string]any)["name"], "high"; got != want {
				t.Fatalf("second name = %#v, want %#v", got, want)
			}

			writeJSON(t, w, http.StatusOK, map[string]any{
				"items": []map[string]any{
					{"name": "low", "position": 1, "icon": ""},
					{"name": "high", "position": 2, "icon": ""},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "space", "priority", "replace", "home", "--name", "low", "--name", "high")
	if err != nil {
		t.Fatalf("execute space priority replace: %v", err)
	}
}

func TestSpacePriorityReplaceAllowsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/task-priority-levels":
			if got, want := r.Method, http.MethodPut; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}

			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			items, ok := body["items"].([]any)
			if !ok {
				t.Fatalf("items missing or wrong type: %#v", body["items"])
			}
			if len(items) != 0 {
				t.Fatalf("item count = %d, want 0", len(items))
			}

			writeJSON(t, w, http.StatusOK, map[string]any{
				"items": []map[string]any{},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "space", "priority", "replace", "home")
	if err != nil {
		t.Fatalf("execute space priority replace: %v", err)
	}
}

func TestTaskListPassesPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/tasks":
			if got, want := r.URL.Query().Get("cursor"), "next-1"; got != want {
				t.Fatalf("cursor = %q, want %q", got, want)
			}
			if got, want := r.URL.Query().Get("limit"), "10"; got != want {
				t.Fatalf("limit = %q, want %q", got, want)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"items": []map[string]any{
					{
						"id":                "T1",
						"spaceSlug":         "home",
						"title":             "First task",
						"description":       "",
						"status":            "todo",
						"effort":            nil,
						"priority":          nil,
						"recurrenceType":    "one_off",
						"recurrenceRule":    nil,
						"lastCompletedAt":   nil,
						"assigneeIds":       []string{},
						"rotationPool":      []string{},
						"tags":              []string{},
						"relations":         []map[string]any{},
						"due":               nil,
						"overdueActionRule": nil,
						"createdAt":         "2026-04-11T12:00:00Z",
						"updatedAt":         "2026-04-11T12:00:00Z",
					},
				},
				"nextCursor": "next-2",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	stdout, _, err := executeRoot(t, "--json", "task", "list", "home", "--cursor", "next-1", "--limit", "10")
	if err != nil {
		t.Fatalf("execute task list: %v", err)
	}

	var out struct {
		NextCursor string `json:"nextCursor"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode output: %v\noutput=%s", err, stdout)
	}
	if got, want := out.NextCursor, "next-2"; got != want {
		t.Fatalf("next cursor = %q, want %q", got, want)
	}
}

func TestTaskMineUsesCurrentUser(t *testing.T) {
	sawWhoami := false
	sawTasks := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/users/me":
			sawWhoami = true
			writeJSON(t, w, http.StatusOK, map[string]any{
				"id":          "U1",
				"email":       "admin@localhost",
				"name":        "Admin",
				"isOwner":     true,
				"hasPassword": true,
				"createdAt":   "2026-04-11T12:00:00Z",
				"updatedAt":   "2026-04-11T12:00:00Z",
			})
		case "/api/users/U1/tasks":
			sawTasks = true
			writeJSON(t, w, http.StatusOK, map[string]any{
				"items":      []map[string]any{},
				"nextCursor": nil,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "task", "mine")
	if err != nil {
		t.Fatalf("execute task mine: %v", err)
	}
	if !sawWhoami || !sawTasks {
		t.Fatalf("expected users/me and users/U1/tasks requests")
	}
}

func TestTaskCreateSendsBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/tasks":
			if got, want := r.Method, http.MethodPost; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}

			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if got, want := body["title"], "Task title"; got != want {
				t.Fatalf("title = %#v, want %#v", got, want)
			}
			if got, want := body["status"], "todo"; got != want {
				t.Fatalf("status = %#v, want %#v", got, want)
			}
			if got, want := body["priority"], "high"; got != want {
				t.Fatalf("priority = %#v, want %#v", got, want)
			}

			writeJSON(t, w, http.StatusCreated, map[string]any{
				"id":                "T1",
				"spaceSlug":         "home",
				"title":             "Task title",
				"description":       "",
				"status":            "todo",
				"effort":            nil,
				"priority":          "high",
				"recurrenceType":    "one_off",
				"recurrenceRule":    nil,
				"lastCompletedAt":   nil,
				"assigneeIds":       []string{},
				"rotationPool":      []string{},
				"tags":              []string{},
				"relations":         []map[string]any{},
				"due":               nil,
				"overdueActionRule": nil,
				"createdAt":         "2026-04-11T12:00:00Z",
				"updatedAt":         "2026-04-11T12:00:00Z",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "task", "create", "home", "--title", "Task title", "--status", "todo", "--priority", "high")
	if err != nil {
		t.Fatalf("execute task create: %v", err)
	}
}

func TestTaskUpdateSendsScalarPatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/tasks/T1":
			if got, want := r.Method, http.MethodPatch; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}

			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if got, want := body["title"], "Updated"; got != want {
				t.Fatalf("title = %#v, want %#v", got, want)
			}
			if body["effort"] != nil {
				t.Fatalf("effort = %#v, want nil", body["effort"])
			}
			if got, want := body["priority"], "low"; got != want {
				t.Fatalf("priority = %#v, want %#v", got, want)
			}

			writeJSON(t, w, http.StatusOK, map[string]any{
				"id":                "T1",
				"spaceSlug":         "home",
				"title":             "Updated",
				"description":       "",
				"status":            "todo",
				"effort":            nil,
				"priority":          "low",
				"recurrenceType":    "one_off",
				"recurrenceRule":    nil,
				"lastCompletedAt":   nil,
				"assigneeIds":       []string{},
				"rotationPool":      []string{},
				"tags":              []string{},
				"relations":         []map[string]any{},
				"due":               nil,
				"overdueActionRule": nil,
				"createdAt":         "2026-04-11T12:00:00Z",
				"updatedAt":         "2026-04-11T13:00:00Z",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "task", "update", "home", "T1", "--title", "Updated", "--clear-effort", "--priority", "low")
	if err != nil {
		t.Fatalf("execute task update: %v", err)
	}
}

func TestTaskCompleteUsesFirstCompletionStatus(t *testing.T) {
	step := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/task-statuses":
			step++
			writeJSON(t, w, http.StatusOK, map[string]any{
				"items": []map[string]any{
					{"name": "todo", "category": "initial", "position": 1, "icon": ""},
					{"name": "done", "category": "completion", "position": 2, "icon": ""},
					{"name": "archived", "category": "completion", "position": 3, "icon": ""},
				},
			})
		case "/api/spaces/home/tasks/T1":
			step++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if got, want := body["status"], "done"; got != want {
				t.Fatalf("status = %#v, want %#v", got, want)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"id":                "T1",
				"spaceSlug":         "home",
				"title":             "Task",
				"description":       "",
				"status":            "done",
				"effort":            nil,
				"priority":          nil,
				"recurrenceType":    "one_off",
				"recurrenceRule":    nil,
				"lastCompletedAt":   "2026-04-11T13:00:00Z",
				"assigneeIds":       []string{},
				"rotationPool":      []string{},
				"tags":              []string{},
				"relations":         []map[string]any{},
				"due":               nil,
				"overdueActionRule": nil,
				"createdAt":         "2026-04-11T12:00:00Z",
				"updatedAt":         "2026-04-11T13:00:00Z",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "task", "complete", "home", "T1")
	if err != nil {
		t.Fatalf("execute task complete: %v", err)
	}
	if got, want := step, 2; got != want {
		t.Fatalf("step count = %d, want %d", got, want)
	}
}

func TestTaskActivityPassesPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/tasks/T1/activity":
			if got, want := r.URL.Query().Get("cursor"), "next-1"; got != want {
				t.Fatalf("cursor = %q, want %q", got, want)
			}
			if got, want := r.URL.Query().Get("limit"), "5"; got != want {
				t.Fatalf("limit = %q, want %q", got, want)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"items":      []map[string]any{},
				"nextCursor": "next-2",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	stdout, _, err := executeRoot(t, "--json", "task", "activity", "home", "T1", "--cursor", "next-1", "--limit", "5")
	if err != nil {
		t.Fatalf("execute task activity: %v", err)
	}

	var out struct {
		NextCursor string `json:"nextCursor"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode output: %v\noutput=%s", err, stdout)
	}
	if got, want := out.NextCursor, "next-2"; got != want {
		t.Fatalf("next cursor = %q, want %q", got, want)
	}
}

func TestTaskDueSetSendsPatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/tasks/T1":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			due, ok := body["due"].(map[string]any)
			if !ok {
				t.Fatalf("due missing or wrong type: %#v", body["due"])
			}
			if got, want := due["at"], "2026-05-01"; got != want {
				t.Fatalf("due.at = %#v, want %#v", got, want)
			}
			if got, want := due["timezone"], "UTC"; got != want {
				t.Fatalf("due.timezone = %#v, want %#v", got, want)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"id":                "T1",
				"spaceSlug":         "home",
				"title":             "Task",
				"description":       "",
				"status":            "todo",
				"effort":            nil,
				"priority":          nil,
				"recurrenceType":    "one_off",
				"recurrenceRule":    nil,
				"lastCompletedAt":   nil,
				"assigneeIds":       []string{},
				"rotationPool":      []string{},
				"tags":              []string{},
				"relations":         []map[string]any{},
				"due":               map[string]any{"at": "2026-05-01", "timezone": "UTC"},
				"overdueActionRule": nil,
				"createdAt":         "2026-04-11T12:00:00Z",
				"updatedAt":         "2026-04-11T13:00:00Z",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "task", "due", "set", "home", "T1", "--date", "2026-05-01", "--timezone", "UTC")
	if err != nil {
		t.Fatalf("execute task due set: %v", err)
	}
}

func TestTaskAssigneeAddReadsThenPatches(t *testing.T) {
	readCount := 0
	patchCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/tasks/T1":
			if r.Method == http.MethodGet {
				readCount++
				writeJSON(t, w, http.StatusOK, map[string]any{
					"id":                "T1",
					"spaceSlug":         "home",
					"title":             "Task",
					"description":       "",
					"status":            "todo",
					"effort":            nil,
					"priority":          nil,
					"recurrenceType":    "one_off",
					"recurrenceRule":    nil,
					"lastCompletedAt":   nil,
					"assigneeIds":       []string{"U1"},
					"rotationPool":      []string{},
					"tags":              []string{},
					"relations":         []map[string]any{},
					"due":               nil,
					"overdueActionRule": nil,
					"createdAt":         "2026-04-11T12:00:00Z",
					"updatedAt":         "2026-04-11T12:00:00Z",
				})
				return
			}
			patchCount++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if got, want := body["assigneeIds"], []any{"U1", "U2"}; !reflect.DeepEqual(got, want) {
				t.Fatalf("assigneeIds = %#v, want %#v", got, want)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"id":                "T1",
				"spaceSlug":         "home",
				"title":             "Task",
				"description":       "",
				"status":            "todo",
				"effort":            nil,
				"priority":          nil,
				"recurrenceType":    "one_off",
				"recurrenceRule":    nil,
				"lastCompletedAt":   nil,
				"assigneeIds":       []string{"U1", "U2"},
				"rotationPool":      []string{},
				"tags":              []string{},
				"relations":         []map[string]any{},
				"due":               nil,
				"overdueActionRule": nil,
				"createdAt":         "2026-04-11T12:00:00Z",
				"updatedAt":         "2026-04-11T13:00:00Z",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "task", "assignee", "add", "home", "T1", "U2")
	if err != nil {
		t.Fatalf("execute task assignee add: %v", err)
	}
	if readCount != 1 || patchCount != 1 {
		t.Fatalf("readCount=%d patchCount=%d, want 1/1", readCount, patchCount)
	}
}

func TestTaskRotationClearSendsEmptyList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/tasks/T1":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if got, want := body["rotationPool"], []any{}; !reflect.DeepEqual(got, want) {
				t.Fatalf("rotationPool = %#v, want %#v", got, want)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"id":                "T1",
				"spaceSlug":         "home",
				"title":             "Task",
				"description":       "",
				"status":            "todo",
				"effort":            nil,
				"priority":          nil,
				"recurrenceType":    "one_off",
				"recurrenceRule":    nil,
				"lastCompletedAt":   nil,
				"assigneeIds":       []string{},
				"rotationPool":      []string{},
				"tags":              []string{},
				"relations":         []map[string]any{},
				"due":               nil,
				"overdueActionRule": nil,
				"createdAt":         "2026-04-11T12:00:00Z",
				"updatedAt":         "2026-04-11T13:00:00Z",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "task", "rotation", "clear", "home", "T1")
	if err != nil {
		t.Fatalf("execute task rotation clear: %v", err)
	}
}

func TestTaskRelationAddUsesCreateEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/tasks/T1/relations":
			if got, want := r.Method, http.MethodPost; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if got, want := body["kind"], "blocks"; got != want {
				t.Fatalf("kind = %#v, want %#v", got, want)
			}
			if got, want := body["relatedTaskId"], "T2"; got != want {
				t.Fatalf("relatedTaskId = %#v, want %#v", got, want)
			}
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"kind":          "blocks",
				"relatedTaskId": "T2",
				"createdAt":     "2026-04-11T12:00:00Z",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "task", "relation", "add", "home", "T1", "blocks", "T2")
	if err != nil {
		t.Fatalf("execute task relation add: %v", err)
	}
}

func TestTaskRecurrenceSetSendsPatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/tasks/T1":
			if got, want := r.Method, http.MethodPatch; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if got, want := body["recurrenceType"], "completion_based"; got != want {
				t.Fatalf("recurrenceType = %#v, want %#v", got, want)
			}
			if got, want := body["recurrenceRule"], "FREQ=WEEKLY"; got != want {
				t.Fatalf("recurrenceRule = %#v, want %#v", got, want)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"id":                "T1",
				"spaceSlug":         "home",
				"title":             "Task",
				"description":       "",
				"status":            "todo",
				"effort":            nil,
				"priority":          nil,
				"recurrenceType":    "completion_based",
				"recurrenceRule":    "FREQ=WEEKLY",
				"lastCompletedAt":   nil,
				"assigneeIds":       []string{},
				"rotationPool":      []string{},
				"tags":              []string{},
				"relations":         []map[string]any{},
				"due":               nil,
				"overdueActionRule": nil,
				"createdAt":         "2026-04-11T12:00:00Z",
				"updatedAt":         "2026-04-11T13:00:00Z",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "task", "recurrence", "set", "home", "T1", "--type", "completion_based", "--rule", "FREQ=WEEKLY")
	if err != nil {
		t.Fatalf("execute task recurrence set: %v", err)
	}
}

func TestTaskRecurrenceClearSetsOneOff(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/tasks/T1":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if got, want := body["recurrenceType"], "one_off"; got != want {
				t.Fatalf("recurrenceType = %#v, want %#v", got, want)
			}
			if _, ok := body["recurrenceRule"]; ok {
				t.Fatalf("recurrenceRule should be omitted when clearing recurrence")
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"id":                "T1",
				"spaceSlug":         "home",
				"title":             "Task",
				"description":       "",
				"status":            "todo",
				"effort":            nil,
				"priority":          nil,
				"recurrenceType":    "one_off",
				"recurrenceRule":    nil,
				"lastCompletedAt":   nil,
				"assigneeIds":       []string{},
				"rotationPool":      []string{},
				"tags":              []string{},
				"relations":         []map[string]any{},
				"due":               nil,
				"overdueActionRule": nil,
				"createdAt":         "2026-04-11T12:00:00Z",
				"updatedAt":         "2026-04-11T13:00:00Z",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "task", "recurrence", "clear", "home", "T1")
	if err != nil {
		t.Fatalf("execute task recurrence clear: %v", err)
	}
}

func TestTaskOverdueActionSetSendsPatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/tasks/T1":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			rule, ok := body["overdueActionRule"].(map[string]any)
			if !ok {
				t.Fatalf("overdueActionRule missing or wrong type: %#v", body["overdueActionRule"])
			}
			if got, want := rule["action"], "set_status"; got != want {
				t.Fatalf("action = %#v, want %#v", got, want)
			}
			if got, want := rule["after"], float64(3); got != want {
				t.Fatalf("after = %#v, want %#v", got, want)
			}
			if got, want := rule["status"], "done"; got != want {
				t.Fatalf("status = %#v, want %#v", got, want)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"id":                "T1",
				"spaceSlug":         "home",
				"title":             "Task",
				"description":       "",
				"status":            "todo",
				"effort":            nil,
				"priority":          nil,
				"recurrenceType":    "completion_based",
				"recurrenceRule":    "FREQ=WEEKLY",
				"lastCompletedAt":   nil,
				"assigneeIds":       []string{},
				"rotationPool":      []string{},
				"tags":              []string{},
				"relations":         []map[string]any{},
				"due":               map[string]any{"at": "2026-05-01", "timezone": "UTC"},
				"overdueActionRule": map[string]any{"after": 3, "action": "set_status", "status": "done"},
				"createdAt":         "2026-04-11T12:00:00Z",
				"updatedAt":         "2026-04-11T13:00:00Z",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "task", "overdue-action", "set", "home", "T1", "--action", "set_status", "--after-days", "3", "--status", "done")
	if err != nil {
		t.Fatalf("execute task overdue-action set: %v", err)
	}
}

func TestTaskOverdueActionClearSendsNullPatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/tasks/T1":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if body["overdueActionRule"] != nil {
				t.Fatalf("overdueActionRule = %#v, want nil", body["overdueActionRule"])
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"id":                "T1",
				"spaceSlug":         "home",
				"title":             "Task",
				"description":       "",
				"status":            "todo",
				"effort":            nil,
				"priority":          nil,
				"recurrenceType":    "completion_based",
				"recurrenceRule":    "FREQ=WEEKLY",
				"lastCompletedAt":   nil,
				"assigneeIds":       []string{},
				"rotationPool":      []string{},
				"tags":              []string{},
				"relations":         []map[string]any{},
				"due":               map[string]any{"at": "2026-05-01", "timezone": "UTC"},
				"overdueActionRule": nil,
				"createdAt":         "2026-04-11T12:00:00Z",
				"updatedAt":         "2026-04-11T13:00:00Z",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "task", "overdue-action", "clear", "home", "T1")
	if err != nil {
		t.Fatalf("execute task overdue-action clear: %v", err)
	}
}

func TestTaskOverdueActionSetValidatesStatusLocally(t *testing.T) {
	setEnvValue(t, "HOROLOGIA_SERVER", nil)
	setEnvValue(t, "HOROLOGIA_TOKEN", nil)

	_, _, err := executeRoot(t, "task", "overdue-action", "set", "home", "T1", "--action", "set_status")
	if err == nil {
		t.Fatalf("expected error")
	}
	if got, want := err.Error(), "status is required when action is set_status"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestAuthTokenCreateSendsName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/tokens":
			if got, want := r.Method, http.MethodPost; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}
			if got, want := r.Header.Get("Authorization"), "Bearer abc123"; got != want {
				t.Fatalf("authorization = %q, want %q", got, want)
			}

			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if got, want := body["name"], "CLI token"; got != want {
				t.Fatalf("name = %#v, want %#v", got, want)
			}

			writeJSON(t, w, http.StatusCreated, map[string]any{
				"token": "secret-token",
				"authToken": map[string]any{
					"id":        "tok_1",
					"name":      "CLI token",
					"kind":      "api",
					"createdAt": "2026-04-11T12:00:00Z",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	stdout, _, err := executeRoot(t, "--json", "auth", "token", "create", "--name", "CLI token")
	if err != nil {
		t.Fatalf("execute auth token create: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode output: %v\noutput=%s", err, stdout)
	}
	if got, want := out["token"], "secret-token"; got != want {
		t.Fatalf("token = %#v, want %#v", got, want)
	}
}

func TestAuthTokenRevokeUsesDelete(t *testing.T) {
	sawDelete := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/tokens/tok_1":
			sawDelete = true
			if got, want := r.Method, http.MethodDelete; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(t, "auth", "token", "revoke", "tok_1")
	if err != nil {
		t.Fatalf("execute auth token revoke: %v", err)
	}
	if !sawDelete {
		t.Fatalf("expected delete request")
	}
}

func TestUserUpdateSendsOptionalFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/users/U1":
			if got, want := r.Method, http.MethodPatch; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}

			if got, want := body["name"], "Updated"; got != want {
				t.Fatalf("name = %#v, want %#v", got, want)
			}
			if got, want := body["email"], "updated@example.com"; got != want {
				t.Fatalf("email = %#v, want %#v", got, want)
			}
			if got, want := body["isOwner"], false; got != want {
				t.Fatalf("isOwner = %#v, want %#v", got, want)
			}
			if got, want := body["setPassword"], "new-password"; got != want {
				t.Fatalf("setPassword = %#v, want %#v", got, want)
			}

			writeJSON(t, w, http.StatusOK, map[string]any{
				"id":          "U1",
				"email":       "updated@example.com",
				"name":        "Updated",
				"isOwner":     false,
				"hasPassword": true,
				"createdAt":   "2026-04-11T12:00:00Z",
				"updatedAt":   "2026-04-11T13:00:00Z",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	_, _, err := executeRoot(
		t,
		"user", "update", "U1",
		"--name", "Updated",
		"--email", "updated@example.com",
		"--owner=false",
		"--set-password", "new-password",
	)
	if err != nil {
		t.Fatalf("execute user update: %v", err)
	}
}

func TestUserTasksPassesPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/users/U1/tasks":
			if got, want := r.URL.Query().Get("cursor"), "next-1"; got != want {
				t.Fatalf("cursor = %q, want %q", got, want)
			}
			if got, want := r.URL.Query().Get("limit"), "25"; got != want {
				t.Fatalf("limit = %q, want %q", got, want)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"items":      []map[string]any{},
				"nextCursor": "next-2",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setEnvValue(t, "HOROLOGIA_SERVER", stringPtr(srv.URL))
	setEnvValue(t, "HOROLOGIA_TOKEN", stringPtr("abc123"))

	stdout, _, err := executeRoot(t, "--json", "user", "tasks", "U1", "--cursor", "next-1", "--limit", "25")
	if err != nil {
		t.Fatalf("execute user tasks: %v", err)
	}

	var out struct {
		NextCursor string `json:"nextCursor"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode output: %v\noutput=%s", err, stdout)
	}
	if got, want := out.NextCursor, "next-2"; got != want {
		t.Fatalf("next cursor = %q, want %q", got, want)
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

func setConfigHome(t *testing.T) {
	t.Helper()

	home := t.TempDir()
	setEnvValue(t, "HOME", stringPtr(home))
	setEnvValue(t, "XDG_CONFIG_HOME", stringPtr(home))
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

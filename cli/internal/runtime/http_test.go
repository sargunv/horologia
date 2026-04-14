package runtime

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAppRefreshesExpiredAccessTokenAndRetriesOnce(t *testing.T) {
	setConfigHome(t)
	withCredentialStore(t, newMemoryCredentialStore())

	protectedCalls := 0
	refreshCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/protected":
			protectedCalls++
			switch protectedCalls {
			case 1:
				if got, want := r.Header.Get("Authorization"), "Bearer stale-access-token"; got != want {
					t.Fatalf("first authorization = %q, want %q", got, want)
				}
				http.Error(w, "unauthorized", http.StatusUnauthorized)
			case 2:
				if got, want := r.Header.Get("Authorization"), "Bearer fresh-access-token"; got != want {
					t.Fatalf("retry authorization = %q, want %q", got, want)
				}
				if got, want := r.Header.Get("X-Horologia-Retry"), "1"; got != want {
					t.Fatalf("retry header = %q, want %q", got, want)
				}
				writeJSONResponse(t, w, http.StatusOK, map[string]string{"status": "ok"})
			default:
				t.Fatalf("unexpected extra protected request %d", protectedCalls)
			}
		case "/oauth/token":
			refreshCalls++
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse refresh form: %v", err)
			}
			if got, want := r.PostForm.Get("grant_type"), "refresh_token"; got != want {
				t.Fatalf("grant_type = %q, want %q", got, want)
			}
			if got, want := r.PostForm.Get("client_id"), "horologia-cli"; got != want {
				t.Fatalf("client_id = %q, want %q", got, want)
			}
			if got, want := r.PostForm.Get("refresh_token"), "stale-refresh-token"; got != want {
				t.Fatalf("refresh_token = %q, want %q", got, want)
			}
			writeJSONResponse(t, w, http.StatusOK, map[string]any{
				"access_token":  "fresh-access-token",
				"refresh_token": "fresh-refresh-token",
				"token_type":    "Bearer",
				"expires_in":    3600,
				"scope":         "profile:read",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	serverURL, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	app := NewApp(Config{
		ServerRaw:    srv.URL,
		Server:       serverURL,
		ServerSource: ValueSourceEnv,
		Token:        "stale-access-token",
		TokenSource:  ValueSourceKeychain,
		OAuth: &OAuthCredentials{
			ClientID:     "horologia-cli",
			AccessToken:  "stale-access-token",
			RefreshToken: "stale-refresh-token",
			TokenType:    "Bearer",
			Scope:        "profile:read",
		},
	}, io.Discard, io.Discard)

	if err := SaveOAuthCredentials(srv.URL, *app.Config.OAuth); err != nil {
		t.Fatalf("save oauth credentials: %v", err)
	}

	var out map[string]string
	err = app.doJSON(context.Background(), requestSpec{
		Method: http.MethodGet,
		URL:    resolveURL(serverURL, "/api/protected"),
		Auth:   true,
	}, &out)
	if err != nil {
		t.Fatalf("do json: %v", err)
	}

	if got, want := out["status"], "ok"; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if got, want := protectedCalls, 2; got != want {
		t.Fatalf("protected calls = %d, want %d", got, want)
	}
	if got, want := refreshCalls, 1; got != want {
		t.Fatalf("refresh calls = %d, want %d", got, want)
	}
	if got, want := app.BearerToken(), "fresh-access-token"; got != want {
		t.Fatalf("bearer token = %q, want %q", got, want)
	}

	saved, err := LoadOAuthCredentials(srv.URL)
	if err != nil {
		t.Fatalf("load oauth credentials: %v", err)
	}
	if saved == nil {
		t.Fatal("expected saved oauth credentials")
	}
	if got, want := saved.RefreshToken, "fresh-refresh-token"; got != want {
		t.Fatalf("saved refresh token = %q, want %q", got, want)
	}
}

func TestAppRefreshFailureRequiresRelogin(t *testing.T) {
	setConfigHome(t)
	withCredentialStore(t, newMemoryCredentialStore())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/protected":
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		case "/oauth/token":
			writeJSONResponse(t, w, http.StatusBadRequest, map[string]string{
				"error":             "invalid_grant",
				"error_description": "refresh token is invalid",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	serverURL, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	app := NewApp(Config{
		ServerRaw:    srv.URL,
		Server:       serverURL,
		ServerSource: ValueSourceEnv,
		Token:        "stale-access-token",
		TokenSource:  ValueSourceKeychain,
		OAuth: &OAuthCredentials{
			ClientID:     "horologia-cli",
			AccessToken:  "stale-access-token",
			RefreshToken: "stale-refresh-token",
			TokenType:    "Bearer",
			Scope:        "profile:read",
		},
	}, io.Discard, io.Discard)

	if err := SaveOAuthCredentials(srv.URL, *app.Config.OAuth); err != nil {
		t.Fatalf("save oauth credentials: %v", err)
	}

	var out map[string]string
	err = app.doJSON(context.Background(), requestSpec{
		Method: http.MethodGet,
		URL:    resolveURL(serverURL, "/api/protected"),
		Auth:   true,
	}, &out)
	if err == nil {
		t.Fatal("expected refresh failure")
	}
	if !strings.Contains(err.Error(), "stored login has expired; run `horo auth login` again") {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := app.BearerToken(); got != "" {
		t.Fatalf("bearer token should be cleared, got %q", got)
	}

	saved, err := LoadOAuthCredentials(srv.URL)
	if err != nil {
		t.Fatalf("load oauth credentials: %v", err)
	}
	if saved != nil {
		t.Fatalf("expected oauth credentials to be cleared, got %#v", saved)
	}
}

func writeJSONResponse(t *testing.T, w http.ResponseWriter, status int, body any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Fatalf("encode json response: %v", err)
	}
}

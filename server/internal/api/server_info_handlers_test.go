package api_test

import (
	"net/http"
	"testing"
)

func TestServerInfoIsPublicAndVersioned(t *testing.T) {
	env := setupTestServer(t)
	resp := doRequestAs(t, env, "", http.MethodGet, "/server-info", "")
	assertStatus(t, resp, http.StatusOK)

	var got struct {
		APIVersion   int      `json:"apiVersion"`
		Capabilities []string `json:"capabilities"`
	}
	readJSON(t, resp, &got)
	if got.APIVersion != 1 {
		t.Fatalf("apiVersion = %d, want 1", got.APIVersion)
	}
	want := []string{"oauth-2.1-pkce", "widget-snapshots-v1"}
	if len(got.Capabilities) != len(want) {
		t.Fatalf("capabilities = %v, want %v", got.Capabilities, want)
	}
	for i := range want {
		if got.Capabilities[i] != want[i] {
			t.Fatalf("capabilities = %v, want %v", got.Capabilities, want)
		}
	}
}

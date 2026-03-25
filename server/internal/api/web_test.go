package api_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestWebErrorCodeIsSnakeCase(t *testing.T) {
	env := setupTestServer(t)

	// Hit the web login endpoint with bad credentials to trigger writeJSONError.
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, env.Server.URL+"/auth/login",
		strings.NewReader(`{"email":"test@example.com","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	assertStatus(t, resp, http.StatusBadRequest)

	var body map[string]any
	readJSON(t, resp, &body)

	code, ok := body["code"].(string)
	if !ok {
		t.Fatal("response missing \"code\" field")
	}
	if code != "bad_request" {
		t.Errorf("code = %q, want %q (must be snake_case, not Title Case)", code, "bad_request")
	}
}

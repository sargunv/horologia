package api_test

import (
	"net/http"
	"testing"
)

func TestUsersMe(t *testing.T) {
	env := setupTestServer(t)

	resp := doRequest(t, env, "GET", "/users/me", "")
	assertStatus(t, resp, http.StatusOK)

	var user map[string]any
	readJSON(t, resp, &user)
	if user["email"] != "test@example.com" {
		t.Errorf("email = %v, want test@example.com", user["email"])
	}
	if user["isOwner"] != true {
		t.Errorf("isOwner = %v, want true", user["isOwner"])
	}
}

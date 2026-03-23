package api_test

import (
	"net/http"
	"testing"
)

func TestSpacesCreate(t *testing.T) {
	env := setupTestServer(t)

	resp := doRequest(t, env, "POST", "/spaces", `{"slug":"home","name":"Home"}`)
	assertStatus(t, resp, http.StatusCreated)

	var space map[string]any
	readJSON(t, resp, &space)
	if space["slug"] != "home" {
		t.Errorf("slug = %v, want home", space["slug"])
	}
	if space["name"] != "Home" {
		t.Errorf("name = %v, want Home", space["name"])
	}
	if space["description"] != "" {
		t.Errorf("description = %v, want empty", space["description"])
	}
	if space["createdAt"] == nil || space["createdAt"] == "" {
		t.Error("createdAt should be set")
	}
	if space["updatedAt"] == nil || space["updatedAt"] == "" {
		t.Error("updatedAt should be set")
	}
}

func TestSpacesCreateDuplicate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "POST", "/spaces", `{"slug":"home","name":"Home 2"}`)
	assertStatusClose(t, resp, http.StatusConflict)
}

func TestSpacesRead(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "GET", "/spaces/home", "")
	assertStatus(t, resp, http.StatusOK)

	var space map[string]any
	readJSON(t, resp, &space)
	if space["slug"] != "home" {
		t.Errorf("slug = %v, want home", space["slug"])
	}
}

func TestSpacesReadNotFound(t *testing.T) {
	env := setupTestServer(t)

	resp := doRequest(t, env, "GET", "/spaces/nonexistent", "")
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestSpacesListEmpty(t *testing.T) {
	env := setupTestServer(t)

	resp := doRequest(t, env, "GET", "/spaces", "")
	assertStatus(t, resp, http.StatusOK)

	var page map[string]any
	readJSON(t, resp, &page)
	items := page["items"].([]any)
	if len(items) != 0 {
		t.Fatalf("got %d items, want 0", len(items))
	}
	if page["nextCursor"] != nil {
		t.Error("expected null nextCursor")
	}
}

func TestSpacesList(t *testing.T) {
	env := setupTestServer(t)

	// Insert in non-alphabetical order to prove sort is applied.
	createSpace(t, env, "gamma", "Gamma")
	createSpace(t, env, "alpha", "Alpha")
	createSpace(t, env, "beta", "Beta")

	resp := doRequest(t, env, "GET", "/spaces", "")
	assertStatus(t, resp, http.StatusOK)

	var page map[string]any
	readJSON(t, resp, &page)
	items := page["items"].([]any)
	slugs := make([]string, len(items))
	for i, item := range items {
		slugs[i] = item.(map[string]any)["slug"].(string)
	}
	if slugs[0] != "alpha" || slugs[1] != "beta" || slugs[2] != "gamma" {
		t.Errorf("slugs = %v, want [alpha beta gamma]", slugs)
	}
}

func TestSpacesListPagination(t *testing.T) {
	env := setupTestServer(t)

	// Create 4 items to test exact page boundary (4 items / limit 2 = 2 full pages).
	createSpace(t, env, "a", "A")
	createSpace(t, env, "b", "B")
	createSpace(t, env, "c", "C")
	createSpace(t, env, "d", "D")

	// Page 1: should return "a" and "b" with a cursor.
	resp := doRequest(t, env, "GET", "/spaces?limit=2", "")
	assertStatus(t, resp, http.StatusOK)
	var page1 map[string]any
	readJSON(t, resp, &page1)
	items1 := page1["items"].([]any)
	if len(items1) != 2 {
		t.Fatalf("page 1: got %d items, want 2", len(items1))
	}
	if items1[0].(map[string]any)["slug"] != "a" || items1[1].(map[string]any)["slug"] != "b" {
		t.Errorf("page 1: got slugs %v/%v, want a/b",
			items1[0].(map[string]any)["slug"], items1[1].(map[string]any)["slug"])
	}
	cursor := page1["nextCursor"]
	if cursor == nil {
		t.Fatal("page 1: expected nextCursor")
	}

	// Page 2: should return "c" and "d" with null cursor (exact boundary).
	resp2 := doRequest(t, env, "GET", "/spaces?limit=2&cursor="+cursor.(string), "")
	assertStatus(t, resp2, http.StatusOK)
	var page2 map[string]any
	readJSON(t, resp2, &page2)
	items2 := page2["items"].([]any)
	if len(items2) != 2 {
		t.Fatalf("page 2: got %d items, want 2", len(items2))
	}
	if items2[0].(map[string]any)["slug"] != "c" || items2[1].(map[string]any)["slug"] != "d" {
		t.Errorf("page 2: got slugs %v/%v, want c/d",
			items2[0].(map[string]any)["slug"], items2[1].(map[string]any)["slug"])
	}
	if page2["nextCursor"] != nil {
		t.Error("page 2: expected null nextCursor")
	}
}

func TestSpacesUpdate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	// First set a description so we can verify partial update preserves it.
	resp := doRequest(t, env, "PATCH", "/spaces/home", `{"description":"My house"}`)
	assertStatusClose(t, resp, http.StatusOK)

	// Now update only the name — description should be preserved.
	resp2 := doRequest(t, env, "PATCH", "/spaces/home", `{"name":"House"}`)
	assertStatus(t, resp2, http.StatusOK)
	var space map[string]any
	readJSON(t, resp2, &space)
	if space["name"] != "House" {
		t.Errorf("name = %v, want House", space["name"])
	}
	if space["description"] != "My house" {
		t.Errorf("description = %v, want My house", space["description"])
	}
}

func TestSpacesDelete(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "DELETE", "/spaces/home", "")
	assertStatusClose(t, resp, http.StatusNoContent)

	// Verify it's gone.
	resp2 := doRequest(t, env, "GET", "/spaces/home", "")
	assertStatusClose(t, resp2, http.StatusNotFound)
}

func TestNonMemberCannotAccessSpace(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "secret", "Secret")
	userToken := createTestUser(t, env, "bob@example.com", "Bob", "pass123")

	// Non-member cannot read the space.
	assertStatusClose(t, doRequestAs(t, env, userToken, "GET", "/spaces/secret", ""), http.StatusNotFound)
	// Non-member cannot update the space.
	assertStatusClose(t, doRequestAs(t, env, userToken, "PATCH", "/spaces/secret", `{"name":"X"}`), http.StatusNotFound)
	// Non-member cannot delete the space.
	assertStatusClose(t, doRequestAs(t, env, userToken, "DELETE", "/spaces/secret", ""), http.StatusNotFound)
	// Non-member cannot list tasks.
	assertStatusClose(t, doRequestAs(t, env, userToken, "GET", "/spaces/secret/tasks", ""), http.StatusNotFound)
	// Non-member cannot create tasks.
	assertStatusClose(t, doRequestAs(t, env, userToken, "POST", "/spaces/secret/tasks", `{"title":"Task"}`), http.StatusNotFound)
}

func TestViewerCannotWriteToSpace(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	userToken, _ := createAndAddMember(t, env, "home", "viewer@example.com", "Viewer", "pass123", "viewer")

	// Viewer can read the space.
	assertStatusClose(t, doRequestAs(t, env, userToken, "GET", "/spaces/home", ""), http.StatusOK)
	// Viewer can list tasks.
	assertStatusClose(t, doRequestAs(t, env, userToken, "GET", "/spaces/home/tasks", ""), http.StatusOK)
	// Viewer cannot create tasks.
	assertStatusClose(t, doRequestAs(t, env, userToken, "POST", "/spaces/home/tasks", `{"title":"X"}`), http.StatusBadRequest)
	// Viewer cannot update space.
	assertStatusClose(t, doRequestAs(t, env, userToken, "PATCH", "/spaces/home", `{"name":"X"}`), http.StatusBadRequest)
	// Viewer cannot delete space.
	assertStatusClose(t, doRequestAs(t, env, userToken, "DELETE", "/spaces/home", ""), http.StatusBadRequest)
}

func TestMemberCannotManageSpace(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	userToken, userID := createAndAddMember(t, env, "home", "member@example.com", "Member", "pass123", "member")

	// Member can create tasks.
	resp2 := doRequestAs(t, env, userToken, "POST", "/spaces/home/tasks", `{"title":"My task"}`)
	assertStatusClose(t, resp2, http.StatusCreated)
	// Member cannot update space settings.
	assertStatusClose(t, doRequestAs(t, env, userToken, "PATCH", "/spaces/home", `{"name":"X"}`), http.StatusBadRequest)
	// Member cannot delete space.
	assertStatusClose(t, doRequestAs(t, env, userToken, "DELETE", "/spaces/home", ""), http.StatusBadRequest)
	// Member cannot manage members.
	assertStatusClose(t, doRequestAs(t, env, userToken, "POST", "/spaces/home/members", `{"userId":"`+userID+`","role":"viewer"}`), http.StatusBadRequest)
}

func TestNonOwnerSpacesListFiltered(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "alpha", "Alpha")
	createSpace(t, env, "beta", "Beta")
	createSpace(t, env, "gamma", "Gamma")

	userToken, _ := createAndAddMember(t, env, "beta", "user@example.com", "User", "pass123", "member")

	// Non-owner should only see "beta".
	resp2 := doRequestAs(t, env, userToken, "GET", "/spaces", "")
	assertStatus(t, resp2, http.StatusOK)
	var page map[string]any
	readJSON(t, resp2, &page)
	items := page["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].(map[string]any)["slug"] != "beta" {
		t.Errorf("slug = %v, want beta", items[0].(map[string]any)["slug"])
	}
}

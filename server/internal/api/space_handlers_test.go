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

	var list map[string]any
	readJSON(t, resp, &list)
	items := jsonAs[[]any](t, list["items"])
	if len(items) != 0 {
		t.Fatalf("got %d items, want 0", len(items))
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

	var list map[string]any
	readJSON(t, resp, &list)
	items := jsonAs[[]any](t, list["items"])
	slugs := make([]string, len(items))
	for i, item := range items {
		slugs[i] = jsonAs[string](t, jsonAs[map[string]any](t, item)["slug"])
	}
	if slugs[0] != "alpha" || slugs[1] != "beta" || slugs[2] != "gamma" {
		t.Errorf("slugs = %v, want [alpha beta gamma]", slugs)
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

func TestSpacesUpdateSlug(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "old-slug", "My Space")

	// Rename the slug.
	resp := doRequest(t, env, "PATCH", "/spaces/old-slug", `{"slug":"new-slug"}`)
	assertStatus(t, resp, http.StatusOK)
	var space map[string]any
	readJSON(t, resp, &space)
	if space["slug"] != "new-slug" {
		t.Errorf("slug = %v, want new-slug", space["slug"])
	}
	if space["name"] != "My Space" {
		t.Errorf("name = %v, want My Space", space["name"])
	}

	// Old slug should 404.
	resp2 := doRequest(t, env, "GET", "/spaces/old-slug", "")
	assertStatus(t, resp2, http.StatusNotFound)

	// New slug should work.
	resp3 := doRequest(t, env, "GET", "/spaces/new-slug", "")
	assertStatus(t, resp3, http.StatusOK)
}

func TestSpacesUpdateSlugCascadesToTasks(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "old-slug", "My Space")

	// Create a task in the space.
	resp := doRequest(t, env, "POST", "/spaces/old-slug/tasks", `{"title":"Test task"}`)
	assertStatusClose(t, resp, http.StatusCreated)

	// Rename the slug.
	resp2 := doRequest(t, env, "PATCH", "/spaces/old-slug", `{"slug":"new-slug"}`)
	assertStatus(t, resp2, http.StatusOK)

	// Tasks should be accessible under the new slug.
	resp3 := doRequest(t, env, "GET", "/spaces/new-slug/tasks", "")
	assertStatus(t, resp3, http.StatusOK)
	var taskList map[string]any
	readJSON(t, resp3, &taskList)
	items, ok := taskList["items"].([]any)
	if !ok {
		t.Fatal("items is not an array")
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 task, got %d", len(items))
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
	assertStatusClose(t, doRequestAs(t, env, userToken, "POST", "/spaces/home/tasks", `{"title":"X"}`), http.StatusForbidden)
	// Viewer cannot update space.
	assertStatusClose(t, doRequestAs(t, env, userToken, "PATCH", "/spaces/home", `{"name":"X"}`), http.StatusForbidden)
	// Viewer cannot delete space.
	assertStatusClose(t, doRequestAs(t, env, userToken, "DELETE", "/spaces/home", ""), http.StatusForbidden)
}

func TestMemberCannotManageSpace(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	userToken, userID := createAndAddMember(t, env, "home", "member@example.com", "Member", "pass123", "member")

	// Member can create tasks.
	resp2 := doRequestAs(t, env, userToken, "POST", "/spaces/home/tasks", `{"title":"My task"}`)
	assertStatusClose(t, resp2, http.StatusCreated)
	// Member cannot update space settings.
	assertStatusClose(t, doRequestAs(t, env, userToken, "PATCH", "/spaces/home", `{"name":"X"}`), http.StatusForbidden)
	// Member cannot delete space.
	assertStatusClose(t, doRequestAs(t, env, userToken, "DELETE", "/spaces/home", ""), http.StatusForbidden)
	// Member cannot manage members.
	assertStatusClose(t, doRequestAs(t, env, userToken, "POST", "/spaces/home/members", `{"userId":"`+userID+`","role":"viewer"}`), http.StatusForbidden)
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
	items := jsonAs[[]any](t, page["items"])
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if jsonAs[map[string]any](t, items[0])["slug"] != "beta" {
		t.Errorf("slug = %v, want beta", jsonAs[map[string]any](t, items[0])["slug"])
	}
}

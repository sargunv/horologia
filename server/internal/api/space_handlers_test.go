package api_test

import (
	"fmt"
	"net/http"
	"testing"
)

func TestSpaceHandlers(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	t.Run("create", func(t *testing.T) {
		resp := doRequest(t, env, "POST", "/spaces", `{"slug":"create-home","name":"Home"}`)
		assertStatus(t, resp, http.StatusCreated)

		var space map[string]any
		readJSON(t, resp, &space)
		if space["slug"] != "create-home" {
			t.Errorf("slug = %v, want create-home", space["slug"])
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
	})

	t.Run("create slug valid", func(t *testing.T) {
		slugs := []string{
			"home",
			"my-project",
			"abc123",
			"プロジェクト",
			"my-プロジェクト",
			"café",
		}
		for i, slug := range slugs {
			resp := doRequest(t, env, "POST", "/spaces", `{"slug":"valid-`+strconv.Itoa(i)+`-`+slug+`","name":"Test"}`)
			assertStatusClose(t, resp, http.StatusCreated)
		}
	})

	t.Run("create slug invalid", func(t *testing.T) {
		slugs := []string{
			"",
			"-leading",
			"trailing-",
			"double--hyphen",
			"has space",
			"has.dot",
			"-",
		}
		for _, slug := range slugs {
			resp := doRequest(t, env, "POST", "/spaces", `{"slug":"`+slug+`","name":"Test"}`)
			assertStatusClose(t, resp, http.StatusBadRequest)
		}
	})

	t.Run("create duplicate", func(t *testing.T) {
		createSpace(t, env, "dup-home", "Home")

		resp := doRequest(t, env, "POST", "/spaces", `{"slug":"dup-home","name":"Home 2"}`)
		assertStatusClose(t, resp, http.StatusConflict)
	})

	t.Run("read", func(t *testing.T) {
		createSpace(t, env, "read-home", "Home")

		resp := doRequest(t, env, "GET", "/spaces/read-home", "")
		assertStatus(t, resp, http.StatusOK)

		var space map[string]any
		readJSON(t, resp, &space)
		if space["slug"] != "read-home" {
			t.Errorf("slug = %v, want read-home", space["slug"])
		}
	})

	t.Run("read not found", func(t *testing.T) {
		resp := doRequest(t, env, "GET", "/spaces/nonexistent", "")
		assertStatusClose(t, resp, http.StatusNotFound)
	})

	t.Run("list empty for non-owner", func(t *testing.T) {
		userToken := createTestUser(t, env, "spaces-empty@example.com", "Spaces Empty", "pass1234")

		resp := doRequestAs(t, env, userToken, "GET", "/spaces", "")
		assertStatus(t, resp, http.StatusOK)

		var list map[string]any
		readJSON(t, resp, &list)
		items := jsonAs[[]any](t, list["items"])
		if len(items) != 0 {
			t.Fatalf("got %d items, want 0", len(items))
		}
	})

	t.Run("list", func(t *testing.T) {
		createSpace(t, env, "list-gamma", "Gamma")
		createSpace(t, env, "list-alpha", "Alpha")
		createSpace(t, env, "list-beta", "Beta")

		resp := doRequest(t, env, "GET", "/spaces", "")
		assertStatus(t, resp, http.StatusOK)

		var list map[string]any
		readJSON(t, resp, &list)
		items := jsonAs[[]any](t, list["items"])
		slugs := make([]string, len(items))
		for i, item := range items {
			slugs[i] = jsonAs[string](t, jsonAs[map[string]any](t, item)["slug"])
		}
		if slugs[0] != "create-home" || slugs[1] != "dup-home" || slugs[len(slugs)-3] != "list-alpha" {
			if !(containsInOrder(slugs, []string{"list-alpha", "list-beta", "list-gamma"})) {
				t.Errorf("slugs = %v, want list-alpha/list-beta/list-gamma in order", slugs)
			}
		}
	})

	t.Run("update", func(t *testing.T) {
		createSpace(t, env, "update-home", "Home")

		resp := doRequest(t, env, "PATCH", "/spaces/update-home", `{"description":"My house"}`)
		assertStatusClose(t, resp, http.StatusOK)

		resp2 := doRequest(t, env, "PATCH", "/spaces/update-home", `{"name":"House"}`)
		assertStatus(t, resp2, http.StatusOK)
		var space map[string]any
		readJSON(t, resp2, &space)
		if space["name"] != "House" {
			t.Errorf("name = %v, want House", space["name"])
		}
		if space["description"] != "My house" {
			t.Errorf("description = %v, want My house", space["description"])
		}
	})

	t.Run("update slug", func(t *testing.T) {
		createSpace(t, env, "rename-old", "My Space")

		resp := doRequest(t, env, "PATCH", "/spaces/rename-old", `{"slug":"rename-new"}`)
		assertStatus(t, resp, http.StatusOK)
		var space map[string]any
		readJSON(t, resp, &space)
		if space["slug"] != "rename-new" {
			t.Errorf("slug = %v, want rename-new", space["slug"])
		}
		if space["name"] != "My Space" {
			t.Errorf("name = %v, want My Space", space["name"])
		}

		resp2 := doRequest(t, env, "GET", "/spaces/rename-old", "")
		assertStatusClose(t, resp2, http.StatusNotFound)

		resp3 := doRequest(t, env, "GET", "/spaces/rename-new", "")
		assertStatusClose(t, resp3, http.StatusOK)
	})

	t.Run("update slug cascades to tasks", func(t *testing.T) {
		createSpace(t, env, "cascade-old", "My Space")

		resp := doRequest(t, env, "POST", "/spaces/cascade-old/tasks", `{"title":"Test task"}`)
		assertStatusClose(t, resp, http.StatusCreated)

		resp2 := doRequest(t, env, "PATCH", "/spaces/cascade-old", `{"slug":"cascade-new"}`)
		assertStatusClose(t, resp2, http.StatusOK)

		resp3 := doRequest(t, env, "GET", "/spaces/cascade-new/tasks", "")
		assertStatus(t, resp3, http.StatusOK)
		var taskList map[string]any
		readJSON(t, resp3, &taskList)
		items := jsonAs[[]any](t, taskList["items"])
		if len(items) != 1 {
			t.Fatalf("expected 1 task, got %d", len(items))
		}
	})

	t.Run("delete", func(t *testing.T) {
		createSpace(t, env, "delete-home", "Home")

		resp := doRequest(t, env, "DELETE", "/spaces/delete-home", "")
		assertStatusClose(t, resp, http.StatusNoContent)

		resp2 := doRequest(t, env, "GET", "/spaces/delete-home", "")
		assertStatusClose(t, resp2, http.StatusNotFound)
	})
}

func TestSpacePermissions(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	t.Run("non member cannot access space", func(t *testing.T) {
		createSpace(t, env, "secret-space", "Secret")
		userToken := createTestUser(t, env, "bob@example.com", "Bob", "pass1234")

		assertStatusClose(t, doRequestAs(t, env, userToken, "GET", "/spaces/secret-space", ""), http.StatusNotFound)
		assertStatusClose(t, doRequestAs(t, env, userToken, "PATCH", "/spaces/secret-space", `{"name":"X"}`), http.StatusNotFound)
		assertStatusClose(t, doRequestAs(t, env, userToken, "DELETE", "/spaces/secret-space", ""), http.StatusNotFound)
		assertStatusClose(t, doRequestAs(t, env, userToken, "GET", "/spaces/secret-space/tasks", ""), http.StatusNotFound)
		assertStatusClose(t, doRequestAs(t, env, userToken, "POST", "/spaces/secret-space/tasks", `{"title":"Task"}`), http.StatusNotFound)
	})

	t.Run("viewer cannot write to space", func(t *testing.T) {
		createSpace(t, env, "viewer-home", "Home")
		userToken, _ := createAndAddMember(t, env, "viewer-home", "viewer@example.com", "Viewer", "pass1234", "viewer")

		assertStatusClose(t, doRequestAs(t, env, userToken, "GET", "/spaces/viewer-home", ""), http.StatusOK)
		assertStatusClose(t, doRequestAs(t, env, userToken, "GET", "/spaces/viewer-home/tasks", ""), http.StatusOK)
		assertStatusClose(t, doRequestAs(t, env, userToken, "POST", "/spaces/viewer-home/tasks", `{"title":"X"}`), http.StatusForbidden)
		assertStatusClose(t, doRequestAs(t, env, userToken, "PATCH", "/spaces/viewer-home", `{"name":"X"}`), http.StatusForbidden)
		assertStatusClose(t, doRequestAs(t, env, userToken, "DELETE", "/spaces/viewer-home", ""), http.StatusForbidden)
	})

	t.Run("member cannot manage space", func(t *testing.T) {
		createSpace(t, env, "member-home", "Home")
		userToken, userID := createAndAddMember(t, env, "member-home", "member@example.com", "Member", "pass1234", "member")

		resp := doRequestAs(t, env, userToken, "POST", "/spaces/member-home/tasks", `{"title":"My task"}`)
		assertStatusClose(t, resp, http.StatusCreated)
		assertStatusClose(t, doRequestAs(t, env, userToken, "PATCH", "/spaces/member-home", `{"name":"X"}`), http.StatusForbidden)
		assertStatusClose(t, doRequestAs(t, env, userToken, "DELETE", "/spaces/member-home", ""), http.StatusForbidden)
		assertStatusClose(t, doRequestAs(t, env, userToken, "POST", "/spaces/member-home/members", `{"userId":"`+userID+`","role":"viewer"}`), http.StatusForbidden)
	})

	t.Run("non owner spaces list filtered", func(t *testing.T) {
		createSpace(t, env, "filter-alpha", "Alpha")
		createSpace(t, env, "filter-beta", "Beta")
		createSpace(t, env, "filter-gamma", "Gamma")

		userToken, _ := createAndAddMember(t, env, "filter-beta", "user@example.com", "User", "pass1234", "member")

		resp := doRequestAs(t, env, userToken, "GET", "/spaces", "")
		assertStatus(t, resp, http.StatusOK)
		var page map[string]any
		readJSON(t, resp, &page)
		items := jsonAs[[]any](t, page["items"])
		if len(items) != 1 {
			t.Fatalf("got %d items, want 1", len(items))
		}
		if jsonAs[map[string]any](t, items[0])["slug"] != "filter-beta" {
			t.Errorf("slug = %v, want filter-beta", jsonAs[map[string]any](t, items[0])["slug"])
		}
	})
}

func containsInOrder(have, want []string) bool {
	if len(want) == 0 {
		return true
	}
	index := 0
	for _, item := range have {
		if item == want[index] {
			index++
			if index == len(want) {
				return true
			}
		}
	}
	return false
}

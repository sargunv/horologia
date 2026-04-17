package api_test

import (
	"net/http"
	"testing"
)

func TestSpaceTagHandlers(t *testing.T) {
	t.Parallel()

	t.Run("create", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "tag-create", "Tag Create")

		resp := doRequest(t, env, "POST", "/spaces/tag-create/tags", `{"name":"Bug"}`)
		assertStatus(t, resp, http.StatusCreated)
		var tag map[string]any
		readJSON(t, resp, &tag)
		if tag["name"] != "Bug" {
			t.Fatalf("got name %v, want Bug", tag["name"])
		}
		if tag["createdAt"] == nil {
			t.Fatal("expected createdAt to be set")
		}
	})

	t.Run("create duplicate", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "tag-dup", "Tag Dup")
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-dup/tags", `{"name":"Bug"}`), http.StatusCreated)
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-dup/tags", `{"name":"Bug"}`), http.StatusConflict)
	})

	t.Run("create case fold duplicate", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "tag-fold", "Tag Fold")
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-fold/tags", `{"name":"Bug"}`), http.StatusCreated)
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-fold/tags", `{"name":"bug"}`), http.StatusConflict)
	})

	t.Run("create empty", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "tag-empty", "Tag Empty")
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-empty/tags", `{"name":""}`), http.StatusBadRequest)
	})

	t.Run("list", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "tag-list", "Tag List")
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-list/tags", `{"name":"Alpha"}`), http.StatusCreated)
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-list/tags", `{"name":"Beta"}`), http.StatusCreated)

		resp := doRequest(t, env, "GET", "/spaces/tag-list/tags", "")
		assertStatus(t, resp, http.StatusOK)
		var page map[string]any
		readJSON(t, resp, &page)
		items := jsonAs[[]any](t, page["items"])
		if len(items) != 2 {
			t.Fatalf("got %d tags, want 2", len(items))
		}
	})

	t.Run("list empty", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "tag-list-empty", "Tag List Empty")
		resp := doRequest(t, env, "GET", "/spaces/tag-list-empty/tags", "")
		assertStatus(t, resp, http.StatusOK)
		var page map[string]any
		readJSON(t, resp, &page)
		items := jsonAs[[]any](t, page["items"])
		if len(items) != 0 {
			t.Fatalf("got %d tags, want 0", len(items))
		}
	})

	t.Run("update", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "tag-update", "Tag Update")
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-update/tags", `{"name":"Bug"}`), http.StatusCreated)

		resp := doRequest(t, env, "PATCH", "/spaces/tag-update/tags/Bug", `{"name":"Defect"}`)
		assertStatus(t, resp, http.StatusOK)
		var tag map[string]any
		readJSON(t, resp, &tag)
		if tag["name"] != "Defect" {
			t.Fatalf("got name %v, want Defect", tag["name"])
		}
	})

	t.Run("update case fold collision", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "tag-collision", "Tag Collision")
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-collision/tags", `{"name":"Bug"}`), http.StatusCreated)
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-collision/tags", `{"name":"Feature"}`), http.StatusCreated)
		assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/tag-collision/tags/Feature", `{"name":"bug"}`), http.StatusConflict)
	})

	t.Run("update not found", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "tag-update-missing", "Tag Update Missing")
		assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/tag-update-missing/tags/nonexistent", `{"name":"X"}`), http.StatusNotFound)
	})

	t.Run("delete", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "tag-delete", "Tag Delete")
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-delete/tags", `{"name":"Bug"}`), http.StatusCreated)
		assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/tag-delete/tags/Bug", ""), http.StatusNoContent)

		resp := doRequest(t, env, "GET", "/spaces/tag-delete/tags", "")
		assertStatus(t, resp, http.StatusOK)
		var page map[string]any
		readJSON(t, resp, &page)
		if len(jsonAs[[]any](t, page["items"])) != 0 {
			t.Fatal("expected no tags after delete")
		}
	})

	t.Run("delete not found", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "tag-delete-missing", "Tag Delete Missing")
		assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/tag-delete-missing/tags/nonexistent", ""), http.StatusNotFound)
	})

	t.Run("cross space isolation", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "tag-a", "Tag A")
		createSpace(t, env, "tag-b", "Tag B")
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-a/tags", `{"name":"Bug"}`), http.StatusCreated)

		resp := doRequest(t, env, "GET", "/spaces/tag-b/tags", "")
		assertStatus(t, resp, http.StatusOK)
		var page map[string]any
		readJSON(t, resp, &page)
		if len(jsonAs[[]any](t, page["items"])) != 0 {
			t.Fatal("space B should have no tags")
		}
	})

	t.Run("non member rejected", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "tag-acl", "Tag ACL")
		outsiderToken := createTestUser(t, env, testEmail(t, "tag-outsider"), "Outsider", "pass1234")
		assertStatusClose(t, doRequestAs(t, env, outsiderToken, "GET", "/spaces/tag-acl/tags", ""), http.StatusNotFound)
		assertStatusClose(t, doRequestAs(t, env, outsiderToken, "POST", "/spaces/tag-acl/tags", `{"name":"Bug"}`), http.StatusNotFound)
	})

	t.Run("viewer cannot write", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "tag-viewer", "Tag Viewer Write")
		viewerToken, _ := createAndAddMember(t, env, "tag-viewer", testEmail(t, "tag-viewer"), "Viewer", "pass1234", "viewer")
		assertStatusClose(t, doRequestAs(t, env, viewerToken, "GET", "/spaces/tag-viewer/tags", ""), http.StatusOK)
		assertStatusClose(t, doRequestAs(t, env, viewerToken, "POST", "/spaces/tag-viewer/tags", `{"name":"Bug"}`), http.StatusForbidden)
	})
}

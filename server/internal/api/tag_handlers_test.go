package api_test

import (
	"net/http"
	"testing"
)

func TestSpaceTagsCreate(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-test", "Tag Test")

	resp := doRequest(t, env, "POST", "/spaces/tag-test/tags", `{"name":"Bug"}`)
	assertStatus(t, resp, http.StatusCreated)
	var tag map[string]any
	readJSON(t, resp, &tag)
	if tag["name"] != "Bug" {
		t.Fatalf("got name %v, want Bug", tag["name"])
	}
	if tag["createdAt"] == nil {
		t.Fatal("expected createdAt to be set")
	}
}

func TestSpaceTagsCreateDuplicate(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-dup", "Tag Dup")

	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-dup/tags", `{"name":"Bug"}`), http.StatusCreated)

	// Same name (exact case) should 409.
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-dup/tags", `{"name":"Bug"}`), http.StatusConflict)
}

func TestSpaceTagsCreateCaseFoldDuplicate(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-fold", "Tag Fold")

	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-fold/tags", `{"name":"Bug"}`), http.StatusCreated)

	// Different case but same folded name should also 409.
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-fold/tags", `{"name":"bug"}`), http.StatusConflict)
}

func TestSpaceTagsCreateEmpty(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-empty", "Tag Empty")

	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-empty/tags", `{"name":""}`), http.StatusBadRequest)
}

func TestSpaceTagsList(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-list", "Tag List")

	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-list/tags", `{"name":"Alpha"}`), http.StatusCreated)
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-list/tags", `{"name":"Beta"}`), http.StatusCreated)

	resp := doRequest(t, env, "GET", "/spaces/tag-list/tags", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := page["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("got %d tags, want 2", len(items))
	}
}

func TestSpaceTagsListEmpty(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-le", "Tag List Empty")

	resp := doRequest(t, env, "GET", "/spaces/tag-le/tags", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := page["items"].([]any)
	if len(items) != 0 {
		t.Fatalf("got %d tags, want 0", len(items))
	}
}

func TestSpaceTagsUpdate(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-upd", "Tag Update")

	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-upd/tags", `{"name":"Bug"}`), http.StatusCreated)

	resp := doRequest(t, env, "PATCH", "/spaces/tag-upd/tags/Bug", `{"name":"Defect"}`)
	assertStatus(t, resp, http.StatusOK)
	var tag map[string]any
	readJSON(t, resp, &tag)
	if tag["name"] != "Defect" {
		t.Fatalf("got name %v, want Defect", tag["name"])
	}
}

func TestSpaceTagsUpdateCaseFoldCollision(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-coll", "Tag Collision")

	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-coll/tags", `{"name":"Bug"}`), http.StatusCreated)
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-coll/tags", `{"name":"Feature"}`), http.StatusCreated)

	// Renaming "Feature" to "bug" should conflict with "Bug".
	assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/tag-coll/tags/Feature", `{"name":"bug"}`), http.StatusConflict)
}

func TestSpaceTagsUpdateNotFound(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-unf", "Tag Update NF")

	assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/tag-unf/tags/nonexistent", `{"name":"X"}`), http.StatusNotFound)
}

func TestSpaceTagsDelete(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-del", "Tag Delete")

	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-del/tags", `{"name":"Bug"}`), http.StatusCreated)

	assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/tag-del/tags/Bug", ""), http.StatusNoContent)

	// List should be empty now.
	resp := doRequest(t, env, "GET", "/spaces/tag-del/tags", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	if len(page["items"].([]any)) != 0 {
		t.Fatal("expected no tags after delete")
	}
}

func TestSpaceTagsDeleteNotFound(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-dnf", "Tag Delete NF")

	assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/tag-dnf/tags/nonexistent", ""), http.StatusNotFound)
}

func TestSpaceTagsCrossSpaceIsolation(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-a", "Tag A")
	createSpace(t, env, "tag-b", "Tag B")

	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-a/tags", `{"name":"Bug"}`), http.StatusCreated)

	// Tag in space A should not appear in space B.
	resp := doRequest(t, env, "GET", "/spaces/tag-b/tags", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	if len(page["items"].([]any)) != 0 {
		t.Fatal("space B should have no tags")
	}
}

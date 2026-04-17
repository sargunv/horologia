package api_test

import (
	"net/http"
	"testing"
)

func TestTaskRelationHandlers(t *testing.T) {
	t.Parallel()

	t.Run("create", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "rel-create", "Relation Test")
		t1 := createTask(t, env, "rel-create", `{"title":"Task 1"}`)
		t2 := createTask(t, env, "rel-create", `{"title":"Task 2"}`)
		t1id, t2id := jsonAs[string](t, t1["id"]), jsonAs[string](t, t2["id"])

		resp := doRequest(t, env, "POST", "/spaces/rel-create/tasks/"+t1id+"/relations", `{"kind":"blocks","relatedTaskId":"`+t2id+`"}`)
		assertStatus(t, resp, http.StatusCreated)
		var rel map[string]any
		readJSON(t, resp, &rel)
		if rel["kind"] != "blocks" {
			t.Fatalf("got kind %v, want blocks", rel["kind"])
		}
		if rel["relatedTaskId"] != t2id {
			t.Fatalf("got relatedTaskId %v, want %v", rel["relatedTaskId"], t2id)
		}

		rels1 := assertTaskRelations(t, env, "rel-create", t1id, 1)
		assertRelationKind(t, rels1[0], "blocks", t2id)
		rels2 := assertTaskRelations(t, env, "rel-create", t2id, 1)
		assertRelationKind(t, rels2[0], "blocked_by", t1id)
	})

	t.Run("symmetric", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "rel-symmetric", "Symmetric Test")
		t1 := createTask(t, env, "rel-symmetric", `{"title":"Task 1"}`)
		t2 := createTask(t, env, "rel-symmetric", `{"title":"Task 2"}`)
		t1id, t2id := jsonAs[string](t, t1["id"]), jsonAs[string](t, t2["id"])
		createRelation(t, env, "rel-symmetric", t1id, "relates_to", t2id)
		assertRelationKind(t, assertTaskRelations(t, env, "rel-symmetric", t1id, 1)[0], "relates_to", t2id)
		assertRelationKind(t, assertTaskRelations(t, env, "rel-symmetric", t2id, 1)[0], "relates_to", t1id)
	})

	t.Run("delete", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "rel-delete", "Delete Test")
		t1 := createTask(t, env, "rel-delete", `{"title":"Task 1"}`)
		t2 := createTask(t, env, "rel-delete", `{"title":"Task 2"}`)
		t1id, t2id := jsonAs[string](t, t1["id"]), jsonAs[string](t, t2["id"])
		createRelation(t, env, "rel-delete", t1id, "blocks", t2id)
		assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/rel-delete/tasks/"+t1id+"/relations/blocks/"+t2id, ""), http.StatusNoContent)
		assertTaskRelations(t, env, "rel-delete", t1id, 0)
		assertTaskRelations(t, env, "rel-delete", t2id, 0)
	})

	t.Run("delete from other side", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "rel-delete-other", "Delete Other Side Test")
		t1 := createTask(t, env, "rel-delete-other", `{"title":"Task 1"}`)
		t2 := createTask(t, env, "rel-delete-other", `{"title":"Task 2"}`)
		t1id, t2id := jsonAs[string](t, t1["id"]), jsonAs[string](t, t2["id"])
		createRelation(t, env, "rel-delete-other", t1id, "blocks", t2id)
		assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/rel-delete-other/tasks/"+t2id+"/relations/blocked_by/"+t1id, ""), http.StatusNoContent)
		assertTaskRelations(t, env, "rel-delete-other", t1id, 0)
		assertTaskRelations(t, env, "rel-delete-other", t2id, 0)
	})

	t.Run("delete non existent", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "rel-delete-missing", "Delete Non-Existent")
		t1 := createTask(t, env, "rel-delete-missing", `{"title":"Task 1"}`)
		t2 := createTask(t, env, "rel-delete-missing", `{"title":"Task 2"}`)
		assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/rel-delete-missing/tasks/"+jsonAs[string](t, t1["id"])+"/relations/blocks/"+jsonAs[string](t, t2["id"]), ""), http.StatusNotFound)
	})

	t.Run("self rejected", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "rel-self", "Self Test")
		t1 := createTask(t, env, "rel-self", `{"title":"Task 1"}`)
		t1id := jsonAs[string](t, t1["id"])
		resp := doRequest(t, env, "POST", "/spaces/rel-self/tasks/"+t1id+"/relations", `{"kind":"blocks","relatedTaskId":"`+t1id+`"}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("cross space rejected", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "rel-space-a", "Space A")
		createSpace(t, env, "rel-space-b", "Space B")
		t1 := createTask(t, env, "rel-space-a", `{"title":"Task 1"}`)
		t2 := createTask(t, env, "rel-space-b", `{"title":"Task 2"}`)
		resp := doRequest(t, env, "POST", "/spaces/rel-space-a/tasks/"+jsonAs[string](t, t1["id"])+"/relations", `{"kind":"blocks","relatedTaskId":"`+jsonAs[string](t, t2["id"])+`"}`)
		assertStatusClose(t, resp, http.StatusNotFound)
	})

	t.Run("non existent task", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "rel-missing-task", "Non-Existent Test")
		t1 := createTask(t, env, "rel-missing-task", `{"title":"Task 1"}`)
		resp := doRequest(t, env, "POST", "/spaces/rel-missing-task/tasks/"+jsonAs[string](t, t1["id"])+"/relations", `{"kind":"blocks","relatedTaskId":"T99999"}`)
		assertStatusClose(t, resp, http.StatusNotFound)
	})

	t.Run("duplicate rejected", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "rel-dup", "Dup Test")
		t1 := createTask(t, env, "rel-dup", `{"title":"Task 1"}`)
		t2 := createTask(t, env, "rel-dup", `{"title":"Task 2"}`)
		t1id, t2id := jsonAs[string](t, t1["id"]), jsonAs[string](t, t2["id"])
		createRelation(t, env, "rel-dup", t1id, "blocks", t2id)
		resp := doRequest(t, env, "POST", "/spaces/rel-dup/tasks/"+t1id+"/relations", `{"kind":"blocks","relatedTaskId":"`+t2id+`"}`)
		assertStatusClose(t, resp, http.StatusConflict)
	})

	t.Run("duplicate via inverse", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "rel-dup-inverse", "Dup Inverse Test")
		t1 := createTask(t, env, "rel-dup-inverse", `{"title":"Task 1"}`)
		t2 := createTask(t, env, "rel-dup-inverse", `{"title":"Task 2"}`)
		t1id, t2id := jsonAs[string](t, t1["id"]), jsonAs[string](t, t2["id"])
		createRelation(t, env, "rel-dup-inverse", t1id, "blocks", t2id)
		resp := doRequest(t, env, "POST", "/spaces/rel-dup-inverse/tasks/"+t2id+"/relations", `{"kind":"blocked_by","relatedTaskId":"`+t1id+`"}`)
		assertStatusClose(t, resp, http.StatusConflict)
	})

	t.Run("create via blocked by", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "rel-blocked-by", "Blocked By Test")
		t1 := createTask(t, env, "rel-blocked-by", `{"title":"Task 1"}`)
		t2 := createTask(t, env, "rel-blocked-by", `{"title":"Task 2"}`)
		t1id, t2id := jsonAs[string](t, t1["id"]), jsonAs[string](t, t2["id"])
		createRelation(t, env, "rel-blocked-by", t2id, "blocked_by", t1id)
		assertRelationKind(t, assertTaskRelations(t, env, "rel-blocked-by", t1id, 1)[0], "blocks", t2id)
		assertRelationKind(t, assertTaskRelations(t, env, "rel-blocked-by", t2id, 1)[0], "blocked_by", t1id)
	})

	t.Run("parent child", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "rel-parent-child", "Parent Child Test")
		parent := createTask(t, env, "rel-parent-child", `{"title":"Parent"}`)
		child := createTask(t, env, "rel-parent-child", `{"title":"Child"}`)
		parentID, childID := jsonAs[string](t, parent["id"]), jsonAs[string](t, child["id"])
		createRelation(t, env, "rel-parent-child", childID, "child_of", parentID)
		assertRelationKind(t, assertTaskRelations(t, env, "rel-parent-child", parentID, 1)[0], "parent_of", childID)
		assertRelationKind(t, assertTaskRelations(t, env, "rel-parent-child", childID, 1)[0], "child_of", parentID)
	})

	t.Run("duplicates kind", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "rel-duplicates", "Duplicates Kind Test")
		t1 := createTask(t, env, "rel-duplicates", `{"title":"Task 1"}`)
		t2 := createTask(t, env, "rel-duplicates", `{"title":"Task 2"}`)
		t1id, t2id := jsonAs[string](t, t1["id"]), jsonAs[string](t, t2["id"])
		createRelation(t, env, "rel-duplicates", t1id, "duplicates", t2id)
		assertRelationKind(t, assertTaskRelations(t, env, "rel-duplicates", t1id, 1)[0], "duplicates", t2id)
		assertRelationKind(t, assertTaskRelations(t, env, "rel-duplicates", t2id, 1)[0], "duplicates", t1id)
	})

	t.Run("in list response", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "rel-list", "List Rel Test")
		t1 := createTask(t, env, "rel-list", `{"title":"Task 1"}`)
		t2 := createTask(t, env, "rel-list", `{"title":"Task 2"}`)
		t1id, t2id := jsonAs[string](t, t1["id"]), jsonAs[string](t, t2["id"])
		createRelation(t, env, "rel-list", t1id, "blocks", t2id)

		resp := doRequest(t, env, "GET", "/spaces/rel-list/tasks", "")
		assertStatus(t, resp, http.StatusOK)
		var page map[string]any
		readJSON(t, resp, &page)
		items := jsonAs[[]any](t, page["items"])
		for _, item := range items {
			task := jsonAs[map[string]any](t, item)
			rels := jsonAs[[]any](t, task["relations"])
			switch task["id"] {
			case t1id:
				if len(rels) != 1 {
					t.Fatalf("T1 in list: got %d relations, want 1", len(rels))
				}
				assertRelationKind(t, rels[0], "blocks", t2id)
			case t2id:
				if len(rels) != 1 {
					t.Fatalf("T2 in list: got %d relations, want 1", len(rels))
				}
				assertRelationKind(t, rels[0], "blocked_by", t1id)
			}
		}
	})

	t.Run("empty by default", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "rel-empty", "Empty Rel Test")
		task := createTask(t, env, "rel-empty", `{"title":"Task 1"}`)
		rels := jsonAs[[]any](t, task["relations"])
		if len(rels) != 0 {
			t.Fatalf("create response: got %d relations, want 0", len(rels))
		}
		assertTaskRelations(t, env, "rel-empty", jsonAs[string](t, task["id"]), 0)
	})

	t.Run("non member rejected", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "rel-perm", "Permission Test")
		t1 := createTask(t, env, "rel-perm", `{"title":"Task 1"}`)
		t2 := createTask(t, env, "rel-perm", `{"title":"Task 2"}`)
		nonMemberToken := createTestUser(t, env, testEmail(t, "rel-outsider"), "Outsider", "password")
		resp := doRequestAs(t, env, nonMemberToken, "POST", "/spaces/rel-perm/tasks/"+jsonAs[string](t, t1["id"])+"/relations", `{"kind":"blocks","relatedTaskId":"`+jsonAs[string](t, t2["id"])+`"}`)
		assertStatusClose(t, resp, http.StatusNotFound)
	})

	t.Run("cascade on task delete", func(t *testing.T) {
		env := setupTestServer(t)
		createSpace(t, env, "rel-cascade", "Cascade Test")
		t1 := createTask(t, env, "rel-cascade", `{"title":"Task 1"}`)
		t2 := createTask(t, env, "rel-cascade", `{"title":"Task 2"}`)
		t2id := jsonAs[string](t, t2["id"])
		createRelation(t, env, "rel-cascade", jsonAs[string](t, t1["id"]), "blocks", t2id)
		assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/rel-cascade/tasks/"+jsonAs[string](t, t1["id"]), ""), http.StatusNoContent)
		assertTaskRelations(t, env, "rel-cascade", t2id, 0)
	})
}

package api_test

import (
	"net/http"
	"testing"
)

func TestTaskRelationsCreate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "rel-test", "Relation Test")
	t1 := createTask(t, env, "rel-test", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "rel-test", `{"title":"Task 2"}`)
	t1id, t2id := t1["id"].(string), t2["id"].(string)

	// Create a "blocks" relation: T1 blocks T2.
	resp := doRequest(t, env, "POST", "/spaces/rel-test/tasks/"+t1id+"/relations",
		`{"kind":"blocks","taskId":"`+t2id+`"}`)
	assertStatus(t, resp, http.StatusCreated)
	var rel map[string]any
	readJSON(t, resp, &rel)
	if rel["kind"] != "blocks" {
		t.Fatalf("got kind %v, want blocks", rel["kind"])
	}
	if rel["taskId"] != t2id {
		t.Fatalf("got taskId %v, want %v", rel["taskId"], t2id)
	}

	// Verify relation appears on T1 as "blocks" and T2 as "blocked_by".
	rels1 := assertTaskRelations(t, env, "rel-test", t1id, 1)
	assertRelationKind(t, rels1[0], "blocks", t2id)

	rels2 := assertTaskRelations(t, env, "rel-test", t2id, 1)
	assertRelationKind(t, rels2[0], "blocked_by", t1id)
}

func TestTaskRelationsSymmetric(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "sym-test", "Symmetric Test")
	t1 := createTask(t, env, "sym-test", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "sym-test", `{"title":"Task 2"}`)
	t1id, t2id := t1["id"].(string), t2["id"].(string)

	createRelation(t, env, "sym-test", t1id, "relates_to", t2id)

	// Both tasks should show "relates_to" pointing at the other.
	rels1 := assertTaskRelations(t, env, "sym-test", t1id, 1)
	assertRelationKind(t, rels1[0], "relates_to", t2id)

	rels2 := assertTaskRelations(t, env, "sym-test", t2id, 1)
	assertRelationKind(t, rels2[0], "relates_to", t1id)
}

func TestTaskRelationsDelete(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "del-test", "Delete Test")
	t1 := createTask(t, env, "del-test", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "del-test", `{"title":"Task 2"}`)
	t1id, t2id := t1["id"].(string), t2["id"].(string)

	createRelation(t, env, "del-test", t1id, "blocks", t2id)

	assertStatusClose(t, doRequest(t, env, "DELETE",
		"/spaces/del-test/tasks/"+t1id+"/relations/blocks/"+t2id, ""),
		http.StatusNoContent)

	// Verify relation is gone from both tasks (T1).
	assertTaskRelations(t, env, "del-test", t1id, 0)
	assertTaskRelations(t, env, "del-test", t2id, 0)
}

func TestTaskRelationsDeleteFromOtherSide(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "del2-test", "Delete Other Side Test")
	t1 := createTask(t, env, "del2-test", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "del2-test", `{"title":"Task 2"}`)
	t1id, t2id := t1["id"].(string), t2["id"].(string)

	createRelation(t, env, "del2-test", t1id, "blocks", t2id)

	// Delete from T2's perspective as "blocked_by".
	assertStatusClose(t, doRequest(t, env, "DELETE",
		"/spaces/del2-test/tasks/"+t2id+"/relations/blocked_by/"+t1id, ""),
		http.StatusNoContent)

	// Verify it's gone from both sides.
	assertTaskRelations(t, env, "del2-test", t1id, 0)
	assertTaskRelations(t, env, "del2-test", t2id, 0)
}

func TestTaskRelationsDeleteNonExistent(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "del-ne", "Delete Non-Existent")
	t1 := createTask(t, env, "del-ne", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "del-ne", `{"title":"Task 2"}`)

	// Delete a relation that was never created.
	assertStatusClose(t, doRequest(t, env, "DELETE",
		"/spaces/del-ne/tasks/"+t1["id"].(string)+"/relations/blocks/"+t2["id"].(string), ""),
		http.StatusNotFound)
}

func TestTaskRelationsSelfRejected(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "self-test", "Self Test")
	t1 := createTask(t, env, "self-test", `{"title":"Task 1"}`)

	resp := doRequest(t, env, "POST", "/spaces/self-test/tasks/"+t1["id"].(string)+"/relations",
		`{"kind":"blocks","taskId":"`+t1["id"].(string)+`"}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTaskRelationsCrossSpaceRejected(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "space-a", "Space A")
	createSpace(t, env, "space-b", "Space B")
	t1 := createTask(t, env, "space-a", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "space-b", `{"title":"Task 2"}`)

	resp := doRequest(t, env, "POST", "/spaces/space-a/tasks/"+t1["id"].(string)+"/relations",
		`{"kind":"blocks","taskId":"`+t2["id"].(string)+`"}`)
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestTaskRelationsNonExistentTask(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "ne-test", "Non-Existent Test")
	t1 := createTask(t, env, "ne-test", `{"title":"Task 1"}`)

	resp := doRequest(t, env, "POST", "/spaces/ne-test/tasks/"+t1["id"].(string)+"/relations",
		`{"kind":"blocks","taskId":"T99999"}`)
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestTaskRelationsDuplicateRejected(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "dup-test", "Dup Test")
	t1 := createTask(t, env, "dup-test", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "dup-test", `{"title":"Task 2"}`)
	t1id, t2id := t1["id"].(string), t2["id"].(string)

	createRelation(t, env, "dup-test", t1id, "blocks", t2id)

	// Same relation again should 409.
	resp := doRequest(t, env, "POST", "/spaces/dup-test/tasks/"+t1id+"/relations",
		`{"kind":"blocks","taskId":"`+t2id+`"}`)
	assertStatusClose(t, resp, http.StatusConflict)
}

func TestTaskRelationsDuplicateViaInverse(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "dup-inv", "Dup Inverse Test")
	t1 := createTask(t, env, "dup-inv", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "dup-inv", `{"title":"Task 2"}`)
	t1id, t2id := t1["id"].(string), t2["id"].(string)

	createRelation(t, env, "dup-inv", t1id, "blocks", t2id)

	// T2 blocked_by T1 should also 409 (same canonical row).
	resp := doRequest(t, env, "POST", "/spaces/dup-inv/tasks/"+t2id+"/relations",
		`{"kind":"blocked_by","taskId":"`+t1id+`"}`)
	assertStatusClose(t, resp, http.StatusConflict)
}

func TestTaskRelationsCreateViaBlockedBy(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "blkby-test", "Blocked By Test")
	t1 := createTask(t, env, "blkby-test", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "blkby-test", `{"title":"Task 2"}`)
	t1id, t2id := t1["id"].(string), t2["id"].(string)

	createRelation(t, env, "blkby-test", t2id, "blocked_by", t1id)

	rels1 := assertTaskRelations(t, env, "blkby-test", t1id, 1)
	assertRelationKind(t, rels1[0], "blocks", t2id)

	rels2 := assertTaskRelations(t, env, "blkby-test", t2id, 1)
	assertRelationKind(t, rels2[0], "blocked_by", t1id)
}

func TestTaskRelationsParentChild(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "pc-test", "Parent Child Test")
	parent := createTask(t, env, "pc-test", `{"title":"Parent"}`)
	child := createTask(t, env, "pc-test", `{"title":"Child"}`)
	parentID, childID := parent["id"].(string), child["id"].(string)

	createRelation(t, env, "pc-test", childID, "child_of", parentID)

	pRels := assertTaskRelations(t, env, "pc-test", parentID, 1)
	assertRelationKind(t, pRels[0], "parent_of", childID)

	cRels := assertTaskRelations(t, env, "pc-test", childID, 1)
	assertRelationKind(t, cRels[0], "child_of", parentID)
}

func TestTaskRelationsDuplicatesKind(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "dup-kind", "Duplicates Kind Test")
	t1 := createTask(t, env, "dup-kind", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "dup-kind", `{"title":"Task 2"}`)
	t1id, t2id := t1["id"].(string), t2["id"].(string)

	createRelation(t, env, "dup-kind", t1id, "duplicates", t2id)

	// Both sides should show "duplicates" (symmetric).
	rels1 := assertTaskRelations(t, env, "dup-kind", t1id, 1)
	assertRelationKind(t, rels1[0], "duplicates", t2id)

	rels2 := assertTaskRelations(t, env, "dup-kind", t2id, 1)
	assertRelationKind(t, rels2[0], "duplicates", t1id)
}

func TestTaskRelationsInListResponse(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "list-rel", "List Rel Test")
	t1 := createTask(t, env, "list-rel", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "list-rel", `{"title":"Task 2"}`)
	t1id, t2id := t1["id"].(string), t2["id"].(string)

	createRelation(t, env, "list-rel", t1id, "blocks", t2id)

	// List tasks and check relations are correctly included.
	resp := doRequest(t, env, "GET", "/spaces/list-rel/tasks", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := page["items"].([]any)

	for _, item := range items {
		task := item.(map[string]any)
		rels := task["relations"].([]any)
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
}

func TestTaskRelationsEmptyByDefault(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "empty-rel", "Empty Rel Test")
	task := createTask(t, env, "empty-rel", `{"title":"Task 1"}`)

	// Check both create response and GET response.
	rels := task["relations"].([]any)
	if len(rels) != 0 {
		t.Fatalf("create response: got %d relations, want 0", len(rels))
	}
	assertTaskRelations(t, env, "empty-rel", task["id"].(string), 0)
}

func TestTaskRelationsNonMemberRejected(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "perm-test", "Permission Test")
	t1 := createTask(t, env, "perm-test", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "perm-test", `{"title":"Task 2"}`)

	// Create a non-member user.
	nonMemberToken := createTestUser(t, env, "outsider@example.com", "Outsider", "password")

	// Non-member should not be able to create relations.
	resp := doRequestAs(t, env, nonMemberToken, "POST", "/spaces/perm-test/tasks/"+t1["id"].(string)+"/relations",
		`{"kind":"blocks","taskId":"`+t2["id"].(string)+`"}`)
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestTaskRelationsCascadeOnTaskDelete(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "cascade-rel", "Cascade Test")
	t1 := createTask(t, env, "cascade-rel", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "cascade-rel", `{"title":"Task 2"}`)
	t2id := t2["id"].(string)

	createRelation(t, env, "cascade-rel", t1["id"].(string), "blocks", t2id)

	// Delete T1.
	assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/cascade-rel/tasks/"+t1["id"].(string), ""), http.StatusNoContent)

	// T2 should have no relations.
	assertTaskRelations(t, env, "cascade-rel", t2id, 0)
}

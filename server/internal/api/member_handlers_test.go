package api_test

import (
	"net/http"
	"testing"
)

func TestSpaceMembersCreate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	aliceToken := createTestUser(t, env, "alice@example.com", "Alice", "pass123")
	userID := getUserID(t, env, aliceToken)

	resp2 := doRequest(t, env, "POST", "/spaces/home/members", `{"userId":"`+userID+`","role":"member"}`)
	assertStatus(t, resp2, http.StatusCreated)
	var member map[string]any
	readJSON(t, resp2, &member)
	if member["userId"] != userID {
		t.Errorf("userId = %v, want %v", member["userId"], userID)
	}
	if member["role"] != "member" {
		t.Errorf("role = %v, want member", member["role"])
	}
	if member["userName"] != "Alice" {
		t.Errorf("userName = %v, want Alice", member["userName"])
	}
}

func TestSpaceMembersCreateDuplicate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	aliceToken := createTestUser(t, env, "alice@example.com", "Alice", "pass123")
	userID := getUserID(t, env, aliceToken)

	// First add succeeds.
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/home/members", `{"userId":"`+userID+`","role":"member"}`), http.StatusCreated)

	// Second add should be 409 Conflict.
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/home/members", `{"userId":"`+userID+`","role":"member"}`), http.StatusConflict)
}

func TestSpaceMembersList(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "GET", "/spaces/home/members", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := page["items"].([]any)
	// Creator is auto-added as admin.
	if len(items) != 1 {
		t.Fatalf("got %d members, want 1", len(items))
	}
	if items[0].(map[string]any)["role"] != "admin" {
		t.Errorf("role = %v, want admin", items[0].(map[string]any)["role"])
	}
}

func TestSpaceMembersListPagination(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	// Space creator is auto-added as admin (1 member already).
	// Add 3 more members for a total of 4.
	createAndAddMember(t, env, "home", "b@example.com", "B", "pass123", "member")
	createAndAddMember(t, env, "home", "c@example.com", "C", "pass123", "member")
	createAndAddMember(t, env, "home", "d@example.com", "D", "pass123", "member")

	// Page 1: limit=2, should return 2 items with a cursor.
	resp := doRequest(t, env, "GET", "/spaces/home/members?limit=2", "")
	assertStatus(t, resp, http.StatusOK)
	var page1 map[string]any
	readJSON(t, resp, &page1)
	items1 := page1["items"].([]any)
	if len(items1) != 2 {
		t.Fatalf("page 1: got %d items, want 2", len(items1))
	}
	cursor := page1["nextCursor"]
	if cursor == nil {
		t.Fatal("page 1: expected nextCursor")
	}

	// Page 2: should return remaining 2 items with null cursor.
	resp2 := doRequest(t, env, "GET", "/spaces/home/members?limit=2&cursor="+cursor.(string), "")
	assertStatus(t, resp2, http.StatusOK)
	var page2 map[string]any
	readJSON(t, resp2, &page2)
	items2 := page2["items"].([]any)
	if len(items2) != 2 {
		t.Fatalf("page 2: got %d items, want 2", len(items2))
	}
	if page2["nextCursor"] != nil {
		t.Error("page 2: expected null nextCursor")
	}
}

func TestSpaceMembersUpdate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	_, userID := createAndAddMember(t, env, "home", "bob@example.com", "Bob", "pass123", "member")

	// Promote to admin.
	resp2 := doRequest(t, env, "PATCH", "/spaces/home/members/"+userID, `{"role":"admin"}`)
	assertStatus(t, resp2, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp2, &updated)
	if updated["role"] != "admin" {
		t.Errorf("role = %v, want admin", updated["role"])
	}
}

func TestSpaceMembersUpdateNotFound(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	// Attempt to update a non-existent member.
	assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/home/members/U999999", `{"role":"admin"}`), http.StatusNotFound)
}

func TestSpaceMembersDelete(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	userToken, userID := createAndAddMember(t, env, "home", "charlie@example.com", "Charlie", "pass123", "member")

	// Remove.
	assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/home/members/"+userID, ""), http.StatusNoContent)

	// User can no longer access the space.
	assertStatusClose(t, doRequestAs(t, env, userToken, "GET", "/spaces/home", ""), http.StatusNotFound)
}

func TestSpaceMembersLastAdminGuard(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	ownerID := getUserID(t, env, env.Token)

	// Try to downgrade the only admin to viewer.
	assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/home/members/"+ownerID, `{"role":"viewer"}`), http.StatusBadRequest)
	// Try to remove the only admin.
	assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/home/members/"+ownerID, ""), http.StatusBadRequest)
}

func TestMemberRemovalClearsAssignments(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	// Create a user and add as member.
	_, bobID := createAndAddMember(t, env, "home", "bob@example.com", "Bob", "pass123", "member")

	// Assign bob to a task.
	task := createTask(t, env, "home", `{"title":"Bob's task","assigneeIds":["`+bobID+`"]}`)
	taskID := task["id"].(string)

	// Verify bob is assigned.
	resp2 := doRequest(t, env, "GET", "/spaces/home/tasks/"+taskID, "")
	assertStatus(t, resp2, http.StatusOK)
	var before map[string]any
	readJSON(t, resp2, &before)
	if len(before["assigneeIds"].([]any)) != 1 {
		t.Fatalf("expected 1 assignee before removal")
	}

	// Remove bob from the space.
	assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/home/members/"+bobID, ""), http.StatusNoContent)

	// Bob's assignment should be cleared.
	resp3 := doRequest(t, env, "GET", "/spaces/home/tasks/"+taskID, "")
	assertStatus(t, resp3, http.StatusOK)
	var after map[string]any
	readJSON(t, resp3, &after)
	if len(after["assigneeIds"].([]any)) != 0 {
		t.Fatalf("expected 0 assignees after member removal, got %d", len(after["assigneeIds"].([]any)))
	}
}

func TestMemberRemovalIsolation(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "alpha", "Alpha")
	createSpace(t, env, "beta", "Beta")

	// Create bob and add to both spaces.
	userToken := createTestUser(t, env, "bob@example.com", "Bob", "pass123")
	bobID := getUserID(t, env, userToken)
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/alpha/members", `{"userId":"`+bobID+`","role":"member"}`), http.StatusCreated)
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/beta/members", `{"userId":"`+bobID+`","role":"member"}`), http.StatusCreated)

	// Assign bob to a task in each space.
	taskAlpha := createTask(t, env, "alpha", `{"title":"Alpha task","assigneeIds":["`+bobID+`"]}`)
	taskBeta := createTask(t, env, "beta", `{"title":"Beta task","assigneeIds":["`+bobID+`"]}`)

	// Remove bob from alpha only.
	assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/alpha/members/"+bobID, ""), http.StatusNoContent)

	// Alpha task should have no assignees.
	resp2 := doRequest(t, env, "GET", "/spaces/alpha/tasks/"+taskAlpha["id"].(string), "")
	assertStatus(t, resp2, http.StatusOK)
	var alpha map[string]any
	readJSON(t, resp2, &alpha)
	if len(alpha["assigneeIds"].([]any)) != 0 {
		t.Fatalf("alpha task: expected 0 assignees, got %d", len(alpha["assigneeIds"].([]any)))
	}

	// Beta task should still have bob assigned.
	resp3 := doRequest(t, env, "GET", "/spaces/beta/tasks/"+taskBeta["id"].(string), "")
	assertStatus(t, resp3, http.StatusOK)
	var beta map[string]any
	readJSON(t, resp3, &beta)
	if len(beta["assigneeIds"].([]any)) != 1 {
		t.Fatalf("beta task: expected 1 assignee, got %d", len(beta["assigneeIds"].([]any)))
	}
}

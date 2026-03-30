package api_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestTasksCreate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "POST", "/spaces/home/tasks", `{"title":"Wash dishes"}`)
	assertStatus(t, resp, http.StatusCreated)

	var task map[string]any
	readJSON(t, resp, &task)
	if task["title"] != "Wash dishes" {
		t.Errorf("title = %v, want Wash dishes", task["title"])
	}
	// Should default to the initial status.
	if task["status"] != "todo" {
		t.Errorf("status = %v, want todo", task["status"])
	}
	// ID should be T-prefixed.
	id := jsonAs[string](t, task["id"])
	if !strings.HasPrefix(id, "T") {
		t.Errorf("id = %v, want T-prefixed", id)
	}
	// Due date should be null.
	if task["due"] != nil {
		t.Errorf("dueAt = %v, want nil", task["due"])
	}
	// Timestamps should be set.
	if task["createdAt"] == nil || task["createdAt"] == "" {
		t.Error("createdAt should be set")
	}
}

func TestTasksCreateWithFields(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	body := `{"title":"Clean","description":"Deep clean","status":"done","due":{"at":"2025-06-15","timezone":"UTC"}}`
	resp := doRequest(t, env, "POST", "/spaces/home/tasks", body)
	assertStatus(t, resp, http.StatusCreated)

	var task map[string]any
	readJSON(t, resp, &task)
	if task["description"] != "Deep clean" {
		t.Errorf("description = %v, want Deep clean", task["description"])
	}
	if task["status"] != "done" {
		t.Errorf("status = %v, want done", task["status"])
	}
	if task["due"] == nil {
		t.Error("due should not be nil")
	}
	due := jsonAs[map[string]any](t, task["due"])
	if due["timezone"] != "UTC" {
		t.Errorf("due.timezone = %v, want UTC", due["timezone"])
	}
}

func TestTasksCreateInNonexistentSpace(t *testing.T) {
	env := setupTestServer(t)

	resp := doRequest(t, env, "POST", "/spaces/nonexistent/tasks", `{"title":"Task"}`)
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestTasksCreateInvalidStatus(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "POST", "/spaces/home/tasks", `{"title":"Task","status":"bogus"}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTasksRead(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	created := createTask(t, env, "home", `{"title":"Mop floors"}`)
	id := jsonAs[string](t, created["id"])

	resp := doRequest(t, env, "GET", "/spaces/home/tasks/"+id, "")
	assertStatus(t, resp, http.StatusOK)
	var task map[string]any
	readJSON(t, resp, &task)
	if task["title"] != "Mop floors" {
		t.Errorf("title = %v, want Mop floors", task["title"])
	}
}

func TestTasksReadNotFound(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "GET", "/spaces/home/tasks/T999", "")
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestTasksReadInvalidID(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "GET", "/spaces/home/tasks/invalid", "")
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTasksListEmpty(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "GET", "/spaces/home/tasks", "")
	assertStatus(t, resp, http.StatusOK)

	var page map[string]any
	readJSON(t, resp, &page)
	items := jsonAs[[]any](t, page["items"])
	if len(items) != 0 {
		t.Fatalf("got %d items, want 0", len(items))
	}
	if page["nextCursor"] != nil {
		t.Error("expected null nextCursor")
	}
}

func TestTasksListPagination(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	// Create 4 tasks with distinct titles for verification.
	var createdIDs []string
	for i := 1; i <= 4; i++ {
		task := createTask(t, env, "home", `{"title":"Task"}`)
		createdIDs = append(createdIDs, jsonAs[string](t, task["id"]))
	}

	// Page 1: should return 2 items with a cursor.
	resp := doRequest(t, env, "GET", "/spaces/home/tasks?limit=2", "")
	assertStatus(t, resp, http.StatusOK)
	var page1 map[string]any
	readJSON(t, resp, &page1)
	items1 := jsonAs[[]any](t, page1["items"])
	if len(items1) != 2 {
		t.Fatalf("page 1: got %d items, want 2", len(items1))
	}
	// Verify page 1 contains the first two created tasks.
	if jsonAs[map[string]any](t, items1[0])["id"] != createdIDs[0] || jsonAs[map[string]any](t, items1[1])["id"] != createdIDs[1] {
		t.Errorf("page 1: got ids %v/%v, want %v/%v",
			jsonAs[map[string]any](t, items1[0])["id"], jsonAs[map[string]any](t, items1[1])["id"],
			createdIDs[0], createdIDs[1])
	}
	cursor := page1["nextCursor"]
	if cursor == nil {
		t.Fatal("page 1: expected nextCursor")
	}

	// Page 2: should return the last 2 items with null cursor (exact boundary).
	resp2 := doRequest(t, env, "GET", "/spaces/home/tasks?limit=2&cursor="+jsonAs[string](t, cursor), "")
	assertStatus(t, resp2, http.StatusOK)
	var page2 map[string]any
	readJSON(t, resp2, &page2)
	items2 := jsonAs[[]any](t, page2["items"])
	if len(items2) != 2 {
		t.Fatalf("page 2: got %d items, want 2", len(items2))
	}
	if jsonAs[map[string]any](t, items2[0])["id"] != createdIDs[2] || jsonAs[map[string]any](t, items2[1])["id"] != createdIDs[3] {
		t.Errorf("page 2: got ids %v/%v, want %v/%v",
			jsonAs[map[string]any](t, items2[0])["id"], jsonAs[map[string]any](t, items2[1])["id"],
			createdIDs[2], createdIDs[3])
	}
	if page2["nextCursor"] != nil {
		t.Error("page 2: expected null nextCursor")
	}
}

func TestTasksListNonexistentSpace(t *testing.T) {
	env := setupTestServer(t)

	resp := doRequest(t, env, "GET", "/spaces/nonexistent/tasks", "")
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestTasksUpdate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	// Create with a non-empty description to test merge preservation.
	created := createTask(t, env, "home", `{"title":"Old title","description":"Keep me"}`)
	id := jsonAs[string](t, created["id"])

	resp := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+id, `{"title":"New title"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)
	if updated["title"] != "New title" {
		t.Errorf("title = %v, want New title", updated["title"])
	}
	// Description should be preserved from the original.
	if updated["description"] != "Keep me" {
		t.Errorf("description = %v, want Keep me", updated["description"])
	}
}

func TestTasksUpdateStatus(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	created := createTask(t, env, "home", `{"title":"Chore"}`)
	id := jsonAs[string](t, created["id"])

	resp := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+id, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)
	if updated["status"] != "done" {
		t.Errorf("status = %v, want done", updated["status"])
	}
}

func TestTasksUpdateInvalidStatus(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	created := createTask(t, env, "home", `{"title":"Chore"}`)
	id := jsonAs[string](t, created["id"])

	resp := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+id, `{"status":"nonexistent"}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTasksUpdateClearDue(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	created := createTask(t, env, "home", `{"title":"Task","due":{"at":"2025-06-15","timezone":"UTC"}}`)
	id := jsonAs[string](t, created["id"])

	// Verify the due date was actually set.
	resp := doRequest(t, env, "GET", "/spaces/home/tasks/"+id, "")
	assertStatus(t, resp, http.StatusOK)
	var fetched map[string]any
	readJSON(t, resp, &fetched)
	if fetched["due"] == nil {
		t.Fatal("dueAt should be set after creation")
	}

	// Clear due date by sending null.
	resp2 := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+id, `{"due":null}`)
	assertStatus(t, resp2, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp2, &updated)
	if updated["due"] != nil {
		t.Errorf("dueAt = %v, want nil", updated["due"])
	}
}

func TestTasksDelete(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	created := createTask(t, env, "home", `{"title":"Temp"}`)
	id := jsonAs[string](t, created["id"])

	resp := doRequest(t, env, "DELETE", "/spaces/home/tasks/"+id, "")
	assertStatusClose(t, resp, http.StatusNoContent)

	// Verify it's gone.
	resp2 := doRequest(t, env, "GET", "/spaces/home/tasks/"+id, "")
	assertStatusClose(t, resp2, http.StatusNotFound)
}

func TestSpaceDeleteCascadesTasks(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	created := createTask(t, env, "home", `{"title":"Chore"}`)
	id := jsonAs[string](t, created["id"])

	// Delete the space - tasks should cascade.
	resp := doRequest(t, env, "DELETE", "/spaces/home", "")
	assertStatusClose(t, resp, http.StatusNoContent)

	// Task should be gone.
	resp2 := doRequest(t, env, "GET", "/spaces/home/tasks/"+id, "")
	assertStatusClose(t, resp2, http.StatusNotFound)
}

// --- Cross-Space Isolation Tests ---

func TestCrossSpaceTaskRead(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "space-a", "Space A")
	createSpace(t, env, "space-b", "Space B")
	task := createTask(t, env, "space-b", `{"title":"Secret"}`)
	id := jsonAs[string](t, task["id"])

	// Reading space-b's task via space-a's URL should 404.
	resp := doRequest(t, env, "GET", "/spaces/space-a/tasks/"+id, "")
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestCrossSpaceTaskUpdate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "space-a", "Space A")
	createSpace(t, env, "space-b", "Space B")
	task := createTask(t, env, "space-b", `{"title":"Secret"}`)
	id := jsonAs[string](t, task["id"])

	// Updating space-b's task via space-a's URL should 404.
	resp := doRequest(t, env, "PATCH", "/spaces/space-a/tasks/"+id, `{"title":"Hacked"}`)
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestCrossSpaceTaskDelete(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "space-a", "Space A")
	createSpace(t, env, "space-b", "Space B")
	task := createTask(t, env, "space-b", `{"title":"Secret"}`)
	id := jsonAs[string](t, task["id"])

	// Deleting space-b's task via space-a's URL should 404.
	resp := doRequest(t, env, "DELETE", "/spaces/space-a/tasks/"+id, "")
	assertStatusClose(t, resp, http.StatusNotFound)

	// Task should still exist in space-b.
	resp2 := doRequest(t, env, "GET", "/spaces/space-b/tasks/"+id, "")
	assertStatusClose(t, resp2, http.StatusOK)
}

// --- Task Assignee Tests ---

func TestTaskAssigneesOnCreate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	// Get owner's user ID.
	ownerID := getUserID(t, env, env.Token)

	// Create task with assignees.
	task := createTask(t, env, "home", `{"title":"With assignees","assigneeIds":["`+ownerID+`"]}`)
	assigneeIds := jsonAs[[]any](t, task["assigneeIds"])
	if len(assigneeIds) != 1 {
		t.Fatalf("got %d assignees, want 1", len(assigneeIds))
	}
	if jsonAs[string](t, assigneeIds[0]) != ownerID {
		t.Errorf("assigneeIds[0] = %v, want %v", assigneeIds[0], ownerID)
	}
}

func TestTaskAssigneesEmptyByDefault(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	task := createTask(t, env, "home", `{"title":"No assignees"}`)
	assigneeIds := jsonAs[[]any](t, task["assigneeIds"])
	if len(assigneeIds) != 0 {
		t.Fatalf("got %d assignees, want 0", len(assigneeIds))
	}
}

func TestTaskAssigneesUpdate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	// Get owner's user ID.
	ownerID := getUserID(t, env, env.Token)

	// Create a second user and add as member.
	_, bobID := createAndAddMember(t, env, "home", "bob@example.com", "Bob", "pass1234", "member")

	// Create task without assignees.
	task := createTask(t, env, "home", `{"title":"Test"}`)
	taskID := jsonAs[string](t, task["id"])

	// Update: set assignees.
	resp3 := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+taskID, `{"assigneeIds":["`+ownerID+`","`+bobID+`"]}`)
	assertStatus(t, resp3, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp3, &updated)
	assigneeIds := jsonAs[[]any](t, updated["assigneeIds"])
	if len(assigneeIds) != 2 {
		t.Fatalf("got %d assignees, want 2", len(assigneeIds))
	}
	// Verify actual IDs are present (order is by user_id).
	ids := make(map[string]bool)
	for _, id := range assigneeIds {
		ids[jsonAs[string](t, id)] = true
	}
	if !ids[ownerID] || !ids[bobID] {
		t.Errorf("assigneeIds = %v, want both %s and %s", assigneeIds, ownerID, bobID)
	}

	// Update: clear assignees.
	resp4 := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+taskID, `{"assigneeIds":[]}`)
	assertStatus(t, resp4, http.StatusOK)
	var cleared map[string]any
	readJSON(t, resp4, &cleared)
	if len(jsonAs[[]any](t, cleared["assigneeIds"])) != 0 {
		t.Error("expected empty assignees after clearing")
	}
}

func TestTaskAssigneesPreservedOnUnrelatedUpdate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	ownerID := getUserID(t, env, env.Token)

	// Create task with assignee.
	task := createTask(t, env, "home", `{"title":"Test","assigneeIds":["`+ownerID+`"]}`)
	taskID := jsonAs[string](t, task["id"])

	// Update title only — assignees should be preserved.
	resp2 := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+taskID, `{"title":"Updated"}`)
	assertStatus(t, resp2, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp2, &updated)
	if updated["title"] != "Updated" {
		t.Errorf("title = %v, want Updated", updated["title"])
	}
	assigneeIds := jsonAs[[]any](t, updated["assigneeIds"])
	if len(assigneeIds) != 1 {
		t.Fatalf("got %d assignees, want 1 (preserved)", len(assigneeIds))
	}
	if jsonAs[string](t, assigneeIds[0]) != ownerID {
		t.Errorf("assigneeIds[0] = %v, want %v", assigneeIds[0], ownerID)
	}
}

func TestTaskAssigneesNonMemberRejected(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	// Create a user but don't add them as a member.
	outsiderToken := createTestUser(t, env, "outsider@example.com", "Outsider", "pass1234")
	outsiderID := getUserID(t, env, outsiderToken)

	// Try to assign the non-member.
	task := createTask(t, env, "home", `{"title":"Test"}`)
	taskID := jsonAs[string](t, task["id"])
	resp2 := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+taskID, `{"assigneeIds":["`+outsiderID+`"]}`)
	assertStatusClose(t, resp2, http.StatusBadRequest)
}

func TestTaskAssigneesCrossSpaceRejected(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "alpha", "Alpha")
	createSpace(t, env, "beta", "Beta")

	// Create user and add to beta only.
	_, bobID := createAndAddMember(t, env, "beta", "bob@example.com", "Bob", "pass1234", "member")

	// Try to assign bob to a task in alpha (where he's not a member).
	task := createTask(t, env, "alpha", `{"title":"Test"}`)
	taskID := jsonAs[string](t, task["id"])
	resp2 := doRequest(t, env, "PATCH", "/spaces/alpha/tasks/"+taskID, `{"assigneeIds":["`+bobID+`"]}`)
	assertStatusClose(t, resp2, http.StatusBadRequest)
}

func TestTaskAssigneesDeduplicated(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	ownerID := getUserID(t, env, env.Token)

	// Create task with duplicate assignee IDs.
	task := createTask(t, env, "home", `{"title":"Dedup","assigneeIds":["`+ownerID+`","`+ownerID+`"]}`)
	assigneeIds := jsonAs[[]any](t, task["assigneeIds"])
	if len(assigneeIds) != 1 {
		t.Fatalf("got %d assignees, want 1 (deduplicated)", len(assigneeIds))
	}
}

func TestTaskDeleteCascadesAssignees(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	ownerID := getUserID(t, env, env.Token)

	task := createTask(t, env, "home", `{"title":"Delete me","assigneeIds":["`+ownerID+`"]}`)
	taskID := jsonAs[string](t, task["id"])

	// Delete the task.
	assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/home/tasks/"+taskID, ""), http.StatusNoContent)

	// Task is gone.
	assertStatusClose(t, doRequest(t, env, "GET", "/spaces/home/tasks/"+taskID, ""), http.StatusNotFound)
}

func TestTaskAssigneesInListResponse(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	ownerID := getUserID(t, env, env.Token)

	// Create two tasks: one with assignees, one without.
	createTask(t, env, "home", `{"title":"Assigned","assigneeIds":["`+ownerID+`"]}`)
	createTask(t, env, "home", `{"title":"Unassigned"}`)

	// List tasks and check assigneeIds are present in both items.
	resp2 := doRequest(t, env, "GET", "/spaces/home/tasks", "")
	assertStatus(t, resp2, http.StatusOK)
	var page map[string]any
	readJSON(t, resp2, &page)
	items := jsonAs[[]any](t, page["items"])
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}

	for _, item := range items {
		task := jsonAs[map[string]any](t, item)
		title := jsonAs[string](t, task["title"])
		assignees := jsonAs[[]any](t, task["assigneeIds"])
		if title == "Assigned" {
			if len(assignees) != 1 || jsonAs[string](t, assignees[0]) != ownerID {
				t.Errorf("Assigned task: assigneeIds = %v, want [%s]", assignees, ownerID)
			}
		} else {
			if len(assignees) != 0 {
				t.Errorf("Unassigned task: assigneeIds = %v, want []", assignees)
			}
		}
	}
}

func TestTaskTagsInListResponse(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	// Create two tasks: one with tags, one without.
	createTask(t, env, "home", `{"title":"Tagged","tags":["bug","urgent"]}`)
	createTask(t, env, "home", `{"title":"Untagged"}`)

	// List tasks and check tags are present in both items.
	resp := doRequest(t, env, "GET", "/spaces/home/tasks", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := jsonAs[[]any](t, page["items"])
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}

	for _, item := range items {
		task := jsonAs[map[string]any](t, item)
		title := jsonAs[string](t, task["title"])
		tags := jsonAs[[]any](t, task["tags"])
		if title == "Tagged" {
			if len(tags) != 2 {
				t.Errorf("Tagged task: got %d tags, want 2", len(tags))
			}
		} else {
			if len(tags) != 0 {
				t.Errorf("Untagged task: tags = %v, want []", tags)
			}
		}
	}
}

func TestTaskAssigneesInvalidIDFormat(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	task := createTask(t, env, "home", `{"title":"Test"}`)
	taskID := jsonAs[string](t, task["id"])

	// Invalid format (no U prefix).
	resp := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+taskID, `{"assigneeIds":["invalid"]}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTaskAssigneesNonExistentUser(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	task := createTask(t, env, "home", `{"title":"Test"}`)
	taskID := jsonAs[string](t, task["id"])

	// Valid format but user doesn't exist.
	resp := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+taskID, `{"assigneeIds":["U99999"]}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTaskTagCaseFoldingPreservesDisplayName(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "tag-fold", "Tag Fold")

	// 1. Explicitly create tag with display name "Bug".
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-fold/tags", `{"name":"Bug"}`), http.StatusCreated)

	// 2. Create a task using lowercase "bug".
	task := createTask(t, env, "tag-fold", `{"title":"Test","tags":["bug"]}`)
	taskID := jsonAs[string](t, task["id"])

	// 3. GET the task — tag should be "Bug" (original display name), not "bug".
	resp := doRequest(t, env, "GET", "/spaces/tag-fold/tasks/"+taskID, "")
	assertStatus(t, resp, http.StatusOK)
	var fetched map[string]any
	readJSON(t, resp, &fetched)
	tags := jsonAs[[]any](t, fetched["tags"])
	if len(tags) != 1 {
		t.Fatalf("got %d tags, want 1", len(tags))
	}
	if tags[0] != "Bug" {
		t.Errorf("tag = %v, want Bug", tags[0])
	}
}

package api_test

import (
	"net/http"
	"testing"
	"time"
)

func TestUsersMe(t *testing.T) {
	t.Parallel()
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

func TestUserTasksList(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	ownerID := getUserID(t, env, env.Token)

	// Create two spaces with different configurations.
	createSpace(t, env, "work", "Work")
	createSpace(t, env, "home", "Home")

	// Set up statuses in work space: todo (initial), doing (intermediate), done (completion).
	assertStatusClose(t, doRequest(t, env, "PUT", "/spaces/work/task-statuses",
		`{"items":[{"name":"todo","category":"initial"},{"name":"doing","category":"intermediate"},{"name":"done","category":"completion"}]}`),
		http.StatusOK)

	// Set up priority levels in work space.
	assertStatusClose(t, doRequest(t, env, "PUT", "/spaces/work/task-priority-levels",
		`{"items":[{"name":"high"},{"name":"low"}]}`),
		http.StatusOK)

	dueDate := time.Now().AddDate(0, 0, 5).Format(time.DateOnly)

	// Assign the owner to tasks in both spaces.
	// Task 1: work space, doing status, due soon, high priority
	task1 := createTask(t, env, "work", `{"title":"Work task 1","status":"doing","due":{"at":"`+dueDate+`","timezone":"UTC"},"priority":"high","assigneeIds":["`+ownerID+`"]}`)
	// Task 2: home space, default status (todo), no due date
	task2 := createTask(t, env, "home", `{"title":"Home task","assigneeIds":["`+ownerID+`"]}`)
	// Task 3: work space, done status — should sort last
	task3 := createTask(t, env, "work", `{"title":"Work task done","status":"done","assigneeIds":["`+ownerID+`"]}`)
	// Task 4: work space, not assigned to owner — should NOT appear
	_ = createTask(t, env, "work", `{"title":"Not mine"}`)

	// Expected order: task1 (doing/-1, due), task2 (todo/0, no due=inf), task3 (done/+, inf)
	expectedOrder := []string{
		jsonAs[string](t, task1["id"]),
		jsonAs[string](t, task2["id"]),
		jsonAs[string](t, task3["id"]),
	}

	resp := doRequest(t, env, "GET", "/users/"+ownerID+"/tasks?limit=10", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := jsonAs[[]any](t, page["items"])
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}

	for i, item := range items {
		task := jsonAs[map[string]any](t, item)
		got := jsonAs[string](t, task["id"])
		if got != expectedOrder[i] {
			t.Errorf("position %d: got %s, want %s", i, got, expectedOrder[i])
		}
		// Verify spaceSlug is present on each task.
		if task["spaceSlug"] == nil || task["spaceSlug"] == "" {
			t.Errorf("position %d: spaceSlug should be set", i)
		}
	}

	// Verify cross-space: first task is from work, second from home.
	if jsonAs[string](t, jsonAs[map[string]any](t, items[0])["spaceSlug"]) != "work" {
		t.Errorf("item 0 spaceSlug = %v, want work", jsonAs[map[string]any](t, items[0])["spaceSlug"])
	}
	if jsonAs[string](t, jsonAs[map[string]any](t, items[1])["spaceSlug"]) != "home" {
		t.Errorf("item 1 spaceSlug = %v, want home", jsonAs[map[string]any](t, items[1])["spaceSlug"])
	}

	// Verify enrichment: each task should have assigneeIds populated.
	for i, item := range items {
		task := jsonAs[map[string]any](t, item)
		assignees := jsonAs[[]any](t, task["assigneeIds"])
		if len(assignees) != 1 || jsonAs[string](t, assignees[0]) != ownerID {
			t.Errorf("position %d: assigneeIds = %v, want [%s]", i, assignees, ownerID)
		}
		// tags, relations, rotationPool should be empty but present.
		if jsonAs[[]any](t, task["tags"]) == nil {
			t.Errorf("position %d: tags should be non-nil", i)
		}
		if jsonAs[[]any](t, task["relations"]) == nil {
			t.Errorf("position %d: relations should be non-nil", i)
		}
		if jsonAs[[]any](t, task["rotationPool"]) == nil {
			t.Errorf("position %d: rotationPool should be non-nil", i)
		}
	}
}

func TestUserTasksListPagination(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	ownerID := getUserID(t, env, env.Token)
	createSpace(t, env, "s", "Space")

	// Create 3 tasks assigned to owner.
	task1 := createTask(t, env, "s", `{"title":"T1","assigneeIds":["`+ownerID+`"]}`)
	task2 := createTask(t, env, "s", `{"title":"T2","assigneeIds":["`+ownerID+`"]}`)
	task3 := createTask(t, env, "s", `{"title":"T3","assigneeIds":["`+ownerID+`"]}`)

	expectedOrder := []string{
		jsonAs[string](t, task1["id"]),
		jsonAs[string](t, task2["id"]),
		jsonAs[string](t, task3["id"]),
	}

	// Page 1: limit=2
	resp := doRequest(t, env, "GET", "/users/"+ownerID+"/tasks?limit=2", "")
	assertStatus(t, resp, http.StatusOK)
	var page1 map[string]any
	readJSON(t, resp, &page1)
	items1 := jsonAs[[]any](t, page1["items"])
	if len(items1) != 2 {
		t.Fatalf("page 1: got %d items, want 2", len(items1))
	}
	if jsonAs[string](t, jsonAs[map[string]any](t, items1[0])["id"]) != expectedOrder[0] {
		t.Errorf("page 1 item 0: got %v, want %v", jsonAs[map[string]any](t, items1[0])["id"], expectedOrder[0])
	}
	if jsonAs[string](t, jsonAs[map[string]any](t, items1[1])["id"]) != expectedOrder[1] {
		t.Errorf("page 1 item 1: got %v, want %v", jsonAs[map[string]any](t, items1[1])["id"], expectedOrder[1])
	}
	cursor := page1["nextCursor"]
	if cursor == nil {
		t.Fatal("page 1: expected nextCursor")
	}

	// Page 2
	resp2 := doRequest(t, env, "GET", "/users/"+ownerID+"/tasks?limit=2&cursor="+jsonAs[string](t, cursor), "")
	assertStatus(t, resp2, http.StatusOK)
	var page2 map[string]any
	readJSON(t, resp2, &page2)
	items2 := jsonAs[[]any](t, page2["items"])
	if len(items2) != 1 {
		t.Fatalf("page 2: got %d items, want 1", len(items2))
	}
	if jsonAs[string](t, jsonAs[map[string]any](t, items2[0])["id"]) != expectedOrder[2] {
		t.Errorf("page 2 item 0: got %v, want %v", jsonAs[map[string]any](t, items2[0])["id"], expectedOrder[2])
	}
	if page2["nextCursor"] != nil {
		t.Errorf("page 2: expected no nextCursor, got %v", page2["nextCursor"])
	}
}

func TestUserTasksListCrossSpaceIdenticalSortKeys(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	ownerID := getUserID(t, env, env.Token)

	// Create three spaces. Tasks will have identical sort keys (default status,
	// no due date, no priority, no effort) — only t.id breaks ties.
	createSpace(t, env, "a", "Alpha")
	createSpace(t, env, "b", "Beta")
	createSpace(t, env, "c", "Gamma")

	task1 := createTask(t, env, "a", `{"title":"A1","assigneeIds":["`+ownerID+`"]}`)
	task2 := createTask(t, env, "b", `{"title":"B1","assigneeIds":["`+ownerID+`"]}`)
	task3 := createTask(t, env, "c", `{"title":"C1","assigneeIds":["`+ownerID+`"]}`)

	expectedOrder := []string{
		jsonAs[string](t, task1["id"]),
		jsonAs[string](t, task2["id"]),
		jsonAs[string](t, task3["id"]),
	}

	// Page 1: limit=2 forces cursor at identical sort keys.
	resp := doRequest(t, env, "GET", "/users/"+ownerID+"/tasks?limit=2", "")
	assertStatus(t, resp, http.StatusOK)
	var page1 map[string]any
	readJSON(t, resp, &page1)
	items1 := jsonAs[[]any](t, page1["items"])
	if len(items1) != 2 {
		t.Fatalf("page 1: got %d items, want 2", len(items1))
	}
	if jsonAs[string](t, jsonAs[map[string]any](t, items1[0])["id"]) != expectedOrder[0] {
		t.Errorf("page 1 item 0: got %v, want %v", jsonAs[map[string]any](t, items1[0])["id"], expectedOrder[0])
	}
	if jsonAs[string](t, jsonAs[map[string]any](t, items1[1])["id"]) != expectedOrder[1] {
		t.Errorf("page 1 item 1: got %v, want %v", jsonAs[map[string]any](t, items1[1])["id"], expectedOrder[1])
	}
	cursor := page1["nextCursor"]
	if cursor == nil {
		t.Fatal("page 1: expected nextCursor")
	}

	// Page 2: should get the third task without duplicates or gaps.
	resp2 := doRequest(t, env, "GET", "/users/"+ownerID+"/tasks?limit=2&cursor="+jsonAs[string](t, cursor), "")
	assertStatus(t, resp2, http.StatusOK)
	var page2 map[string]any
	readJSON(t, resp2, &page2)
	items2 := jsonAs[[]any](t, page2["items"])
	if len(items2) != 1 {
		t.Fatalf("page 2: got %d items, want 1", len(items2))
	}
	if jsonAs[string](t, jsonAs[map[string]any](t, items2[0])["id"]) != expectedOrder[2] {
		t.Errorf("page 2 item 0: got %v, want %v", jsonAs[map[string]any](t, items2[0])["id"], expectedOrder[2])
	}
	if page2["nextCursor"] != nil {
		t.Errorf("page 2: expected no nextCursor, got %v", page2["nextCursor"])
	}
}

func TestUserTasksListForbiddenForOtherUser(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	// Create a non-owner user.
	token := createTestUser(t, env, "alice@test.com", "Alice", "password")
	aliceID := getUserID(t, env, token)

	// Alice tries to view the owner's tasks — should be forbidden.
	ownerID := getUserID(t, env, env.Token)
	resp := doRequestAs(t, env, token, "GET", "/users/"+ownerID+"/tasks", "")
	assertStatusClose(t, resp, http.StatusForbidden)

	// Owner (is_owner=true) can view Alice's tasks.
	resp2 := doRequest(t, env, "GET", "/users/"+aliceID+"/tasks", "")
	assertStatus(t, resp2, http.StatusOK)
	var page map[string]any
	readJSON(t, resp2, &page)
	items := jsonAs[[]any](t, page["items"])
	if len(items) != 0 {
		t.Errorf("expected 0 tasks for alice, got %d", len(items))
	}
}

func TestUserTasksListEmpty(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	ownerID := getUserID(t, env, env.Token)
	createSpace(t, env, "s", "Space")

	// Create a task NOT assigned to owner.
	createTask(t, env, "s", `{"title":"Unassigned"}`)

	resp := doRequest(t, env, "GET", "/users/"+ownerID+"/tasks", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := jsonAs[[]any](t, page["items"])
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

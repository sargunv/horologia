package api_test

import (
	"net/http"
	"strings"
	"testing"
	"time"
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

	futureDate := time.Now().AddDate(0, 0, 30).Format(time.DateOnly)
	body := `{"title":"Clean","description":"Deep clean","status":"done","due":{"at":"` + futureDate + `","timezone":"UTC"}}`
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

func TestTasksListSortOrder(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "proj", "Project")

	// Set up statuses: backlog (initial), in_progress (intermediate), done (completion).
	assertStatusClose(t, doRequest(t, env, "PUT", "/spaces/proj/task-statuses",
		`{"items":[{"name":"backlog","category":"initial"},{"name":"in_progress","category":"intermediate"},{"name":"done","category":"completion"}]}`),
		http.StatusOK)

	// Set up priority levels: urgent (0), normal (1), low (2).
	assertStatusClose(t, doRequest(t, env, "PUT", "/spaces/proj/task-priority-levels",
		`{"items":[{"name":"urgent"},{"name":"normal"},{"name":"low"}]}`),
		http.StatusOK)

	// Set up effort levels: small (0), medium (1), large (2).
	assertStatusClose(t, doRequest(t, env, "PUT", "/spaces/proj/task-effort-levels",
		`{"items":[{"name":"small"},{"name":"medium"},{"name":"large"}]}`),
		http.StatusOK)

	futureDate := time.Now().AddDate(0, 0, 30).Format(time.DateOnly)
	pastDate := time.Now().AddDate(0, 0, -5).Format(time.DateOnly)

	// Create tasks in an order that differs from the expected sort.
	// Task A: backlog, no due date, no priority — should sort late (initial status, no due)
	taskA := createTask(t, env, "proj", `{"title":"A-backlog-nodue"}`)

	// Task B: in_progress, past due, urgent, small effort — sort first
	taskB := createTask(t, env, "proj", `{"title":"B","status":"in_progress","due":{"at":"`+pastDate+`","timezone":"UTC"},"priority":"urgent","effort":"small"}`)

	// Task C: in_progress, past due, urgent, large effort — same status+due+priority as B, effort tiebreaker
	taskC := createTask(t, env, "proj", `{"title":"C","status":"in_progress","due":{"at":"`+pastDate+`","timezone":"UTC"},"priority":"urgent","effort":"large"}`)

	// Task D: in_progress, future due, low priority — same status as B/C, later due
	taskD := createTask(t, env, "proj", `{"title":"D","status":"in_progress","due":{"at":"`+futureDate+`","timezone":"UTC"},"priority":"low"}`)

	// Task E: backlog, past due, normal priority — lower status than in_progress
	taskE := createTask(t, env, "proj", `{"title":"E","due":{"at":"`+pastDate+`","timezone":"UTC"},"priority":"normal"}`)

	// Task F: backlog, no due date — no due date sorts after all dated tasks
	taskF := createTask(t, env, "proj", `{"title":"F"}`)

	// Task G: done — completion status sorts last
	taskG := createTask(t, env, "proj", `{"title":"G","status":"done"}`)

	// Expected order exercises all sort dimensions:
	// 1. B: in_progress(-1), pastDate, urgent(0), small(0)
	// 2. C: in_progress(-1), pastDate, urgent(0), large(2)  — effort tiebreaker
	// 3. D: in_progress(-1), futureDate, low(2), no effort   — due date tiebreaker
	// 4. A: backlog(0), no due(inf), no priority, no effort  — status tiebreaker (but no due date)
	// 5. E: backlog(0), pastDate, normal(1), no effort        — status tiebreaker
	// 6. F: backlog(0), no due(inf), no priority, no effort  — due date tiebreaker vs E
	// 7. G: done(2), no due(inf)                              — completion last
	expectedOrder := []string{
		jsonAs[string](t, taskB["id"]),
		jsonAs[string](t, taskC["id"]),
		jsonAs[string](t, taskD["id"]),
		jsonAs[string](t, taskE["id"]),
		jsonAs[string](t, taskA["id"]),
		jsonAs[string](t, taskF["id"]),
		jsonAs[string](t, taskG["id"]),
	}

	resp := doRequest(t, env, "GET", "/spaces/proj/tasks?limit=10", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := jsonAs[[]any](t, page["items"])
	if len(items) != 7 {
		t.Fatalf("got %d items, want 7", len(items))
	}

	for i, item := range items {
		got := jsonAs[string](t, jsonAs[map[string]any](t, item)["id"])
		if got != expectedOrder[i] {
			t.Errorf("position %d: got %s, want %s", i, got, expectedOrder[i])
		}
	}
}

func TestTasksSearchOwnerAcrossSpaces(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	createSpace(t, env, "work", "Work")
	homeTask := createTask(t, env, "home", `{"title":"Buy oat milk"}`)
	workTask := createTask(t, env, "work", `{"title":"Buy office snacks"}`)

	resp := doRequest(t, env, "GET", "/tasks/search?q=buy", "")
	assertStatus(t, resp, http.StatusOK)

	var result map[string]any
	readJSON(t, resp, &result)
	items := jsonAs[[]any](t, result["items"])
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}

	got := map[string]string{}
	for _, item := range items {
		row := jsonAs[map[string]any](t, item)
		got[jsonAs[string](t, row["id"])] = jsonAs[string](t, row["spaceSlug"])
	}
	if got[jsonAs[string](t, homeTask["id"])] != "home" {
		t.Fatalf("missing home task in results: %v", got)
	}
	if got[jsonAs[string](t, workTask["id"])] != "work" {
		t.Fatalf("missing work task in results: %v", got)
	}
}

func TestTasksSearchFiltersToVisibleSpaces(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "alpha", "Alpha")
	createSpace(t, env, "beta", "Beta")
	alphaTask := createTask(t, env, "alpha", `{"title":"Shared title"}`)
	createTask(t, env, "beta", `{"title":"Shared title"}`)

	userToken, _ := createAndAddMember(t, env, "alpha", "viewer@example.com", "Viewer", "pass1234", "viewer")

	resp := doRequestAs(t, env, userToken, "GET", "/tasks/search?q=shared", "")
	assertStatus(t, resp, http.StatusOK)

	var result map[string]any
	readJSON(t, resp, &result)
	items := jsonAs[[]any](t, result["items"])
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}

	row := jsonAs[map[string]any](t, items[0])
	if row["id"] != alphaTask["id"] {
		t.Fatalf("got id %v, want %v", row["id"], alphaTask["id"])
	}
	if row["spaceSlug"] != "alpha" {
		t.Fatalf("got spaceSlug %v, want alpha", row["spaceSlug"])
	}
}

func TestTasksSearchOptionalSpaceFilter(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "alpha", "Alpha")
	createSpace(t, env, "beta", "Beta")
	alphaTask := createTask(t, env, "alpha", `{"title":"Release notes"}`)
	createTask(t, env, "beta", `{"title":"Release notes"}`)

	resp := doRequest(t, env, "GET", "/tasks/search?q=release&spaceSlug=alpha", "")
	assertStatus(t, resp, http.StatusOK)

	var result map[string]any
	readJSON(t, resp, &result)
	items := jsonAs[[]any](t, result["items"])
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}

	row := jsonAs[map[string]any](t, items[0])
	if row["id"] != alphaTask["id"] {
		t.Fatalf("got id %v, want %v", row["id"], alphaTask["id"])
	}
	if row["spaceSlug"] != "alpha" {
		t.Fatalf("got spaceSlug %v, want alpha", row["spaceSlug"])
	}
}

func TestTasksSearchExcludeTaskIDAndExactID(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	first := createTask(t, env, "home", `{"title":"Plan sprint"}`)
	second := createTask(t, env, "home", `{"title":"Plan retrospective"}`)
	firstID := jsonAs[string](t, first["id"])

	resp := doRequest(t, env, "GET", "/tasks/search?q=plan&excludeTaskId="+firstID, "")
	assertStatus(t, resp, http.StatusOK)

	var result map[string]any
	readJSON(t, resp, &result)
	items := jsonAs[[]any](t, result["items"])
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}

	row := jsonAs[map[string]any](t, items[0])
	if row["id"] != second["id"] {
		t.Fatalf("got id %v, want %v", row["id"], second["id"])
	}

	resp = doRequest(t, env, "GET", "/tasks/search?q="+firstID, "")
	assertStatus(t, resp, http.StatusOK)

	readJSON(t, resp, &result)
	items = jsonAs[[]any](t, result["items"])
	if len(items) != 1 {
		t.Fatalf("exact-id search got %d items, want 1", len(items))
	}
	row = jsonAs[map[string]any](t, items[0])
	if row["id"] != firstID {
		t.Fatalf("got id %v, want %v", row["id"], firstID)
	}
}

func TestTasksSearchLeadingTStillDoesTextSearch(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	task := createTask(t, env, "home", `{"title":"Task planning"}`)

	resp := doRequest(t, env, "GET", "/tasks/search?q=Task", "")
	assertStatus(t, resp, http.StatusOK)

	var result map[string]any
	readJSON(t, resp, &result)
	items := jsonAs[[]any](t, result["items"])
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}

	row := jsonAs[map[string]any](t, items[0])
	if row["id"] != task["id"] {
		t.Fatalf("got id %v, want %v", row["id"], task["id"])
	}
}

func TestTasksListSortPagination(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	// Priority levels: high (pos 0), medium (pos 1), low (pos 2).
	assertStatusClose(t, doRequest(t, env, "PUT", "/spaces/home/task-priority-levels",
		`{"items":[{"name":"high"},{"name":"medium"},{"name":"low"}]}`),
		http.StatusOK)

	dueDate := time.Now().AddDate(0, 0, 1).Format(time.DateOnly)

	// 3 tasks with same status+due, differing only in priority.
	// Page break at limit=2 forces the cursor to cross a priority boundary.
	task1 := createTask(t, env, "home", `{"title":"T1","due":{"at":"`+dueDate+`","timezone":"UTC"},"priority":"high"}`)
	task2 := createTask(t, env, "home", `{"title":"T2","due":{"at":"`+dueDate+`","timezone":"UTC"},"priority":"medium"}`)
	task3 := createTask(t, env, "home", `{"title":"T3","due":{"at":"`+dueDate+`","timezone":"UTC"},"priority":"low"}`)

	expectedOrder := []string{
		jsonAs[string](t, task1["id"]),
		jsonAs[string](t, task2["id"]),
		jsonAs[string](t, task3["id"]),
	}

	// Page 1: limit=2, cursor ends mid-priority-group.
	resp := doRequest(t, env, "GET", "/spaces/home/tasks?limit=2", "")
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

	// Page 2: cursor crosses priority boundary (medium → low).
	resp2 := doRequest(t, env, "GET", "/spaces/home/tasks?limit=2&cursor="+jsonAs[string](t, cursor), "")
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
	futureDate := time.Now().AddDate(0, 0, 30).Format(time.DateOnly)
	created := createTask(t, env, "home", `{"title":"Task","due":{"at":"`+futureDate+`","timezone":"UTC"}}`)
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

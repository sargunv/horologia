package api_test

import (
	"net/http"
	"testing"
)

// --- Task Statuses ---

func TestTaskStatusesList(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "st", "Status Test")

	resp := doRequest(t, env, "GET", "/spaces/st/task-statuses", "")
	assertStatus(t, resp, http.StatusOK)
	var body map[string]any
	readJSON(t, resp, &body)
	items, ok := body["items"].([]any)
	if !ok {
		t.Fatal("items field missing or wrong type")
	}
	if len(items) != 2 {
		t.Fatalf("got %d statuses, want 2", len(items))
	}
	// Default statuses: todo (initial), done (completion).
	first := jsonAs[map[string]any](t, items[0])
	if first["name"] != "todo" || first["category"] != "initial" {
		t.Errorf("first status = %v, want todo/initial", first)
	}
	second := jsonAs[map[string]any](t, items[1])
	if second["name"] != "done" || second["category"] != "completion" {
		t.Errorf("second status = %v, want done/completion", second)
	}
}

func TestTaskStatusesReplaceBasic(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "st", "Status Test")

	resp := doRequest(t, env, "PUT", "/spaces/st/task-statuses", `{
		"items": [
			{"name": "backlog", "category": "initial"},
			{"name": "in-progress", "category": "intermediate"},
			{"name": "review", "category": "intermediate"},
			{"name": "done", "category": "completion"}
		]
	}`)
	assertStatus(t, resp, http.StatusOK)
	var body map[string]any
	readJSON(t, resp, &body)
	items := jsonAs[[]any](t, body["items"])
	if len(items) != 4 {
		t.Fatalf("got %d statuses, want 4", len(items))
	}
	// Verify ordering by position.
	names := make([]string, len(items))
	for i, item := range items {
		names[i] = jsonAs[string](t, jsonAs[map[string]any](t, item)["name"])
	}
	if names[0] != "backlog" || names[1] != "in-progress" || names[2] != "review" || names[3] != "done" {
		t.Errorf("statuses = %v, want [backlog in-progress review done]", names)
	}
}

func TestTaskStatusesReplaceRejectOutOfOrderCategories(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "st", "Status Test")

	resp := doRequest(t, env, "PUT", "/spaces/st/task-statuses", `{
		"items": [
			{"name": "done", "category": "completion"},
			{"name": "todo", "category": "initial"}
		]
	}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTaskStatusesReplaceRejectIntermediateAfterCompletion(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "st", "Status Test")

	resp := doRequest(t, env, "PUT", "/spaces/st/task-statuses", `{
		"items": [
			{"name": "todo", "category": "initial"},
			{"name": "done", "category": "completion"},
			{"name": "review", "category": "intermediate"}
		]
	}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTaskStatusesReplaceRejectRemoveWithTasks(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "st", "Status Test")

	// Create a task (defaults to "todo" status).
	createTask(t, env, "st", `{"title":"Blocker"}`)

	// Try to remove "todo" — should fail.
	resp := doRequest(t, env, "PUT", "/spaces/st/task-statuses", `{
		"items": [
			{"name": "done", "category": "completion"},
			{"name": "new-initial", "category": "initial"}
		]
	}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTaskStatusesReplaceAllowRemoveUnusedStatus(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "st", "Status Test")

	// "done" has no tasks, so removing it should succeed (as long as we add another completion).
	resp := doRequest(t, env, "PUT", "/spaces/st/task-statuses", `{
		"items": [
			{"name": "todo", "category": "initial"},
			{"name": "finished", "category": "completion"}
		]
	}`)
	assertStatus(t, resp, http.StatusOK)
	var body map[string]any
	readJSON(t, resp, &body)
	items := jsonAs[[]any](t, body["items"])
	if len(items) != 2 {
		t.Fatalf("got %d statuses, want 2", len(items))
	}
}

func TestTaskStatusesReplaceRejectNoInitial(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "st", "Status Test")

	resp := doRequest(t, env, "PUT", "/spaces/st/task-statuses", `{
		"items": [
			{"name": "done", "category": "completion"}
		]
	}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTaskStatusesReplaceRejectMultipleInitial(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "st", "Status Test")

	resp := doRequest(t, env, "PUT", "/spaces/st/task-statuses", `{
		"items": [
			{"name": "todo", "category": "initial"},
			{"name": "new", "category": "initial"},
			{"name": "done", "category": "completion"}
		]
	}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTaskStatusesReplaceRejectNoCompletion(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "st", "Status Test")

	resp := doRequest(t, env, "PUT", "/spaces/st/task-statuses", `{
		"items": [
			{"name": "todo", "category": "initial"}
		]
	}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTaskStatusesReplaceRejectDuplicateNames(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "st", "Status Test")

	resp := doRequest(t, env, "PUT", "/spaces/st/task-statuses", `{
		"items": [
			{"name": "todo", "category": "initial"},
			{"name": "todo", "category": "completion"}
		]
	}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTaskStatusesReplaceNonAdminRejected(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "st", "Status Test")

	outsiderToken := createTestUser(t, env, "outsider@example.com", "Outsider", "pass1234")
	resp := doRequestAs(t, env, outsiderToken, "PUT", "/spaces/st/task-statuses", `{
		"items": [
			{"name": "todo", "category": "initial"},
			{"name": "done", "category": "completion"}
		]
	}`)
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestTaskStatusesReplaceUpdateCategory(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "st", "Status Test")

	// Change "todo" from initial to intermediate, add a new initial.
	resp := doRequest(t, env, "PUT", "/spaces/st/task-statuses", `{
		"items": [
			{"name": "backlog", "category": "initial"},
			{"name": "todo", "category": "intermediate"},
			{"name": "done", "category": "completion"}
		]
	}`)
	assertStatus(t, resp, http.StatusOK)
	var body map[string]any
	readJSON(t, resp, &body)
	items := jsonAs[[]any](t, body["items"])
	for _, item := range items {
		m := jsonAs[map[string]any](t, item)
		if m["name"] == "todo" {
			if m["category"] != "intermediate" {
				t.Errorf("category for todo = %v, want intermediate", m["category"])
			}
		}
	}
}

func TestTaskStatusesCrossSpaceIsolation(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "iso-a", "Iso A")
	createSpace(t, env, "iso-b", "Iso B")

	// Replace space A's statuses.
	resp := doRequest(t, env, "PUT", "/spaces/iso-a/task-statuses", `{
		"items": [
			{"name": "custom-initial", "category": "initial"},
			{"name": "custom-done", "category": "completion"}
		]
	}`)
	assertStatus(t, resp, http.StatusOK)

	// Space B should still have its original defaults.
	resp2 := doRequest(t, env, "GET", "/spaces/iso-b/task-statuses", "")
	assertStatus(t, resp2, http.StatusOK)
	var bodyB map[string]any
	readJSON(t, resp2, &bodyB)
	items := jsonAs[[]any](t, bodyB["items"])
	if len(items) != 2 {
		t.Fatalf("space B: got %d statuses, want 2", len(items))
	}
	first := jsonAs[map[string]any](t, items[0])
	if first["name"] != "todo" {
		t.Errorf("space B first status = %v, want todo", first["name"])
	}
}

func TestTaskStatusesListNonMemberRejected(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "st-acl", "Status ACL")

	outsiderToken := createTestUser(t, env, "outsider@example.com", "Outsider", "pass1234")
	assertStatusClose(t, doRequestAs(t, env, outsiderToken, "GET", "/spaces/st-acl/task-statuses", ""), http.StatusNotFound)
}

// --- Task Effort Levels ---

func TestTaskEffortLevelsList(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "eff", "Effort Test")

	resp := doRequest(t, env, "GET", "/spaces/eff/task-effort-levels", "")
	assertStatus(t, resp, http.StatusOK)
	var body map[string]any
	readJSON(t, resp, &body)
	items, ok := body["items"].([]any)
	if !ok {
		t.Fatal("items field missing or wrong type")
	}
	if len(items) != 3 {
		t.Fatalf("got %d effort levels, want 3", len(items))
	}
	// Verify ordering by position.
	names := make([]string, len(items))
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("item %d: wrong type %T", i, item)
		}
		name, ok := m["name"].(string)
		if !ok {
			t.Fatalf("item %d: name missing or wrong type", i)
		}
		names[i] = name
	}
	if names[0] != "small" || names[1] != "medium" || names[2] != "large" {
		t.Errorf("effort levels = %v, want [small medium large]", names)
	}
}

func TestTaskEffortLevelsReplaceBasic(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "eff", "Effort Test")

	resp := doRequest(t, env, "PUT", "/spaces/eff/task-effort-levels", `{
		"items": [{"name": "xs"}, {"name": "s"}, {"name": "m"}, {"name": "l"}, {"name": "xl"}]
	}`)
	assertStatus(t, resp, http.StatusOK)
	var body map[string]any
	readJSON(t, resp, &body)
	items := jsonAs[[]any](t, body["items"])
	if len(items) != 5 {
		t.Fatalf("got %d effort levels, want 5", len(items))
	}
	names := make([]string, len(items))
	for i, item := range items {
		names[i] = jsonAs[string](t, jsonAs[map[string]any](t, item)["name"])
	}
	if names[0] != "xs" || names[4] != "xl" {
		t.Errorf("effort levels = %v", names)
	}
}

func TestTaskEffortLevelsReplaceEmpty(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "eff", "Effort Test")

	resp := doRequest(t, env, "PUT", "/spaces/eff/task-effort-levels", `{"items": []}`)
	assertStatus(t, resp, http.StatusOK)
	var body map[string]any
	readJSON(t, resp, &body)
	items := jsonAs[[]any](t, body["items"])
	if len(items) != 0 {
		t.Fatalf("got %d effort levels, want 0", len(items))
	}
}

func TestTaskEffortLevelsReplaceNullsTasksOnRemoval(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "eff", "Effort Test")

	task := createTask(t, env, "eff", `{"title":"Task","effort":"medium"}`)
	taskID := jsonAs[string](t, task["id"])

	// Remove "medium" from the list.
	resp := doRequest(t, env, "PUT", "/spaces/eff/task-effort-levels", `{
		"items": [{"name": "small"}, {"name": "large"}]
	}`)
	assertStatus(t, resp, http.StatusOK)

	// Task's effort should now be null.
	taskResp := doRequest(t, env, "GET", "/spaces/eff/tasks/"+taskID, "")
	assertStatus(t, taskResp, http.StatusOK)
	var taskBody map[string]any
	readJSON(t, taskResp, &taskBody)
	if taskBody["effort"] != nil {
		t.Errorf("effort = %v, want nil after level removal", taskBody["effort"])
	}
}

func TestTaskEffortLevelsReplaceRejectDuplicateNames(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "eff", "Effort Test")

	resp := doRequest(t, env, "PUT", "/spaces/eff/task-effort-levels", `{
		"items": [{"name": "small"}, {"name": "small"}]
	}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTaskEffortLevelsReplaceNonAdminRejected(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "eff", "Effort Test")

	outsiderToken := createTestUser(t, env, "outsider@example.com", "Outsider", "pass1234")
	resp := doRequestAs(t, env, outsiderToken, "PUT", "/spaces/eff/task-effort-levels", `{
		"items": [{"name": "small"}]
	}`)
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestTaskEffortLevelsCrossSpaceIsolation(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "iso-a", "Iso A")
	createSpace(t, env, "iso-b", "Iso B")

	// Replace space A's levels.
	resp := doRequest(t, env, "PUT", "/spaces/iso-a/task-effort-levels", `{
		"items": [{"name": "tiny"}]
	}`)
	assertStatus(t, resp, http.StatusOK)

	// Space B should still have defaults.
	resp2 := doRequest(t, env, "GET", "/spaces/iso-b/task-effort-levels", "")
	assertStatus(t, resp2, http.StatusOK)
	var bodyB map[string]any
	readJSON(t, resp2, &bodyB)
	items := jsonAs[[]any](t, bodyB["items"])
	if len(items) != 3 {
		t.Fatalf("space B: got %d effort levels, want 3", len(items))
	}
}

// --- Task Priority Levels ---

func TestTaskPriorityLevelsList(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "pri", "Priority Test")

	resp := doRequest(t, env, "GET", "/spaces/pri/task-priority-levels", "")
	assertStatus(t, resp, http.StatusOK)
	var body map[string]any
	readJSON(t, resp, &body)
	items, ok := body["items"].([]any)
	if !ok {
		t.Fatal("items field missing or wrong type")
	}
	if len(items) != 3 {
		t.Fatalf("got %d priority levels, want 3", len(items))
	}
	names := make([]string, len(items))
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("item %d: wrong type %T", i, item)
		}
		name, ok := m["name"].(string)
		if !ok {
			t.Fatalf("item %d: name missing or wrong type", i)
		}
		names[i] = name
	}
	if names[0] != "low" || names[1] != "medium" || names[2] != "high" {
		t.Errorf("priority levels = %v, want [low medium high]", names)
	}
}

func TestTaskPriorityLevelsReplaceBasic(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "pri", "Priority Test")

	resp := doRequest(t, env, "PUT", "/spaces/pri/task-priority-levels", `{
		"items": [{"name": "p0"}, {"name": "p1"}, {"name": "p2"}, {"name": "p3"}]
	}`)
	assertStatus(t, resp, http.StatusOK)
	var body map[string]any
	readJSON(t, resp, &body)
	items := jsonAs[[]any](t, body["items"])
	if len(items) != 4 {
		t.Fatalf("got %d priority levels, want 4", len(items))
	}
}

func TestTaskPriorityLevelsReplaceNullsTasksOnRemoval(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "pri", "Priority Test")

	task := createTask(t, env, "pri", `{"title":"Task","priority":"high"}`)
	taskID := jsonAs[string](t, task["id"])

	// Remove "high" from the list.
	resp := doRequest(t, env, "PUT", "/spaces/pri/task-priority-levels", `{
		"items": [{"name": "low"}, {"name": "medium"}]
	}`)
	assertStatus(t, resp, http.StatusOK)

	// Task's priority should now be null.
	taskResp := doRequest(t, env, "GET", "/spaces/pri/tasks/"+taskID, "")
	assertStatus(t, taskResp, http.StatusOK)
	var taskBody map[string]any
	readJSON(t, taskResp, &taskBody)
	if taskBody["priority"] != nil {
		t.Errorf("priority = %v, want nil after level removal", taskBody["priority"])
	}
}

func TestTaskPriorityLevelsReplaceRejectDuplicateNames(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "pri", "Priority Test")

	resp := doRequest(t, env, "PUT", "/spaces/pri/task-priority-levels", `{
		"items": [{"name": "low"}, {"name": "low"}]
	}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTaskPriorityLevelsReplaceNonAdminRejected(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "pri", "Priority Test")

	outsiderToken := createTestUser(t, env, "outsider@example.com", "Outsider", "pass1234")
	resp := doRequestAs(t, env, outsiderToken, "PUT", "/spaces/pri/task-priority-levels", `{
		"items": [{"name": "low"}]
	}`)
	assertStatusClose(t, resp, http.StatusNotFound)
}

// --- Existing effort/priority tests on tasks ---

func TestTaskCreateWithEffortAndPriority(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	task := createTask(t, env, "home", `{"title":"Task","effort":"medium","priority":"high"}`)
	if task["effort"] != "medium" {
		t.Errorf("effort = %v, want medium", task["effort"])
	}
	if task["priority"] != "high" {
		t.Errorf("priority = %v, want high", task["priority"])
	}
}

func TestTaskCreateEffortAndPriorityNullByDefault(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	task := createTask(t, env, "home", `{"title":"Task"}`)
	if task["effort"] != nil {
		t.Errorf("effort = %v, want nil", task["effort"])
	}
	if task["priority"] != nil {
		t.Errorf("priority = %v, want nil", task["priority"])
	}
}

func TestTaskCreateInvalidEffort(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "POST", "/spaces/home/tasks", `{"title":"Task","effort":"bogus"}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTaskCreateInvalidPriority(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "POST", "/spaces/home/tasks", `{"title":"Task","priority":"bogus"}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTaskUpdateEffort(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	task := createTask(t, env, "home", `{"title":"Task"}`)
	taskID, ok := task["id"].(string)
	if !ok {
		t.Fatal("task id missing or wrong type")
	}

	// Set effort.
	resp := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+taskID, `{"effort":"large"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)
	if updated["effort"] != "large" {
		t.Errorf("effort = %v, want large", updated["effort"])
	}

	// Clear effort.
	resp2 := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+taskID, `{"effort":null}`)
	assertStatus(t, resp2, http.StatusOK)
	var cleared map[string]any
	readJSON(t, resp2, &cleared)
	if cleared["effort"] != nil {
		t.Errorf("effort = %v, want nil", cleared["effort"])
	}
}

func TestTaskUpdatePriority(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	task := createTask(t, env, "home", `{"title":"Task"}`)
	taskID, ok := task["id"].(string)
	if !ok {
		t.Fatal("task id missing or wrong type")
	}

	// Set priority.
	resp := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+taskID, `{"priority":"low"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)
	if updated["priority"] != "low" {
		t.Errorf("priority = %v, want low", updated["priority"])
	}

	// Clear priority.
	resp2 := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+taskID, `{"priority":null}`)
	assertStatus(t, resp2, http.StatusOK)
	var cleared map[string]any
	readJSON(t, resp2, &cleared)
	if cleared["priority"] != nil {
		t.Errorf("priority = %v, want nil", cleared["priority"])
	}
}

func TestTaskUpdateInvalidEffort(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	task := createTask(t, env, "home", `{"title":"Task"}`)
	taskID, ok := task["id"].(string)
	if !ok {
		t.Fatal("task id missing or wrong type")
	}

	resp := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+taskID, `{"effort":"nonexistent"}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTaskEffortPreservedOnUnrelatedUpdate(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	task := createTask(t, env, "home", `{"title":"Task","effort":"small","priority":"high"}`)
	taskID, ok := task["id"].(string)
	if !ok {
		t.Fatal("task id missing or wrong type")
	}

	// Update title only — effort and priority should be preserved.
	resp := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+taskID, `{"title":"Updated"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)
	if updated["title"] != "Updated" {
		t.Errorf("title = %v, want Updated", updated["title"])
	}
	if updated["effort"] != "small" {
		t.Errorf("effort = %v, want small (preserved)", updated["effort"])
	}
	if updated["priority"] != "high" {
		t.Errorf("priority = %v, want high (preserved)", updated["priority"])
	}
}

func TestTaskEffortInListResponse(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	createTask(t, env, "home", `{"title":"With effort","effort":"large"}`)
	createTask(t, env, "home", `{"title":"Without effort"}`)

	resp := doRequest(t, env, "GET", "/spaces/home/tasks", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items, ok := page["items"].([]any)
	if !ok {
		t.Fatal("items field missing or wrong type")
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}

	for _, item := range items {
		task, ok := item.(map[string]any)
		if !ok {
			t.Fatal("task item wrong type")
		}
		title := jsonAs[string](t, task["title"])
		if title == "With effort" {
			if task["effort"] != "large" {
				t.Errorf("task with effort: effort = %v, want large", task["effort"])
			}
		} else {
			if task["effort"] != nil {
				t.Errorf("task without effort: effort = %v, want nil", task["effort"])
			}
		}
	}
}

func TestTaskEffortLevelsNonMemberRejected(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "eff-acl", "Effort ACL")

	outsiderToken := createTestUser(t, env, "outsider@example.com", "Outsider", "pass1234")
	assertStatusClose(t, doRequestAs(t, env, outsiderToken, "GET", "/spaces/eff-acl/task-effort-levels", ""), http.StatusNotFound)
}

func TestTaskPriorityLevelsNonMemberRejected(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "pri-acl", "Priority ACL")

	outsiderToken := createTestUser(t, env, "outsider@example.com", "Outsider", "pass1234")
	assertStatusClose(t, doRequestAs(t, env, outsiderToken, "GET", "/spaces/pri-acl/task-priority-levels", ""), http.StatusNotFound)
}

func TestTaskPriorityLevelsCrossSpaceIsolation(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "piso-a", "Piso A")
	createSpace(t, env, "piso-b", "Piso B")

	// Replace space A's priority levels.
	resp := doRequest(t, env, "PUT", "/spaces/piso-a/task-priority-levels", `{
		"items": [{"name": "critical"}]
	}`)
	assertStatus(t, resp, http.StatusOK)

	// Space B should still have defaults.
	resp2 := doRequest(t, env, "GET", "/spaces/piso-b/task-priority-levels", "")
	assertStatus(t, resp2, http.StatusOK)
	var bodyB map[string]any
	readJSON(t, resp2, &bodyB)
	itemsB, ok := bodyB["items"].([]any)
	if !ok {
		t.Fatal("space B: items field missing or wrong type")
	}
	if len(itemsB) != 3 {
		t.Fatalf("space B: got %d priority levels, want 3", len(itemsB))
	}
}

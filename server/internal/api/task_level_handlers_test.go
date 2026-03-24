package api_test

import (
	"net/http"
	"testing"
)

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

func TestTaskEffortLevelsCrossSpaceIsolation(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "iso-a", "Iso A")
	createSpace(t, env, "iso-b", "Iso B")

	// Both spaces should have their own set of 3 default levels.
	resp := doRequest(t, env, "GET", "/spaces/iso-a/task-effort-levels", "")
	assertStatus(t, resp, http.StatusOK)
	var bodyA map[string]any
	readJSON(t, resp, &bodyA)
	itemsA, ok := bodyA["items"].([]any)
	if !ok {
		t.Fatal("space A: items field missing or wrong type")
	}
	if len(itemsA) != 3 {
		t.Fatalf("space A: got %d effort levels, want 3", len(itemsA))
	}

	resp2 := doRequest(t, env, "GET", "/spaces/iso-b/task-effort-levels", "")
	assertStatus(t, resp2, http.StatusOK)
	var bodyB map[string]any
	readJSON(t, resp2, &bodyB)
	itemsB, ok := bodyB["items"].([]any)
	if !ok {
		t.Fatal("space B: items field missing or wrong type")
	}
	if len(itemsB) != 3 {
		t.Fatalf("space B: got %d effort levels, want 3", len(itemsB))
	}
}

func TestTaskPriorityLevelsCrossSpaceIsolation(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "piso-a", "Piso A")
	createSpace(t, env, "piso-b", "Piso B")

	resp := doRequest(t, env, "GET", "/spaces/piso-a/task-priority-levels", "")
	assertStatus(t, resp, http.StatusOK)
	var bodyA map[string]any
	readJSON(t, resp, &bodyA)
	itemsA, ok := bodyA["items"].([]any)
	if !ok {
		t.Fatal("space A: items field missing or wrong type")
	}
	if len(itemsA) != 3 {
		t.Fatalf("space A: got %d priority levels, want 3", len(itemsA))
	}

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

func TestTaskEffortLevelsNonMemberRejected(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "eff-acl", "Effort ACL")

	outsiderToken := createTestUser(t, env, "outsider@example.com", "Outsider", "pass123")
	assertStatusClose(t, doRequestAs(t, env, outsiderToken, "GET", "/spaces/eff-acl/task-effort-levels", ""), http.StatusNotFound)
}

func TestTaskPriorityLevelsNonMemberRejected(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "pri-acl", "Priority ACL")

	outsiderToken := createTestUser(t, env, "outsider@example.com", "Outsider", "pass123")
	assertStatusClose(t, doRequestAs(t, env, outsiderToken, "GET", "/spaces/pri-acl/task-priority-levels", ""), http.StatusNotFound)
}

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
		title, _ := task["title"].(string)
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

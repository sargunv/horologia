package api_test

import (
	"net/http"
	"testing"
)

func TestTaskStatusHandlers(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	t.Run("list", func(t *testing.T) {
		createSpace(t, env, "st-list", "Status Test")
		resp := doRequest(t, env, "GET", "/spaces/st-list/task-statuses", "")
		assertStatus(t, resp, http.StatusOK)
		var body map[string]any
		readJSON(t, resp, &body)
		items := jsonAs[[]any](t, body["items"])
		if len(items) != 2 {
			t.Fatalf("got %d statuses, want 2", len(items))
		}
		first := jsonAs[map[string]any](t, items[0])
		if first["name"] != "todo" || first["category"] != "initial" {
			t.Errorf("first status = %v, want todo/initial", first)
		}
		second := jsonAs[map[string]any](t, items[1])
		if second["name"] != "done" || second["category"] != "completion" {
			t.Errorf("second status = %v, want done/completion", second)
		}
	})

	t.Run("replace basic", func(t *testing.T) {
		createSpace(t, env, "st-basic", "Status Test")
		resp := doRequest(t, env, "PUT", "/spaces/st-basic/task-statuses", `{"items":[{"name":"backlog","category":"initial"},{"name":"in-progress","category":"intermediate"},{"name":"review","category":"intermediate"},{"name":"done","category":"completion"}]}`)
		assertStatus(t, resp, http.StatusOK)
		var body map[string]any
		readJSON(t, resp, &body)
		items := jsonAs[[]any](t, body["items"])
		if len(items) != 4 {
			t.Fatalf("got %d statuses, want 4", len(items))
		}
		names := make([]string, len(items))
		for i, item := range items {
			names[i] = jsonAs[string](t, jsonAs[map[string]any](t, item)["name"])
		}
		if names[0] != "backlog" || names[1] != "in-progress" || names[2] != "review" || names[3] != "done" {
			t.Errorf("statuses = %v, want [backlog in-progress review done]", names)
		}
	})

	t.Run("reject out of order categories", func(t *testing.T) {
		createSpace(t, env, "st-order", "Status Test")
		resp := doRequest(t, env, "PUT", "/spaces/st-order/task-statuses", `{"items":[{"name":"done","category":"completion"},{"name":"todo","category":"initial"}]}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("reject intermediate after completion", func(t *testing.T) {
		createSpace(t, env, "st-after-complete", "Status Test")
		resp := doRequest(t, env, "PUT", "/spaces/st-after-complete/task-statuses", `{"items":[{"name":"todo","category":"initial"},{"name":"done","category":"completion"},{"name":"review","category":"intermediate"}]}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("reject remove with tasks", func(t *testing.T) {
		createSpace(t, env, "st-remove-used", "Status Test")
		createTask(t, env, "st-remove-used", `{"title":"Blocker"}`)
		resp := doRequest(t, env, "PUT", "/spaces/st-remove-used/task-statuses", `{"items":[{"name":"done","category":"completion"},{"name":"new-initial","category":"initial"}]}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("allow remove unused status", func(t *testing.T) {
		createSpace(t, env, "st-remove-unused", "Status Test")
		resp := doRequest(t, env, "PUT", "/spaces/st-remove-unused/task-statuses", `{"items":[{"name":"todo","category":"initial"},{"name":"finished","category":"completion"}]}`)
		assertStatus(t, resp, http.StatusOK)
		var body map[string]any
		readJSON(t, resp, &body)
		items := jsonAs[[]any](t, body["items"])
		if len(items) != 2 {
			t.Fatalf("got %d statuses, want 2", len(items))
		}
	})

	t.Run("reject no initial", func(t *testing.T) {
		createSpace(t, env, "st-no-initial", "Status Test")
		resp := doRequest(t, env, "PUT", "/spaces/st-no-initial/task-statuses", `{"items":[{"name":"done","category":"completion"}]}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("reject multiple initial", func(t *testing.T) {
		createSpace(t, env, "st-multi-initial", "Status Test")
		resp := doRequest(t, env, "PUT", "/spaces/st-multi-initial/task-statuses", `{"items":[{"name":"todo","category":"initial"},{"name":"new","category":"initial"},{"name":"done","category":"completion"}]}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("reject no completion", func(t *testing.T) {
		createSpace(t, env, "st-no-completion", "Status Test")
		resp := doRequest(t, env, "PUT", "/spaces/st-no-completion/task-statuses", `{"items":[{"name":"todo","category":"initial"}]}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("reject duplicate names", func(t *testing.T) {
		createSpace(t, env, "st-dup-names", "Status Test")
		resp := doRequest(t, env, "PUT", "/spaces/st-dup-names/task-statuses", `{"items":[{"name":"todo","category":"initial"},{"name":"todo","category":"completion"}]}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("replace non admin rejected", func(t *testing.T) {
		createSpace(t, env, "st-non-admin", "Status Test")
		outsiderToken := createTestUser(t, env, "status-outsider@example.com", "Outsider", "pass1234")
		resp := doRequestAs(t, env, outsiderToken, "PUT", "/spaces/st-non-admin/task-statuses", `{"items":[{"name":"todo","category":"initial"},{"name":"done","category":"completion"}]}`)
		assertStatusClose(t, resp, http.StatusNotFound)
	})

	t.Run("replace update category", func(t *testing.T) {
		createSpace(t, env, "st-update-category", "Status Test")
		resp := doRequest(t, env, "PUT", "/spaces/st-update-category/task-statuses", `{"items":[{"name":"backlog","category":"initial"},{"name":"todo","category":"intermediate"},{"name":"done","category":"completion"}]}`)
		assertStatus(t, resp, http.StatusOK)
		var body map[string]any
		readJSON(t, resp, &body)
		items := jsonAs[[]any](t, body["items"])
		for _, item := range items {
			m := jsonAs[map[string]any](t, item)
			if m["name"] == "todo" && m["category"] != "intermediate" {
				t.Errorf("category for todo = %v, want intermediate", m["category"])
			}
		}
	})

	t.Run("cross space isolation", func(t *testing.T) {
		createSpace(t, env, "st-iso-a", "Iso A")
		createSpace(t, env, "st-iso-b", "Iso B")
		resp := doRequest(t, env, "PUT", "/spaces/st-iso-a/task-statuses", `{"items":[{"name":"custom-initial","category":"initial"},{"name":"custom-done","category":"completion"}]}`)
		assertStatusClose(t, resp, http.StatusOK)
		resp2 := doRequest(t, env, "GET", "/spaces/st-iso-b/task-statuses", "")
		assertStatus(t, resp2, http.StatusOK)
		var body map[string]any
		readJSON(t, resp2, &body)
		items := jsonAs[[]any](t, body["items"])
		if len(items) != 2 {
			t.Fatalf("space B: got %d statuses, want 2", len(items))
		}
		first := jsonAs[map[string]any](t, items[0])
		if first["name"] != "todo" {
			t.Errorf("space B first status = %v, want todo", first["name"])
		}
	})

	t.Run("list non member rejected", func(t *testing.T) {
		createSpace(t, env, "st-acl", "Status ACL")
		outsiderToken := createTestUser(t, env, "status-list-outsider@example.com", "Outsider", "pass1234")
		assertStatusClose(t, doRequestAs(t, env, outsiderToken, "GET", "/spaces/st-acl/task-statuses", ""), http.StatusNotFound)
	})
}

func TestTaskEffortLevelHandlers(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	t.Run("list", func(t *testing.T) {
		createSpace(t, env, "eff-list", "Effort Test")
		resp := doRequest(t, env, "GET", "/spaces/eff-list/task-effort-levels", "")
		assertStatus(t, resp, http.StatusOK)
		var body map[string]any
		readJSON(t, resp, &body)
		items := jsonAs[[]any](t, body["items"])
		if len(items) != 3 {
			t.Fatalf("got %d effort levels, want 3", len(items))
		}
		names := make([]string, len(items))
		for i, item := range items {
			names[i] = jsonAs[string](t, jsonAs[map[string]any](t, item)["name"])
		}
		if names[0] != "small" || names[1] != "moderate" || names[2] != "large" {
			t.Errorf("effort levels = %v, want [small moderate large]", names)
		}
	})

	t.Run("replace basic", func(t *testing.T) {
		createSpace(t, env, "eff-basic", "Effort Test")
		resp := doRequest(t, env, "PUT", "/spaces/eff-basic/task-effort-levels", `{"items":[{"name":"xs"},{"name":"s"},{"name":"m"},{"name":"l"},{"name":"xl"}]}`)
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
	})

	t.Run("replace empty", func(t *testing.T) {
		createSpace(t, env, "eff-empty", "Effort Test")
		resp := doRequest(t, env, "PUT", "/spaces/eff-empty/task-effort-levels", `{"items": []}`)
		assertStatus(t, resp, http.StatusOK)
		var body map[string]any
		readJSON(t, resp, &body)
		items := jsonAs[[]any](t, body["items"])
		if len(items) != 0 {
			t.Fatalf("got %d effort levels, want 0", len(items))
		}
	})

	t.Run("replace nulls tasks on removal", func(t *testing.T) {
		createSpace(t, env, "eff-remove", "Effort Test")
		task := createTask(t, env, "eff-remove", `{"title":"Task","effort":"moderate"}`)
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "PUT", "/spaces/eff-remove/task-effort-levels", `{"items":[{"name":"small"},{"name":"large"}]}`)
		assertStatusClose(t, resp, http.StatusOK)
		taskResp := doRequest(t, env, "GET", "/spaces/eff-remove/tasks/"+taskID, "")
		assertStatus(t, taskResp, http.StatusOK)
		var taskBody map[string]any
		readJSON(t, taskResp, &taskBody)
		if taskBody["effort"] != nil {
			t.Errorf("effort = %v, want nil after level removal", taskBody["effort"])
		}
	})

	t.Run("replace reject duplicate names", func(t *testing.T) {
		createSpace(t, env, "eff-dup", "Effort Test")
		resp := doRequest(t, env, "PUT", "/spaces/eff-dup/task-effort-levels", `{"items":[{"name":"small"},{"name":"small"}]}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("replace non admin rejected", func(t *testing.T) {
		createSpace(t, env, "eff-non-admin", "Effort Test")
		outsiderToken := createTestUser(t, env, "eff-outsider@example.com", "Outsider", "pass1234")
		resp := doRequestAs(t, env, outsiderToken, "PUT", "/spaces/eff-non-admin/task-effort-levels", `{"items":[{"name":"small"}]}`)
		assertStatusClose(t, resp, http.StatusNotFound)
	})

	t.Run("cross space isolation", func(t *testing.T) {
		createSpace(t, env, "eff-iso-a", "Iso A")
		createSpace(t, env, "eff-iso-b", "Iso B")
		resp := doRequest(t, env, "PUT", "/spaces/eff-iso-a/task-effort-levels", `{"items":[{"name":"tiny"}]}`)
		assertStatusClose(t, resp, http.StatusOK)
		resp2 := doRequest(t, env, "GET", "/spaces/eff-iso-b/task-effort-levels", "")
		assertStatus(t, resp2, http.StatusOK)
		var body map[string]any
		readJSON(t, resp2, &body)
		items := jsonAs[[]any](t, body["items"])
		if len(items) != 3 {
			t.Fatalf("space B: got %d effort levels, want 3", len(items))
		}
	})

	t.Run("non member rejected", func(t *testing.T) {
		createSpace(t, env, "eff-acl", "Effort ACL")
		outsiderToken := createTestUser(t, env, "eff-list-outsider@example.com", "Outsider", "pass1234")
		assertStatusClose(t, doRequestAs(t, env, outsiderToken, "GET", "/spaces/eff-acl/task-effort-levels", ""), http.StatusNotFound)
	})
}

func TestTaskPriorityLevelHandlers(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	t.Run("list", func(t *testing.T) {
		createSpace(t, env, "pri-list", "Priority Test")
		resp := doRequest(t, env, "GET", "/spaces/pri-list/task-priority-levels", "")
		assertStatus(t, resp, http.StatusOK)
		var body map[string]any
		readJSON(t, resp, &body)
		items := jsonAs[[]any](t, body["items"])
		if len(items) != 3 {
			t.Fatalf("got %d priority levels, want 3", len(items))
		}
		names := make([]string, len(items))
		for i, item := range items {
			names[i] = jsonAs[string](t, jsonAs[map[string]any](t, item)["name"])
		}
		if names[0] != "low" || names[1] != "medium" || names[2] != "high" {
			t.Errorf("priority levels = %v, want [low medium high]", names)
		}
	})

	t.Run("replace basic", func(t *testing.T) {
		createSpace(t, env, "pri-basic", "Priority Test")
		resp := doRequest(t, env, "PUT", "/spaces/pri-basic/task-priority-levels", `{"items":[{"name":"p0"},{"name":"p1"},{"name":"p2"},{"name":"p3"}]}`)
		assertStatus(t, resp, http.StatusOK)
		var body map[string]any
		readJSON(t, resp, &body)
		items := jsonAs[[]any](t, body["items"])
		if len(items) != 4 {
			t.Fatalf("got %d priority levels, want 4", len(items))
		}
	})

	t.Run("replace nulls tasks on removal", func(t *testing.T) {
		createSpace(t, env, "pri-remove", "Priority Test")
		task := createTask(t, env, "pri-remove", `{"title":"Task","priority":"high"}`)
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "PUT", "/spaces/pri-remove/task-priority-levels", `{"items":[{"name":"low"},{"name":"medium"}]}`)
		assertStatusClose(t, resp, http.StatusOK)
		taskResp := doRequest(t, env, "GET", "/spaces/pri-remove/tasks/"+taskID, "")
		assertStatus(t, taskResp, http.StatusOK)
		var taskBody map[string]any
		readJSON(t, taskResp, &taskBody)
		if taskBody["priority"] != nil {
			t.Errorf("priority = %v, want nil after level removal", taskBody["priority"])
		}
	})

	t.Run("replace reject duplicate names", func(t *testing.T) {
		createSpace(t, env, "pri-dup", "Priority Test")
		resp := doRequest(t, env, "PUT", "/spaces/pri-dup/task-priority-levels", `{"items":[{"name":"low"},{"name":"low"}]}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("replace non admin rejected", func(t *testing.T) {
		createSpace(t, env, "pri-non-admin", "Priority Test")
		outsiderToken := createTestUser(t, env, "pri-outsider@example.com", "Outsider", "pass1234")
		resp := doRequestAs(t, env, outsiderToken, "PUT", "/spaces/pri-non-admin/task-priority-levels", `{"items":[{"name":"low"}]}`)
		assertStatusClose(t, resp, http.StatusNotFound)
	})

	t.Run("non member rejected", func(t *testing.T) {
		createSpace(t, env, "pri-acl", "Priority ACL")
		outsiderToken := createTestUser(t, env, "pri-list-outsider@example.com", "Outsider", "pass1234")
		assertStatusClose(t, doRequestAs(t, env, outsiderToken, "GET", "/spaces/pri-acl/task-priority-levels", ""), http.StatusNotFound)
	})

	t.Run("cross space isolation", func(t *testing.T) {
		createSpace(t, env, "pri-iso-a", "Piso A")
		createSpace(t, env, "pri-iso-b", "Piso B")
		resp := doRequest(t, env, "PUT", "/spaces/pri-iso-a/task-priority-levels", `{"items":[{"name":"critical"}]}`)
		assertStatusClose(t, resp, http.StatusOK)
		resp2 := doRequest(t, env, "GET", "/spaces/pri-iso-b/task-priority-levels", "")
		assertStatus(t, resp2, http.StatusOK)
		var body map[string]any
		readJSON(t, resp2, &body)
		items := jsonAs[[]any](t, body["items"])
		if len(items) != 3 {
			t.Fatalf("space B: got %d priority levels, want 3", len(items))
		}
	})
}

func TestTaskEffortPriorityFields(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	t.Run("create with effort and priority", func(t *testing.T) {
		createSpace(t, env, "task-create-values", "Home")
		task := createTask(t, env, "task-create-values", `{"title":"Task","effort":"moderate","priority":"high"}`)
		if task["effort"] != "moderate" {
			t.Errorf("effort = %v, want moderate", task["effort"])
		}
		if task["priority"] != "high" {
			t.Errorf("priority = %v, want high", task["priority"])
		}
	})

	t.Run("create defaults null", func(t *testing.T) {
		createSpace(t, env, "task-create-null", "Home")
		task := createTask(t, env, "task-create-null", `{"title":"Task"}`)
		if task["effort"] != nil {
			t.Errorf("effort = %v, want nil", task["effort"])
		}
		if task["priority"] != nil {
			t.Errorf("priority = %v, want nil", task["priority"])
		}
	})

	t.Run("create invalid effort", func(t *testing.T) {
		createSpace(t, env, "task-invalid-effort", "Home")
		resp := doRequest(t, env, "POST", "/spaces/task-invalid-effort/tasks", `{"title":"Task","effort":"bogus"}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("create invalid priority", func(t *testing.T) {
		createSpace(t, env, "task-invalid-priority", "Home")
		resp := doRequest(t, env, "POST", "/spaces/task-invalid-priority/tasks", `{"title":"Task","priority":"bogus"}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("update effort", func(t *testing.T) {
		createSpace(t, env, "task-update-effort", "Home")
		task := createTask(t, env, "task-update-effort", `{"title":"Task"}`)
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "PATCH", "/spaces/task-update-effort/tasks/"+taskID, `{"effort":"large"}`)
		assertStatus(t, resp, http.StatusOK)
		var updated map[string]any
		readJSON(t, resp, &updated)
		if updated["effort"] != "large" {
			t.Errorf("effort = %v, want large", updated["effort"])
		}
		resp2 := doRequest(t, env, "PATCH", "/spaces/task-update-effort/tasks/"+taskID, `{"effort":null}`)
		assertStatus(t, resp2, http.StatusOK)
		var cleared map[string]any
		readJSON(t, resp2, &cleared)
		if cleared["effort"] != nil {
			t.Errorf("effort = %v, want nil", cleared["effort"])
		}
	})

	t.Run("update priority", func(t *testing.T) {
		createSpace(t, env, "task-update-priority", "Home")
		task := createTask(t, env, "task-update-priority", `{"title":"Task"}`)
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "PATCH", "/spaces/task-update-priority/tasks/"+taskID, `{"priority":"low"}`)
		assertStatus(t, resp, http.StatusOK)
		var updated map[string]any
		readJSON(t, resp, &updated)
		if updated["priority"] != "low" {
			t.Errorf("priority = %v, want low", updated["priority"])
		}
		resp2 := doRequest(t, env, "PATCH", "/spaces/task-update-priority/tasks/"+taskID, `{"priority":null}`)
		assertStatus(t, resp2, http.StatusOK)
		var cleared map[string]any
		readJSON(t, resp2, &cleared)
		if cleared["priority"] != nil {
			t.Errorf("priority = %v, want nil", cleared["priority"])
		}
	})

	t.Run("update invalid effort", func(t *testing.T) {
		createSpace(t, env, "task-update-invalid-effort", "Home")
		task := createTask(t, env, "task-update-invalid-effort", `{"title":"Task"}`)
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "PATCH", "/spaces/task-update-invalid-effort/tasks/"+taskID, `{"effort":"nonexistent"}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("effort preserved on unrelated update", func(t *testing.T) {
		createSpace(t, env, "task-preserve-values", "Home")
		task := createTask(t, env, "task-preserve-values", `{"title":"Task","effort":"small","priority":"high"}`)
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "PATCH", "/spaces/task-preserve-values/tasks/"+taskID, `{"title":"Updated"}`)
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
	})

	t.Run("effort in list response", func(t *testing.T) {
		createSpace(t, env, "task-list-effort", "Home")
		createTask(t, env, "task-list-effort", `{"title":"With effort","effort":"large"}`)
		createTask(t, env, "task-list-effort", `{"title":"Without effort"}`)
		resp := doRequest(t, env, "GET", "/spaces/task-list-effort/tasks", "")
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
			if title == "With effort" {
				if task["effort"] != "large" {
					t.Errorf("task with effort: effort = %v, want large", task["effort"])
				}
			} else if task["effort"] != nil {
				t.Errorf("task without effort: effort = %v, want nil", task["effort"])
			}
		}
	})
}

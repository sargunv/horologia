package api_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestTaskCRUDAndSearch(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	t.Run("create", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		resp := doRequest(t, env, "POST", "/spaces/"+space+"/tasks", `{"title":"Wash dishes"}`)
		assertStatus(t, resp, http.StatusCreated)
		var task map[string]any
		readJSON(t, resp, &task)
		if task["title"] != "Wash dishes" || task["status"] != "todo" {
			t.Fatalf("unexpected task body: %v", task)
		}
		if !strings.HasPrefix(jsonAs[string](t, task["id"]), "T") || task["due"] != nil || task["createdAt"] == nil {
			t.Fatalf("unexpected task metadata: %v", task)
		}
	})

	t.Run("create with fields", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		futureDate := time.Now().AddDate(0, 0, 30).Format(time.DateOnly)
		body := `{"title":"Clean","description":"Deep clean","status":"done","due":{"at":"` + futureDate + `","timezone":"UTC"}}`
		resp := doRequest(t, env, "POST", "/spaces/"+space+"/tasks", body)
		assertStatus(t, resp, http.StatusCreated)
		var task map[string]any
		readJSON(t, resp, &task)
		if task["description"] != "Deep clean" || task["status"] != "done" {
			t.Fatalf("unexpected task body: %v", task)
		}
		due := jsonAs[map[string]any](t, task["due"])
		if due["timezone"] != "UTC" {
			t.Fatalf("unexpected due: %v", due)
		}
	})

	t.Run("create in nonexistent space", func(t *testing.T) {
		resp := doRequest(t, env, "POST", "/spaces/nonexistent/tasks", `{"title":"Task"}`)
		assertStatusClose(t, resp, http.StatusNotFound)
	})

	t.Run("create invalid status", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		resp := doRequest(t, env, "POST", "/spaces/"+space+"/tasks", `{"title":"Task","status":"bogus"}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("read", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		created := createTask(t, env, space, `{"title":"Mop floors"}`)
		id := jsonAs[string](t, created["id"])
		resp := doRequest(t, env, "GET", "/spaces/"+space+"/tasks/"+id, "")
		assertStatus(t, resp, http.StatusOK)
		var task map[string]any
		readJSON(t, resp, &task)
		if task["title"] != "Mop floors" {
			t.Fatalf("title = %v, want Mop floors", task["title"])
		}
	})

	t.Run("read not found", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		assertStatusClose(t, doRequest(t, env, "GET", "/spaces/"+space+"/tasks/T999", ""), http.StatusNotFound)
	})

	t.Run("read invalid id", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		assertStatusClose(t, doRequest(t, env, "GET", "/spaces/"+space+"/tasks/invalid", ""), http.StatusBadRequest)
	})

	t.Run("list empty", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		resp := doRequest(t, env, "GET", "/spaces/"+space+"/tasks", "")
		assertStatus(t, resp, http.StatusOK)
		var page map[string]any
		readJSON(t, resp, &page)
		if len(jsonAs[[]any](t, page["items"])) != 0 || page["nextCursor"] != nil {
			t.Fatalf("unexpected page: %v", page)
		}
	})

	t.Run("list pagination", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		var createdIDs []string
		for range 4 {
			task := createTask(t, env, space, `{"title":"Task"}`)
			createdIDs = append(createdIDs, jsonAs[string](t, task["id"]))
		}
		resp := doRequest(t, env, "GET", "/spaces/"+space+"/tasks?limit=2", "")
		assertStatus(t, resp, http.StatusOK)
		var page1 map[string]any
		readJSON(t, resp, &page1)
		items1 := jsonAs[[]any](t, page1["items"])
		if len(items1) != 2 || jsonAs[map[string]any](t, items1[0])["id"] != createdIDs[0] || jsonAs[map[string]any](t, items1[1])["id"] != createdIDs[1] {
			t.Fatalf("unexpected page 1: %v", page1)
		}
		cursor := jsonAs[string](t, page1["nextCursor"])
		resp2 := doRequest(t, env, "GET", "/spaces/"+space+"/tasks?limit=2&cursor="+cursor, "")
		assertStatus(t, resp2, http.StatusOK)
		var page2 map[string]any
		readJSON(t, resp2, &page2)
		items2 := jsonAs[[]any](t, page2["items"])
		if len(items2) != 2 || jsonAs[map[string]any](t, items2[0])["id"] != createdIDs[2] || jsonAs[map[string]any](t, items2[1])["id"] != createdIDs[3] || page2["nextCursor"] != nil {
			t.Fatalf("unexpected page 2: %v", page2)
		}
	})

	t.Run("list sort order", func(t *testing.T) {
		space := testSlug(t, "proj")
		createSpace(t, env, space, "Project")
		assertStatusClose(t, doRequest(t, env, "PUT", "/spaces/"+space+"/task-statuses", `{"items":[{"name":"backlog","category":"initial"},{"name":"in_progress","category":"intermediate"},{"name":"done","category":"completion"}]}`), http.StatusOK)
		assertStatusClose(t, doRequest(t, env, "PUT", "/spaces/"+space+"/task-priority-levels", `{"items":[{"name":"urgent"},{"name":"normal"},{"name":"low"}]}`), http.StatusOK)
		assertStatusClose(t, doRequest(t, env, "PUT", "/spaces/"+space+"/task-effort-levels", `{"items":[{"name":"small"},{"name":"medium"},{"name":"large"}]}`), http.StatusOK)
		futureDate := time.Now().AddDate(0, 0, 30).Format(time.DateOnly)
		pastDate := time.Now().AddDate(0, 0, -5).Format(time.DateOnly)
		taskA := createTask(t, env, space, `{"title":"A-backlog-nodue"}`)
		taskB := createTask(t, env, space, `{"title":"B","status":"in_progress","due":{"at":"`+pastDate+`","timezone":"UTC"},"priority":"urgent","effort":"small"}`)
		taskC := createTask(t, env, space, `{"title":"C","status":"in_progress","due":{"at":"`+pastDate+`","timezone":"UTC"},"priority":"urgent","effort":"large"}`)
		taskD := createTask(t, env, space, `{"title":"D","status":"in_progress","due":{"at":"`+futureDate+`","timezone":"UTC"},"priority":"low"}`)
		taskE := createTask(t, env, space, `{"title":"E","due":{"at":"`+pastDate+`","timezone":"UTC"},"priority":"normal"}`)
		taskF := createTask(t, env, space, `{"title":"F"}`)
		taskG := createTask(t, env, space, `{"title":"G","status":"done"}`)
		expected := []string{jsonAs[string](t, taskB["id"]), jsonAs[string](t, taskC["id"]), jsonAs[string](t, taskD["id"]), jsonAs[string](t, taskE["id"]), jsonAs[string](t, taskA["id"]), jsonAs[string](t, taskF["id"]), jsonAs[string](t, taskG["id"])}
		resp := doRequest(t, env, "GET", "/spaces/"+space+"/tasks?limit=10", "")
		assertStatus(t, resp, http.StatusOK)
		var page map[string]any
		readJSON(t, resp, &page)
		items := jsonAs[[]any](t, page["items"])
		for i, item := range items {
			if got := jsonAs[string](t, jsonAs[map[string]any](t, item)["id"]); got != expected[i] {
				t.Fatalf("position %d: got %s want %s", i, got, expected[i])
			}
		}
	})

	t.Run("search owner across spaces", func(t *testing.T) {
		home := testSlug(t, "home")
		work := testSlug(t, "work")
		createSpace(t, env, home, "Home")
		createSpace(t, env, work, "Work")
		homeTask := createTask(t, env, home, `{"title":"Buy oat milk"}`)
		workTask := createTask(t, env, work, `{"title":"Buy office snacks"}`)
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
		if got[jsonAs[string](t, homeTask["id"])] != home || got[jsonAs[string](t, workTask["id"])] != work {
			t.Fatalf("unexpected search map: %v", got)
		}
	})

	t.Run("search filters to visible spaces", func(t *testing.T) {
		alpha := testSlug(t, "alpha")
		beta := testSlug(t, "beta")
		createSpace(t, env, alpha, "Alpha")
		createSpace(t, env, beta, "Beta")
		alphaTask := createTask(t, env, alpha, `{"title":"Shared title"}`)
		createTask(t, env, beta, `{"title":"Shared title"}`)
		userToken, _ := createAndAddMember(t, env, alpha, testEmail(t, "viewer"), "Viewer", "pass1234", "viewer")
		resp := doRequestAs(t, env, userToken, "GET", "/tasks/search?q=shared", "")
		assertStatus(t, resp, http.StatusOK)
		var result map[string]any
		readJSON(t, resp, &result)
		items := jsonAs[[]any](t, result["items"])
		if len(items) != 1 {
			t.Fatalf("got %d items, want 1", len(items))
		}
		row := jsonAs[map[string]any](t, items[0])
		if row["id"] != alphaTask["id"] || row["spaceSlug"] != alpha {
			t.Fatalf("unexpected filtered result: %v", row)
		}
	})

	t.Run("search optional space filter", func(t *testing.T) {
		alpha := testSlug(t, "alpha")
		beta := testSlug(t, "beta")
		createSpace(t, env, alpha, "Alpha")
		createSpace(t, env, beta, "Beta")
		alphaTask := createTask(t, env, alpha, `{"title":"Release notes"}`)
		createTask(t, env, beta, `{"title":"Release notes"}`)
		resp := doRequest(t, env, "GET", "/tasks/search?q=release&spaceSlug="+alpha, "")
		assertStatus(t, resp, http.StatusOK)
		var result map[string]any
		readJSON(t, resp, &result)
		items := jsonAs[[]any](t, result["items"])
		row := jsonAs[map[string]any](t, items[0])
		if len(items) != 1 || row["id"] != alphaTask["id"] || row["spaceSlug"] != alpha {
			t.Fatalf("unexpected filtered result: %v", result)
		}
	})

	t.Run("search exclude task id and exact id", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		first := createTask(t, env, space, `{"title":"Plan sprint"}`)
		second := createTask(t, env, space, `{"title":"Plan retrospective"}`)
		firstID := jsonAs[string](t, first["id"])
		resp := doRequest(t, env, "GET", "/tasks/search?q=plan&excludeTaskId="+firstID, "")
		assertStatus(t, resp, http.StatusOK)
		var result map[string]any
		readJSON(t, resp, &result)
		items := jsonAs[[]any](t, result["items"])
		if len(items) != 1 || jsonAs[map[string]any](t, items[0])["id"] != second["id"] {
			t.Fatalf("unexpected exclusion result: %v", result)
		}
		resp = doRequest(t, env, "GET", "/tasks/search?q="+firstID, "")
		assertStatus(t, resp, http.StatusOK)
		readJSON(t, resp, &result)
		items = jsonAs[[]any](t, result["items"])
		if len(items) != 1 || jsonAs[map[string]any](t, items[0])["id"] != firstID {
			t.Fatalf("unexpected exact-id result: %v", result)
		}
	})

	t.Run("search leading t still does text search", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		task := createTask(t, env, space, `{"title":"Task planning unique"}`)
		resp := doRequest(t, env, "GET", "/tasks/search?q=planning%20unique", "")
		assertStatus(t, resp, http.StatusOK)
		var result map[string]any
		readJSON(t, resp, &result)
		items := jsonAs[[]any](t, result["items"])
		if len(items) != 1 || jsonAs[map[string]any](t, items[0])["id"] != task["id"] {
			t.Fatalf("unexpected search result: %v", result)
		}
	})

	t.Run("list sort pagination", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		assertStatusClose(t, doRequest(t, env, "PUT", "/spaces/"+space+"/task-priority-levels", `{"items":[{"name":"high"},{"name":"medium"},{"name":"low"}]}`), http.StatusOK)
		dueDate := time.Now().AddDate(0, 0, 1).Format(time.DateOnly)
		task1 := createTask(t, env, space, `{"title":"T1","due":{"at":"`+dueDate+`","timezone":"UTC"},"priority":"high"}`)
		task2 := createTask(t, env, space, `{"title":"T2","due":{"at":"`+dueDate+`","timezone":"UTC"},"priority":"medium"}`)
		task3 := createTask(t, env, space, `{"title":"T3","due":{"at":"`+dueDate+`","timezone":"UTC"},"priority":"low"}`)
		expected := []string{jsonAs[string](t, task1["id"]), jsonAs[string](t, task2["id"]), jsonAs[string](t, task3["id"])}
		resp := doRequest(t, env, "GET", "/spaces/"+space+"/tasks?limit=2", "")
		assertStatus(t, resp, http.StatusOK)
		var page1 map[string]any
		readJSON(t, resp, &page1)
		items1 := jsonAs[[]any](t, page1["items"])
		if len(items1) != 2 || jsonAs[string](t, jsonAs[map[string]any](t, items1[0])["id"]) != expected[0] || jsonAs[string](t, jsonAs[map[string]any](t, items1[1])["id"]) != expected[1] {
			t.Fatalf("unexpected page 1: %v", page1)
		}
		cursor := jsonAs[string](t, page1["nextCursor"])
		resp2 := doRequest(t, env, "GET", "/spaces/"+space+"/tasks?limit=2&cursor="+cursor, "")
		assertStatus(t, resp2, http.StatusOK)
		var page2 map[string]any
		readJSON(t, resp2, &page2)
		items2 := jsonAs[[]any](t, page2["items"])
		if len(items2) != 1 || jsonAs[string](t, jsonAs[map[string]any](t, items2[0])["id"]) != expected[2] || page2["nextCursor"] != nil {
			t.Fatalf("unexpected page 2: %v", page2)
		}
	})

	t.Run("list nonexistent space", func(t *testing.T) {
		assertStatusClose(t, doRequest(t, env, "GET", "/spaces/nonexistent/tasks", ""), http.StatusNotFound)
	})

	t.Run("update", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		created := createTask(t, env, space, `{"title":"Old title","description":"Keep me"}`)
		id := jsonAs[string](t, created["id"])
		resp := doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+id, `{"title":"New title"}`)
		assertStatus(t, resp, http.StatusOK)
		var updated map[string]any
		readJSON(t, resp, &updated)
		if updated["title"] != "New title" || updated["description"] != "Keep me" {
			t.Fatalf("unexpected updated task: %v", updated)
		}
	})

	t.Run("update status", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		created := createTask(t, env, space, `{"title":"Chore"}`)
		id := jsonAs[string](t, created["id"])
		resp := doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+id, `{"status":"done"}`)
		assertStatus(t, resp, http.StatusOK)
		var updated map[string]any
		readJSON(t, resp, &updated)
		if updated["status"] != "done" {
			t.Fatalf("unexpected updated task: %v", updated)
		}
	})

	t.Run("update invalid status", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		created := createTask(t, env, space, `{"title":"Chore"}`)
		id := jsonAs[string](t, created["id"])
		assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+id, `{"status":"nonexistent"}`), http.StatusBadRequest)
	})

	t.Run("update clear due", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		futureDate := time.Now().AddDate(0, 0, 30).Format(time.DateOnly)
		created := createTask(t, env, space, `{"title":"Task","due":{"at":"`+futureDate+`","timezone":"UTC"}}`)
		id := jsonAs[string](t, created["id"])
		resp := doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+id, `{"due":null}`)
		assertStatus(t, resp, http.StatusOK)
		var updated map[string]any
		readJSON(t, resp, &updated)
		if updated["due"] != nil {
			t.Fatalf("dueAt = %v, want nil", updated["due"])
		}
	})

	t.Run("delete", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		created := createTask(t, env, space, `{"title":"Temp"}`)
		id := jsonAs[string](t, created["id"])
		assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/"+space+"/tasks/"+id, ""), http.StatusNoContent)
		assertStatusClose(t, doRequest(t, env, "GET", "/spaces/"+space+"/tasks/"+id, ""), http.StatusNotFound)
	})

	t.Run("space delete cascades tasks", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		created := createTask(t, env, space, `{"title":"Chore"}`)
		id := jsonAs[string](t, created["id"])
		assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/"+space, ""), http.StatusNoContent)
		assertStatusClose(t, doRequest(t, env, "GET", "/spaces/"+space+"/tasks/"+id, ""), http.StatusNotFound)
	})

	t.Run("cross space read update delete", func(t *testing.T) {
		spaceA := testSlug(t, "space-a")
		spaceB := testSlug(t, "space-b")
		createSpace(t, env, spaceA, "Space A")
		createSpace(t, env, spaceB, "Space B")
		task := createTask(t, env, spaceB, `{"title":"Secret"}`)
		id := jsonAs[string](t, task["id"])
		assertStatusClose(t, doRequest(t, env, "GET", "/spaces/"+spaceA+"/tasks/"+id, ""), http.StatusNotFound)
		assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/"+spaceA+"/tasks/"+id, `{"title":"Hacked"}`), http.StatusNotFound)
		assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/"+spaceA+"/tasks/"+id, ""), http.StatusNotFound)
		assertStatusClose(t, doRequest(t, env, "GET", "/spaces/"+spaceB+"/tasks/"+id, ""), http.StatusOK)
	})
}

func TestConcurrentTaskPatchPreservesDisjointFields(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	space := testSlug(t, "concurrent")
	createSpace(t, env, space, "Concurrent")
	created := createTask(t, env, space, `{"title":"Old title","description":"Old description"}`)
	id := jsonAs[string](t, created["id"])
	dbID := mustParseTaskID(t, id)

	tx, err := env.pool.Begin(t.Context())
	if err != nil {
		t.Fatalf("begin lock transaction: %v", err)
	}
	defer func() { _ = tx.Rollback(t.Context()) }()
	if _, err := tx.Exec(t.Context(), `SELECT id FROM tasks WHERE id = $1 AND space_slug = $2 FOR UPDATE`, dbID, space); err != nil {
		t.Fatalf("lock task row: %v", err)
	}

	type patchResult struct {
		status int
		body   string
		err    error
	}
	patch := func(body string) patchResult {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPatch, env.Server.URL+"/spaces/"+space+"/tasks/"+id, strings.NewReader(body))
		if err != nil {
			return patchResult{err: err}
		}
		req.Header.Set("Authorization", "Bearer "+env.Token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return patchResult{err: err}
		}
		defer func() { _ = resp.Body.Close() }()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return patchResult{err: err}
		}
		return patchResult{status: resp.StatusCode, body: string(data)}
	}

	patchDone := make(chan patchResult, 1)
	go func() {
		patchDone <- patch(`{"description":"Concurrent description"}`)
	}()
	time.Sleep(100 * time.Millisecond)

	if _, err := tx.Exec(t.Context(), `UPDATE tasks SET title = $1 WHERE id = $2 AND space_slug = $3`, "Concurrent title", dbID, space); err != nil {
		t.Fatalf("update title while patch is waiting: %v", err)
	}
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatalf("commit lock transaction: %v", err)
	}

	patched := <-patchDone
	if patched.err != nil {
		t.Fatalf("patch: %v", patched.err)
	}
	if patched.status != http.StatusOK {
		t.Fatalf("patch status = %d, want %d; body: %s", patched.status, http.StatusOK, patched.body)
	}

	resp := doRequest(t, env, "GET", "/spaces/"+space+"/tasks/"+id, "")
	assertStatus(t, resp, http.StatusOK)
	var fetched map[string]any
	readJSON(t, resp, &fetched)
	if fetched["title"] != "Concurrent title" || fetched["description"] != "Concurrent description" {
		t.Fatalf("concurrent patches were not preserved: %v", fetched)
	}
}

func TestTaskAssigneeAndTagFields(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	t.Run("assignees on create", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		ownerID := getUserID(t, env, env.Token)
		task := createTask(t, env, space, `{"title":"With assignees","assigneeIds":["`+ownerID+`"]}`)
		assigneeIDs := jsonAs[[]any](t, task["assigneeIds"])
		if len(assigneeIDs) != 1 || jsonAs[string](t, assigneeIDs[0]) != ownerID {
			t.Fatalf("unexpected assignees: %v", assigneeIDs)
		}
	})

	t.Run("assignees empty by default", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		task := createTask(t, env, space, `{"title":"No assignees"}`)
		if len(jsonAs[[]any](t, task["assigneeIds"])) != 0 {
			t.Fatalf("unexpected assignees: %v", task["assigneeIds"])
		}
	})

	t.Run("assignees update", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		ownerID := getUserID(t, env, env.Token)
		_, bobID := createAndAddMember(t, env, space, testEmail(t, "bob"), "Bob", "pass1234", "member")
		task := createTask(t, env, space, `{"title":"Test"}`)
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+taskID, `{"assigneeIds":["`+ownerID+`","`+bobID+`"]}`)
		assertStatus(t, resp, http.StatusOK)
		var updated map[string]any
		readJSON(t, resp, &updated)
		assigneeIDs := jsonAs[[]any](t, updated["assigneeIds"])
		ids := map[string]bool{}
		for _, id := range assigneeIDs {
			ids[jsonAs[string](t, id)] = true
		}
		if len(assigneeIDs) != 2 || !ids[ownerID] || !ids[bobID] {
			t.Fatalf("unexpected assignees: %v", assigneeIDs)
		}
		resp = doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+taskID, `{"assigneeIds":[]}`)
		assertStatus(t, resp, http.StatusOK)
		var cleared map[string]any
		readJSON(t, resp, &cleared)
		if len(jsonAs[[]any](t, cleared["assigneeIds"])) != 0 {
			t.Fatal("expected empty assignees after clearing")
		}
	})

	t.Run("assignees preserved on unrelated update", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		ownerID := getUserID(t, env, env.Token)
		task := createTask(t, env, space, `{"title":"Test","assigneeIds":["`+ownerID+`"]}`)
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+taskID, `{"title":"Updated"}`)
		assertStatus(t, resp, http.StatusOK)
		var updated map[string]any
		readJSON(t, resp, &updated)
		assigneeIDs := jsonAs[[]any](t, updated["assigneeIds"])
		if updated["title"] != "Updated" || len(assigneeIDs) != 1 || jsonAs[string](t, assigneeIDs[0]) != ownerID {
			t.Fatalf("unexpected updated task: %v", updated)
		}
	})

	t.Run("assignees validation", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		outsiderToken := createTestUser(t, env, testEmail(t, "outsider"), "Outsider", "pass1234")
		outsiderID := getUserID(t, env, outsiderToken)
		task := createTask(t, env, space, `{"title":"Test"}`)
		taskID := jsonAs[string](t, task["id"])
		assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+taskID, `{"assigneeIds":["`+outsiderID+`"]}`), http.StatusBadRequest)
		assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+taskID, `{"assigneeIds":["invalid"]}`), http.StatusBadRequest)
		assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+taskID, `{"assigneeIds":["U99999"]}`), http.StatusBadRequest)
	})

	t.Run("assignees cross space rejected", func(t *testing.T) {
		alpha := testSlug(t, "alpha")
		beta := testSlug(t, "beta")
		createSpace(t, env, alpha, "Alpha")
		createSpace(t, env, beta, "Beta")
		_, bobID := createAndAddMember(t, env, beta, testEmail(t, "bob"), "Bob", "pass1234", "member")
		task := createTask(t, env, alpha, `{"title":"Test"}`)
		taskID := jsonAs[string](t, task["id"])
		assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/"+alpha+"/tasks/"+taskID, `{"assigneeIds":["`+bobID+`"]}`), http.StatusBadRequest)
	})

	t.Run("assignees deduplicated and delete cascades", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		ownerID := getUserID(t, env, env.Token)
		task := createTask(t, env, space, `{"title":"Dedup","assigneeIds":["`+ownerID+`","`+ownerID+`"]}`)
		assigneeIDs := jsonAs[[]any](t, task["assigneeIds"])
		if len(assigneeIDs) != 1 {
			t.Fatalf("got %d assignees, want 1", len(assigneeIDs))
		}
		taskID := jsonAs[string](t, task["id"])
		assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/"+space+"/tasks/"+taskID, ""), http.StatusNoContent)
		assertStatusClose(t, doRequest(t, env, "GET", "/spaces/"+space+"/tasks/"+taskID, ""), http.StatusNotFound)
	})

	t.Run("assignees and tags in list response", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		ownerID := getUserID(t, env, env.Token)
		createTask(t, env, space, `{"title":"Assigned","assigneeIds":["`+ownerID+`"]}`)
		createTask(t, env, space, `{"title":"Unassigned"}`)
		createTask(t, env, space, `{"title":"Tagged","tags":["bug","urgent"]}`)
		createTask(t, env, space, `{"title":"Untagged"}`)
		resp := doRequest(t, env, "GET", "/spaces/"+space+"/tasks", "")
		assertStatus(t, resp, http.StatusOK)
		var page map[string]any
		readJSON(t, resp, &page)
		items := jsonAs[[]any](t, page["items"])
		if len(items) != 4 {
			t.Fatalf("got %d items, want 4", len(items))
		}
		for _, item := range items {
			task := jsonAs[map[string]any](t, item)
			title := jsonAs[string](t, task["title"])
			switch title {
			case "Assigned":
				assignees := jsonAs[[]any](t, task["assigneeIds"])
				if len(assignees) != 1 || jsonAs[string](t, assignees[0]) != ownerID {
					t.Fatalf("Assigned task assigneeIds = %v", assignees)
				}
			case "Unassigned":
				if len(jsonAs[[]any](t, task["assigneeIds"])) != 0 {
					t.Fatalf("Unassigned task assigneeIds = %v", task["assigneeIds"])
				}
			case "Tagged":
				if len(jsonAs[[]any](t, task["tags"])) != 2 {
					t.Fatalf("Tagged task tags = %v", task["tags"])
				}
			case "Untagged":
				if len(jsonAs[[]any](t, task["tags"])) != 0 {
					t.Fatalf("Untagged task tags = %v", task["tags"])
				}
			}
		}
	})

	t.Run("tag case folding preserves display name", func(t *testing.T) {
		space := testSlug(t, "tag-fold")
		createSpace(t, env, space, "Tag Fold")
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/"+space+"/tags", `{"name":"Bug"}`), http.StatusCreated)
		task := createTask(t, env, space, `{"title":"Test","tags":["bug"]}`)
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "GET", "/spaces/"+space+"/tasks/"+taskID, "")
		assertStatus(t, resp, http.StatusOK)
		var fetched map[string]any
		readJSON(t, resp, &fetched)
		tags := jsonAs[[]any](t, fetched["tags"])
		if len(tags) != 1 || tags[0] != "Bug" {
			t.Fatalf("tag = %v, want Bug", tags)
		}
	})
}

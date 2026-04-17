package api_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/sargunv/horologia/server/internal/taskengine"
)

func TestOverdueActionHandlers(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	t.Run("create with advance recurrence", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		future := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":null,"action":"advance_recurrence"}}`, future))
		rule := jsonAs[map[string]any](t, task["overdueActionRule"])
		if rule["action"] != "advance_recurrence" || rule["after"] != nil {
			t.Fatalf("unexpected rule: %v", rule)
		}
	})

	t.Run("create with set status", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		future := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":3,"action":"set_status","status":"done"}}`, future))
		rule := jsonAs[map[string]any](t, task["overdueActionRule"])
		if rule["action"] != "set_status" || jsonAs[float64](t, rule["after"]) != 3 || rule["status"] != "done" {
			t.Fatalf("unexpected rule: %v", rule)
		}
	})

	t.Run("validation rejects invalid combinations", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		future := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/"+space+"/tasks", fmt.Sprintf(`{"title":"One-off","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":null,"action":"advance_recurrence"}}`, future)), http.StatusBadRequest)
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/"+space+"/tasks", `{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","overdueActionRule":{"after":null,"action":"advance_recurrence"}}`), http.StatusBadRequest)
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/"+space+"/tasks", fmt.Sprintf(`{"title":"Accum","recurrenceType":"fixed_accumulating","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":null,"action":"advance_recurrence"}}`, future)), http.StatusBadRequest)
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/"+space+"/tasks", fmt.Sprintf(`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":null,"action":"set_status"}}`, future)), http.StatusBadRequest)
	})

	t.Run("patch preserves rule", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		future := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":2,"action":"clear_due_date"}}`, future))
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+taskID, `{"title":"Updated"}`)
		assertStatus(t, resp, http.StatusOK)
		var updated map[string]any
		readJSON(t, resp, &updated)
		rule := jsonAs[map[string]any](t, updated["overdueActionRule"])
		if rule["action"] != "clear_due_date" {
			t.Fatalf("unexpected rule after patch: %v", rule)
		}
	})

	t.Run("patch clears rule", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		future := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":null,"action":"advance_recurrence"}}`, future))
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+taskID, `{"overdueActionRule":null}`)
		assertStatus(t, resp, http.StatusOK)
		var updated map[string]any
		readJSON(t, resp, &updated)
		if updated["overdueActionRule"] != nil {
			t.Fatalf("overdueActionRule = %v, want nil", updated["overdueActionRule"])
		}
	})
}

func TestOverdueActionCron(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	t.Run("advance recurrence", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		threeDaysAgo := time.Now().UTC().AddDate(0, 0, -3).Format("2006-01-02")
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"Weekly","recurrenceType":"fixed_non_accumulating","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":null,"action":"advance_recurrence"}}`, threeDaysAgo))
		taskID := jsonAs[string](t, task["id"])
		must(t, taskengine.ProcessOverdueActionTasks(t.Context(), env.pool, time.Now(), nil))
		resp := doRequest(t, env, "GET", "/spaces/"+space+"/tasks/"+taskID, "")
		assertStatus(t, resp, http.StatusOK)
		var updated map[string]any
		readJSON(t, resp, &updated)
		due, hasDue := dueAtFromResponse(updated)
		if !hasDue {
			t.Fatal("due should be set after advance_recurrence")
		}
		newDue, _ := time.Parse("2006-01-02", due)
		origDue, _ := time.Parse("2006-01-02", threeDaysAgo)
		if !newDue.After(origDue) || updated["overdueActionRule"] == nil {
			t.Fatalf("unexpected updated task: %v", updated)
		}
	})

	t.Run("set status", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":null,"action":"set_status","status":"done"}}`, yesterday))
		taskID := jsonAs[string](t, task["id"])
		must(t, taskengine.ProcessOverdueActionTasks(t.Context(), env.pool, time.Now(), nil))
		resp := doRequest(t, env, "GET", "/spaces/"+space+"/tasks/"+taskID, "")
		assertStatus(t, resp, http.StatusOK)
		var updated map[string]any
		readJSON(t, resp, &updated)
		if updated["status"] != "done" {
			t.Fatalf("unexpected updated task: %v", updated)
		}
	})

	t.Run("clear due date", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":null,"action":"clear_due_date"}}`, yesterday))
		taskID := jsonAs[string](t, task["id"])
		must(t, taskengine.ProcessOverdueActionTasks(t.Context(), env.pool, time.Now(), nil))
		resp := doRequest(t, env, "GET", "/spaces/"+space+"/tasks/"+taskID, "")
		assertStatus(t, resp, http.StatusOK)
		var updated map[string]any
		readJSON(t, resp, &updated)
		if updated["due"] != nil || updated["overdueActionRule"] != nil {
			t.Fatalf("unexpected updated task: %v", updated)
		}
	})

	t.Run("grace period not yet elapsed", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":3,"action":"advance_recurrence"}}`, yesterday))
		taskID := jsonAs[string](t, task["id"])
		originalDue, _ := dueAtFromResponse(task)
		must(t, taskengine.ProcessOverdueActionTasks(t.Context(), env.pool, time.Now(), nil))
		resp := doRequest(t, env, "GET", "/spaces/"+space+"/tasks/"+taskID, "")
		assertStatus(t, resp, http.StatusOK)
		var updated map[string]any
		readJSON(t, resp, &updated)
		newDue, _ := dueAtFromResponse(updated)
		if newDue != originalDue {
			t.Fatalf("due changed from %s to %s", originalDue, newDue)
		}
	})

	t.Run("skips silently when status deleted", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		resp := doRequest(t, env, "PUT", "/spaces/"+space+"/task-statuses", `{"items":[{"name":"todo","category":"initial"},{"name":"needs-review","category":"intermediate"},{"name":"done","category":"completion"}]}`)
		assertStatusClose(t, resp, http.StatusOK)
		yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":null,"action":"set_status","status":"needs-review"}}`, yesterday))
		taskID := jsonAs[string](t, task["id"])
		originalStatus := jsonAs[string](t, task["status"])
		resp = doRequest(t, env, "PUT", "/spaces/"+space+"/task-statuses", `{"items":[{"name":"todo","category":"initial"},{"name":"done","category":"completion"}]}`)
		assertStatusClose(t, resp, http.StatusOK)
		var cronErr error
		must(t, taskengine.ProcessOverdueActionTasks(t.Context(), env.pool, time.Now(), func(_ int64, _ string, err error) { cronErr = err }))
		if cronErr != nil {
			t.Fatalf("expected silent skip, got %v", cronErr)
		}
		resp = doRequest(t, env, "GET", "/spaces/"+space+"/tasks/"+taskID, "")
		assertStatus(t, resp, http.StatusOK)
		var updated map[string]any
		readJSON(t, resp, &updated)
		if updated["status"] != originalStatus {
			t.Fatalf("status changed from %v to %v", originalStatus, updated["status"])
		}
	})

	t.Run("does not fire for future due", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		tomorrow := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":null,"action":"clear_due_date"}}`, tomorrow))
		taskID := jsonAs[string](t, task["id"])
		must(t, taskengine.ProcessOverdueActionTasks(t.Context(), env.pool, time.Now(), nil))
		resp := doRequest(t, env, "GET", "/spaces/"+space+"/tasks/"+taskID, "")
		assertStatus(t, resp, http.StatusOK)
		var updated map[string]any
		readJSON(t, resp, &updated)
		if updated["due"] == nil {
			t.Fatalf("due should not have been cleared: %v", updated)
		}
	})
}

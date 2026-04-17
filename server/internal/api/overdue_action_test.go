package api_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/sargunv/horologia/server/internal/taskengine"
)

// ─── Handler tests ────────────────────────────────────────────────────────────

func TestOverdueActionCreateWithAdvanceRecurrence(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	future := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	task := createTask(t, env, "home", fmt.Sprintf(
		`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":null,"action":"advance_recurrence"}}`,
		future,
	))

	rule, ok := task["overdueActionRule"].(map[string]any)
	if !ok {
		t.Fatalf("overdueActionRule missing or wrong type: %T", task["overdueActionRule"])
	}
	if rule["action"] != "advance_recurrence" {
		t.Errorf("action = %v, want advance_recurrence", rule["action"])
	}
	if rule["after"] != nil {
		t.Errorf("after = %v, want nil", rule["after"])
	}
}

func TestOverdueActionCreateWithSetStatus(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	future := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	task := createTask(t, env, "home", fmt.Sprintf(
		`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":3,"action":"set_status","status":"done"}}`,
		future,
	))

	rule, ok := task["overdueActionRule"].(map[string]any)
	if !ok {
		t.Fatalf("overdueActionRule missing or wrong type: %T", task["overdueActionRule"])
	}
	if rule["action"] != "set_status" {
		t.Errorf("action = %v, want set_status", rule["action"])
	}
	after := jsonAs[float64](t, rule["after"])
	if after != 3 {
		t.Errorf("after = %v, want 3", after)
	}
	if rule["status"] != "done" {
		t.Errorf("status = %v, want done", rule["status"])
	}
}

func TestOverdueActionRejectedOnOneOff(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	future := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	resp := doRequest(t, env, "POST", "/spaces/home/tasks", fmt.Sprintf(
		`{"title":"One-off","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":null,"action":"advance_recurrence"}}`,
		future,
	))
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestOverdueActionRejectedWithoutDue(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "POST", "/spaces/home/tasks",
		`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","overdueActionRule":{"after":null,"action":"advance_recurrence"}}`,
	)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestOverdueActionRejectedAdvanceOnFixedAccumulating(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	future := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	resp := doRequest(t, env, "POST", "/spaces/home/tasks", fmt.Sprintf(
		`{"title":"Accum","recurrenceType":"fixed_accumulating","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":null,"action":"advance_recurrence"}}`,
		future,
	))
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestOverdueActionRejectedSetStatusWithoutStatus(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	future := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	resp := doRequest(t, env, "POST", "/spaces/home/tasks", fmt.Sprintf(
		`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":null,"action":"set_status"}}`,
		future,
	))
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestOverdueActionPatchPreservesRule(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	future := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	task := createTask(t, env, "home", fmt.Sprintf(
		`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":2,"action":"clear_due_date"}}`,
		future,
	))
	taskID := jsonAs[string](t, task["id"])

	// Patch title only — overdueActionRule should be preserved.
	resp := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+taskID, `{"title":"Updated"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	rule, ok := updated["overdueActionRule"].(map[string]any)
	if !ok {
		t.Fatalf("overdueActionRule missing after title-only patch: %T", updated["overdueActionRule"])
	}
	if rule["action"] != "clear_due_date" {
		t.Errorf("action after patch = %v, want clear_due_date", rule["action"])
	}
}

func TestOverdueActionPatchClearsRule(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	future := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	task := createTask(t, env, "home", fmt.Sprintf(
		`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":null,"action":"advance_recurrence"}}`,
		future,
	))
	taskID := jsonAs[string](t, task["id"])

	// Patch overdueActionRule to null — clears it.
	resp := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+taskID, `{"overdueActionRule":null}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	if updated["overdueActionRule"] != nil {
		t.Errorf("overdueActionRule = %v after null patch, want nil", updated["overdueActionRule"])
	}
}

// ─── Cron integration tests ───────────────────────────────────────────────────

func TestOverdueActionCronAdvanceRecurrence(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	// Create a weekly task due 3 days ago with advance_recurrence rule (no grace period).
	threeDaysAgo := time.Now().UTC().AddDate(0, 0, -3).Format("2006-01-02")
	task := createTask(t, env, "home", fmt.Sprintf(
		`{"title":"Weekly","recurrenceType":"fixed_non_accumulating","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":null,"action":"advance_recurrence"}}`,
		threeDaysAgo,
	))
	taskID := jsonAs[string](t, task["id"])

	must(t, taskengine.ProcessOverdueActionTasks(t.Context(), env.pool, time.Now(), nil))

	resp := doRequest(t, env, "GET", "/spaces/home/tasks/"+taskID, "")
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	due, hasDue := dueAtFromResponse(updated)
	if !hasDue {
		t.Fatal("due should be set after advance_recurrence")
	}
	// Due should have advanced by one week from threeDaysAgo.
	newDue, err := time.Parse("2006-01-02", due)
	if err != nil {
		t.Fatalf("parse new due %q: %v", due, err)
	}
	origDue, _ := time.Parse("2006-01-02", threeDaysAgo)
	if !newDue.After(origDue) {
		t.Errorf("new due %s should be after original %s", due, threeDaysAgo)
	}

	// Rule should still be present (not exhausted).
	if updated["overdueActionRule"] == nil {
		t.Error("overdueActionRule should be preserved after advance_recurrence")
	}
}

func TestOverdueActionCronSetStatus(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	// Create a weekly task due yesterday with set_status rule (immediate = no grace).
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	task := createTask(t, env, "home", fmt.Sprintf(
		`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":null,"action":"set_status","status":"done"}}`,
		yesterday,
	))
	taskID := jsonAs[string](t, task["id"])

	must(t, taskengine.ProcessOverdueActionTasks(t.Context(), env.pool, time.Now(), nil))

	resp := doRequest(t, env, "GET", "/spaces/home/tasks/"+taskID, "")
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	if updated["status"] != "done" {
		t.Errorf("status = %v after set_status cron, want done", updated["status"])
	}
}

func TestOverdueActionCronClearDueDate(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	// Create a weekly task due yesterday with clear_due_date rule.
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	task := createTask(t, env, "home", fmt.Sprintf(
		`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":null,"action":"clear_due_date"}}`,
		yesterday,
	))
	taskID := jsonAs[string](t, task["id"])

	must(t, taskengine.ProcessOverdueActionTasks(t.Context(), env.pool, time.Now(), nil))

	resp := doRequest(t, env, "GET", "/spaces/home/tasks/"+taskID, "")
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	if updated["due"] != nil {
		t.Errorf("due = %v after clear_due_date cron, want nil", updated["due"])
	}
	// Overdue action rule should be cleared too (requires due).
	if updated["overdueActionRule"] != nil {
		t.Errorf("overdueActionRule = %v after clear_due_date, want nil", updated["overdueActionRule"])
	}
}

func TestOverdueActionCronGracePeriodNotYetElapsed(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	// Create a weekly task due yesterday with 3-day grace period — should NOT fire yet.
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	task := createTask(t, env, "home", fmt.Sprintf(
		`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":3,"action":"advance_recurrence"}}`,
		yesterday,
	))
	taskID := jsonAs[string](t, task["id"])
	originalDue, _ := dueAtFromResponse(task)

	must(t, taskengine.ProcessOverdueActionTasks(t.Context(), env.pool, time.Now(), nil))

	resp := doRequest(t, env, "GET", "/spaces/home/tasks/"+taskID, "")
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	// Due should be unchanged — grace period not elapsed.
	newDue, _ := dueAtFromResponse(updated)
	if newDue != originalDue {
		t.Errorf("due changed from %s to %s, expected no change (grace period not elapsed)", originalDue, newDue)
	}
}

func TestOverdueActionCronSkipsSilentlyWhenStatusDeleted(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	// Add a custom status "needs-review" to the space.
	resp := doRequest(t, env, "PUT", "/spaces/home/task-statuses", `{"items":[
		{"name":"todo","category":"initial"},
		{"name":"needs-review","category":"intermediate"},
		{"name":"done","category":"completion"}
	]}`)
	assertStatusClose(t, resp, http.StatusOK)

	// Create a weekly task due yesterday with set_status rule pointing to "needs-review".
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	task := createTask(t, env, "home", fmt.Sprintf(
		`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":null,"action":"set_status","status":"needs-review"}}`,
		yesterday,
	))
	taskID := jsonAs[string](t, task["id"])
	originalStatus := jsonAs[string](t, task["status"])

	// Remove "needs-review" by replacing the status list (simulating status deletion).
	resp = doRequest(t, env, "PUT", "/spaces/home/task-statuses", `{"items":[
		{"name":"todo","category":"initial"},
		{"name":"done","category":"completion"}
	]}`)
	assertStatusClose(t, resp, http.StatusOK)

	// Cron should not return an error — silently skips missing status.
	var cronErr error
	must(t, taskengine.ProcessOverdueActionTasks(t.Context(), env.pool, time.Now(), func(_ int64, _ string, err error) {
		cronErr = err
	}))
	if cronErr != nil {
		t.Errorf("expected silent skip for deleted status, got error: %v", cronErr)
	}

	// Task status should be unchanged.
	resp = doRequest(t, env, "GET", "/spaces/home/tasks/"+taskID, "")
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)
	if updated["status"] != originalStatus {
		t.Errorf("status changed from %v to %v, expected no change (status deleted)", originalStatus, updated["status"])
	}
}

func TestOverdueActionCronDoesNotFireForFutureDue(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	createSpace(t, env, "home", "Home")

	// Create a weekly task due tomorrow — should NOT fire.
	tomorrow := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	task := createTask(t, env, "home", fmt.Sprintf(
		`{"title":"Weekly","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"overdueActionRule":{"after":null,"action":"clear_due_date"}}`,
		tomorrow,
	))
	taskID := jsonAs[string](t, task["id"])

	must(t, taskengine.ProcessOverdueActionTasks(t.Context(), env.pool, time.Now(), nil))

	resp := doRequest(t, env, "GET", "/spaces/home/tasks/"+taskID, "")
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	if updated["due"] == nil {
		t.Error("due should not have been cleared for future task")
	}
}

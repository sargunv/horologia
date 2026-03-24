package api_test

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// dueAtFromResponse extracts the "at" string from a task's nested "due" field.
// Returns ("", false) if due is nil.
func dueAtFromResponse(task map[string]any) (string, bool) {
	due := task["due"]
	if due == nil {
		return "", false
	}
	return due.(map[string]any)["at"].(string), true
}

// dueTzFromResponse extracts the "timezone" string from a task's nested "due" field.
// Returns ("", false) if due is nil.
func dueTzFromResponse(task map[string]any) (string, bool) {
	due := task["due"]
	if due == nil {
		return "", false
	}
	return due.(map[string]any)["timezone"].(string), true
}

func TestRecurrenceOneOffDefault(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-def", "Rec Default")

	task := createTask(t, env, "rec-def", `{"title":"Normal task"}`)
	if task["recurrenceType"] != "one_off" {
		t.Fatalf("got recurrenceType %v, want one_off", task["recurrenceType"])
	}
	if task["recurrenceRule"] != nil {
		t.Fatalf("got recurrenceRule %v, want nil", task["recurrenceRule"])
	}
	if task["lastCompletedAt"] != nil {
		t.Fatalf("got lastCompletedAt %v, want nil", task["lastCompletedAt"])
	}
}

func TestRecurrenceOneOffRejectsRule(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-rej", "Rec Reject")

	assertStatusClose(t,
		doRequest(t, env, "POST", "/spaces/rec-rej/tasks",
			`{"title":"Bad","recurrenceType":"one_off","recurrenceRule":"FREQ=WEEKLY"}`),
		http.StatusBadRequest)
}

func TestRecurrenceRequiresRule(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-req", "Rec Require")

	assertStatusClose(t,
		doRequest(t, env, "POST", "/spaces/rec-req/tasks",
			`{"title":"Bad","recurrenceType":"completion_based"}`),
		http.StatusBadRequest)
}

func TestRecurrenceInvalidRRule(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-inv", "Rec Invalid")

	assertStatusClose(t,
		doRequest(t, env, "POST", "/spaces/rec-inv/tasks",
			`{"title":"Bad","recurrenceType":"completion_based","recurrenceRule":"FREQ=BOGUS"}`),
		http.StatusBadRequest)
}

func TestRecurrenceRejectsSubDayFrequency(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-sub", "Rec SubDay")

	assertStatusClose(t,
		doRequest(t, env, "POST", "/spaces/rec-sub/tasks",
			`{"title":"Bad","recurrenceType":"completion_based","recurrenceRule":"FREQ=HOURLY"}`),
		http.StatusBadRequest)
}

func TestRecurrenceRejectsUnsupportedProperty(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-prop", "Rec Prop")

	assertStatusClose(t,
		doRequest(t, env, "POST", "/spaces/rec-prop/tasks",
			`{"title":"Bad","recurrenceType":"completion_based","recurrenceRule":"FREQ=DAILY;BYHOUR=10"}`),
		http.StatusBadRequest)
}

func TestRecurrenceCompletionBased(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-cb", "Rec CB")

	// Create a completion-based task recurring weekly.
	task := createTask(t, env, "rec-cb",
		`{"title":"Weekly chore","recurrenceType":"completion_based","recurrenceRule":"FREQ=WEEKLY","due":{"at":"2026-03-23T00:00:00Z","timezone":"UTC"}}`)

	if task["recurrenceType"] != "completion_based" {
		t.Fatalf("got recurrenceType %v, want completion_based", task["recurrenceType"])
	}
	if task["recurrenceRule"] != "FREQ=WEEKLY" {
		t.Fatalf("got recurrenceRule %v, want FREQ=WEEKLY", task["recurrenceRule"])
	}
	taskID := task["id"].(string)

	// Complete the task by setting status to "done".
	resp := doRequest(t, env, "PATCH", "/spaces/rec-cb/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	// Should be reset to initial status.
	if updated["status"] != "todo" {
		t.Fatalf("got status %v, want todo", updated["status"])
	}
	// lastCompletedAt should be set.
	if updated["lastCompletedAt"] == nil {
		t.Fatal("expected lastCompletedAt to be set")
	}
	// Due date should be ~7 days from now (completion-based: DTSTART = now).
	if updated["due"] == nil {
		t.Fatal("expected dueAt to be set after completion")
	}
	newDue := updated["due"].(map[string]any)["at"].(string)
	parsedDue, err := time.Parse(time.RFC3339, newDue)
	if err != nil {
		t.Fatalf("parse dueAt %q: %v", newDue, err)
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	daysUntilDue := int(parsedDue.Sub(today).Hours() / 24)
	if daysUntilDue < 6 || daysUntilDue > 8 {
		t.Fatalf("expected dueAt ~7 days from now, got %s (%d days)", newDue, daysUntilDue)
	}
}

func TestRecurrenceFixedNonAccumulating(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-fna", "Rec FNA")

	// Create a fixed non-accumulating task due every Saturday.
	task := createTask(t, env, "rec-fna",
		`{"title":"Saturday task","recurrenceType":"fixed_non_accumulating","recurrenceRule":"FREQ=WEEKLY;BYDAY=SA","due":{"at":"2026-03-21T00:00:00Z","timezone":"UTC"}}`)

	taskID := task["id"].(string)

	// Complete the task.
	resp := doRequest(t, env, "PATCH", "/spaces/rec-fna/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	// Should be reset to initial status.
	if updated["status"] != "todo" {
		t.Fatalf("got status %v, want todo", updated["status"])
	}
	// lastCompletedAt should be set.
	if updated["lastCompletedAt"] == nil {
		t.Fatal("expected lastCompletedAt to be set")
	}
	// Due date should be the next Saturday after now.
	if updated["due"] == nil {
		t.Fatal("expected dueAt to be set")
	}
	parsedDue, err := time.Parse(time.RFC3339, updated["due"].(map[string]any)["at"].(string))
	if err != nil {
		t.Fatalf("parse dueAt: %v", err)
	}
	if parsedDue.Weekday() != time.Saturday {
		t.Fatalf("expected dueAt to be a Saturday, got %s (%s)", updated["due"], parsedDue.Weekday())
	}
	if !parsedDue.After(time.Now().Truncate(24 * time.Hour)) {
		t.Fatalf("expected dueAt after today, got %s", updated["due"])
	}
}

func TestRecurrenceOneOffCompletion(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-oc", "Rec One Off Complete")

	task := createTask(t, env, "rec-oc", `{"title":"One-off task"}`)
	taskID := task["id"].(string)

	// Complete the task.
	resp := doRequest(t, env, "PATCH", "/spaces/rec-oc/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	// Status should stay at "done" (no reset for one_off).
	if updated["status"] != "done" {
		t.Fatalf("got status %v, want done", updated["status"])
	}
	// lastCompletedAt should be set.
	if updated["lastCompletedAt"] == nil {
		t.Fatal("expected lastCompletedAt to be set")
	}
}

func TestRecurrenceFixedAccumulatingCompletion(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-fa", "Rec FA")

	task := createTask(t, env, "rec-fa",
		`{"title":"Monthly task","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=MONTHLY","due":{"at":"2026-04-01T00:00:00Z","timezone":"UTC"}}`)
	taskID := task["id"].(string)

	// Complete the task.
	resp := doRequest(t, env, "PATCH", "/spaces/rec-fa/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	// The completed task stays at "done" and becomes one_off.
	if updated["status"] != "done" {
		t.Fatalf("got status %v, want done", updated["status"])
	}
	if updated["recurrenceType"] != "one_off" {
		t.Fatalf("got recurrenceType %v, want one_off", updated["recurrenceType"])
	}
	if updated["recurrenceRule"] != nil {
		t.Fatalf("got recurrenceRule %v, want nil", updated["recurrenceRule"])
	}
	if updated["lastCompletedAt"] == nil {
		t.Fatal("expected lastCompletedAt to be set")
	}

	// A new fixed_accumulating task should have been spawned.
	// Check via the spawns relation on the completed task.
	var spawnsRelation map[string]any
	for _, rel := range updated["relations"].([]any) {
		r := rel.(map[string]any)
		if r["kind"] == "spawns" {
			spawnsRelation = r
			break
		}
	}
	if spawnsRelation == nil {
		t.Fatal("expected a 'spawns' relation on the completed task")
	}

	// Fetch the spawned task.
	spawnedID := spawnsRelation["taskId"].(string)
	resp2 := doRequest(t, env, "GET", "/spaces/rec-fa/tasks/"+spawnedID, "")
	assertStatus(t, resp2, http.StatusOK)
	var spawned map[string]any
	readJSON(t, resp2, &spawned)

	if spawned["status"] != "todo" {
		t.Fatalf("spawned task status = %v, want todo", spawned["status"])
	}
	if spawned["recurrenceType"] != "fixed_accumulating" {
		t.Fatalf("spawned task recurrenceType = %v, want fixed_accumulating", spawned["recurrenceType"])
	}
	if spawned["due"] == nil {
		t.Fatal("spawned task should have a due date")
	}
}

func TestRecurrenceNoRetriggerOnNonTransition(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-nrt", "Rec No Retrigger")

	task := createTask(t, env, "rec-nrt",
		`{"title":"Weekly chore","recurrenceType":"completion_based","recurrenceRule":"FREQ=WEEKLY","due":{"at":"2026-03-23T00:00:00Z","timezone":"UTC"}}`)
	taskID := task["id"].(string)

	// Complete the task.
	resp := doRequest(t, env, "PATCH", "/spaces/rec-nrt/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var first map[string]any
	readJSON(t, resp, &first)
	firstDueAt, _ := dueAtFromResponse(first)

	// Update title while already reset to "todo" — should not change due date.
	resp2 := doRequest(t, env, "PATCH", "/spaces/rec-nrt/tasks/"+taskID, `{"title":"Updated title"}`)
	assertStatus(t, resp2, http.StatusOK)
	var second map[string]any
	readJSON(t, resp2, &second)

	secondDueAt, _ := dueAtFromResponse(second)
	if secondDueAt != firstDueAt {
		t.Fatalf("due.at changed on non-status update: got %v, want %v", secondDueAt, firstDueAt)
	}
}

func TestRecurrenceOnDependencyTrigger(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-dep", "Rec Dep")

	// Create task A (one_off) and task B (on_dependency).
	taskA := createTask(t, env, "rec-dep", `{"title":"Load dishwasher"}`)
	taskB := createTask(t, env, "rec-dep", `{"title":"Unload dishwasher","recurrenceType":"on_dependency"}`)
	taskAID := taskA["id"].(string)
	taskBID := taskB["id"].(string)

	// Complete B first so it's in "done" state.
	assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/rec-dep/tasks/"+taskBID, `{"status":"done"}`), http.StatusOK)

	// Create "triggers" relation: A triggers B.
	createRelation(t, env, "rec-dep", taskAID, "triggers", taskBID)

	// Complete A.
	assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/rec-dep/tasks/"+taskAID, `{"status":"done"}`), http.StatusOK)

	// B should be reset to initial status.
	resp := doRequest(t, env, "GET", "/spaces/rec-dep/tasks/"+taskBID, "")
	assertStatus(t, resp, http.StatusOK)
	var taskBState map[string]any
	readJSON(t, resp, &taskBState)

	if taskBState["status"] != "todo" {
		t.Fatalf("task B status = %v, want todo", taskBState["status"])
	}
}

func TestRecurrenceTriggersRelationKind(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-trk", "Rec Trigger Kind")

	taskA := createTask(t, env, "rec-trk", `{"title":"Task A"}`)
	taskB := createTask(t, env, "rec-trk", `{"title":"Task B"}`)
	taskAID := taskA["id"].(string)
	taskBID := taskB["id"].(string)

	// Create triggers relation.
	createRelation(t, env, "rec-trk", taskAID, "triggers", taskBID)

	// A should show "triggers" B.
	relsA := assertTaskRelations(t, env, "rec-trk", taskAID, 1)
	assertRelationKind(t, relsA[0], "triggers", taskBID)

	// B should show "triggered_by" A.
	relsB := assertTaskRelations(t, env, "rec-trk", taskBID, 1)
	assertRelationKind(t, relsB[0], "triggered_by", taskAID)
}

func TestRecurrenceUpdateTypeFromOneOff(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-upd", "Rec Update")

	task := createTask(t, env, "rec-upd", `{"title":"Was one-off"}`)
	taskID := task["id"].(string)

	// Update to completion_based with a rule.
	resp := doRequest(t, env, "PATCH", "/spaces/rec-upd/tasks/"+taskID,
		`{"recurrenceType":"completion_based","recurrenceRule":"FREQ=DAILY;INTERVAL=3"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	if updated["recurrenceType"] != "completion_based" {
		t.Fatalf("got recurrenceType %v, want completion_based", updated["recurrenceType"])
	}
	if updated["recurrenceRule"] != "FREQ=DAILY;INTERVAL=3" {
		t.Fatalf("got recurrenceRule %v, want FREQ=DAILY;INTERVAL=3", updated["recurrenceRule"])
	}
}

func TestRecurrenceUpdateTypeRequiresRule(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-ur", "Rec Update Req")

	task := createTask(t, env, "rec-ur", `{"title":"Was one-off"}`)
	taskID := task["id"].(string)

	// Changing to completion_based without a rule should fail.
	assertStatusClose(t,
		doRequest(t, env, "PATCH", "/spaces/rec-ur/tasks/"+taskID,
			`{"recurrenceType":"completion_based"}`),
		http.StatusBadRequest)
}

func TestRecurrenceCrossSpaceIsolation(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-iso-a", "Rec Iso A")
	createSpace(t, env, "rec-iso-b", "Rec Iso B")

	task := createTask(t, env, "rec-iso-a",
		`{"title":"Recurring","recurrenceType":"completion_based","recurrenceRule":"FREQ=WEEKLY"}`)

	// Task in space A should not be accessible from space B.
	taskID := task["id"].(string)
	assertStatusClose(t,
		doRequest(t, env, "GET", "/spaces/rec-iso-b/tasks/"+taskID, ""),
		http.StatusNotFound)
}

func TestRecurrenceOnDependencyDirectCompletion(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-odc", "Rec OnDep Complete")

	// Create an on_dependency task and complete it directly.
	// It should stay at "done" — on_dependency reset only happens when
	// the trigger source completes, not when the task itself is completed.
	task := createTask(t, env, "rec-odc", `{"title":"Dependent task","recurrenceType":"on_dependency"}`)
	taskID := task["id"].(string)

	resp := doRequest(t, env, "PATCH", "/spaces/rec-odc/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	if updated["status"] != "done" {
		t.Fatalf("got status %v, want done (on_dependency should stay done when completed directly)", updated["status"])
	}
	if updated["lastCompletedAt"] == nil {
		t.Fatal("expected lastCompletedAt to be set")
	}
}

func TestRecurrenceOnDependencyRejectsRule(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-odr", "Rec OnDep Rule")

	assertStatusClose(t,
		doRequest(t, env, "POST", "/spaces/rec-odr/tasks",
			`{"title":"Bad","recurrenceType":"on_dependency","recurrenceRule":"FREQ=WEEKLY"}`),
		http.StatusBadRequest)
}

func TestRecurrenceUntilExhaustion(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-unt", "Rec Until")

	// Create a completion_based task with UNTIL in the past. The rule is
	// already exhausted — the task should stay at the completion status.
	task := createTask(t, env, "rec-unt",
		`{"title":"Expired","recurrenceType":"completion_based","recurrenceRule":"FREQ=WEEKLY;UNTIL=20200101T000000Z","due":{"at":"2026-03-23T00:00:00Z","timezone":"UTC"}}`)
	taskID := task["id"].(string)

	resp := doRequest(t, env, "PATCH", "/spaces/rec-unt/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	// Rule is exhausted — task should stay "done", not reset.
	if updated["status"] != "done" {
		t.Fatalf("got status %v, want done (rule exhausted)", updated["status"])
	}
	if updated["lastCompletedAt"] == nil {
		t.Fatal("expected lastCompletedAt to be set")
	}
}

func TestRecurrenceDoubleCompletion(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-dbl", "Rec Double")

	task := createTask(t, env, "rec-dbl",
		`{"title":"Every 3 days","recurrenceType":"completion_based","recurrenceRule":"FREQ=DAILY;INTERVAL=3","due":{"at":"2026-03-23T00:00:00Z","timezone":"UTC"}}`)
	taskID := task["id"].(string)

	// First completion — should reset to todo with due date 3 days from now.
	resp1 := doRequest(t, env, "PATCH", "/spaces/rec-dbl/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp1, http.StatusOK)
	var first map[string]any
	readJSON(t, resp1, &first)

	if first["status"] != "todo" {
		t.Fatalf("first completion: got status %v, want todo", first["status"])
	}
	firstDue, err := time.Parse(time.RFC3339, first["due"].(map[string]any)["at"].(string))
	if err != nil {
		t.Fatalf("parse first dueAt: %v", err)
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	daysUntil := int(firstDue.Sub(today).Hours() / 24)
	if daysUntil < 2 || daysUntil > 4 {
		t.Fatalf("first completion: expected dueAt ~3 days from now, got %s (%d days)", first["due"], daysUntil)
	}

	// Second completion — recurrence fires again; task should reset.
	resp2 := doRequest(t, env, "PATCH", "/spaces/rec-dbl/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp2, http.StatusOK)
	var second map[string]any
	readJSON(t, resp2, &second)

	if second["status"] != "todo" {
		t.Fatalf("second completion: got status %v, want todo", second["status"])
	}
	if second["lastCompletedAt"] == nil {
		t.Fatal("expected lastCompletedAt to be set after second completion")
	}
	// Both completions happen at ~same instant, so due dates should be equal.
	secondDueStr, _ := dueAtFromResponse(second)
	secondDue, err2 := time.Parse(time.RFC3339, secondDueStr)
	if err2 != nil {
		t.Fatalf("parse second due.at: %v", err2)
	}
	if secondDue != firstDue {
		t.Fatalf("second due %s should equal first due %s (same completion time)", secondDueStr, first["due"])
	}
}

func TestRecurrenceAutoCleanRuleOnTypeChange(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-acr", "Rec Auto Clean")

	// Create a completion_based task with a rule.
	task := createTask(t, env, "rec-acr",
		`{"title":"Was recurring","recurrenceType":"completion_based","recurrenceRule":"FREQ=WEEKLY"}`)
	taskID := task["id"].(string)

	// Change type to one_off WITHOUT explicitly nulling recurrenceRule.
	// The server should auto-clear the rule.
	resp := doRequest(t, env, "PATCH", "/spaces/rec-acr/tasks/"+taskID,
		`{"recurrenceType":"one_off"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	if updated["recurrenceType"] != "one_off" {
		t.Fatalf("got recurrenceType %v, want one_off", updated["recurrenceType"])
	}
	if updated["recurrenceRule"] != nil {
		t.Fatalf("got recurrenceRule %v, want nil (should auto-clear)", updated["recurrenceRule"])
	}
}

func TestRecurrenceFixedAccumulatingDueDateAdvances(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-fap", "Rec FA Advance")

	// Use a past due date so the next occurrence is clearly in the future.
	task := createTask(t, env, "rec-fap",
		`{"title":"Monthly","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=MONTHLY","due":{"at":"2026-01-01T00:00:00Z","timezone":"UTC"}}`)
	taskID := task["id"].(string)

	resp := doRequest(t, env, "PATCH", "/spaces/rec-fap/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	// The completed task becomes one_off with its original due date.
	if updated["recurrenceType"] != "one_off" {
		t.Fatalf("got recurrenceType %v, want one_off", updated["recurrenceType"])
	}

	// Find the spawned task via the spawns relation.
	var spawnedID string
	for _, rel := range updated["relations"].([]any) {
		r := rel.(map[string]any)
		if r["kind"] == "spawns" {
			spawnedID = r["taskId"].(string)
			break
		}
	}
	if spawnedID == "" {
		t.Fatal("expected a 'spawns' relation on the completed task")
	}

	resp2 := doRequest(t, env, "GET", "/spaces/rec-fap/tasks/"+spawnedID, "")
	assertStatus(t, resp2, http.StatusOK)
	var spawned map[string]any
	readJSON(t, resp2, &spawned)

	// Spawned task should have a future due date on the 1st of a month.
	if spawned["due"] == nil {
		t.Fatal("spawned task should have a due date")
	}
	newDue := spawned["due"].(map[string]any)["at"].(string)
	parsedDue, err := time.Parse(time.RFC3339, newDue)
	if err != nil {
		t.Fatalf("parse dueAt: %v", err)
	}
	if !parsedDue.After(time.Now().Truncate(24 * time.Hour)) {
		t.Fatalf("expected dueAt after today, got %s", newDue)
	}
	if parsedDue.Day() != 1 {
		t.Fatalf("expected dueAt on the 1st, got day %d", parsedDue.Day())
	}
	if spawned["recurrenceType"] != "fixed_accumulating" {
		t.Fatalf("spawned recurrenceType = %v, want fixed_accumulating", spawned["recurrenceType"])
	}
}

func TestRecurrenceCountRejected(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-cnt", "Rec Count")

	// COUNT is not supported — use UNTIL instead.
	assertStatusClose(t,
		doRequest(t, env, "POST", "/spaces/rec-cnt/tasks",
			`{"title":"Bad","recurrenceType":"completion_based","recurrenceRule":"FREQ=DAILY;COUNT=5"}`),
		http.StatusBadRequest)
}

func TestRecurrenceSameStatusNoRetrigger(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-snr", "Rec Same No Retrigger")

	// Complete a one_off task, then PATCH status "done" again.
	// The second PATCH should not update lastCompletedAt.
	task := createTask(t, env, "rec-snr", `{"title":"One-off"}`)
	taskID := task["id"].(string)

	resp := doRequest(t, env, "PATCH", "/spaces/rec-snr/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var first map[string]any
	readJSON(t, resp, &first)
	firstCompleted := first["lastCompletedAt"]

	// Same status again — not a transition, should not re-trigger.
	resp2 := doRequest(t, env, "PATCH", "/spaces/rec-snr/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp2, http.StatusOK)
	var second map[string]any
	readJSON(t, resp2, &second)

	if second["lastCompletedAt"] != firstCompleted {
		t.Fatalf("lastCompletedAt changed on same-status update: got %v, want %v", second["lastCompletedAt"], firstCompleted)
	}
}

func TestRecurrenceCompletionBasedNonUTCTimezone(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-tz", "Rec TZ")

	// Create a completion-based task with America/New_York timezone.
	// Due at midnight Eastern (which is 05:00 UTC during EST).
	task := createTask(t, env, "rec-tz",
		`{"title":"Weekly chore","recurrenceType":"completion_based","recurrenceRule":"FREQ=WEEKLY","due":{"at":"2026-03-23T05:00:00Z","timezone":"America/New_York"}}`)
	taskID := task["id"].(string)

	// Verify timezone is stored and returned.
	tz, ok := dueTzFromResponse(task)
	if !ok || tz != "America/New_York" {
		t.Fatalf("got timezone %q, want America/New_York", tz)
	}

	// Complete the task.
	resp := doRequest(t, env, "PATCH", "/spaces/rec-tz/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	if updated["status"] != "todo" {
		t.Fatalf("got status %v, want todo", updated["status"])
	}

	// Timezone should be preserved after completion.
	updatedTz, ok := dueTzFromResponse(updated)
	if !ok || updatedTz != "America/New_York" {
		t.Fatalf("timezone after completion: got %q, want America/New_York", updatedTz)
	}

	// The new due date should be midnight Eastern (~7 days from now).
	// In UTC, midnight Eastern is either 05:00 (EST) or 04:00 (EDT).
	dueStr, ok := dueAtFromResponse(updated)
	if !ok {
		t.Fatal("expected due to be set after completion")
	}
	parsedDue, err := time.Parse(time.RFC3339, dueStr)
	if err != nil {
		t.Fatalf("parse due.at: %v", err)
	}

	// Verify the time component is midnight in New York (not midnight UTC).
	loc, _ := time.LoadLocation("America/New_York")
	localDue := parsedDue.In(loc)
	if localDue.Hour() != 0 || localDue.Minute() != 0 {
		t.Fatalf("expected midnight in America/New_York, got %s", localDue.Format(time.RFC3339))
	}
}

func TestRecurrenceFixedNonAccumulatingNonUTCTimezone(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-ftz", "Rec Fixed TZ")

	// Create a fixed non-accumulating task recurring every Saturday in Los Angeles.
	// Due at midnight Pacific (08:00 UTC during PST).
	task := createTask(t, env, "rec-ftz",
		`{"title":"Saturday task","recurrenceType":"fixed_non_accumulating","recurrenceRule":"FREQ=WEEKLY;BYDAY=SA","due":{"at":"2026-03-21T08:00:00Z","timezone":"America/Los_Angeles"}}`)
	taskID := task["id"].(string)

	// Complete the task.
	resp := doRequest(t, env, "PATCH", "/spaces/rec-ftz/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	if updated["status"] != "todo" {
		t.Fatalf("got status %v, want todo", updated["status"])
	}

	// Due date should be a Saturday at midnight Pacific.
	dueStr, ok := dueAtFromResponse(updated)
	if !ok {
		t.Fatal("expected due to be set")
	}
	parsedDue, err := time.Parse(time.RFC3339, dueStr)
	if err != nil {
		t.Fatalf("parse due.at: %v", err)
	}

	loc, _ := time.LoadLocation("America/Los_Angeles")
	localDue := parsedDue.In(loc)
	if localDue.Weekday() != time.Saturday {
		t.Fatalf("expected Saturday in LA, got %s (%s)", localDue.Format("2006-01-02"), localDue.Weekday())
	}
	if localDue.Hour() != 0 || localDue.Minute() != 0 {
		t.Fatalf("expected midnight in America/Los_Angeles, got %s", localDue.Format(time.RFC3339))
	}
}

func TestDueTimezoneRoundTrip(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "due-rt", "Due Round Trip")

	// Create with a non-UTC timezone.
	task := createTask(t, env, "due-rt",
		`{"title":"TZ task","due":{"at":"2026-06-15T04:00:00Z","timezone":"America/New_York"}}`)

	// Verify round-trip: both at and timezone returned correctly.
	dueStr, ok := dueAtFromResponse(task)
	if !ok {
		t.Fatal("expected due to be set")
	}
	if dueStr != "2026-06-15T04:00:00Z" {
		t.Fatalf("due.at = %v, want 2026-06-15T04:00:00Z", dueStr)
	}
	tz, ok := dueTzFromResponse(task)
	if !ok || tz != "America/New_York" {
		t.Fatalf("due.timezone = %v, want America/New_York", tz)
	}

	// Update to a different timezone.
	taskID := task["id"].(string)
	resp := doRequest(t, env, "PATCH", "/spaces/due-rt/tasks/"+taskID,
		`{"due":{"at":"2026-06-15T07:00:00Z","timezone":"America/Los_Angeles"}}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	updatedTz, ok := dueTzFromResponse(updated)
	if !ok || updatedTz != "America/Los_Angeles" {
		t.Fatalf("updated due.timezone = %v, want America/Los_Angeles", updatedTz)
	}

	// Clear due.
	resp2 := doRequest(t, env, "PATCH", "/spaces/due-rt/tasks/"+taskID, `{"due":null}`)
	assertStatus(t, resp2, http.StatusOK)
	var cleared map[string]any
	readJSON(t, resp2, &cleared)

	if cleared["due"] != nil {
		t.Fatalf("due = %v, want nil after clear", cleared["due"])
	}
}

func TestDueInvalidTimezoneRejected(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "due-inv", "Due Invalid TZ")

	assertStatusClose(t,
		doRequest(t, env, "POST", "/spaces/due-inv/tasks",
			`{"title":"Bad","due":{"at":"2026-06-15T00:00:00Z","timezone":"America/Bogus"}}`),
		http.StatusBadRequest)
}

func TestFixedAccumulatingCompletionCopiesFields(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "fa-copy", "FA Copy")

	// Create task with all user-settable fields.
	task := createTask(t, env, "fa-copy",
		`{"title":"Chore","description":"Do the thing","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=WEEKLY","due":{"at":"2026-04-01T00:00:00Z","timezone":"UTC"},"tags":["weekly"]}`)
	taskID := task["id"].(string)

	// Complete it.
	resp := doRequest(t, env, "PATCH", "/spaces/fa-copy/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	// Find spawned task.
	var spawnedID string
	for _, rel := range updated["relations"].([]any) {
		r := rel.(map[string]any)
		if r["kind"] == "spawns" {
			spawnedID = r["taskId"].(string)
			break
		}
	}
	if spawnedID == "" {
		t.Fatal("expected spawns relation")
	}

	resp2 := doRequest(t, env, "GET", "/spaces/fa-copy/tasks/"+spawnedID, "")
	assertStatus(t, resp2, http.StatusOK)
	var spawned map[string]any
	readJSON(t, resp2, &spawned)

	// Verify fields were copied.
	if spawned["title"] != "Chore" {
		t.Errorf("title = %v, want Chore", spawned["title"])
	}
	if spawned["description"] != "Do the thing" {
		t.Errorf("description = %v, want Do the thing", spawned["description"])
	}
	tags := spawned["tags"].([]any)
	if len(tags) != 1 || tags[0] != "weekly" {
		t.Errorf("tags = %v, want [weekly]", tags)
	}
	if spawned["recurrenceRule"] != "FREQ=WEEKLY" {
		t.Errorf("recurrenceRule = %v, want FREQ=WEEKLY", spawned["recurrenceRule"])
	}

	// Spawned task should have a spawned_by relation back.
	var hasSpawnedBy bool
	for _, rel := range spawned["relations"].([]any) {
		r := rel.(map[string]any)
		if r["kind"] == "spawned_by" && r["taskId"] == taskID {
			hasSpawnedBy = true
			break
		}
	}
	if !hasSpawnedBy {
		t.Error("spawned task should have spawned_by relation")
	}
}

func TestFixedAccumulatingCompletionCopiesRelations(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "fa-rel", "FA Relations")

	// Create a parent task.
	parent := createTask(t, env, "fa-rel", `{"title":"Parent"}`)
	parentID := parent["id"].(string)

	// Create the accumulating task.
	task := createTask(t, env, "fa-rel",
		`{"title":"Child","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=WEEKLY","due":{"at":"2026-04-01T00:00:00Z","timezone":"UTC"}}`)
	taskID := task["id"].(string)

	// Create a parent_of relation: parent -> task.
	resp := doRequest(t, env, "POST", "/spaces/fa-rel/tasks/"+parentID+"/relations",
		`{"kind":"parent_of","taskId":"`+taskID+`"}`)
	assertStatus(t, resp, http.StatusCreated)

	// Create a duplicates relation (should NOT be copied).
	other := createTask(t, env, "fa-rel", `{"title":"Other"}`)
	otherID := other["id"].(string)
	resp = doRequest(t, env, "POST", "/spaces/fa-rel/tasks/"+taskID+"/relations",
		`{"kind":"duplicates","taskId":"`+otherID+`"}`)
	assertStatus(t, resp, http.StatusCreated)

	// Complete the accumulating task.
	resp = doRequest(t, env, "PATCH", "/spaces/fa-rel/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	// Find spawned task.
	var spawnedID string
	for _, rel := range updated["relations"].([]any) {
		r := rel.(map[string]any)
		if r["kind"] == "spawns" {
			spawnedID = r["taskId"].(string)
			break
		}
	}
	if spawnedID == "" {
		t.Fatal("expected spawns relation")
	}

	resp2 := doRequest(t, env, "GET", "/spaces/fa-rel/tasks/"+spawnedID, "")
	assertStatus(t, resp2, http.StatusOK)
	var spawned map[string]any
	readJSON(t, resp2, &spawned)

	// Check relations on spawned task.
	var hasParent, hasDuplicates, hasSpawnedBy bool
	for _, rel := range spawned["relations"].([]any) {
		r := rel.(map[string]any)
		switch r["kind"] {
		case "child_of":
			if r["taskId"] == parentID {
				hasParent = true
			}
		case "duplicates":
			hasDuplicates = true
		case "spawned_by":
			hasSpawnedBy = true
		}
	}

	if !hasParent {
		t.Error("spawned task should inherit child_of relation (copyOnSpawn=true)")
	}
	if hasDuplicates {
		t.Error("spawned task should NOT inherit duplicates relation (copyOnSpawn=false)")
	}
	if !hasSpawnedBy {
		t.Error("spawned task should have spawned_by relation")
	}
}

func TestFixedAccumulatingCronBackfill(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "fa-cron", "FA Cron")

	// Create a weekly task with due date 3 weeks in the past.
	task := createTask(t, env, "fa-cron",
		`{"title":"Weekly","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=WEEKLY","due":{"at":"2026-03-03T00:00:00Z","timezone":"UTC"}}`)
	taskID := task["id"].(string)

	// Directly call processOverdueTasks to simulate the cron.
	env.Handler.ProcessOverdueTasksForTest(context.Background())

	// Fetch the original task — should now be one_off.
	resp := doRequest(t, env, "GET", "/spaces/fa-cron/tasks/"+taskID, "")
	assertStatus(t, resp, http.StatusOK)
	var original map[string]any
	readJSON(t, resp, &original)

	if original["recurrenceType"] != "one_off" {
		t.Fatalf("original recurrenceType = %v, want one_off", original["recurrenceType"])
	}

	// List all tasks in the space — should have the original + missed occurrences + continuation.
	resp2 := doRequest(t, env, "GET", "/spaces/fa-cron/tasks?limit=100", "")
	assertStatus(t, resp2, http.StatusOK)
	var page map[string]any
	readJSON(t, resp2, &page)
	items := page["items"].([]any)

	// Due 2026-03-03, missed: 2026-03-10, 2026-03-17, 2026-03-24.
	// Continuation: next week after now (2026-03-31).
	// Total: 1 original + 3 missed + 1 continuation = 5.
	if len(items) != 5 {
		t.Fatalf("expected exactly 5 tasks after cron backfill (1 original + 3 missed + 1 continuation), got %d", len(items))
	}

	// Find the continuing fixed_accumulating task.
	var continuation map[string]any
	for _, item := range items {
		task := item.(map[string]any)
		if task["recurrenceType"] == "fixed_accumulating" {
			continuation = task
			break
		}
	}
	if continuation == nil {
		t.Fatal("expected one task to still be fixed_accumulating")
	}

	// The continuation's due date should be in the future.
	dueStr, ok := dueAtFromResponse(continuation)
	if !ok {
		t.Fatal("continuation should have a due date")
	}
	parsedDue, err := time.Parse(time.RFC3339, dueStr)
	if err != nil {
		t.Fatalf("parse due: %v", err)
	}
	if !parsedDue.After(time.Now().Truncate(24 * time.Hour)) {
		t.Fatalf("continuation due date should be in the future, got %s", dueStr)
	}
}

func TestFixedAccumulatingCronIdempotent(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "fa-idem", "FA Idempotent")

	// Create a weekly task with due date in the past.
	createTask(t, env, "fa-idem",
		`{"title":"Weekly","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=WEEKLY","due":{"at":"2026-03-10T00:00:00Z","timezone":"UTC"}}`)

	// Run the cron twice.
	env.Handler.ProcessOverdueTasksForTest(context.Background())
	env.Handler.ProcessOverdueTasksForTest(context.Background())

	// List all tasks — second run should not have created duplicates.
	resp := doRequest(t, env, "GET", "/spaces/fa-idem/tasks?limit=100", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := page["items"].([]any)

	// Count fixed_accumulating tasks — should be exactly 1.
	var accumulatingCount int
	for _, item := range items {
		task := item.(map[string]any)
		if task["recurrenceType"] == "fixed_accumulating" {
			accumulatingCount++
		}
	}
	if accumulatingCount != 1 {
		t.Fatalf("expected exactly 1 fixed_accumulating task, got %d", accumulatingCount)
	}
}

func TestFixedAccumulatingCronExhaustedRule(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "fa-exh", "FA Exhausted")

	// Create a task with UNTIL in the past — rule is already exhausted.
	task := createTask(t, env, "fa-exh",
		`{"title":"Expired","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=WEEKLY;UNTIL=20260310T000000Z","due":{"at":"2026-03-03T00:00:00Z","timezone":"UTC"}}`)
	taskID := task["id"].(string)

	env.Handler.ProcessOverdueTasksForTest(context.Background())

	// The original should be converted to one_off.
	resp := doRequest(t, env, "GET", "/spaces/fa-exh/tasks/"+taskID, "")
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	if updated["recurrenceType"] != "one_off" {
		t.Fatalf("expected one_off after exhausted rule, got %v", updated["recurrenceType"])
	}
}

func TestFixedAccumulatingCronNonUTCTimezone(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "fa-tz", "FA TZ")

	// Weekly task due Mondays at midnight Eastern (05:00 UTC during EST).
	// Due date is 2 weeks in the past.
	task := createTask(t, env, "fa-tz",
		`{"title":"Weekly Eastern","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=WEEKLY","due":{"at":"2026-03-09T05:00:00Z","timezone":"America/New_York"}}`)
	taskID := task["id"].(string)

	env.Handler.ProcessOverdueTasksForTest(context.Background())

	// Original should be one_off.
	resp := doRequest(t, env, "GET", "/spaces/fa-tz/tasks/"+taskID, "")
	assertStatus(t, resp, http.StatusOK)
	var original map[string]any
	readJSON(t, resp, &original)
	if original["recurrenceType"] != "one_off" {
		t.Fatalf("original recurrenceType = %v, want one_off", original["recurrenceType"])
	}

	// Find the continuation task.
	resp2 := doRequest(t, env, "GET", "/spaces/fa-tz/tasks?limit=100", "")
	assertStatus(t, resp2, http.StatusOK)
	var page map[string]any
	readJSON(t, resp2, &page)

	var continuation map[string]any
	for _, item := range page["items"].([]any) {
		task := item.(map[string]any)
		if task["recurrenceType"] == "fixed_accumulating" {
			continuation = task
			break
		}
	}
	if continuation == nil {
		t.Fatal("expected a fixed_accumulating continuation task")
	}

	// Continuation's due should be midnight in New York.
	dueStr, ok := dueAtFromResponse(continuation)
	if !ok {
		t.Fatal("continuation should have a due date")
	}
	parsedDue, err := time.Parse(time.RFC3339, dueStr)
	if err != nil {
		t.Fatalf("parse due: %v", err)
	}
	loc, _ := time.LoadLocation("America/New_York")
	localDue := parsedDue.In(loc)
	if localDue.Hour() != 0 || localDue.Minute() != 0 {
		t.Fatalf("expected midnight in America/New_York, got %s", localDue.Format(time.RFC3339))
	}

	// Timezone should be preserved.
	tz, ok := dueTzFromResponse(continuation)
	if !ok || tz != "America/New_York" {
		t.Fatalf("continuation timezone = %v, want America/New_York", tz)
	}
}

func TestFixedAccumulatingCronAfterCompletion(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "fa-race", "FA Race")

	// Create a task with due date in the past.
	task := createTask(t, env, "fa-race",
		`{"title":"Weekly","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=WEEKLY","due":{"at":"2026-03-17T00:00:00Z","timezone":"UTC"}}`)
	taskID := task["id"].(string)

	// Complete the task via HTTP — this converts it to one_off and spawns a new task.
	resp := doRequest(t, env, "PATCH", "/spaces/fa-race/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)

	// Now run the cron — it should NOT re-process the original (already one_off).
	env.Handler.ProcessOverdueTasksForTest(context.Background())

	// Count tasks — should be exactly 2 (original one_off + 1 spawned by completion).
	resp2 := doRequest(t, env, "GET", "/spaces/fa-race/tasks?limit=100", "")
	assertStatus(t, resp2, http.StatusOK)
	var page map[string]any
	readJSON(t, resp2, &page)
	items := page["items"].([]any)

	// Count fixed_accumulating tasks — should be exactly 1 (the one spawned by completion).
	var accumulatingCount int
	for _, item := range items {
		task := item.(map[string]any)
		if task["recurrenceType"] == "fixed_accumulating" {
			accumulatingCount++
		}
	}
	if accumulatingCount != 1 {
		t.Fatalf("expected exactly 1 fixed_accumulating task after completion + cron, got %d", accumulatingCount)
	}
}

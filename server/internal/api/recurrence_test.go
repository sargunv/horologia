package api_test

import (
	"net/http"
	"testing"
	"time"
)

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
		`{"title":"Weekly chore","recurrenceType":"completion_based","recurrenceRule":"FREQ=WEEKLY","dueDate":"2026-03-23"}`)

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
	if updated["dueDate"] == nil {
		t.Fatal("expected dueDate to be set after completion")
	}
	newDue := updated["dueDate"].(string)
	parsedDue, err := time.Parse("2006-01-02", newDue)
	if err != nil {
		t.Fatalf("parse dueDate %q: %v", newDue, err)
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	daysUntilDue := int(parsedDue.Sub(today).Hours() / 24)
	if daysUntilDue < 6 || daysUntilDue > 8 {
		t.Fatalf("expected dueDate ~7 days from now, got %s (%d days)", newDue, daysUntilDue)
	}
}

func TestRecurrenceFixedNonAccumulating(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-fna", "Rec FNA")

	// Create a fixed non-accumulating task due every Saturday.
	task := createTask(t, env, "rec-fna",
		`{"title":"Saturday task","recurrenceType":"fixed_non_accumulating","recurrenceRule":"FREQ=WEEKLY;BYDAY=SA","dueDate":"2026-03-21"}`)

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
	if updated["dueDate"] == nil {
		t.Fatal("expected dueDate to be set")
	}
	parsedDue, err := time.Parse("2006-01-02", updated["dueDate"].(string))
	if err != nil {
		t.Fatalf("parse dueDate: %v", err)
	}
	if parsedDue.Weekday() != time.Saturday {
		t.Fatalf("expected dueDate to be a Saturday, got %s (%s)", updated["dueDate"], parsedDue.Weekday())
	}
	if !parsedDue.After(time.Now().Truncate(24 * time.Hour)) {
		t.Fatalf("expected dueDate after today, got %s", updated["dueDate"])
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
		`{"title":"Monthly task","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=MONTHLY","dueDate":"2026-04-01"}`)
	taskID := task["id"].(string)

	// Complete the task.
	resp := doRequest(t, env, "PATCH", "/spaces/rec-fa/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	// Should be reset to initial status (same as fixed_non_accumulating).
	// The "accumulating" part is about what happens when due dates pass without
	// completion (cron spawns new tasks), not about completion behavior.
	if updated["status"] != "todo" {
		t.Fatalf("got status %v, want todo", updated["status"])
	}
	// lastCompletedAt should be set.
	if updated["lastCompletedAt"] == nil {
		t.Fatal("expected lastCompletedAt to be set")
	}
	// Due date should advance to next monthly occurrence.
	if updated["dueDate"] == nil {
		t.Fatal("expected dueDate to be set after completion")
	}
}

func TestRecurrenceNoRetriggerOnNonTransition(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-nrt", "Rec No Retrigger")

	task := createTask(t, env, "rec-nrt",
		`{"title":"Weekly chore","recurrenceType":"completion_based","recurrenceRule":"FREQ=WEEKLY","dueDate":"2026-03-23"}`)
	taskID := task["id"].(string)

	// Complete the task.
	resp := doRequest(t, env, "PATCH", "/spaces/rec-nrt/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var first map[string]any
	readJSON(t, resp, &first)
	firstDue := first["dueDate"]

	// Update title while already reset to "todo" — should not change due date.
	resp2 := doRequest(t, env, "PATCH", "/spaces/rec-nrt/tasks/"+taskID, `{"title":"Updated title"}`)
	assertStatus(t, resp2, http.StatusOK)
	var second map[string]any
	readJSON(t, resp2, &second)

	if second["dueDate"] != firstDue {
		t.Fatalf("dueDate changed on non-status update: got %v, want %v", second["dueDate"], firstDue)
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

func TestRecurrenceCountExhaustion(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-cnt", "Rec Count")

	// Create a completion_based task with COUNT=1. After one completion the
	// rule is exhausted — the task should stay at the completion status.
	task := createTask(t, env, "rec-cnt",
		`{"title":"Once only","recurrenceType":"completion_based","recurrenceRule":"FREQ=WEEKLY;COUNT=1","dueDate":"2026-03-23"}`)
	taskID := task["id"].(string)

	resp := doRequest(t, env, "PATCH", "/spaces/rec-cnt/tasks/"+taskID, `{"status":"done"}`)
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
		`{"title":"Every 3 days","recurrenceType":"completion_based","recurrenceRule":"FREQ=DAILY;INTERVAL=3","dueDate":"2026-03-23"}`)
	taskID := task["id"].(string)

	// First completion — should reset to todo with due date 3 days from now.
	resp1 := doRequest(t, env, "PATCH", "/spaces/rec-dbl/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp1, http.StatusOK)
	var first map[string]any
	readJSON(t, resp1, &first)

	if first["status"] != "todo" {
		t.Fatalf("first completion: got status %v, want todo", first["status"])
	}
	firstDue, err := time.Parse("2006-01-02", first["dueDate"].(string))
	if err != nil {
		t.Fatalf("parse first dueDate: %v", err)
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	daysUntil := int(firstDue.Sub(today).Hours() / 24)
	if daysUntil < 2 || daysUntil > 4 {
		t.Fatalf("first completion: expected dueDate ~3 days from now, got %s (%d days)", first["dueDate"], daysUntil)
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
	secondDue, err2 := time.Parse("2006-01-02", second["dueDate"].(string))
	if err2 != nil {
		t.Fatalf("parse second dueDate: %v", err2)
	}
	if secondDue != firstDue {
		t.Fatalf("second due %s should equal first due %s (same completion time)", second["dueDate"], first["dueDate"])
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
		`{"title":"Monthly","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=MONTHLY","dueDate":"2026-01-01"}`)
	taskID := task["id"].(string)

	resp := doRequest(t, env, "PATCH", "/spaces/rec-fap/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	// Due date should advance to the next monthly occurrence after now
	// (from the schedule anchor of 2026-01-01).
	if updated["dueDate"] == nil {
		t.Fatal("expected dueDate to be set")
	}
	newDue := updated["dueDate"].(string)
	parsedDue, err := time.Parse("2006-01-02", newDue)
	if err != nil {
		t.Fatalf("parse dueDate: %v", err)
	}
	if !parsedDue.After(time.Now().Truncate(24 * time.Hour)) {
		t.Fatalf("expected dueDate after today, got %s", newDue)
	}
	// Should be on the 1st of a month (FREQ=MONTHLY from Jan 1).
	if parsedDue.Day() != 1 {
		t.Fatalf("expected dueDate on the 1st, got day %d", parsedDue.Day())
	}
}

func TestRecurrenceCountNotExhausted(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-cnt2", "Rec Count 2")

	// COUNT=2: first completion should reset (one occurrence remains).
	task := createTask(t, env, "rec-cnt2",
		`{"title":"Twice","recurrenceType":"completion_based","recurrenceRule":"FREQ=WEEKLY;COUNT=2","dueDate":"2026-03-23"}`)
	taskID := task["id"].(string)

	resp := doRequest(t, env, "PATCH", "/spaces/rec-cnt2/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	// COUNT=2 means two occurrences from DTSTART. The first is at DTSTART (now),
	// excluded by After(now, false). The second is now+7 days. Task should reset.
	if updated["status"] != "todo" {
		t.Fatalf("got status %v, want todo (COUNT=2 not yet exhausted)", updated["status"])
	}
	if updated["dueDate"] == nil {
		t.Fatal("expected dueDate to be set")
	}
}

func TestRecurrenceCountOverLimitRejected(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "rec-col", "Rec Count Limit")

	// COUNT=1000 should be accepted.
	assertStatusClose(t,
		doRequest(t, env, "POST", "/spaces/rec-col/tasks",
			`{"title":"OK","recurrenceType":"completion_based","recurrenceRule":"FREQ=DAILY;COUNT=1000"}`),
		http.StatusCreated)

	// COUNT=1001 should be rejected.
	assertStatusClose(t,
		doRequest(t, env, "POST", "/spaces/rec-col/tasks",
			`{"title":"Bad","recurrenceType":"completion_based","recurrenceRule":"FREQ=DAILY;COUNT=1001"}`),
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

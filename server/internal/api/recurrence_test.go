package api_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/sargunv/horologia/server/internal/taskengine"
)

// dueAtFromResponse extracts the "at" string from a task's nested "due" field.
// Returns ("", false) if due is nil.
func dueAtFromResponse(task map[string]any) (string, bool) {
	due, ok := task["due"].(map[string]any)
	if !ok {
		return "", false
	}
	at, ok := due["at"].(string)
	return at, ok
}

// dueTzFromResponse extracts the "timezone" string from a task's nested "due" field.
// Returns ("", false) if due is nil.
func dueTzFromResponse(task map[string]any) (string, bool) {
	due, ok := task["due"].(map[string]any)
	if !ok {
		return "", false
	}
	tz, ok := due["timezone"].(string)
	return tz, ok
}

func TestRecurrenceValidation(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	t.Run("one off default", func(t *testing.T) { testRecurrenceOneOffDefault(t, env) })
	t.Run("one off rejects rule", func(t *testing.T) { testRecurrenceOneOffRejectsRule(t, env) })
	t.Run("requires rule", func(t *testing.T) { testRecurrenceRequiresRule(t, env) })
	t.Run("invalid rrule", func(t *testing.T) { testRecurrenceInvalidRRule(t, env) })
	t.Run("rejects sub day frequency", func(t *testing.T) { testRecurrenceRejectsSubDayFrequency(t, env) })
	t.Run("rejects unsupported property", func(t *testing.T) { testRecurrenceRejectsUnsupportedProperty(t, env) })
	t.Run("update type from one off", func(t *testing.T) { testRecurrenceUpdateTypeFromOneOff(t, env) })
	t.Run("update type requires rule", func(t *testing.T) { testRecurrenceUpdateTypeRequiresRule(t, env) })
	t.Run("cross space isolation", func(t *testing.T) { testRecurrenceCrossSpaceIsolation(t, env) })
	t.Run("on dependency rejects rule", func(t *testing.T) { testRecurrenceOnDependencyRejectsRule(t, env) })
	t.Run("count rejected", func(t *testing.T) { testRecurrenceCountRejected(t, env) })
	t.Run("due invalid timezone rejected", func(t *testing.T) { testDueInvalidTimezoneRejected(t, env) })
}

func TestRecurrenceCompletionFlows(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	t.Run("completion based", func(t *testing.T) { testRecurrenceCompletionBased(t, env) })
	t.Run("fixed non accumulating", func(t *testing.T) { testRecurrenceFixedNonAccumulating(t, env) })
	t.Run("one off completion", func(t *testing.T) { testRecurrenceOneOffCompletion(t, env) })
	t.Run("fixed accumulating completion", func(t *testing.T) { testRecurrenceFixedAccumulatingCompletion(t, env) })
	t.Run("no retrigger on non transition", func(t *testing.T) { testRecurrenceNoRetriggerOnNonTransition(t, env) })
	t.Run("on dependency trigger", func(t *testing.T) { testRecurrenceOnDependencyTrigger(t, env) })
	t.Run("triggers relation kind", func(t *testing.T) { testRecurrenceTriggersRelationKind(t, env) })
	t.Run("on dependency direct completion", func(t *testing.T) { testRecurrenceOnDependencyDirectCompletion(t, env) })
	t.Run("until exhaustion", func(t *testing.T) { testRecurrenceUntilExhaustion(t, env) })
	t.Run("double completion", func(t *testing.T) { testRecurrenceDoubleCompletion(t, env) })
	t.Run("auto clean rule on type change", func(t *testing.T) { testRecurrenceAutoCleanRuleOnTypeChange(t, env) })
	t.Run("fixed accumulating due date advances", func(t *testing.T) { testRecurrenceFixedAccumulatingDueDateAdvances(t, env) })
	t.Run("same status no retrigger", func(t *testing.T) { testRecurrenceSameStatusNoRetrigger(t, env) })
	t.Run("completion based non utc timezone", func(t *testing.T) { testRecurrenceCompletionBasedNonUTCTimezone(t, env) })
	t.Run("fixed non accumulating non utc timezone", func(t *testing.T) { testRecurrenceFixedNonAccumulatingNonUTCTimezone(t, env) })
	t.Run("due timezone round trip", func(t *testing.T) { testDueTimezoneRoundTrip(t, env) })
	t.Run("fixed accumulating completion copies fields", func(t *testing.T) { testFixedAccumulatingCompletionCopiesFields(t, env) })
	t.Run("fixed accumulating completion copies relations", func(t *testing.T) { testFixedAccumulatingCompletionCopiesRelations(t, env) })
}

func TestRecurrenceCronFlows(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	t.Run("fixed accumulating cron backfill", func(t *testing.T) { testFixedAccumulatingCronBackfill(t, env) })
	t.Run("fixed accumulating cron idempotent", func(t *testing.T) { testFixedAccumulatingCronIdempotent(t, env) })
	t.Run("fixed accumulating cron exhausted rule", func(t *testing.T) { testFixedAccumulatingCronExhaustedRule(t, env) })
	t.Run("fixed accumulating cron non utc timezone", func(t *testing.T) { testFixedAccumulatingCronNonUTCTimezone(t, env) })
	t.Run("fixed accumulating cron after completion", func(t *testing.T) { testFixedAccumulatingCronAfterCompletion(t, env) })
}

func testRecurrenceOneOffDefault(t *testing.T, env *testEnv) {
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

func testRecurrenceOneOffRejectsRule(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-rej", "Rec Reject")

	assertStatusClose(t,
		doRequest(t, env, "POST", "/spaces/rec-rej/tasks",
			`{"title":"Bad","recurrenceType":"one_off","recurrenceRule":"FREQ=WEEKLY"}`),
		http.StatusBadRequest)
}

func testRecurrenceRequiresRule(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-req", "Rec Require")

	assertStatusClose(t,
		doRequest(t, env, "POST", "/spaces/rec-req/tasks",
			`{"title":"Bad","recurrenceType":"completion_based"}`),
		http.StatusBadRequest)
}

func testRecurrenceInvalidRRule(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-inv", "Rec Invalid")

	assertStatusClose(t,
		doRequest(t, env, "POST", "/spaces/rec-inv/tasks",
			`{"title":"Bad","recurrenceType":"completion_based","recurrenceRule":"FREQ=BOGUS"}`),
		http.StatusBadRequest)
}

func testRecurrenceRejectsSubDayFrequency(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-sub", "Rec SubDay")

	assertStatusClose(t,
		doRequest(t, env, "POST", "/spaces/rec-sub/tasks",
			`{"title":"Bad","recurrenceType":"completion_based","recurrenceRule":"FREQ=HOURLY"}`),
		http.StatusBadRequest)
}

func testRecurrenceRejectsUnsupportedProperty(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-prop", "Rec Prop")

	assertStatusClose(t,
		doRequest(t, env, "POST", "/spaces/rec-prop/tasks",
			`{"title":"Bad","recurrenceType":"completion_based","recurrenceRule":"FREQ=DAILY;BYHOUR=10"}`),
		http.StatusBadRequest)
}

func testRecurrenceCompletionBased(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-cb", "Rec CB")

	// Create a completion-based task recurring weekly.
	pastDue := time.Now().AddDate(0, 0, -7).Format(time.DateOnly)
	task := createTask(t, env, "rec-cb",
		`{"title":"Weekly chore","recurrenceType":"completion_based","recurrenceRule":"FREQ=WEEKLY","due":{"at":"`+pastDue+`","timezone":"UTC"}}`)

	if task["recurrenceType"] != "completion_based" {
		t.Fatalf("got recurrenceType %v, want completion_based", task["recurrenceType"])
	}
	if task["recurrenceRule"] != "FREQ=WEEKLY" {
		t.Fatalf("got recurrenceRule %v, want FREQ=WEEKLY", task["recurrenceRule"])
	}
	taskID := jsonAs[string](t, task["id"])

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
	newDue := jsonAs[string](t, jsonAs[map[string]any](t, updated["due"])["at"])
	parsedDue, err := time.Parse(time.DateOnly, newDue)
	if err != nil {
		t.Fatalf("parse dueAt %q: %v", newDue, err)
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	daysUntilDue := int(parsedDue.Sub(today).Hours() / 24)
	if daysUntilDue < 6 || daysUntilDue > 8 {
		t.Fatalf("expected dueAt ~7 days from now, got %s (%d days)", newDue, daysUntilDue)
	}
}

func testRecurrenceFixedNonAccumulating(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-fna", "Rec FNA")

	// Create a fixed non-accumulating task due every Saturday.
	// Find a past Saturday that's at least 8 days ago so the next occurrence is strictly in the future.
	pastSat := time.Now().AddDate(0, 0, -8)
	for pastSat.Weekday() != time.Saturday {
		pastSat = pastSat.AddDate(0, 0, -1)
	}
	pastSatStr := pastSat.Format(time.DateOnly)
	task := createTask(t, env, "rec-fna",
		`{"title":"Saturday task","recurrenceType":"fixed_non_accumulating","recurrenceRule":"FREQ=WEEKLY;BYDAY=SA","due":{"at":"`+pastSatStr+`","timezone":"UTC"}}`)

	taskID := jsonAs[string](t, task["id"])

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
	parsedDue, err := time.Parse(time.DateOnly, jsonAs[string](t, jsonAs[map[string]any](t, updated["due"])["at"]))
	if err != nil {
		t.Fatalf("parse dueAt: %v", err)
	}
	if parsedDue.Weekday() != time.Saturday {
		t.Fatalf("expected dueAt to be a Saturday, got %s (%s)", updated["due"], parsedDue.Weekday())
	}
	if parsedDue.Before(time.Now().Truncate(24 * time.Hour)) {
		t.Fatalf("expected dueAt after today, got %s", updated["due"])
	}
}

func testRecurrenceOneOffCompletion(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-oc", "Rec One Off Complete")

	task := createTask(t, env, "rec-oc", `{"title":"One-off task"}`)
	taskID := jsonAs[string](t, task["id"])

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

func testRecurrenceFixedAccumulatingCompletion(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-fa", "Rec FA")

	nextMonth := time.Date(time.Now().Year(), time.Now().Month()+1, 1, 0, 0, 0, 0, time.UTC)
	nextMonthStr := nextMonth.Format(time.DateOnly)
	task := createTask(t, env, "rec-fa",
		`{"title":"Monthly task","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=MONTHLY","due":{"at":"`+nextMonthStr+`","timezone":"UTC"}}`)
	taskID := jsonAs[string](t, task["id"])

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
	for _, rel := range jsonAs[[]any](t, updated["relations"]) {
		r := jsonAs[map[string]any](t, rel)
		if r["kind"] == "spawns" {
			spawnsRelation = r
			break
		}
	}
	if spawnsRelation == nil {
		t.Fatal("expected a 'spawns' relation on the completed task")
	}

	// Fetch the spawned task.
	spawnedID := jsonAs[string](t, spawnsRelation["relatedTaskId"])
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

func testRecurrenceNoRetriggerOnNonTransition(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-nrt", "Rec No Retrigger")

	pastDue := time.Now().AddDate(0, 0, -7).Format(time.DateOnly)
	task := createTask(t, env, "rec-nrt",
		`{"title":"Weekly chore","recurrenceType":"completion_based","recurrenceRule":"FREQ=WEEKLY","due":{"at":"`+pastDue+`","timezone":"UTC"}}`)
	taskID := jsonAs[string](t, task["id"])

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

func testRecurrenceOnDependencyTrigger(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-dep", "Rec Dep")

	// Create task A (one_off) and task B (on_dependency).
	taskA := createTask(t, env, "rec-dep", `{"title":"Load dishwasher"}`)
	taskB := createTask(t, env, "rec-dep", `{"title":"Unload dishwasher","recurrenceType":"on_dependency"}`)
	taskAID := jsonAs[string](t, taskA["id"])
	taskBID := jsonAs[string](t, taskB["id"])

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

func testRecurrenceTriggersRelationKind(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-trk", "Rec Trigger Kind")

	taskA := createTask(t, env, "rec-trk", `{"title":"Task A"}`)
	taskB := createTask(t, env, "rec-trk", `{"title":"Task B"}`)
	taskAID := jsonAs[string](t, taskA["id"])
	taskBID := jsonAs[string](t, taskB["id"])

	// Create triggers relation.
	createRelation(t, env, "rec-trk", taskAID, "triggers", taskBID)

	// A should show "triggers" B.
	relsA := assertTaskRelations(t, env, "rec-trk", taskAID, 1)
	assertRelationKind(t, relsA[0], "triggers", taskBID)

	// B should show "triggered_by" A.
	relsB := assertTaskRelations(t, env, "rec-trk", taskBID, 1)
	assertRelationKind(t, relsB[0], "triggered_by", taskAID)
}

func testRecurrenceUpdateTypeFromOneOff(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-upd", "Rec Update")

	task := createTask(t, env, "rec-upd", `{"title":"Was one-off"}`)
	taskID := jsonAs[string](t, task["id"])

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

func testRecurrenceUpdateTypeRequiresRule(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-ur", "Rec Update Req")

	task := createTask(t, env, "rec-ur", `{"title":"Was one-off"}`)
	taskID := jsonAs[string](t, task["id"])

	// Changing to completion_based without a rule should fail.
	assertStatusClose(t,
		doRequest(t, env, "PATCH", "/spaces/rec-ur/tasks/"+taskID,
			`{"recurrenceType":"completion_based"}`),
		http.StatusBadRequest)
}

func testRecurrenceCrossSpaceIsolation(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-iso-a", "Rec Iso A")
	createSpace(t, env, "rec-iso-b", "Rec Iso B")

	task := createTask(t, env, "rec-iso-a",
		`{"title":"Recurring","recurrenceType":"completion_based","recurrenceRule":"FREQ=WEEKLY"}`)

	// Task in space A should not be accessible from space B.
	taskID := jsonAs[string](t, task["id"])
	assertStatusClose(t,
		doRequest(t, env, "GET", "/spaces/rec-iso-b/tasks/"+taskID, ""),
		http.StatusNotFound)
}

func testRecurrenceOnDependencyDirectCompletion(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-odc", "Rec OnDep Complete")

	// Create an on_dependency task and complete it directly.
	// It should stay at "done" — on_dependency reset only happens when
	// the trigger source completes, not when the task itself is completed.
	task := createTask(t, env, "rec-odc", `{"title":"Dependent task","recurrenceType":"on_dependency"}`)
	taskID := jsonAs[string](t, task["id"])

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

func testRecurrenceOnDependencyRejectsRule(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-odr", "Rec OnDep Rule")

	assertStatusClose(t,
		doRequest(t, env, "POST", "/spaces/rec-odr/tasks",
			`{"title":"Bad","recurrenceType":"on_dependency","recurrenceRule":"FREQ=WEEKLY"}`),
		http.StatusBadRequest)
}

func testRecurrenceUntilExhaustion(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-unt", "Rec Until")

	// Create a completion_based task with UNTIL in the past. The rule is
	// already exhausted — the task should stay at the completion status.
	pastDue := time.Now().AddDate(0, 0, -7).Format(time.DateOnly)
	untilDate := time.Now().AddDate(0, 0, -30).Format("20060102T150405Z")
	task := createTask(t, env, "rec-unt",
		`{"title":"Expired","recurrenceType":"completion_based","recurrenceRule":"FREQ=WEEKLY;UNTIL=`+untilDate+`","due":{"at":"`+pastDue+`","timezone":"UTC"}}`)
	taskID := jsonAs[string](t, task["id"])

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

func testRecurrenceDoubleCompletion(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-dbl", "Rec Double")

	pastDue := time.Now().AddDate(0, 0, -7).Format(time.DateOnly)
	task := createTask(t, env, "rec-dbl",
		`{"title":"Every 3 days","recurrenceType":"completion_based","recurrenceRule":"FREQ=DAILY;INTERVAL=3","due":{"at":"`+pastDue+`","timezone":"UTC"}}`)
	taskID := jsonAs[string](t, task["id"])

	// First completion — should reset to todo with due date 3 days from now.
	resp1 := doRequest(t, env, "PATCH", "/spaces/rec-dbl/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp1, http.StatusOK)
	var first map[string]any
	readJSON(t, resp1, &first)

	if first["status"] != "todo" {
		t.Fatalf("first completion: got status %v, want todo", first["status"])
	}
	firstDue, err := time.Parse(time.DateOnly, jsonAs[string](t, jsonAs[map[string]any](t, first["due"])["at"]))
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
	secondDue, err2 := time.Parse(time.DateOnly, secondDueStr)
	if err2 != nil {
		t.Fatalf("parse second due.at: %v", err2)
	}
	if secondDue != firstDue {
		t.Fatalf("second due %s should equal first due %s (same completion time)", secondDueStr, first["due"])
	}
}

func testRecurrenceAutoCleanRuleOnTypeChange(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-acr", "Rec Auto Clean")

	// Create a completion_based task with a rule.
	task := createTask(t, env, "rec-acr",
		`{"title":"Was recurring","recurrenceType":"completion_based","recurrenceRule":"FREQ=WEEKLY"}`)
	taskID := jsonAs[string](t, task["id"])

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

func testRecurrenceFixedAccumulatingDueDateAdvances(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-fap", "Rec FA Advance")

	// Use a past due date so the next occurrence is clearly in the future.
	pastFirst := time.Date(time.Now().Year(), time.Now().Month()-3, 1, 0, 0, 0, 0, time.UTC)
	pastFirstStr := pastFirst.Format(time.DateOnly)
	task := createTask(t, env, "rec-fap",
		`{"title":"Monthly","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=MONTHLY","due":{"at":"`+pastFirstStr+`","timezone":"UTC"}}`)
	taskID := jsonAs[string](t, task["id"])

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
	for _, rel := range jsonAs[[]any](t, updated["relations"]) {
		r := jsonAs[map[string]any](t, rel)
		if r["kind"] == "spawns" {
			spawnedID = jsonAs[string](t, r["relatedTaskId"])
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
	newDue := jsonAs[string](t, jsonAs[map[string]any](t, spawned["due"])["at"])
	parsedDue, err := time.Parse(time.DateOnly, newDue)
	if err != nil {
		t.Fatalf("parse dueAt: %v", err)
	}
	if parsedDue.Before(time.Now().Truncate(24 * time.Hour)) {
		t.Fatalf("expected dueAt after today, got %s", newDue)
	}
	if parsedDue.Day() != 1 {
		t.Fatalf("expected dueAt on the 1st, got day %d", parsedDue.Day())
	}
	if spawned["recurrenceType"] != "fixed_accumulating" {
		t.Fatalf("spawned recurrenceType = %v, want fixed_accumulating", spawned["recurrenceType"])
	}
}

func testRecurrenceCountRejected(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-cnt", "Rec Count")

	// COUNT is not supported — use UNTIL instead.
	assertStatusClose(t,
		doRequest(t, env, "POST", "/spaces/rec-cnt/tasks",
			`{"title":"Bad","recurrenceType":"completion_based","recurrenceRule":"FREQ=DAILY;COUNT=5"}`),
		http.StatusBadRequest)
}

func testRecurrenceSameStatusNoRetrigger(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-snr", "Rec Same No Retrigger")

	// Complete a one_off task, then PATCH status "done" again.
	// The second PATCH should not update lastCompletedAt.
	task := createTask(t, env, "rec-snr", `{"title":"One-off"}`)
	taskID := jsonAs[string](t, task["id"])

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

func testRecurrenceCompletionBasedNonUTCTimezone(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-tz", "Rec TZ")

	// Create a completion-based task with America/New_York timezone.
	pastDue := time.Now().AddDate(0, 0, -7).Format(time.DateOnly)
	task := createTask(t, env, "rec-tz",
		`{"title":"Weekly chore","recurrenceType":"completion_based","recurrenceRule":"FREQ=WEEKLY","due":{"at":"`+pastDue+`","timezone":"America/New_York"}}`)
	taskID := jsonAs[string](t, task["id"])

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

	// The new due date should be ~7 days from now.
	dueStr, ok := dueAtFromResponse(updated)
	if !ok {
		t.Fatal("expected due to be set after completion")
	}
	parsedDue, err := time.Parse(time.DateOnly, dueStr)
	if err != nil {
		t.Fatalf("parse due.at: %v", err)
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	daysUntilDue := int(parsedDue.Sub(today).Hours() / 24)
	if daysUntilDue < 6 || daysUntilDue > 8 {
		t.Fatalf("expected dueAt ~7 days from now, got %s (%d days)", dueStr, daysUntilDue)
	}
}

func testRecurrenceFixedNonAccumulatingNonUTCTimezone(t *testing.T, env *testEnv) {
	createSpace(t, env, "rec-ftz", "Rec Fixed TZ")

	// Create a fixed non-accumulating task recurring every Saturday in Los Angeles.
	// Find a past Saturday that's at least 8 days ago so the next occurrence is strictly in the future.
	pastSat := time.Now().AddDate(0, 0, -8)
	for pastSat.Weekday() != time.Saturday {
		pastSat = pastSat.AddDate(0, 0, -1)
	}
	pastSatStr := pastSat.Format(time.DateOnly)
	task := createTask(t, env, "rec-ftz",
		`{"title":"Saturday task","recurrenceType":"fixed_non_accumulating","recurrenceRule":"FREQ=WEEKLY;BYDAY=SA","due":{"at":"`+pastSatStr+`","timezone":"America/Los_Angeles"}}`)
	taskID := jsonAs[string](t, task["id"])

	// Complete the task.
	resp := doRequest(t, env, "PATCH", "/spaces/rec-ftz/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	if updated["status"] != "todo" {
		t.Fatalf("got status %v, want todo", updated["status"])
	}

	// Due date should be a Saturday.
	dueStr, ok := dueAtFromResponse(updated)
	if !ok {
		t.Fatal("expected due to be set")
	}
	parsedDue, err := time.Parse(time.DateOnly, dueStr)
	if err != nil {
		t.Fatalf("parse due.at: %v", err)
	}
	if parsedDue.Weekday() != time.Saturday {
		t.Fatalf("expected Saturday, got %s (%s)", dueStr, parsedDue.Weekday())
	}
}

func testDueTimezoneRoundTrip(t *testing.T, env *testEnv) {
	createSpace(t, env, "due-rt", "Due Round Trip")

	// Create with a non-UTC timezone.
	futureDate := time.Now().AddDate(0, 0, 30).Format(time.DateOnly)
	task := createTask(t, env, "due-rt",
		`{"title":"TZ task","due":{"at":"`+futureDate+`","timezone":"America/New_York"}}`)

	// Verify round-trip: both at and timezone returned correctly.
	dueStr, ok := dueAtFromResponse(task)
	if !ok {
		t.Fatal("expected due to be set")
	}
	if dueStr != futureDate {
		t.Fatalf("due.at = %v, want %s", dueStr, futureDate)
	}
	tz, ok := dueTzFromResponse(task)
	if !ok || tz != "America/New_York" {
		t.Fatalf("due.timezone = %v, want America/New_York", tz)
	}

	// Update to a different timezone.
	taskID := jsonAs[string](t, task["id"])
	resp := doRequest(t, env, "PATCH", "/spaces/due-rt/tasks/"+taskID,
		`{"due":{"at":"`+futureDate+`","timezone":"America/Los_Angeles"}}`)
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

func testDueInvalidTimezoneRejected(t *testing.T, env *testEnv) {
	createSpace(t, env, "due-inv", "Due Invalid TZ")

	futureDate := time.Now().AddDate(0, 0, 30).Format(time.DateOnly)
	assertStatusClose(t,
		doRequest(t, env, "POST", "/spaces/due-inv/tasks",
			`{"title":"Bad","due":{"at":"`+futureDate+`","timezone":"America/Bogus"}}`),
		http.StatusBadRequest)
}

func testFixedAccumulatingCompletionCopiesFields(t *testing.T, env *testEnv) {
	createSpace(t, env, "fa-copy", "FA Copy")

	// Create task with all user-settable fields.
	nextMonth := time.Date(time.Now().Year(), time.Now().Month()+1, 1, 0, 0, 0, 0, time.UTC)
	nextMonthStr := nextMonth.Format(time.DateOnly)
	task := createTask(t, env, "fa-copy",
		`{"title":"Chore","description":"Do the thing","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=WEEKLY","due":{"at":"`+nextMonthStr+`","timezone":"UTC"},"tags":["weekly"]}`)
	taskID := jsonAs[string](t, task["id"])

	// Complete it.
	resp := doRequest(t, env, "PATCH", "/spaces/fa-copy/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	// Find spawned task.
	var spawnedID string
	for _, rel := range jsonAs[[]any](t, updated["relations"]) {
		r := jsonAs[map[string]any](t, rel)
		if r["kind"] == "spawns" {
			spawnedID = jsonAs[string](t, r["relatedTaskId"])
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
	tags := jsonAs[[]any](t, spawned["tags"])
	if len(tags) != 1 || tags[0] != "weekly" {
		t.Errorf("tags = %v, want [weekly]", tags)
	}
	if spawned["recurrenceRule"] != "FREQ=WEEKLY" {
		t.Errorf("recurrenceRule = %v, want FREQ=WEEKLY", spawned["recurrenceRule"])
	}

	// Spawned task should have a spawned_by relation back.
	var hasSpawnedBy bool
	for _, rel := range jsonAs[[]any](t, spawned["relations"]) {
		r := jsonAs[map[string]any](t, rel)
		if r["kind"] == "spawned_by" && r["relatedTaskId"] == taskID {
			hasSpawnedBy = true
			break
		}
	}
	if !hasSpawnedBy {
		t.Error("spawned task should have spawned_by relation")
	}
}

func testFixedAccumulatingCompletionCopiesRelations(t *testing.T, env *testEnv) {
	createSpace(t, env, "fa-rel", "FA Relations")

	// Create a parent task.
	parent := createTask(t, env, "fa-rel", `{"title":"Parent"}`)
	parentID := jsonAs[string](t, parent["id"])

	// Create the accumulating task.
	nextMonth := time.Date(time.Now().Year(), time.Now().Month()+1, 1, 0, 0, 0, 0, time.UTC)
	nextMonthStr := nextMonth.Format(time.DateOnly)
	task := createTask(t, env, "fa-rel",
		`{"title":"Child","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=WEEKLY","due":{"at":"`+nextMonthStr+`","timezone":"UTC"}}`)
	taskID := jsonAs[string](t, task["id"])

	// Create a parent_of relation: parent -> task.
	resp := doRequest(t, env, "POST", "/spaces/fa-rel/tasks/"+parentID+"/relations",
		`{"kind":"parent_of","relatedTaskId":"`+taskID+`"}`)
	assertStatus(t, resp, http.StatusCreated)

	// Create a duplicates relation (should NOT be copied).
	other := createTask(t, env, "fa-rel", `{"title":"Other"}`)
	otherID := jsonAs[string](t, other["id"])
	resp = doRequest(t, env, "POST", "/spaces/fa-rel/tasks/"+taskID+"/relations",
		`{"kind":"duplicates","relatedTaskId":"`+otherID+`"}`)
	assertStatus(t, resp, http.StatusCreated)

	// Complete the accumulating task.
	resp = doRequest(t, env, "PATCH", "/spaces/fa-rel/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	// Find spawned task.
	var spawnedID string
	for _, rel := range jsonAs[[]any](t, updated["relations"]) {
		r := jsonAs[map[string]any](t, rel)
		if r["kind"] == "spawns" {
			spawnedID = jsonAs[string](t, r["relatedTaskId"])
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
	for _, rel := range jsonAs[[]any](t, spawned["relations"]) {
		r := jsonAs[map[string]any](t, rel)
		switch r["kind"] {
		case "child_of":
			if r["relatedTaskId"] == parentID {
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

func testFixedAccumulatingCronBackfill(t *testing.T, env *testEnv) {
	createSpace(t, env, "fa-cron", "FA Cron")

	// Create a weekly task with due date ~3.5 weeks in the past (dynamically computed).
	// Using 25 days ensures we clearly have 3 missed weekly occurrences in the past
	// and the 4th occurrence lands in the future as the continuation.
	dueDate := time.Now().AddDate(0, 0, -25).Format(time.DateOnly)
	task := createTask(t, env, "fa-cron",
		`{"title":"Weekly","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=WEEKLY","due":{"at":"`+dueDate+`","timezone":"UTC"}}`)
	taskID := jsonAs[string](t, task["id"])

	// Directly call processOverdueTasks to simulate the cron.
	must(t, taskengine.ProcessOverdueTasks(t.Context(), env.pool, time.Now(), nil))

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
	items := jsonAs[[]any](t, page["items"])

	// Due date is ~25 days ago. Missed occurrences at due+7d (~18d ago), due+14d (~11d ago),
	// due+21d (~4d ago). Continuation at due+28d (~3d in future).
	// Total: 1 original + 3 missed + 1 continuation = 5.
	if len(items) != 5 {
		t.Fatalf("expected exactly 5 tasks after cron backfill (1 original + 3 missed + 1 continuation), got %d", len(items))
	}

	// Find the continuing fixed_accumulating task.
	var continuation map[string]any
	for _, item := range items {
		task := jsonAs[map[string]any](t, item)
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
	parsedDue, err := time.Parse(time.DateOnly, dueStr)
	if err != nil {
		t.Fatalf("parse due: %v", err)
	}
	if parsedDue.Before(time.Now().Truncate(24 * time.Hour)) {
		t.Fatalf("continuation due date should be in the future, got %s", dueStr)
	}
}

func testFixedAccumulatingCronIdempotent(t *testing.T, env *testEnv) {
	createSpace(t, env, "fa-idem", "FA Idempotent")

	// Create a weekly task with due date in the past (dynamically computed).
	idemDue := time.Now().AddDate(0, 0, -14).Format(time.DateOnly)
	createTask(t, env, "fa-idem",
		`{"title":"Weekly","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=WEEKLY","due":{"at":"`+idemDue+`","timezone":"UTC"}}`)

	// Run the cron twice.
	must(t, taskengine.ProcessOverdueTasks(t.Context(), env.pool, time.Now(), nil))
	must(t, taskengine.ProcessOverdueTasks(t.Context(), env.pool, time.Now(), nil))

	// List all tasks — second run should not have created duplicates.
	resp := doRequest(t, env, "GET", "/spaces/fa-idem/tasks?limit=100", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := jsonAs[[]any](t, page["items"])

	// Count fixed_accumulating tasks — should be exactly 1.
	var accumulatingCount int
	for _, item := range items {
		task := jsonAs[map[string]any](t, item)
		if task["recurrenceType"] == "fixed_accumulating" {
			accumulatingCount++
		}
	}
	if accumulatingCount != 1 {
		t.Fatalf("expected exactly 1 fixed_accumulating task, got %d", accumulatingCount)
	}
}

func testFixedAccumulatingCronExhaustedRule(t *testing.T, env *testEnv) {
	createSpace(t, env, "fa-exh", "FA Exhausted")

	// Create a task with UNTIL in the past — rule is already exhausted.
	exhDue := time.Now().AddDate(0, 0, -21).Format(time.DateOnly)
	exhUntil := time.Now().AddDate(0, 0, -14).Format("20060102T150405Z")
	task := createTask(t, env, "fa-exh",
		`{"title":"Expired","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=WEEKLY;UNTIL=`+exhUntil+`","due":{"at":"`+exhDue+`","timezone":"UTC"}}`)
	taskID := jsonAs[string](t, task["id"])

	must(t, taskengine.ProcessOverdueTasks(t.Context(), env.pool, time.Now(), nil))

	// The original should be converted to one_off.
	resp := doRequest(t, env, "GET", "/spaces/fa-exh/tasks/"+taskID, "")
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)

	if updated["recurrenceType"] != "one_off" {
		t.Fatalf("expected one_off after exhausted rule, got %v", updated["recurrenceType"])
	}
}

func testFixedAccumulatingCronNonUTCTimezone(t *testing.T, env *testEnv) {
	createSpace(t, env, "fa-tz", "FA TZ")

	// Weekly task due in Eastern timezone.
	// Due date is 2 weeks in the past (dynamically computed).
	tzDue := time.Now().AddDate(0, 0, -14).Format(time.DateOnly)
	task := createTask(t, env, "fa-tz",
		`{"title":"Weekly Eastern","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=WEEKLY","due":{"at":"`+tzDue+`","timezone":"America/New_York"}}`)
	taskID := jsonAs[string](t, task["id"])

	must(t, taskengine.ProcessOverdueTasks(t.Context(), env.pool, time.Now(), nil))

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
	for _, item := range jsonAs[[]any](t, page["items"]) {
		task := jsonAs[map[string]any](t, item)
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
	parsedDue, err := time.Parse(time.DateOnly, dueStr)
	if err != nil {
		t.Fatalf("parse due: %v", err)
	}
	if parsedDue.Before(time.Now().Truncate(24 * time.Hour)) {
		t.Fatalf("continuation due date should be in the future, got %s", dueStr)
	}

	// Timezone should be preserved.
	tz, ok := dueTzFromResponse(continuation)
	if !ok || tz != "America/New_York" {
		t.Fatalf("continuation timezone = %v, want America/New_York", tz)
	}
}

func testFixedAccumulatingCronAfterCompletion(t *testing.T, env *testEnv) {
	createSpace(t, env, "fa-race", "FA Race")

	// Create a task with due date in the past (dynamically computed).
	raceDue := time.Now().AddDate(0, 0, -7).Format(time.DateOnly)
	task := createTask(t, env, "fa-race",
		`{"title":"Weekly","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=WEEKLY","due":{"at":"`+raceDue+`","timezone":"UTC"}}`)
	taskID := jsonAs[string](t, task["id"])

	// Complete the task via HTTP — this converts it to one_off and spawns a new task.
	resp := doRequest(t, env, "PATCH", "/spaces/fa-race/tasks/"+taskID, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)

	// Now run the cron — it should NOT re-process the original (already one_off).
	must(t, taskengine.ProcessOverdueTasks(t.Context(), env.pool, time.Now(), nil))

	// Count tasks — should be exactly 2 (original one_off + 1 spawned by completion).
	resp2 := doRequest(t, env, "GET", "/spaces/fa-race/tasks?limit=100", "")
	assertStatus(t, resp2, http.StatusOK)
	var page map[string]any
	readJSON(t, resp2, &page)
	items := jsonAs[[]any](t, page["items"])

	// Count fixed_accumulating tasks — should be exactly 1 (the one spawned by completion).
	var accumulatingCount int
	for _, item := range items {
		task := jsonAs[map[string]any](t, item)
		if task["recurrenceType"] == "fixed_accumulating" {
			accumulatingCount++
		}
	}
	if accumulatingCount != 1 {
		t.Fatalf("expected exactly 1 fixed_accumulating task after completion + cron, got %d", accumulatingCount)
	}
}

package api_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/sargunv/tend/server/internal/taskengine"
)

func TestRotationPoolRoundTrip(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "alpha", "Alpha")
	_, bobID := createAndAddMember(t, env, "alpha", "bob@example.com", "Bob", "password", "member")
	ownerID := getUserID(t, env, env.Token)

	// Create task with rotation pool.
	task := createTask(t, env, "alpha",
		fmt.Sprintf(`{"title":"Chore","rotationPool":[%q,%q]}`, ownerID, bobID))

	pool, ok := task["rotationPool"].([]any)
	if !ok {
		t.Fatalf("rotationPool missing or wrong type: %T", task["rotationPool"])
	}
	if len(pool) != 2 {
		t.Fatalf("got %d pool members, want 2", len(pool))
	}
	if pool[0] != ownerID || pool[1] != bobID {
		t.Fatalf("got pool %v, want [%s, %s]", pool, ownerID, bobID)
	}

	// Update pool to reorder.
	taskID := jsonAs[string](t, task["id"])
	resp := doRequest(t, env, "PATCH", "/spaces/alpha/tasks/"+taskID,
		fmt.Sprintf(`{"rotationPool":[%q,%q]}`, bobID, ownerID))
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)
	pool2 := jsonAs[[]any](t, updated["rotationPool"])
	if pool2[0] != bobID || pool2[1] != ownerID {
		t.Fatalf("got reordered pool %v, want [%s, %s]", pool2, bobID, ownerID)
	}

	// Clear pool.
	resp = doRequest(t, env, "PATCH", "/spaces/alpha/tasks/"+taskID,
		`{"rotationPool":[]}`)
	assertStatus(t, resp, http.StatusOK)
	var cleared map[string]any
	readJSON(t, resp, &cleared)
	pool3 := jsonAs[[]any](t, cleared["rotationPool"])
	if len(pool3) != 0 {
		t.Fatalf("got pool %v after clear, want empty", pool3)
	}
}

func TestRotationPoolDefaultEmpty(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "alpha", "Alpha")

	task := createTask(t, env, "alpha", `{"title":"No pool"}`)
	pool := jsonAs[[]any](t, task["rotationPool"])
	if len(pool) != 0 {
		t.Fatalf("default pool should be empty, got %v", pool)
	}
}

func TestRotationPoolPreservedOnUnrelatedUpdate(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "alpha", "Alpha")
	ownerID := getUserID(t, env, env.Token)

	task := createTask(t, env, "alpha",
		fmt.Sprintf(`{"title":"Chore","rotationPool":[%q]}`, ownerID))
	taskID := jsonAs[string](t, task["id"])

	// PATCH only the title.
	resp := doRequest(t, env, "PATCH", "/spaces/alpha/tasks/"+taskID,
		`{"title":"Updated Chore"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)
	pool := jsonAs[[]any](t, updated["rotationPool"])
	if len(pool) != 1 || pool[0] != ownerID {
		t.Fatalf("pool should be preserved, got %v", pool)
	}
}

func TestRotationOnCompletionBased(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "alpha", "Alpha")
	_, bobID := createAndAddMember(t, env, "alpha", "bob@example.com", "Bob", "password", "member")
	_, charlieID := createAndAddMember(t, env, "alpha", "charlie@example.com", "Charlie", "password", "member")
	ownerID := getUserID(t, env, env.Token)

	// Create completion_based task: pool=[owner, bob, charlie], assignee=[owner].
	task := createTask(t, env, "alpha", fmt.Sprintf(
		`{"title":"Chore","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=DAILY;INTERVAL=7","assigneeIds":[%q],"rotationPool":[%q,%q,%q]}`,
		ownerID, ownerID, bobID, charlieID,
	))
	taskID := jsonAs[string](t, task["id"])

	// Complete: assignee should rotate owner → bob.
	resp := doRequest(t, env, "PATCH", "/spaces/alpha/tasks/"+taskID,
		`{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var after1 map[string]any
	readJSON(t, resp, &after1)
	assignees1 := jsonAs[[]any](t, after1["assigneeIds"])
	if len(assignees1) != 1 || assignees1[0] != bobID {
		t.Fatalf("after 1st completion: got assignees %v, want [%s]", assignees1, bobID)
	}

	// Complete again: bob → charlie.
	resp = doRequest(t, env, "PATCH", "/spaces/alpha/tasks/"+taskID,
		`{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var after2 map[string]any
	readJSON(t, resp, &after2)
	assignees2 := jsonAs[[]any](t, after2["assigneeIds"])
	if len(assignees2) != 1 || assignees2[0] != charlieID {
		t.Fatalf("after 2nd completion: got assignees %v, want [%s]", assignees2, charlieID)
	}

	// Complete again: charlie → owner (wraps).
	resp = doRequest(t, env, "PATCH", "/spaces/alpha/tasks/"+taskID,
		`{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var after3 map[string]any
	readJSON(t, resp, &after3)
	assignees3 := jsonAs[[]any](t, after3["assigneeIds"])
	if len(assignees3) != 1 || assignees3[0] != ownerID {
		t.Fatalf("after 3rd completion (wrap): got assignees %v, want [%s]", assignees3, ownerID)
	}
}

func TestRotationNoOpWithEmptyPool(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "alpha", "Alpha")
	ownerID := getUserID(t, env, env.Token)

	// Create completion_based task with no pool, assigned to owner.
	task := createTask(t, env, "alpha", fmt.Sprintf(
		`{"title":"Chore","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=DAILY;INTERVAL=7","assigneeIds":[%q]}`,
		ownerID,
	))
	taskID := jsonAs[string](t, task["id"])

	// Complete: assignee should stay the same (no pool).
	resp := doRequest(t, env, "PATCH", "/spaces/alpha/tasks/"+taskID,
		`{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var after map[string]any
	readJSON(t, resp, &after)
	assignees := jsonAs[[]any](t, after["assigneeIds"])
	if len(assignees) != 1 || assignees[0] != ownerID {
		t.Fatalf("got assignees %v, want [%s] (unchanged with empty pool)", assignees, ownerID)
	}
}

func TestRotationCurrentAssigneeNotInPool(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "alpha", "Alpha")
	_, bobID := createAndAddMember(t, env, "alpha", "bob@example.com", "Bob", "password", "member")
	_, charlieID := createAndAddMember(t, env, "alpha", "charlie@example.com", "Charlie", "password", "member")
	ownerID := getUserID(t, env, env.Token)

	// Create task: pool=[bob, charlie], but assignee=[owner] (not in pool).
	task := createTask(t, env, "alpha", fmt.Sprintf(
		`{"title":"Chore","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=DAILY;INTERVAL=7","assigneeIds":[%q],"rotationPool":[%q,%q]}`,
		ownerID, bobID, charlieID,
	))
	taskID := jsonAs[string](t, task["id"])

	// Complete: should use first pool member (bob) since owner is not in pool.
	resp := doRequest(t, env, "PATCH", "/spaces/alpha/tasks/"+taskID,
		`{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var after map[string]any
	readJSON(t, resp, &after)
	assignees := jsonAs[[]any](t, after["assigneeIds"])
	if len(assignees) != 1 || assignees[0] != bobID {
		t.Fatalf("got assignees %v, want [%s] (first in pool)", assignees, bobID)
	}
}

func TestRotationOneOffNoRotation(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "alpha", "Alpha")
	_, bobID := createAndAddMember(t, env, "alpha", "bob@example.com", "Bob", "password", "member")
	ownerID := getUserID(t, env, env.Token)

	// Create one_off task with pool.
	task := createTask(t, env, "alpha", fmt.Sprintf(
		`{"title":"One-off","assigneeIds":[%q],"rotationPool":[%q,%q]}`,
		ownerID, ownerID, bobID,
	))
	taskID := jsonAs[string](t, task["id"])

	// Complete: one_off stays completed, no rotation.
	resp := doRequest(t, env, "PATCH", "/spaces/alpha/tasks/"+taskID,
		`{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var after map[string]any
	readJSON(t, resp, &after)
	assignees := jsonAs[[]any](t, after["assigneeIds"])
	if len(assignees) != 1 || assignees[0] != ownerID {
		t.Fatalf("one_off should not rotate: got assignees %v, want [%s]", assignees, ownerID)
	}
}

func TestRotationFixedAccumulating(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "alpha", "Alpha")
	_, bobID := createAndAddMember(t, env, "alpha", "bob@example.com", "Bob", "password", "member")
	ownerID := getUserID(t, env, env.Token)

	// Create fixed_accumulating task.
	// Find a past Saturday that's at least 8 days ago so the next occurrence is strictly in the future.
	pastSat := time.Now().AddDate(0, 0, -8)
	for pastSat.Weekday() != time.Saturday {
		pastSat = pastSat.AddDate(0, 0, -1)
	}
	pastSatStr := pastSat.Format(time.DateOnly)
	task := createTask(t, env, "alpha", fmt.Sprintf(
		`{"title":"Accumulating","recurrenceType":"fixed_accumulating","recurrenceRule":"RRULE:FREQ=WEEKLY;BYDAY=SA","due":{"at":"`+pastSatStr+`","timezone":"UTC"},"assigneeIds":[%q],"rotationPool":[%q,%q]}`,
		ownerID, ownerID, bobID,
	))
	taskID := jsonAs[string](t, task["id"])

	// Complete: old task → one_off with pool cleared, spawned task gets rotated assignee.
	resp := doRequest(t, env, "PATCH", "/spaces/alpha/tasks/"+taskID,
		`{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var completed map[string]any
	readJSON(t, resp, &completed)

	// Old task should be one_off with empty pool.
	if completed["recurrenceType"] != "one_off" {
		t.Fatalf("completed task should be one_off, got %v", completed["recurrenceType"])
	}
	completedPool := jsonAs[[]any](t, completed["rotationPool"])
	if len(completedPool) != 0 {
		t.Fatalf("completed task pool should be empty, got %v", completedPool)
	}

	// Find spawned task via relations.
	rels := jsonAs[[]any](t, completed["relations"])
	var spawnedID string
	for _, r := range rels {
		rel := jsonAs[map[string]any](t, r)
		if rel["kind"] == "spawns" {
			spawnedID = jsonAs[string](t, rel["relatedTaskId"])
		}
	}
	if spawnedID == "" {
		t.Fatal("no spawns relation found on completed task")
	}

	// Spawned task should have rotated assignee (bob) and the full pool.
	resp = doRequest(t, env, "GET", "/spaces/alpha/tasks/"+spawnedID, "")
	assertStatus(t, resp, http.StatusOK)
	var spawned map[string]any
	readJSON(t, resp, &spawned)

	spawnedAssignees := jsonAs[[]any](t, spawned["assigneeIds"])
	if len(spawnedAssignees) != 1 || spawnedAssignees[0] != bobID {
		t.Fatalf("spawned task assignees should be [%s], got %v", bobID, spawnedAssignees)
	}
	spawnedPool := jsonAs[[]any](t, spawned["rotationPool"])
	if len(spawnedPool) != 2 {
		t.Fatalf("spawned task pool should have 2 members, got %v", spawnedPool)
	}
	if spawned["recurrenceType"] != "fixed_accumulating" {
		t.Fatalf("spawned task should be fixed_accumulating, got %v", spawned["recurrenceType"])
	}
}

func TestRotationCronMultiSpawn(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "alpha", "Alpha")
	_, bobID := createAndAddMember(t, env, "alpha", "bob@example.com", "Bob", "password", "member")
	_, charlieID := createAndAddMember(t, env, "alpha", "charlie@example.com", "Charlie", "password", "member")
	ownerID := getUserID(t, env, env.Token)

	// Create fixed_accumulating task with weekly recurrence, due 3 weeks ago.
	// This should produce 3 missed occurrences + 1 continuation.
	threeWeeksAgo := time.Now().UTC().AddDate(0, 0, -21).Format("2006-01-02")
	task := createTask(t, env, "alpha", fmt.Sprintf(
		`{"title":"Weekly","recurrenceType":"fixed_accumulating","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"assigneeIds":[%q],"rotationPool":[%q,%q,%q]}`,
		threeWeeksAgo, ownerID, ownerID, bobID, charlieID,
	))
	taskID := jsonAs[string](t, task["id"])

	// Run cron to process overdue tasks.
	must(t, taskengine.ProcessOverdueTasks(t.Context(), env.pool, time.Now(), nil))

	// Original task should now be one_off with empty pool.
	resp := doRequest(t, env, "GET", "/spaces/alpha/tasks/"+taskID, "")
	assertStatus(t, resp, http.StatusOK)
	var original map[string]any
	readJSON(t, resp, &original)
	if original["recurrenceType"] != "one_off" {
		t.Fatalf("original should be one_off, got %v", original["recurrenceType"])
	}
	originalPool := jsonAs[[]any](t, original["rotationPool"])
	if len(originalPool) != 0 {
		t.Fatalf("original pool should be empty, got %v", originalPool)
	}

	// List all tasks and find spawned ones.
	resp = doRequest(t, env, "GET", "/spaces/alpha/tasks", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := jsonAs[[]any](t, page["items"])

	// Collect spawned tasks (not the original).
	var spawnedAssignees []string
	var continuationTask map[string]any
	for _, item := range items {
		task := jsonAs[map[string]any](t, item)
		if task["id"] == taskID {
			continue
		}
		assignees := jsonAs[[]any](t, task["assigneeIds"])
		if len(assignees) == 1 {
			spawnedAssignees = append(spawnedAssignees, jsonAs[string](t, assignees[0]))
		}
		if task["recurrenceType"] == "fixed_accumulating" {
			continuationTask = task
		}
	}

	// The continuation task should have the pool.
	if continuationTask == nil {
		t.Fatal("no continuation fixed_accumulating task found")
	}
	contPool := jsonAs[[]any](t, continuationTask["rotationPool"])
	if len(contPool) != 3 {
		t.Fatalf("continuation pool should have 3 members, got %v", contPool)
	}

	// Verify exact rotation sequence across spawned one_off tasks.
	// Pool is [owner, bob, charlie], current assignee is owner (index 0).
	// step=0 → bob (index 1), step=1 → charlie (index 2), step=2 → owner (index 0).
	expectedOneOff := []string{bobID, charlieID, ownerID}
	if len(spawnedAssignees) < len(expectedOneOff) {
		t.Fatalf("expected at least %d one_off spawns, got %d", len(expectedOneOff), len(spawnedAssignees))
	}
	for i, want := range expectedOneOff {
		if i < len(spawnedAssignees) && spawnedAssignees[i] != want {
			t.Fatalf("spawned task %d: got assignee %s, want %s", i, spawnedAssignees[i], want)
		}
	}

	// Verify the continuation task's assignee is advanced past all missed occurrences.
	// step=len(missed)=3 → (0+1+3) % 3 = 1 → bob.
	contAssignees := jsonAs[[]any](t, continuationTask["assigneeIds"])
	if len(contAssignees) != 1 || contAssignees[0] != bobID {
		t.Fatalf("continuation task assignee: got %v, want [%s]", contAssignees, bobID)
	}
}

func TestRotationPoolNonMemberRejected(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "alpha", "Alpha")

	// Try to set a non-existent user in the pool.
	resp := doRequest(t, env, "POST", "/spaces/alpha/tasks",
		`{"title":"Test","rotationPool":["U99999"]}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestRotationPoolDeduplication(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "alpha", "Alpha")
	ownerID := getUserID(t, env, env.Token)

	// Send duplicate IDs in pool.
	task := createTask(t, env, "alpha", fmt.Sprintf(
		`{"title":"Dedup","rotationPool":[%q,%q]}`, ownerID, ownerID))
	pool := jsonAs[[]any](t, task["rotationPool"])
	if len(pool) != 1 {
		t.Fatalf("pool should deduplicate, got %v", pool)
	}
}

func TestRotationPoolInListResponse(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "alpha", "Alpha")
	ownerID := getUserID(t, env, env.Token)

	createTask(t, env, "alpha", fmt.Sprintf(
		`{"title":"With pool","rotationPool":[%q]}`, ownerID))
	createTask(t, env, "alpha", `{"title":"No pool"}`)

	resp := doRequest(t, env, "GET", "/spaces/alpha/tasks", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := jsonAs[[]any](t, page["items"])
	if len(items) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(items))
	}
	for _, item := range items {
		task := jsonAs[map[string]any](t, item)
		pool := jsonAs[[]any](t, task["rotationPool"])
		title := jsonAs[string](t, task["title"])
		if title == "With pool" && len(pool) != 1 {
			t.Fatalf("'With pool' task should have 1 pool member, got %v", pool)
		}
		if title == "No pool" && len(pool) != 0 {
			t.Fatalf("'No pool' task should have empty pool, got %v", pool)
		}
	}
}

func TestRotationOnFixedNonAccumulating(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "alpha", "Alpha")
	_, bobID := createAndAddMember(t, env, "alpha", "bob@example.com", "Bob", "password", "member")
	_, charlieID := createAndAddMember(t, env, "alpha", "charlie@example.com", "Charlie", "password", "member")
	ownerID := getUserID(t, env, env.Token)

	// Create fixed_non_accumulating task: pool=[owner, bob, charlie], assignee=[owner].
	// Find a past Saturday that's at least 8 days ago so the next occurrence is strictly in the future.
	pastSat := time.Now().AddDate(0, 0, -8)
	for pastSat.Weekday() != time.Saturday {
		pastSat = pastSat.AddDate(0, 0, -1)
	}
	pastSatStr := pastSat.Format(time.DateOnly)
	task := createTask(t, env, "alpha", fmt.Sprintf(
		`{"title":"Weekly","recurrenceType":"fixed_non_accumulating","recurrenceRule":"RRULE:FREQ=WEEKLY;BYDAY=SA","due":{"at":"`+pastSatStr+`","timezone":"UTC"},"assigneeIds":[%q],"rotationPool":[%q,%q,%q]}`,
		ownerID, ownerID, bobID, charlieID,
	))
	taskID := jsonAs[string](t, task["id"])

	// Complete: assignee should rotate owner → bob, task resets in place.
	resp := doRequest(t, env, "PATCH", "/spaces/alpha/tasks/"+taskID,
		`{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var after1 map[string]any
	readJSON(t, resp, &after1)

	// Task should reset to initial status (not stay completed).
	if after1["status"] == "done" {
		t.Fatal("fixed_non_accumulating should reset to initial status on completion")
	}
	// Assignee should have rotated.
	assignees1 := jsonAs[[]any](t, after1["assigneeIds"])
	if len(assignees1) != 1 || assignees1[0] != bobID {
		t.Fatalf("after completion: got assignees %v, want [%s]", assignees1, bobID)
	}
	// Pool should be preserved (not cleared like fixed_accumulating).
	pool1 := jsonAs[[]any](t, after1["rotationPool"])
	if len(pool1) != 3 {
		t.Fatalf("pool should be preserved, got %v", pool1)
	}
}

func TestRotationExplicitAssigneesOverrideRotation(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "alpha", "Alpha")
	_, bobID := createAndAddMember(t, env, "alpha", "bob@example.com", "Bob", "password", "member")
	_, charlieID := createAndAddMember(t, env, "alpha", "charlie@example.com", "Charlie", "password", "member")
	ownerID := getUserID(t, env, env.Token)

	// Create completion_based task: pool=[owner, bob, charlie], assignee=[owner].
	task := createTask(t, env, "alpha", fmt.Sprintf(
		`{"title":"Chore","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=DAILY;INTERVAL=7","assigneeIds":[%q],"rotationPool":[%q,%q,%q]}`,
		ownerID, ownerID, bobID, charlieID,
	))
	taskID := jsonAs[string](t, task["id"])

	// Complete with explicit assigneeIds in the same request.
	resp := doRequest(t, env, "PATCH", "/spaces/alpha/tasks/"+taskID,
		fmt.Sprintf(`{"status":"done","assigneeIds":[%q]}`, charlieID))
	assertStatus(t, resp, http.StatusOK)
	var after map[string]any
	readJSON(t, resp, &after)
	assignees := jsonAs[[]any](t, after["assigneeIds"])
	// Explicit assigneeIds should override rotation.
	if len(assignees) != 1 || assignees[0] != charlieID {
		t.Fatalf("explicit assignees should override rotation: got %v, want [%s]", assignees, charlieID)
	}
}

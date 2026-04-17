package api_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/sargunv/horologia/server/internal/taskengine"
)

func TestRotationPoolHandlers(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	t.Run("pool round trip", func(t *testing.T) {
		space := testSlug(t, "alpha")
		createSpace(t, env, space, "Alpha")
		_, bobID := createAndAddMember(t, env, space, testEmail(t, "bob"), "Bob", "password", "member")
		ownerID := getUserID(t, env, env.Token)
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"Chore","rotationPool":[%q,%q]}`, ownerID, bobID))
		pool := jsonAs[[]any](t, task["rotationPool"])
		if len(pool) != 2 || pool[0] != ownerID || pool[1] != bobID {
			t.Fatalf("got pool %v, want [%s, %s]", pool, ownerID, bobID)
		}
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+taskID, fmt.Sprintf(`{"rotationPool":[%q,%q]}`, bobID, ownerID))
		assertStatus(t, resp, http.StatusOK)
		var updated map[string]any
		readJSON(t, resp, &updated)
		pool2 := jsonAs[[]any](t, updated["rotationPool"])
		if pool2[0] != bobID || pool2[1] != ownerID {
			t.Fatalf("got reordered pool %v, want [%s, %s]", pool2, bobID, ownerID)
		}
		resp = doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+taskID, `{"rotationPool":[]}`)
		assertStatus(t, resp, http.StatusOK)
		var cleared map[string]any
		readJSON(t, resp, &cleared)
		if len(jsonAs[[]any](t, cleared["rotationPool"])) != 0 {
			t.Fatalf("got pool %v after clear, want empty", cleared["rotationPool"])
		}
	})

	t.Run("pool default empty", func(t *testing.T) {
		space := testSlug(t, "alpha")
		createSpace(t, env, space, "Alpha")
		task := createTask(t, env, space, `{"title":"No pool"}`)
		if len(jsonAs[[]any](t, task["rotationPool"])) != 0 {
			t.Fatalf("default pool should be empty, got %v", task["rotationPool"])
		}
	})

	t.Run("pool preserved on unrelated update", func(t *testing.T) {
		space := testSlug(t, "alpha")
		createSpace(t, env, space, "Alpha")
		ownerID := getUserID(t, env, env.Token)
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"Chore","rotationPool":[%q]}`, ownerID))
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+taskID, `{"title":"Updated Chore"}`)
		assertStatus(t, resp, http.StatusOK)
		var updated map[string]any
		readJSON(t, resp, &updated)
		pool := jsonAs[[]any](t, updated["rotationPool"])
		if len(pool) != 1 || pool[0] != ownerID {
			t.Fatalf("pool should be preserved, got %v", pool)
		}
	})

	t.Run("pool non member rejected", func(t *testing.T) {
		space := testSlug(t, "alpha")
		createSpace(t, env, space, "Alpha")
		resp := doRequest(t, env, "POST", "/spaces/"+space+"/tasks", `{"title":"Test","rotationPool":["U99999"]}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("pool deduplication", func(t *testing.T) {
		space := testSlug(t, "alpha")
		createSpace(t, env, space, "Alpha")
		ownerID := getUserID(t, env, env.Token)
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"Dedup","rotationPool":[%q,%q]}`, ownerID, ownerID))
		pool := jsonAs[[]any](t, task["rotationPool"])
		if len(pool) != 1 {
			t.Fatalf("pool should deduplicate, got %v", pool)
		}
	})

	t.Run("pool in list response", func(t *testing.T) {
		space := testSlug(t, "alpha")
		createSpace(t, env, space, "Alpha")
		ownerID := getUserID(t, env, env.Token)
		createTask(t, env, space, fmt.Sprintf(`{"title":"With pool","rotationPool":[%q]}`, ownerID))
		createTask(t, env, space, `{"title":"No pool"}`)
		resp := doRequest(t, env, "GET", "/spaces/"+space+"/tasks", "")
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
	})
}

func TestRotationRecurrence(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	t.Run("on completion based", func(t *testing.T) {
		space := testSlug(t, "alpha")
		createSpace(t, env, space, "Alpha")
		_, bobID := createAndAddMember(t, env, space, testEmail(t, "bob"), "Bob", "password", "member")
		_, charlieID := createAndAddMember(t, env, space, testEmail(t, "charlie"), "Charlie", "password", "member")
		ownerID := getUserID(t, env, env.Token)
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"Chore","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=DAILY;INTERVAL=7","assigneeIds":[%q],"rotationPool":[%q,%q,%q]}`, ownerID, ownerID, bobID, charlieID))
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+taskID, `{"status":"done"}`)
		assertStatus(t, resp, http.StatusOK)
		var after1 map[string]any
		readJSON(t, resp, &after1)
		if assignees := jsonAs[[]any](t, after1["assigneeIds"]); len(assignees) != 1 || assignees[0] != bobID {
			t.Fatalf("after 1st completion: got assignees %v, want [%s]", assignees, bobID)
		}
		resp = doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+taskID, `{"status":"done"}`)
		assertStatus(t, resp, http.StatusOK)
		var after2 map[string]any
		readJSON(t, resp, &after2)
		if assignees := jsonAs[[]any](t, after2["assigneeIds"]); len(assignees) != 1 || assignees[0] != charlieID {
			t.Fatalf("after 2nd completion: got assignees %v, want [%s]", assignees, charlieID)
		}
		resp = doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+taskID, `{"status":"done"}`)
		assertStatus(t, resp, http.StatusOK)
		var after3 map[string]any
		readJSON(t, resp, &after3)
		if assignees := jsonAs[[]any](t, after3["assigneeIds"]); len(assignees) != 1 || assignees[0] != ownerID {
			t.Fatalf("after 3rd completion: got assignees %v, want [%s]", assignees, ownerID)
		}
	})

	t.Run("no op with empty pool", func(t *testing.T) {
		space := testSlug(t, "alpha")
		createSpace(t, env, space, "Alpha")
		ownerID := getUserID(t, env, env.Token)
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"Chore","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=DAILY;INTERVAL=7","assigneeIds":[%q]}`, ownerID))
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+taskID, `{"status":"done"}`)
		assertStatus(t, resp, http.StatusOK)
		var after map[string]any
		readJSON(t, resp, &after)
		assignees := jsonAs[[]any](t, after["assigneeIds"])
		if len(assignees) != 1 || assignees[0] != ownerID {
			t.Fatalf("got assignees %v, want [%s]", assignees, ownerID)
		}
	})

	t.Run("current assignee not in pool", func(t *testing.T) {
		space := testSlug(t, "alpha")
		createSpace(t, env, space, "Alpha")
		_, bobID := createAndAddMember(t, env, space, testEmail(t, "bob"), "Bob", "password", "member")
		_, charlieID := createAndAddMember(t, env, space, testEmail(t, "charlie"), "Charlie", "password", "member")
		ownerID := getUserID(t, env, env.Token)
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"Chore","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=DAILY;INTERVAL=7","assigneeIds":[%q],"rotationPool":[%q,%q]}`, ownerID, bobID, charlieID))
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+taskID, `{"status":"done"}`)
		assertStatus(t, resp, http.StatusOK)
		var after map[string]any
		readJSON(t, resp, &after)
		if assignees := jsonAs[[]any](t, after["assigneeIds"]); len(assignees) != 1 || assignees[0] != bobID {
			t.Fatalf("got assignees %v, want [%s]", assignees, bobID)
		}
	})

	t.Run("one off no rotation", func(t *testing.T) {
		space := testSlug(t, "alpha")
		createSpace(t, env, space, "Alpha")
		_, bobID := createAndAddMember(t, env, space, testEmail(t, "bob"), "Bob", "password", "member")
		ownerID := getUserID(t, env, env.Token)
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"One-off","assigneeIds":[%q],"rotationPool":[%q,%q]}`, ownerID, ownerID, bobID))
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+taskID, `{"status":"done"}`)
		assertStatus(t, resp, http.StatusOK)
		var after map[string]any
		readJSON(t, resp, &after)
		if assignees := jsonAs[[]any](t, after["assigneeIds"]); len(assignees) != 1 || assignees[0] != ownerID {
			t.Fatalf("one_off should not rotate: got %v, want [%s]", assignees, ownerID)
		}
	})

	t.Run("fixed accumulating", func(t *testing.T) {
		space := testSlug(t, "alpha")
		createSpace(t, env, space, "Alpha")
		_, bobID := createAndAddMember(t, env, space, testEmail(t, "bob"), "Bob", "password", "member")
		ownerID := getUserID(t, env, env.Token)
		pastSat := time.Now().AddDate(0, 0, -8)
		for pastSat.Weekday() != time.Saturday {
			pastSat = pastSat.AddDate(0, 0, -1)
		}
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"Accumulating","recurrenceType":"fixed_accumulating","recurrenceRule":"RRULE:FREQ=WEEKLY;BYDAY=SA","due":{"at":"%s","timezone":"UTC"},"assigneeIds":[%q],"rotationPool":[%q,%q]}`, pastSat.Format(time.DateOnly), ownerID, ownerID, bobID))
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+taskID, `{"status":"done"}`)
		assertStatus(t, resp, http.StatusOK)
		var completed map[string]any
		readJSON(t, resp, &completed)
		if completed["recurrenceType"] != "one_off" {
			t.Fatalf("completed task should be one_off, got %v", completed["recurrenceType"])
		}
		if len(jsonAs[[]any](t, completed["rotationPool"])) != 0 {
			t.Fatalf("completed task pool should be empty, got %v", completed["rotationPool"])
		}
		var spawnedID string
		for _, rel := range jsonAs[[]any](t, completed["relations"]) {
			r := jsonAs[map[string]any](t, rel)
			if r["kind"] == "spawns" {
				spawnedID = jsonAs[string](t, r["relatedTaskId"])
			}
		}
		resp = doRequest(t, env, "GET", "/spaces/"+space+"/tasks/"+spawnedID, "")
		assertStatus(t, resp, http.StatusOK)
		var spawned map[string]any
		readJSON(t, resp, &spawned)
		if assignees := jsonAs[[]any](t, spawned["assigneeIds"]); len(assignees) != 1 || assignees[0] != bobID {
			t.Fatalf("spawned task assignees should be [%s], got %v", bobID, assignees)
		}
	})

	t.Run("cron multi spawn", func(t *testing.T) {
		space := testSlug(t, "alpha")
		createSpace(t, env, space, "Alpha")
		_, bobID := createAndAddMember(t, env, space, testEmail(t, "bob"), "Bob", "password", "member")
		_, charlieID := createAndAddMember(t, env, space, testEmail(t, "charlie"), "Charlie", "password", "member")
		ownerID := getUserID(t, env, env.Token)
		threeWeeksAgo := time.Now().UTC().AddDate(0, 0, -21).Format("2006-01-02")
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"Weekly","recurrenceType":"fixed_accumulating","recurrenceRule":"RRULE:FREQ=WEEKLY","due":{"at":%q,"timezone":"UTC"},"assigneeIds":[%q],"rotationPool":[%q,%q,%q]}`, threeWeeksAgo, ownerID, ownerID, bobID, charlieID))
		taskID := jsonAs[string](t, task["id"])
		must(t, taskengine.ProcessOverdueTasks(t.Context(), env.pool, time.Now(), nil))
		resp := doRequest(t, env, "GET", "/spaces/"+space+"/tasks/"+taskID, "")
		assertStatus(t, resp, http.StatusOK)
		var original map[string]any
		readJSON(t, resp, &original)
		if original["recurrenceType"] != "one_off" || len(jsonAs[[]any](t, original["rotationPool"])) != 0 {
			t.Fatalf("original should be one_off with empty pool, got %v / %v", original["recurrenceType"], original["rotationPool"])
		}
		resp = doRequest(t, env, "GET", "/spaces/"+space+"/tasks", "")
		assertStatus(t, resp, http.StatusOK)
		var page map[string]any
		readJSON(t, resp, &page)
		items := jsonAs[[]any](t, page["items"])
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
		if continuationTask == nil || len(jsonAs[[]any](t, continuationTask["rotationPool"])) != 3 {
			t.Fatal("expected continuation task with preserved pool")
		}
		expectedOneOff := []string{bobID, charlieID, ownerID}
		for i, want := range expectedOneOff {
			if i >= len(spawnedAssignees) || spawnedAssignees[i] != want {
				t.Fatalf("spawned task %d: got assignee %v, want %s", i, spawnedAssignees, want)
			}
		}
		contAssignees := jsonAs[[]any](t, continuationTask["assigneeIds"])
		if len(contAssignees) != 1 || contAssignees[0] != bobID {
			t.Fatalf("continuation assignee: got %v, want [%s]", contAssignees, bobID)
		}
	})

	t.Run("on fixed non accumulating", func(t *testing.T) {
		space := testSlug(t, "alpha")
		createSpace(t, env, space, "Alpha")
		_, bobID := createAndAddMember(t, env, space, testEmail(t, "bob"), "Bob", "password", "member")
		_, charlieID := createAndAddMember(t, env, space, testEmail(t, "charlie"), "Charlie", "password", "member")
		ownerID := getUserID(t, env, env.Token)
		pastSat := time.Now().AddDate(0, 0, -8)
		for pastSat.Weekday() != time.Saturday {
			pastSat = pastSat.AddDate(0, 0, -1)
		}
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"Weekly","recurrenceType":"fixed_non_accumulating","recurrenceRule":"RRULE:FREQ=WEEKLY;BYDAY=SA","due":{"at":"%s","timezone":"UTC"},"assigneeIds":[%q],"rotationPool":[%q,%q,%q]}`, pastSat.Format(time.DateOnly), ownerID, ownerID, bobID, charlieID))
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+taskID, `{"status":"done"}`)
		assertStatus(t, resp, http.StatusOK)
		var after map[string]any
		readJSON(t, resp, &after)
		if after["status"] == "done" {
			t.Fatal("fixed_non_accumulating should reset to initial status")
		}
		if assignees := jsonAs[[]any](t, after["assigneeIds"]); len(assignees) != 1 || assignees[0] != bobID {
			t.Fatalf("after completion: got assignees %v, want [%s]", assignees, bobID)
		}
		if len(jsonAs[[]any](t, after["rotationPool"])) != 3 {
			t.Fatalf("pool should be preserved, got %v", after["rotationPool"])
		}
	})

	t.Run("explicit assignees override rotation", func(t *testing.T) {
		space := testSlug(t, "alpha")
		createSpace(t, env, space, "Alpha")
		_, bobID := createAndAddMember(t, env, space, testEmail(t, "bob"), "Bob", "password", "member")
		_, charlieID := createAndAddMember(t, env, space, testEmail(t, "charlie"), "Charlie", "password", "member")
		ownerID := getUserID(t, env, env.Token)
		task := createTask(t, env, space, fmt.Sprintf(`{"title":"Chore","recurrenceType":"completion_based","recurrenceRule":"RRULE:FREQ=DAILY;INTERVAL=7","assigneeIds":[%q],"rotationPool":[%q,%q,%q]}`, ownerID, ownerID, bobID, charlieID))
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "PATCH", "/spaces/"+space+"/tasks/"+taskID, fmt.Sprintf(`{"status":"done","assigneeIds":[%q]}`, charlieID))
		assertStatus(t, resp, http.StatusOK)
		var after map[string]any
		readJSON(t, resp, &after)
		assignees := jsonAs[[]any](t, after["assigneeIds"])
		if len(assignees) != 1 || assignees[0] != charlieID {
			t.Fatalf("explicit assignees should override rotation: got %v, want [%s]", assignees, charlieID)
		}
	})
}

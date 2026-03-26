package api_test

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/taskengine"
	"github.com/sargunv/tend/server/internal/types"
)

// activityItems returns the items array from a GET activity log response.
func activityItems(t *testing.T, env *testEnv, path string) []any {
	t.Helper()
	return activityItemsAs(t, env, env.Token, path)
}

func activityItemsAs(t *testing.T, env *testEnv, token, path string) []any {
	t.Helper()
	resp := doRequestAs(t, env, token, "GET", path, "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	return jsonAs[[]any](t, page["items"])
}

func entryMap(t *testing.T, item any) map[string]any {
	t.Helper()
	return jsonAs[map[string]any](t, item)
}

func entryDetails(t *testing.T, entry map[string]any) []map[string]any {
	t.Helper()
	raw := jsonAs[[]any](t, entry["details"])
	details := make([]map[string]any, len(raw))
	for i, d := range raw {
		details[i] = jsonAs[map[string]any](t, d)
	}
	return details
}

func findDetail(t *testing.T, details []map[string]any, field string) map[string]any {
	t.Helper()
	for _, d := range details {
		if d["field"] == field {
			return d
		}
	}
	t.Fatalf("no detail with field %q found in %v", field, details)
	return nil
}

func mustParseTaskID(t *testing.T, s string) int64 {
	t.Helper()
	id, err := types.ParseTaskID(s)
	if err != nil {
		t.Fatalf("parse task ID %q: %v", s, err)
	}
	return id
}

func hasDetail(details []map[string]any, field string) bool {
	for _, d := range details {
		if d["field"] == field {
			return true
		}
	}
	return false
}

// --- Test 1: SpaceActivityLog_BasicSmoke ---

func TestSpaceActivityLog_BasicSmoke(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "ws", "WS")
	task := createTask(t, env, "ws", `{"title":"Do laundry"}`)
	taskID := jsonAs[string](t, task["id"])

	// Update the task title.
	assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/ws/tasks/"+taskID,
		`{"title":"Do all laundry"}`), http.StatusOK)

	items := activityItems(t, env, "/spaces/ws/activity")
	if len(items) < 3 {
		t.Fatalf("expected at least 3 entries (space create + task create + task update), got %d", len(items))
	}

	// Newest first (descending).
	newest := entryMap(t, items[0])
	if newest["action"] != "updated" || newest["entityType"] != "task" {
		t.Errorf("newest entry: action=%v entityType=%v, want updated/task", newest["action"], newest["entityType"])
	}
	if newest["actorId"] == nil {
		t.Error("expected non-null actorId on user-driven action")
	}
}

// --- Test 2: TaskActivityLog_CreateAndUpdateDetails ---

func TestTaskActivityLog_CreateAndUpdateDetails(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "ws", "WS")
	task := createTask(t, env, "ws", `{"title":"Old"}`)
	taskID := jsonAs[string](t, task["id"])

	assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/ws/tasks/"+taskID,
		`{"title":"New"}`), http.StatusOK)

	items := activityItems(t, env, "/spaces/ws/tasks/"+taskID+"/activity")
	if len(items) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(items))
	}

	// Items are newest-first: [update, create].
	updateEntry := entryMap(t, items[0])
	createEntry := entryMap(t, items[1])

	// Check create details.
	createDetails := entryDetails(t, createEntry)
	titleDetail := findDetail(t, createDetails, "title")
	if titleDetail["from"] != nil {
		t.Errorf("create title.from = %v, want null", titleDetail["from"])
	}
	if titleDetail["to"] != "Old" {
		t.Errorf("create title.to = %v, want Old", titleDetail["to"])
	}
	statusDetail := findDetail(t, createDetails, "status")
	if statusDetail["to"] != "todo" {
		t.Errorf("create status.to = %v, want todo", statusDetail["to"])
	}

	// Check update details — only title changed.
	updateDetails := entryDetails(t, updateEntry)
	if len(updateDetails) != 1 {
		t.Fatalf("update: expected 1 detail (title only), got %d: %v", len(updateDetails), updateDetails)
	}
	if updateDetails[0]["field"] != "title" {
		t.Errorf("update detail field = %v, want title", updateDetails[0]["field"])
	}
	if updateDetails[0]["from"] != "Old" || updateDetails[0]["to"] != "New" {
		t.Errorf("update detail from/to = %v/%v, want Old/New", updateDetails[0]["from"], updateDetails[0]["to"])
	}
}

// --- Test 3: TaskActivityLog_AssigneeDiff ---

func TestTaskActivityLog_AssigneeDiff(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "ws", "WS")
	_, aliceID := createAndAddMember(t, env, "ws", "alice@test.com", "Alice", "pass123", "member")
	_, bobID := createAndAddMember(t, env, "ws", "bob@test.com", "Bob", "pass123", "member")

	task := createTask(t, env, "ws", `{"title":"Chore","assigneeIds":["`+aliceID+`"]}`)
	taskID := jsonAs[string](t, task["id"])

	// Replace Alice with Bob.
	assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/ws/tasks/"+taskID,
		`{"assigneeIds":["`+bobID+`"]}`), http.StatusOK)

	items := activityItems(t, env, "/spaces/ws/tasks/"+taskID+"/activity")

	// Find the update entry (newest).
	updateEntry := entryMap(t, items[0])
	if updateEntry["action"] != "updated" {
		t.Fatalf("expected updated action, got %v", updateEntry["action"])
	}

	details := entryDetails(t, updateEntry)
	var hasAdded, hasRemoved bool
	for _, d := range details {
		if d["field"] == "assignee" && d["from"] == nil && d["to"] == bobID {
			hasAdded = true
		}
		if d["field"] == "assignee" && d["from"] == aliceID && d["to"] == nil {
			hasRemoved = true
		}
	}
	if !hasAdded {
		t.Error("expected assignee added detail for Bob")
	}
	if !hasRemoved {
		t.Error("expected assignee removed detail for Alice")
	}
}

// --- Test 4: TaskActivityLog_SurvivesTaskDeletion ---

func TestTaskActivityLog_SurvivesTaskDeletion(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "ws", "WS")
	task := createTask(t, env, "ws", `{"title":"Temporary"}`)
	taskID := jsonAs[string](t, task["id"])

	assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/ws/tasks/"+taskID, ""), http.StatusNoContent)

	// Space activity should still contain entries for the deleted task.
	items := activityItems(t, env, "/spaces/ws/activity")

	var foundCreate, foundDelete bool
	for _, item := range items {
		e := entryMap(t, item)
		if e["entityType"] == "task" && e["entityId"] == taskID {
			switch e["action"] {
			case "created":
				foundCreate = true
			case "deleted":
				foundDelete = true
				details := entryDetails(t, e)
				titleDetail := findDetail(t, details, "title")
				if titleDetail["from"] != "Temporary" {
					t.Errorf("delete title.from = %v, want Temporary", titleDetail["from"])
				}
				if titleDetail["to"] != nil {
					t.Errorf("delete title.to = %v, want nil", titleDetail["to"])
				}
			}
		}
	}
	if !foundCreate {
		t.Error("task create entry was lost after deletion")
	}
	if !foundDelete {
		t.Error("task delete entry not found")
	}
}

// --- Test 5: SpaceActivityLog_SurvivesSpaceDeletion ---

func TestSpaceActivityLog_SurvivesSpaceDeletion(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tmp", "Tmp")
	createTask(t, env, "tmp", `{"title":"Something"}`)

	assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/tmp", ""), http.StatusNoContent)

	// Space is gone so we can't use the API. Query DB directly.
	var count int64
	err := env.pool.QueryRow(t.Context(),
		"SELECT count(*) FROM activity_log WHERE space_slug = 'tmp'").Scan(&count)
	if err != nil {
		t.Fatalf("query activity_log: %v", err)
	}
	if count < 2 {
		t.Errorf("expected at least 2 activity entries for deleted space, got %d", count)
	}
}

// --- Test 6: UserActivityList_SelfAccess ---

func TestUserActivityList_SelfAccess(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "ws", "WS")
	aliceToken, aliceID := createAndAddMember(t, env, "ws", "alice@test.com", "Alice", "pass123", "member")

	// Alice creates a task.
	resp := doRequestAs(t, env, aliceToken, "POST", "/spaces/ws/tasks", `{"title":"Alice task"}`)
	assertStatusClose(t, resp, http.StatusCreated)

	items := activityItemsAs(t, env, aliceToken, "/users/"+aliceID+"/activity")
	if len(items) == 0 {
		t.Fatal("expected at least one activity entry for Alice")
	}
	for _, item := range items {
		e := entryMap(t, item)
		if e["actorId"] != aliceID {
			t.Errorf("entry actorId = %v, want %s", e["actorId"], aliceID)
		}
	}
}

// --- Test 7: UserActivityList_NonOwnerForbiddenForOtherUser ---

func TestUserActivityList_NonOwnerForbiddenForOtherUser(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "ws", "WS")
	aliceToken, _ := createAndAddMember(t, env, "ws", "alice@test.com", "Alice", "pass123", "member")
	_, bobID := createAndAddMember(t, env, "ws", "bob@test.com", "Bob", "pass123", "member")

	resp := doRequestAs(t, env, aliceToken, "GET", "/users/"+bobID+"/activity", "")
	assertStatusClose(t, resp, http.StatusForbidden)
}

// --- Test 8: UserActivityList_CrossSpaceFiltering ---

func TestUserActivityList_CrossSpaceFiltering(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "visible", "Visible")
	createSpace(t, env, "hidden", "Hidden")
	aliceToken, aliceID := createAndAddMember(t, env, "visible", "alice@test.com", "Alice", "pass123", "member")

	// Alice creates a task in 'visible'.
	resp := doRequestAs(t, env, aliceToken, "POST", "/spaces/visible/tasks", `{"title":"Alice task"}`)
	assertStatusClose(t, resp, http.StatusCreated)

	// Owner creates a task in 'hidden' (Alice is not a member).
	createTask(t, env, "hidden", `{"title":"Hidden task"}`)

	// Fabricate an activity entry for Alice in the hidden space (simulates a
	// hypothetical cross-space leak).
	aliceNumericID, err := types.ParseUserID(aliceID)
	if err != nil {
		t.Fatalf("parse alice user ID %q: %v", aliceID, err)
	}
	_, err = env.pool.Exec(t.Context(),
		`INSERT INTO activity_log (space_slug, actor_id, entity_type, entity_id, action, created_at)
		 VALUES ('hidden', $1, 'task', 'T999', 'created', now())`,
		aliceNumericID)
	if err != nil {
		t.Fatalf("insert fabricated entry: %v", err)
	}

	items := activityItemsAs(t, env, aliceToken, "/users/"+aliceID+"/activity")
	for _, item := range items {
		e := entryMap(t, item)
		if e["spaceSlug"] == "hidden" {
			t.Error("user activity returned entry from a space the user is not a member of")
		}
	}
}

// --- Test 9: UserActivityList_OwnerSeesAllSpaces ---

func TestUserActivityList_OwnerSeesAllSpaces(t *testing.T) {
	env := setupTestServer(t)
	ownerID := getUserID(t, env, env.Token)

	// Alice creates a space where the owner is NOT a member.
	aliceToken := createTestUser(t, env, "alice@test.com", "Alice", "pass123")
	resp := doRequestAs(t, env, aliceToken, "POST", "/spaces",
		`{"slug":"alice-only","name":"Alice Only"}`)
	assertStatusClose(t, resp, http.StatusCreated)

	// Fabricate an activity entry attributed to the owner in Alice's space.
	// The owner has no membership in "alice-only", so only the IsOwner bypass
	// in the SQL query should make this entry visible.
	ownerNumericID, err := types.ParseUserID(ownerID)
	if err != nil {
		t.Fatalf("parse owner user ID %q: %v", ownerID, err)
	}
	_, err = env.pool.Exec(t.Context(),
		`INSERT INTO activity_log (space_slug, actor_id, entity_type, entity_id, action, created_at)
		 VALUES ('alice-only', $1, 'task', 'T888', 'created', now())`,
		ownerNumericID)
	if err != nil {
		t.Fatalf("insert fabricated entry: %v", err)
	}

	items := activityItems(t, env, "/users/"+ownerID+"/activity")
	var foundAliceOnly bool
	for _, item := range items {
		e := entryMap(t, item)
		if e["spaceSlug"] == "alice-only" {
			foundAliceOnly = true
		}
	}
	if !foundAliceOnly {
		t.Error("owner should see activity from spaces they are not a member of via the IsOwner bypass")
	}
}

// --- Test 10: SpaceActivityLog_Pagination ---

func TestSpaceActivityLog_Pagination(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "ws", "WS")
	// Space creation produces 1 entry. Create 3 tasks for 3 more = 4 total.
	createTask(t, env, "ws", `{"title":"A"}`)
	createTask(t, env, "ws", `{"title":"B"}`)
	createTask(t, env, "ws", `{"title":"C"}`)

	// Page 1: limit=2.
	resp := doRequest(t, env, "GET", "/spaces/ws/activity?limit=2", "")
	assertStatus(t, resp, http.StatusOK)
	var page1 map[string]any
	readJSON(t, resp, &page1)
	items1 := jsonAs[[]any](t, page1["items"])
	if len(items1) != 2 {
		t.Fatalf("page 1: got %d items, want 2", len(items1))
	}
	cursor := page1["nextCursor"]
	if cursor == nil {
		t.Fatal("page 1: expected nextCursor")
	}

	// Verify descending order (compare numerically, not lexicographically).
	id0str := jsonAs[string](t, entryMap(t, items1[0])["id"])
	id1str := jsonAs[string](t, entryMap(t, items1[1])["id"])
	id0, err := strconv.ParseInt(id0str, 10, 64)
	if err != nil {
		t.Fatalf("parse id0 %q: %v", id0str, err)
	}
	id1, err := strconv.ParseInt(id1str, 10, 64)
	if err != nil {
		t.Fatalf("parse id1 %q: %v", id1str, err)
	}
	if id0 <= id1 {
		t.Errorf("page 1: expected descending IDs, got %d <= %d", id0, id1)
	}

	// Page 2.
	resp2 := doRequest(t, env, "GET", "/spaces/ws/activity?limit=2&cursor="+jsonAs[string](t, cursor), "")
	assertStatus(t, resp2, http.StatusOK)
	var page2 map[string]any
	readJSON(t, resp2, &page2)
	items2 := jsonAs[[]any](t, page2["items"])
	if len(items2) != 2 {
		t.Fatalf("page 2: got %d items, want 2", len(items2))
	}

	// Page 2 IDs should be less than page 1's smallest.
	id2str := jsonAs[string](t, entryMap(t, items2[0])["id"])
	id2, err := strconv.ParseInt(id2str, 10, 64)
	if err != nil {
		t.Fatalf("parse id2 %q: %v", id2str, err)
	}
	if id2 >= id1 {
		t.Errorf("page 2 ID %d should be less than page 1 last ID %d", id2, id1)
	}
}

// --- Test 11: SystemNullActor ---

func TestSpaceTaskActivityLog_SystemNullActor(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "ws", "WS")

	// Create a fixed_accumulating task with a past due date so we can call
	// ProcessAccumulatingTask to trigger system-driven spawn (null actor).
	task := createTask(t, env, "ws",
		`{"title":"Weekly","recurrenceType":"fixed_accumulating","recurrenceRule":"FREQ=WEEKLY;INTERVAL=1","due":{"at":"2020-01-01","timezone":"UTC"}}`)
	taskID := jsonAs[string](t, task["id"])

	// Directly invoke the overdue processing (simulates the cron).
	q := dbgen.New(env.pool)
	dbTask, err := q.GetTask(t.Context(), dbgen.GetTaskParams{
		ID:        mustParseTaskID(t, taskID),
		SpaceSlug: "ws",
	})
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if err := taskengine.ProcessAccumulatingTask(t.Context(), env.pool, dbTask, time.Now()); err != nil {
		t.Fatalf("process accumulating task: %v", err)
	}

	// The spawned task should have a null-actor "created" entry.
	items := activityItems(t, env, "/spaces/ws/activity")
	var foundSystemCreate bool
	for _, item := range items {
		e := entryMap(t, item)
		if e["entityType"] == "task" && e["action"] == "created" && e["actorId"] == nil {
			foundSystemCreate = true
			details := entryDetails(t, e)
			if !hasDetail(details, "spawned_from") {
				t.Error("expected spawned_from detail on system-created task")
			}
		}
	}
	if !foundSystemCreate {
		t.Error("expected a system-created task entry with null actorId")
	}
}

// --- Test 12: NoUpdateEntryOnNoOp ---

func TestSpaceTaskActivityLog_NoUpdateEntryOnNoOp(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "ws", "WS")
	task := createTask(t, env, "ws", `{"title":"Static"}`)
	taskID := jsonAs[string](t, task["id"])

	// Empty PATCH — no fields change.
	assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/ws/tasks/"+taskID, `{}`), http.StatusOK)

	items := activityItems(t, env, "/spaces/ws/tasks/"+taskID+"/activity")
	if len(items) != 1 {
		t.Errorf("expected 1 entry (create only), got %d", len(items))
	}
}

// --- Test 13: MemberActivity ---

func TestMemberActivity(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "ws", "WS")
	_, aliceID := createAndAddMember(t, env, "ws", "alice@test.com", "Alice", "pass123", "member")

	// Update Alice's role.
	assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/ws/members/"+aliceID,
		`{"role":"admin"}`), http.StatusOK)

	// Remove Alice.
	assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/ws/members/"+aliceID, ""), http.StatusNoContent)

	items := activityItems(t, env, "/spaces/ws/activity")

	var foundAdd, foundUpdate, foundRemove bool
	for _, item := range items {
		e := entryMap(t, item)
		if e["entityType"] != "member" {
			continue
		}
		switch e["action"] {
		case "created":
			foundAdd = true
			details := entryDetails(t, e)
			d := findDetail(t, details, "role")
			if d["to"] != "member" {
				t.Errorf("member add role.to = %v, want member", d["to"])
			}
		case "updated":
			foundUpdate = true
			details := entryDetails(t, e)
			d := findDetail(t, details, "role")
			if d["from"] != "member" || d["to"] != "admin" {
				t.Errorf("member update role = %v→%v, want member→admin", d["from"], d["to"])
			}
		case "deleted":
			foundRemove = true
		}
	}
	if !foundAdd {
		t.Error("member.created entry not found")
	}
	if !foundUpdate {
		t.Error("member.updated entry not found")
	}
	if !foundRemove {
		t.Error("member.deleted entry not found")
	}
}

// --- Test 14: TagActivity ---

func TestTagActivity(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "ws", "WS")

	// Create a tag.
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/ws/tags",
		`{"name":"chores"}`), http.StatusCreated)

	// Rename the tag.
	assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/ws/tags/chores",
		`{"name":"housework"}`), http.StatusOK)

	items := activityItems(t, env, "/spaces/ws/activity")

	var foundCreate, foundUpdate bool
	for _, item := range items {
		e := entryMap(t, item)
		if e["entityType"] != "tag" {
			continue
		}
		switch e["action"] {
		case "created":
			foundCreate = true
		case "updated":
			foundUpdate = true
			details := entryDetails(t, e)
			d := findDetail(t, details, "name")
			if d["from"] != "chores" || d["to"] != "housework" {
				t.Errorf("tag rename = %v→%v, want chores→housework", d["from"], d["to"])
			}
		}
	}
	if !foundCreate {
		t.Error("tag.created entry not found")
	}
	if !foundUpdate {
		t.Error("tag.updated entry not found")
	}
}

// --- Test 15: RelationActivity ---

func TestRelationActivity(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "ws", "WS")
	taskA := createTask(t, env, "ws", `{"title":"A"}`)
	taskAID := jsonAs[string](t, taskA["id"])
	taskB := createTask(t, env, "ws", `{"title":"B"}`)
	taskBID := jsonAs[string](t, taskB["id"])

	// Create a "blocks" relation from A to B.
	createRelation(t, env, "ws", taskAID, "blocks", taskBID)

	// Delete it.
	assertStatusClose(t, doRequest(t, env, "DELETE",
		"/spaces/ws/tasks/"+taskAID+"/relations/blocks/"+taskBID, ""), http.StatusNoContent)

	items := activityItems(t, env, "/spaces/ws/activity")

	var foundCreate, foundDelete bool
	for _, item := range items {
		e := entryMap(t, item)
		if e["entityType"] != "relation" {
			continue
		}
		details := entryDetails(t, e)
		switch e["action"] {
		case "created":
			foundCreate = true
			d := findDetail(t, details, "kind")
			if d["to"] != "blocks" {
				t.Errorf("relation create kind.to = %v, want blocks", d["to"])
			}
		case "deleted":
			foundDelete = true
		}
	}
	if !foundCreate {
		t.Error("relation.created entry not found")
	}
	if !foundDelete {
		t.Error("relation.deleted entry not found")
	}
}

// --- Test 16: TokenIdAndNameInActivityLog ---

func TestTokenIdAndNameInActivityLog(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "ws", "WS")

	// Create an API token.
	resp := doRequest(t, env, "POST", "/auth/tokens", `{"name":"audit-token"}`)
	assertStatus(t, resp, http.StatusCreated)
	var tokenResult map[string]any
	readJSON(t, resp, &tokenResult)
	rawToken := jsonAs[string](t, tokenResult["token"])
	authToken := jsonAs[map[string]any](t, tokenResult["authToken"])
	tokenID := jsonAs[string](t, authToken["id"])
	tokenName := jsonAs[string](t, authToken["name"])

	// Use the API token to perform a logged action (create a task).
	resp2 := doRequestAs(t, env, rawToken, "POST", "/spaces/ws/tasks", `{"title":"Token task"}`)
	assertStatus(t, resp2, http.StatusCreated)
	var createdTask map[string]any
	readJSON(t, resp2, &createdTask)
	taskID := jsonAs[string](t, createdTask["id"])

	// Read the activity log and find the task-create entry.
	items := activityItems(t, env, "/spaces/ws/tasks/"+taskID+"/activity")
	if len(items) == 0 {
		t.Fatal("expected at least one activity entry for the created task")
	}

	// The newest (and only) entry should be the create.
	entry := entryMap(t, items[0])
	if entry["action"] != "created" || entry["entityType"] != "task" {
		t.Fatalf("unexpected entry: action=%v entityType=%v", entry["action"], entry["entityType"])
	}

	// Assert tokenId and tokenName are present and match.
	if entry["tokenId"] == nil {
		t.Error("expected non-null tokenId on action performed via API token")
	} else if jsonAs[string](t, entry["tokenId"]) != tokenID {
		t.Errorf("tokenId = %v, want %s", entry["tokenId"], tokenID)
	}
	if entry["tokenName"] == nil {
		t.Error("expected non-null tokenName on action performed via API token")
	} else if jsonAs[string](t, entry["tokenName"]) != tokenName {
		t.Errorf("tokenName = %v, want %s", entry["tokenName"], tokenName)
	}
}

// --- Test 17: StatusReplaceActivity ---

func TestStatusReplaceActivity(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "ws", "WS")

	// Default statuses are [todo, done]. Replace with [todo, in_progress, done].
	assertStatusClose(t, doRequest(t, env, "PUT", "/spaces/ws/task-statuses",
		`{"items":[{"name":"todo","category":"initial"},{"name":"in_progress","category":"intermediate"},{"name":"done","category":"completion"}]}`),
		http.StatusOK)

	items := activityItems(t, env, "/spaces/ws/activity")

	var foundStatusCreated bool
	for _, item := range items {
		e := entryMap(t, item)
		if e["entityType"] == "status" && e["action"] == "created" && e["entityId"] == "in_progress" {
			foundStatusCreated = true
		}
	}
	if !foundStatusCreated {
		t.Error("status created entry for 'in_progress' not found")
	}
}

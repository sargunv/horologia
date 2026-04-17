package api_test

import (
	"net/http"
	"testing"
	"time"
)

func TestUserHandlers(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	t.Run("users me", func(t *testing.T) {
		resp := doRequest(t, env, "GET", "/users/me", "")
		assertStatus(t, resp, http.StatusOK)
		var user map[string]any
		readJSON(t, resp, &user)
		if user["email"] != "test@example.com" {
			t.Errorf("email = %v, want test@example.com", user["email"])
		}
		if user["isOwner"] != true {
			t.Errorf("isOwner = %v, want true", user["isOwner"])
		}
	})

	t.Run("user tasks list", func(t *testing.T) {
		userToken := createTestUser(t, env, testEmail(t, "owner"), "Owner", "password")
		ownerID := getUserID(t, env, userToken)
		work := testSlug(t, "work")
		home := testSlug(t, "home")
		createSpace(t, env, work, "Work")
		createSpace(t, env, home, "Home")
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/"+work+"/members", `{"userId":"`+ownerID+`","role":"member"}`), http.StatusCreated)
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/"+home+"/members", `{"userId":"`+ownerID+`","role":"member"}`), http.StatusCreated)
		assertStatusClose(t, doRequest(t, env, "PUT", "/spaces/"+work+"/task-statuses", `{"items":[{"name":"todo","category":"initial"},{"name":"doing","category":"intermediate"},{"name":"done","category":"completion"}]}`), http.StatusOK)
		assertStatusClose(t, doRequest(t, env, "PUT", "/spaces/"+work+"/task-priority-levels", `{"items":[{"name":"high"},{"name":"low"}]}`), http.StatusOK)
		dueDate := time.Now().AddDate(0, 0, 5).Format(time.DateOnly)
		task1 := createTask(t, env, work, `{"title":"Work task 1","status":"doing","due":{"at":"`+dueDate+`","timezone":"UTC"},"priority":"high","assigneeIds":["`+ownerID+`"]}`)
		task2 := createTask(t, env, home, `{"title":"Home task","assigneeIds":["`+ownerID+`"]}`)
		task3 := createTask(t, env, work, `{"title":"Work task done","status":"done","assigneeIds":["`+ownerID+`"]}`)
		_ = createTask(t, env, work, `{"title":"Not mine"}`)
		expectedOrder := []string{jsonAs[string](t, task1["id"]), jsonAs[string](t, task2["id"]), jsonAs[string](t, task3["id"])}
		resp := doRequest(t, env, "GET", "/users/"+ownerID+"/tasks?limit=10", "")
		assertStatus(t, resp, http.StatusOK)
		var page map[string]any
		readJSON(t, resp, &page)
		items := jsonAs[[]any](t, page["items"])
		if len(items) != 3 {
			t.Fatalf("got %d items, want 3", len(items))
		}
		for i, item := range items {
			task := jsonAs[map[string]any](t, item)
			got := jsonAs[string](t, task["id"])
			if got != expectedOrder[i] {
				t.Errorf("position %d: got %s, want %s", i, got, expectedOrder[i])
			}
			if task["spaceSlug"] == nil || task["spaceSlug"] == "" {
				t.Errorf("position %d: spaceSlug should be set", i)
			}
			assignees := jsonAs[[]any](t, task["assigneeIds"])
			if len(assignees) != 1 || jsonAs[string](t, assignees[0]) != ownerID {
				t.Errorf("position %d: assigneeIds = %v, want [%s]", i, assignees, ownerID)
			}
			if jsonAs[[]any](t, task["tags"]) == nil || jsonAs[[]any](t, task["relations"]) == nil || jsonAs[[]any](t, task["rotationPool"]) == nil {
				t.Errorf("position %d: enriched arrays should be non-nil", i)
			}
		}
		if jsonAs[string](t, jsonAs[map[string]any](t, items[0])["spaceSlug"]) != work {
			t.Errorf("item 0 spaceSlug = %v, want %s", jsonAs[map[string]any](t, items[0])["spaceSlug"], work)
		}
		if jsonAs[string](t, jsonAs[map[string]any](t, items[1])["spaceSlug"]) != home {
			t.Errorf("item 1 spaceSlug = %v, want %s", jsonAs[map[string]any](t, items[1])["spaceSlug"], home)
		}
	})

	t.Run("user tasks list pagination", func(t *testing.T) {
		userToken := createTestUser(t, env, testEmail(t, "owner"), "Owner", "password")
		ownerID := getUserID(t, env, userToken)
		space := testSlug(t, "space")
		createSpace(t, env, space, "Space")
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/"+space+"/members", `{"userId":"`+ownerID+`","role":"member"}`), http.StatusCreated)
		task1 := createTask(t, env, space, `{"title":"T1","assigneeIds":["`+ownerID+`"]}`)
		task2 := createTask(t, env, space, `{"title":"T2","assigneeIds":["`+ownerID+`"]}`)
		task3 := createTask(t, env, space, `{"title":"T3","assigneeIds":["`+ownerID+`"]}`)
		expectedOrder := []string{jsonAs[string](t, task1["id"]), jsonAs[string](t, task2["id"]), jsonAs[string](t, task3["id"])}
		resp := doRequest(t, env, "GET", "/users/"+ownerID+"/tasks?limit=2", "")
		assertStatus(t, resp, http.StatusOK)
		var page1 map[string]any
		readJSON(t, resp, &page1)
		items1 := jsonAs[[]any](t, page1["items"])
		if len(items1) != 2 {
			t.Fatalf("page 1: got %d items, want 2", len(items1))
		}
		if jsonAs[string](t, jsonAs[map[string]any](t, items1[0])["id"]) != expectedOrder[0] || jsonAs[string](t, jsonAs[map[string]any](t, items1[1])["id"]) != expectedOrder[1] {
			t.Fatalf("page 1 ids = %v, want first two expected ids", items1)
		}
		cursor := jsonAs[string](t, page1["nextCursor"])
		resp2 := doRequest(t, env, "GET", "/users/"+ownerID+"/tasks?limit=2&cursor="+cursor, "")
		assertStatus(t, resp2, http.StatusOK)
		var page2 map[string]any
		readJSON(t, resp2, &page2)
		items2 := jsonAs[[]any](t, page2["items"])
		if len(items2) != 1 || jsonAs[string](t, jsonAs[map[string]any](t, items2[0])["id"]) != expectedOrder[2] {
			t.Fatalf("page 2 items = %v, want final expected id", items2)
		}
		if page2["nextCursor"] != nil {
			t.Errorf("page 2: expected no nextCursor, got %v", page2["nextCursor"])
		}
	})

	t.Run("user tasks list cross space identical sort keys", func(t *testing.T) {
		userToken := createTestUser(t, env, testEmail(t, "owner"), "Owner", "password")
		ownerID := getUserID(t, env, userToken)
		a := testSlug(t, "a")
		b := testSlug(t, "b")
		c := testSlug(t, "c")
		createSpace(t, env, a, "Alpha")
		createSpace(t, env, b, "Beta")
		createSpace(t, env, c, "Gamma")
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/"+a+"/members", `{"userId":"`+ownerID+`","role":"member"}`), http.StatusCreated)
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/"+b+"/members", `{"userId":"`+ownerID+`","role":"member"}`), http.StatusCreated)
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/"+c+"/members", `{"userId":"`+ownerID+`","role":"member"}`), http.StatusCreated)
		task1 := createTask(t, env, a, `{"title":"A1","assigneeIds":["`+ownerID+`"]}`)
		task2 := createTask(t, env, b, `{"title":"B1","assigneeIds":["`+ownerID+`"]}`)
		task3 := createTask(t, env, c, `{"title":"C1","assigneeIds":["`+ownerID+`"]}`)
		expectedOrder := []string{jsonAs[string](t, task1["id"]), jsonAs[string](t, task2["id"]), jsonAs[string](t, task3["id"])}
		resp := doRequest(t, env, "GET", "/users/"+ownerID+"/tasks?limit=2", "")
		assertStatus(t, resp, http.StatusOK)
		var page1 map[string]any
		readJSON(t, resp, &page1)
		items1 := jsonAs[[]any](t, page1["items"])
		if len(items1) != 2 {
			t.Fatalf("page 1: got %d items, want 2", len(items1))
		}
		if jsonAs[string](t, jsonAs[map[string]any](t, items1[0])["id"]) != expectedOrder[0] || jsonAs[string](t, jsonAs[map[string]any](t, items1[1])["id"]) != expectedOrder[1] {
			t.Fatalf("page 1 ids = %v, want first two expected ids", items1)
		}
		cursor := jsonAs[string](t, page1["nextCursor"])
		resp2 := doRequest(t, env, "GET", "/users/"+ownerID+"/tasks?limit=2&cursor="+cursor, "")
		assertStatus(t, resp2, http.StatusOK)
		var page2 map[string]any
		readJSON(t, resp2, &page2)
		items2 := jsonAs[[]any](t, page2["items"])
		if len(items2) != 1 || jsonAs[string](t, jsonAs[map[string]any](t, items2[0])["id"]) != expectedOrder[2] {
			t.Fatalf("page 2 items = %v, want final expected id", items2)
		}
		if page2["nextCursor"] != nil {
			t.Errorf("page 2: expected no nextCursor, got %v", page2["nextCursor"])
		}
	})

	t.Run("user tasks list forbidden for other user", func(t *testing.T) {
		token := createTestUser(t, env, testEmail(t, "alice"), "Alice", "password")
		aliceID := getUserID(t, env, token)
		ownerID := getUserID(t, env, env.Token)
		resp := doRequestAs(t, env, token, "GET", "/users/"+ownerID+"/tasks", "")
		assertStatusClose(t, resp, http.StatusForbidden)
		resp2 := doRequest(t, env, "GET", "/users/"+aliceID+"/tasks", "")
		assertStatus(t, resp2, http.StatusOK)
		var page map[string]any
		readJSON(t, resp2, &page)
		items := jsonAs[[]any](t, page["items"])
		if len(items) != 0 {
			t.Errorf("expected 0 tasks for alice, got %d", len(items))
		}
	})

	t.Run("user tasks list empty", func(t *testing.T) {
		userToken := createTestUser(t, env, testEmail(t, "owner"), "Owner", "password")
		ownerID := getUserID(t, env, userToken)
		space := testSlug(t, "space")
		createSpace(t, env, space, "Space")
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/"+space+"/members", `{"userId":"`+ownerID+`","role":"member"}`), http.StatusCreated)
		createTask(t, env, space, `{"title":"Unassigned"}`)
		resp := doRequest(t, env, "GET", "/users/"+ownerID+"/tasks", "")
		assertStatus(t, resp, http.StatusOK)
		var page map[string]any
		readJSON(t, resp, &page)
		items := jsonAs[[]any](t, page["items"])
		if len(items) != 0 {
			t.Errorf("expected 0 items, got %d", len(items))
		}
	})
}

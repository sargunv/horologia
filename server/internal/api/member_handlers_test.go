package api_test

import (
	"net/http"
	"testing"
)

func TestSpaceMemberHandlers(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	t.Run("create", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		aliceToken := createTestUser(t, env, testEmail(t, "alice"), "Alice", "pass1234")
		userID := getUserID(t, env, aliceToken)
		resp := doRequest(t, env, "POST", "/spaces/"+space+"/members", `{"userId":"`+userID+`","role":"member"}`)
		assertStatus(t, resp, http.StatusCreated)
		var member map[string]any
		readJSON(t, resp, &member)
		if member["userId"] != userID || member["role"] != "member" || member["userName"] != "Alice" {
			t.Fatalf("unexpected member: %v", member)
		}
	})

	t.Run("create duplicate", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		aliceToken := createTestUser(t, env, testEmail(t, "alice"), "Alice", "pass1234")
		userID := getUserID(t, env, aliceToken)
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/"+space+"/members", `{"userId":"`+userID+`","role":"member"}`), http.StatusCreated)
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/"+space+"/members", `{"userId":"`+userID+`","role":"member"}`), http.StatusConflict)
	})

	t.Run("list", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		resp := doRequest(t, env, "GET", "/spaces/"+space+"/members", "")
		assertStatus(t, resp, http.StatusOK)
		var list map[string]any
		readJSON(t, resp, &list)
		items := jsonAs[[]any](t, list["items"])
		if len(items) != 1 || jsonAs[map[string]any](t, items[0])["role"] != "admin" {
			t.Fatalf("unexpected members list: %v", list)
		}
	})

	t.Run("update", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		_, userID := createAndAddMember(t, env, space, testEmail(t, "bob"), "Bob", "pass1234", "member")
		resp := doRequest(t, env, "PATCH", "/spaces/"+space+"/members/"+userID, `{"role":"admin"}`)
		assertStatus(t, resp, http.StatusOK)
		var updated map[string]any
		readJSON(t, resp, &updated)
		if updated["role"] != "admin" {
			t.Fatalf("unexpected member: %v", updated)
		}
	})

	t.Run("update not found", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/"+space+"/members/U999999", `{"role":"admin"}`), http.StatusNotFound)
	})

	t.Run("delete", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		userToken, userID := createAndAddMember(t, env, space, testEmail(t, "charlie"), "Charlie", "pass1234", "member")
		assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/"+space+"/members/"+userID, ""), http.StatusNoContent)
		assertStatusClose(t, doRequestAs(t, env, userToken, "GET", "/spaces/"+space, ""), http.StatusNotFound)
	})

	t.Run("last admin guard", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		ownerID := getUserID(t, env, env.Token)
		assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/"+space+"/members/"+ownerID, `{"role":"viewer"}`), http.StatusBadRequest)
		assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/"+space+"/members/"+ownerID, ""), http.StatusBadRequest)
	})

	t.Run("member removal clears assignments", func(t *testing.T) {
		space := testSlug(t, "home")
		createSpace(t, env, space, "Home")
		_, bobID := createAndAddMember(t, env, space, testEmail(t, "bob"), "Bob", "pass1234", "member")
		task := createTask(t, env, space, `{"title":"Bob's task","assigneeIds":["`+bobID+`"]}`)
		taskID := jsonAs[string](t, task["id"])
		resp := doRequest(t, env, "GET", "/spaces/"+space+"/tasks/"+taskID, "")
		assertStatus(t, resp, http.StatusOK)
		var before map[string]any
		readJSON(t, resp, &before)
		if len(jsonAs[[]any](t, before["assigneeIds"])) != 1 {
			t.Fatal("expected 1 assignee before removal")
		}
		assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/"+space+"/members/"+bobID, ""), http.StatusNoContent)
		resp = doRequest(t, env, "GET", "/spaces/"+space+"/tasks/"+taskID, "")
		assertStatus(t, resp, http.StatusOK)
		var after map[string]any
		readJSON(t, resp, &after)
		if len(jsonAs[[]any](t, after["assigneeIds"])) != 0 {
			t.Fatalf("expected 0 assignees after member removal, got %d", len(jsonAs[[]any](t, after["assigneeIds"])))
		}
	})

	t.Run("member removal isolation", func(t *testing.T) {
		alpha := testSlug(t, "alpha")
		beta := testSlug(t, "beta")
		createSpace(t, env, alpha, "Alpha")
		createSpace(t, env, beta, "Beta")
		userToken := createTestUser(t, env, testEmail(t, "bob"), "Bob", "pass1234")
		bobID := getUserID(t, env, userToken)
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/"+alpha+"/members", `{"userId":"`+bobID+`","role":"member"}`), http.StatusCreated)
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/"+beta+"/members", `{"userId":"`+bobID+`","role":"member"}`), http.StatusCreated)
		taskAlpha := createTask(t, env, alpha, `{"title":"Alpha task","assigneeIds":["`+bobID+`"]}`)
		taskBeta := createTask(t, env, beta, `{"title":"Beta task","assigneeIds":["`+bobID+`"]}`)
		assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/"+alpha+"/members/"+bobID, ""), http.StatusNoContent)
		resp := doRequest(t, env, "GET", "/spaces/"+alpha+"/tasks/"+jsonAs[string](t, taskAlpha["id"]), "")
		assertStatus(t, resp, http.StatusOK)
		var alphaTask map[string]any
		readJSON(t, resp, &alphaTask)
		if len(jsonAs[[]any](t, alphaTask["assigneeIds"])) != 0 {
			t.Fatalf("alpha task assignees = %v", alphaTask["assigneeIds"])
		}
		resp = doRequest(t, env, "GET", "/spaces/"+beta+"/tasks/"+jsonAs[string](t, taskBeta["id"]), "")
		assertStatus(t, resp, http.StatusOK)
		var betaTask map[string]any
		readJSON(t, resp, &betaTask)
		if len(jsonAs[[]any](t, betaTask["assigneeIds"])) != 1 {
			t.Fatalf("beta task assignees = %v", betaTask["assigneeIds"])
		}
	})
}

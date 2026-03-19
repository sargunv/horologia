package api_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sargunv/tend/server/internal/api"
	"github.com/sargunv/tend/server/internal/database"
)

func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrator, err := database.NewMigrator(db)
	if err != nil {
		t.Fatalf("new migrator: %v", err)
	}
	if _, err := migrator.Up(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := &api.Handler{DB: db, Log: log}
	h, err := api.NewServer(handler, log)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	return httptest.NewServer(h)
}

func doRequest(t *testing.T, ts *httptest.Server, method, path, body string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, ts.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func readJSON(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("unmarshal %q: %v", string(data), err)
	}
}

// assertStatus checks the response status code. On failure it drains and closes
// the body. On success the body is left open for the caller to read or close.
func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("got status %d, want %d; body: %s", resp.StatusCode, want, string(body))
	}
}

// assertStatusClose checks the response status code and always closes the body.
// Use when the caller does not need to read the response body.
func assertStatusClose(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	assertStatus(t, resp, want)
	_ = resp.Body.Close()
}

// createSpace is a test helper that creates a space and returns the response.
func createSpace(t *testing.T, ts *httptest.Server, slug, name string) map[string]any {
	t.Helper()
	body := `{"slug":"` + slug + `","name":"` + name + `"}`
	resp := doRequest(t, ts, "POST", "/spaces", body)
	if resp.StatusCode != http.StatusCreated {
		data, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("createSpace %s: got status %d; body: %s", slug, resp.StatusCode, data)
	}
	var result map[string]any
	readJSON(t, resp, &result)
	return result
}

// createTask is a test helper that creates a task and returns the response.
func createTask(t *testing.T, ts *httptest.Server, spaceSlug, jsonBody string) map[string]any {
	t.Helper()
	resp := doRequest(t, ts, "POST", "/spaces/"+spaceSlug+"/tasks", jsonBody)
	if resp.StatusCode != http.StatusCreated {
		data, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("createTask: got status %d; body: %s", resp.StatusCode, data)
	}
	var result map[string]any
	readJSON(t, resp, &result)
	return result
}

// --- Space Tests ---

func TestSpacesCreate(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	resp := doRequest(t, ts, "POST", "/spaces", `{"slug":"home","name":"Home"}`)
	assertStatus(t, resp, http.StatusCreated)

	var space map[string]any
	readJSON(t, resp, &space)
	if space["slug"] != "home" {
		t.Errorf("slug = %v, want home", space["slug"])
	}
	if space["name"] != "Home" {
		t.Errorf("name = %v, want Home", space["name"])
	}
	if space["description"] != "" {
		t.Errorf("description = %v, want empty", space["description"])
	}
	if space["createdAt"] == nil || space["createdAt"] == "" {
		t.Error("createdAt should be set")
	}
	if space["updatedAt"] == nil || space["updatedAt"] == "" {
		t.Error("updatedAt should be set")
	}
}

func TestSpacesCreateDuplicate(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	createSpace(t, ts, "home", "Home")

	resp := doRequest(t, ts, "POST", "/spaces", `{"slug":"home","name":"Home 2"}`)
	assertStatusClose(t, resp, http.StatusConflict)
}

func TestSpacesRead(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	createSpace(t, ts, "home", "Home")

	resp := doRequest(t, ts, "GET", "/spaces/home", "")
	assertStatus(t, resp, http.StatusOK)

	var space map[string]any
	readJSON(t, resp, &space)
	if space["slug"] != "home" {
		t.Errorf("slug = %v, want home", space["slug"])
	}
}

func TestSpacesReadNotFound(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	resp := doRequest(t, ts, "GET", "/spaces/nonexistent", "")
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestSpacesListEmpty(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	resp := doRequest(t, ts, "GET", "/spaces", "")
	assertStatus(t, resp, http.StatusOK)

	var page map[string]any
	readJSON(t, resp, &page)
	items := page["items"].([]any)
	if len(items) != 0 {
		t.Fatalf("got %d items, want 0", len(items))
	}
	if page["nextCursor"] != nil {
		t.Error("expected null nextCursor")
	}
}

func TestSpacesList(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	// Insert in non-alphabetical order to prove sort is applied.
	createSpace(t, ts, "gamma", "Gamma")
	createSpace(t, ts, "alpha", "Alpha")
	createSpace(t, ts, "beta", "Beta")

	resp := doRequest(t, ts, "GET", "/spaces", "")
	assertStatus(t, resp, http.StatusOK)

	var page map[string]any
	readJSON(t, resp, &page)
	items := page["items"].([]any)
	slugs := make([]string, len(items))
	for i, item := range items {
		slugs[i] = item.(map[string]any)["slug"].(string)
	}
	if slugs[0] != "alpha" || slugs[1] != "beta" || slugs[2] != "gamma" {
		t.Errorf("slugs = %v, want [alpha beta gamma]", slugs)
	}
}

func TestSpacesListPagination(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	// Create 4 items to test exact page boundary (4 items / limit 2 = 2 full pages).
	createSpace(t, ts, "a", "A")
	createSpace(t, ts, "b", "B")
	createSpace(t, ts, "c", "C")
	createSpace(t, ts, "d", "D")

	// Page 1: should return "a" and "b" with a cursor.
	resp := doRequest(t, ts, "GET", "/spaces?limit=2", "")
	assertStatus(t, resp, http.StatusOK)
	var page1 map[string]any
	readJSON(t, resp, &page1)
	items1 := page1["items"].([]any)
	if len(items1) != 2 {
		t.Fatalf("page 1: got %d items, want 2", len(items1))
	}
	if items1[0].(map[string]any)["slug"] != "a" || items1[1].(map[string]any)["slug"] != "b" {
		t.Errorf("page 1: got slugs %v/%v, want a/b",
			items1[0].(map[string]any)["slug"], items1[1].(map[string]any)["slug"])
	}
	cursor := page1["nextCursor"]
	if cursor == nil {
		t.Fatal("page 1: expected nextCursor")
	}

	// Page 2: should return "c" and "d" with null cursor (exact boundary).
	resp2 := doRequest(t, ts, "GET", "/spaces?limit=2&cursor="+cursor.(string), "")
	assertStatus(t, resp2, http.StatusOK)
	var page2 map[string]any
	readJSON(t, resp2, &page2)
	items2 := page2["items"].([]any)
	if len(items2) != 2 {
		t.Fatalf("page 2: got %d items, want 2", len(items2))
	}
	if items2[0].(map[string]any)["slug"] != "c" || items2[1].(map[string]any)["slug"] != "d" {
		t.Errorf("page 2: got slugs %v/%v, want c/d",
			items2[0].(map[string]any)["slug"], items2[1].(map[string]any)["slug"])
	}
	if page2["nextCursor"] != nil {
		t.Error("page 2: expected null nextCursor")
	}
}

func TestSpacesUpdate(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	createSpace(t, ts, "home", "Home")

	// First set a description so we can verify partial update preserves it.
	resp := doRequest(t, ts, "PATCH", "/spaces/home", `{"description":"My house"}`)
	assertStatusClose(t, resp, http.StatusOK)

	// Now update only the name — description should be preserved.
	resp2 := doRequest(t, ts, "PATCH", "/spaces/home", `{"name":"House"}`)
	assertStatus(t, resp2, http.StatusOK)
	var space map[string]any
	readJSON(t, resp2, &space)
	if space["name"] != "House" {
		t.Errorf("name = %v, want House", space["name"])
	}
	if space["description"] != "My house" {
		t.Errorf("description = %v, want My house", space["description"])
	}
}

func TestSpacesDelete(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	createSpace(t, ts, "home", "Home")

	resp := doRequest(t, ts, "DELETE", "/spaces/home", "")
	assertStatusClose(t, resp, http.StatusNoContent)

	// Verify it's gone.
	resp2 := doRequest(t, ts, "GET", "/spaces/home", "")
	assertStatusClose(t, resp2, http.StatusNotFound)
}

// --- Task Tests ---

func TestTasksCreate(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	createSpace(t, ts, "home", "Home")

	resp := doRequest(t, ts, "POST", "/spaces/home/tasks", `{"title":"Wash dishes"}`)
	assertStatus(t, resp, http.StatusCreated)

	var task map[string]any
	readJSON(t, resp, &task)
	if task["title"] != "Wash dishes" {
		t.Errorf("title = %v, want Wash dishes", task["title"])
	}
	// Should default to the initial status.
	status := task["status"].(map[string]any)
	if status["name"] != "todo" {
		t.Errorf("status.name = %v, want todo", status["name"])
	}
	if status["category"] != "initial" {
		t.Errorf("status.category = %v, want initial", status["category"])
	}
	// ID should be T-prefixed.
	id := task["id"].(string)
	if !strings.HasPrefix(id, "T") {
		t.Errorf("id = %v, want T-prefixed", id)
	}
	// Due date should be null.
	if task["dueDate"] != nil {
		t.Errorf("dueDate = %v, want nil", task["dueDate"])
	}
	// Timestamps should be set.
	if task["createdAt"] == nil || task["createdAt"] == "" {
		t.Error("createdAt should be set")
	}
}

func TestTasksCreateWithFields(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	createSpace(t, ts, "home", "Home")

	body := `{"title":"Clean","description":"Deep clean","statusName":"done","dueDate":"2025-06-15"}`
	resp := doRequest(t, ts, "POST", "/spaces/home/tasks", body)
	assertStatus(t, resp, http.StatusCreated)

	var task map[string]any
	readJSON(t, resp, &task)
	if task["description"] != "Deep clean" {
		t.Errorf("description = %v, want Deep clean", task["description"])
	}
	status := task["status"].(map[string]any)
	if status["name"] != "done" {
		t.Errorf("status.name = %v, want done", status["name"])
	}
	if status["category"] != "completion" {
		t.Errorf("status.category = %v, want completion", status["category"])
	}
	if task["dueDate"] == nil {
		t.Error("dueDate should not be nil")
	}
}

func TestTasksCreateInNonexistentSpace(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	resp := doRequest(t, ts, "POST", "/spaces/nonexistent/tasks", `{"title":"Task"}`)
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestTasksCreateInvalidStatus(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	createSpace(t, ts, "home", "Home")

	resp := doRequest(t, ts, "POST", "/spaces/home/tasks", `{"title":"Task","statusName":"bogus"}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTasksRead(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	createSpace(t, ts, "home", "Home")
	created := createTask(t, ts, "home", `{"title":"Mop floors"}`)
	id := created["id"].(string)

	resp := doRequest(t, ts, "GET", "/tasks/"+id, "")
	assertStatus(t, resp, http.StatusOK)
	var task map[string]any
	readJSON(t, resp, &task)
	if task["title"] != "Mop floors" {
		t.Errorf("title = %v, want Mop floors", task["title"])
	}
}

func TestTasksReadNotFound(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	resp := doRequest(t, ts, "GET", "/tasks/T999", "")
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestTasksReadInvalidID(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	resp := doRequest(t, ts, "GET", "/tasks/invalid", "")
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTasksListEmpty(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	createSpace(t, ts, "home", "Home")

	resp := doRequest(t, ts, "GET", "/spaces/home/tasks", "")
	assertStatus(t, resp, http.StatusOK)

	var page map[string]any
	readJSON(t, resp, &page)
	items := page["items"].([]any)
	if len(items) != 0 {
		t.Fatalf("got %d items, want 0", len(items))
	}
	if page["nextCursor"] != nil {
		t.Error("expected null nextCursor")
	}
}

func TestTasksListPagination(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	createSpace(t, ts, "home", "Home")
	// Create 4 tasks with distinct titles for verification.
	var createdIDs []string
	for i := 1; i <= 4; i++ {
		task := createTask(t, ts, "home", `{"title":"Task"}`)
		createdIDs = append(createdIDs, task["id"].(string))
	}

	// Page 1: should return 2 items with a cursor.
	resp := doRequest(t, ts, "GET", "/spaces/home/tasks?limit=2", "")
	assertStatus(t, resp, http.StatusOK)
	var page1 map[string]any
	readJSON(t, resp, &page1)
	items1 := page1["items"].([]any)
	if len(items1) != 2 {
		t.Fatalf("page 1: got %d items, want 2", len(items1))
	}
	// Verify page 1 contains the first two created tasks.
	if items1[0].(map[string]any)["id"] != createdIDs[0] || items1[1].(map[string]any)["id"] != createdIDs[1] {
		t.Errorf("page 1: got ids %v/%v, want %v/%v",
			items1[0].(map[string]any)["id"], items1[1].(map[string]any)["id"],
			createdIDs[0], createdIDs[1])
	}
	cursor := page1["nextCursor"]
	if cursor == nil {
		t.Fatal("page 1: expected nextCursor")
	}

	// Page 2: should return the last 2 items with null cursor (exact boundary).
	resp2 := doRequest(t, ts, "GET", "/spaces/home/tasks?limit=2&cursor="+cursor.(string), "")
	assertStatus(t, resp2, http.StatusOK)
	var page2 map[string]any
	readJSON(t, resp2, &page2)
	items2 := page2["items"].([]any)
	if len(items2) != 2 {
		t.Fatalf("page 2: got %d items, want 2", len(items2))
	}
	if items2[0].(map[string]any)["id"] != createdIDs[2] || items2[1].(map[string]any)["id"] != createdIDs[3] {
		t.Errorf("page 2: got ids %v/%v, want %v/%v",
			items2[0].(map[string]any)["id"], items2[1].(map[string]any)["id"],
			createdIDs[2], createdIDs[3])
	}
	if page2["nextCursor"] != nil {
		t.Error("page 2: expected null nextCursor")
	}
}

func TestTasksListNonexistentSpace(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	resp := doRequest(t, ts, "GET", "/spaces/nonexistent/tasks", "")
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestTasksUpdate(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	createSpace(t, ts, "home", "Home")
	// Create with a non-empty description to test merge preservation.
	created := createTask(t, ts, "home", `{"title":"Old title","description":"Keep me"}`)
	id := created["id"].(string)

	resp := doRequest(t, ts, "PATCH", "/tasks/"+id, `{"title":"New title"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)
	if updated["title"] != "New title" {
		t.Errorf("title = %v, want New title", updated["title"])
	}
	// Description should be preserved from the original.
	if updated["description"] != "Keep me" {
		t.Errorf("description = %v, want Keep me", updated["description"])
	}
}

func TestTasksUpdateStatus(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	createSpace(t, ts, "home", "Home")
	created := createTask(t, ts, "home", `{"title":"Chore"}`)
	id := created["id"].(string)

	resp := doRequest(t, ts, "PATCH", "/tasks/"+id, `{"statusName":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)
	status := updated["status"].(map[string]any)
	if status["name"] != "done" {
		t.Errorf("status.name = %v, want done", status["name"])
	}
	if status["category"] != "completion" {
		t.Errorf("status.category = %v, want completion", status["category"])
	}
}

func TestTasksUpdateInvalidStatus(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	createSpace(t, ts, "home", "Home")
	created := createTask(t, ts, "home", `{"title":"Chore"}`)
	id := created["id"].(string)

	resp := doRequest(t, ts, "PATCH", "/tasks/"+id, `{"statusName":"nonexistent"}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTasksUpdateClearDueDate(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	createSpace(t, ts, "home", "Home")
	created := createTask(t, ts, "home", `{"title":"Task","dueDate":"2025-06-15"}`)
	id := created["id"].(string)

	// Verify the due date was actually set.
	resp := doRequest(t, ts, "GET", "/tasks/"+id, "")
	assertStatus(t, resp, http.StatusOK)
	var fetched map[string]any
	readJSON(t, resp, &fetched)
	if fetched["dueDate"] == nil {
		t.Fatal("dueDate should be set after creation")
	}

	// Clear due date by sending null.
	resp2 := doRequest(t, ts, "PATCH", "/tasks/"+id, `{"dueDate":null}`)
	assertStatus(t, resp2, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp2, &updated)
	if updated["dueDate"] != nil {
		t.Errorf("dueDate = %v, want nil", updated["dueDate"])
	}
}

func TestTasksDelete(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	createSpace(t, ts, "home", "Home")
	created := createTask(t, ts, "home", `{"title":"Temp"}`)
	id := created["id"].(string)

	resp := doRequest(t, ts, "DELETE", "/tasks/"+id, "")
	assertStatusClose(t, resp, http.StatusNoContent)

	// Verify it's gone.
	resp2 := doRequest(t, ts, "GET", "/tasks/"+id, "")
	assertStatusClose(t, resp2, http.StatusNotFound)
}

func TestSpaceDeleteCascadesTasks(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	createSpace(t, ts, "home", "Home")
	created := createTask(t, ts, "home", `{"title":"Chore"}`)
	id := created["id"].(string)

	// Delete the space - tasks should cascade.
	resp := doRequest(t, ts, "DELETE", "/spaces/home", "")
	assertStatusClose(t, resp, http.StatusNoContent)

	// Task should be gone.
	resp2 := doRequest(t, ts, "GET", "/tasks/"+id, "")
	assertStatusClose(t, resp2, http.StatusNotFound)
}

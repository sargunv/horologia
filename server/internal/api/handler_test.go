package api_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sargunv/tend/server/internal/api"
	"github.com/sargunv/tend/server/internal/database"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

type testEnv struct {
	Server *httptest.Server
	Token  string
	db     *sql.DB
}

func setupTestServer(t *testing.T) *testEnv {
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

	// Create a test owner user.
	user, err := database.CreateUserWithPassword(context.Background(), db, "test@example.com", "Test User", "password", true)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Create a test auth token.
	rawToken := "test-token-for-integration-tests"
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	q := dbgen.New(db)
	_, err = q.CreateAuthToken(context.Background(), dbgen.CreateAuthTokenParams{
		UserID:    user.ID,
		TokenHash: tokenHash,
		Name:      "test",
		Kind:      "session",
		CreatedAt: types.Now(),
	})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := &api.Handler{DB: db, Log: log}
	h, err := api.NewServer(handler, log)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	return &testEnv{
		Server: srv,
		Token:  rawToken,
		db:     db,
	}
}

func doRequestAs(t *testing.T, env *testEnv, token, method, path, body string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, env.Server.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func doRequest(t *testing.T, env *testEnv, method, path, body string) *http.Response {
	t.Helper()
	return doRequestAs(t, env, env.Token, method, path, body)
}

// createTestUser creates a non-owner user via the DB and logs them in to get a token.
func createTestUser(t *testing.T, env *testEnv, email, name, password string) string {
	t.Helper()
	_, err := database.CreateUserWithPassword(context.Background(), env.db, email, name, password, false)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	// Login to get a token.
	resp := doRequestAs(t, env, "", "POST", "/auth/login",
		`{"email":"`+email+`","password":"`+password+`"}`)
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("login test user: got status %d; body: %s", resp.StatusCode, data)
	}
	var result map[string]any
	readJSON(t, resp, &result)
	return result["token"].(string)
}

// getUserID calls GET /users/me with the given token and returns the user ID.
func getUserID(t *testing.T, env *testEnv, token string) string {
	t.Helper()
	resp := doRequestAs(t, env, token, "GET", "/users/me", "")
	assertStatus(t, resp, http.StatusOK)
	var me map[string]any
	readJSON(t, resp, &me)
	return me["id"].(string)
}

// createAndAddMember creates a test user, resolves their ID, and adds them to
// the given space with the specified role. Returns (token, userID).
func createAndAddMember(t *testing.T, env *testEnv, spaceSlug, email, name, password, role string) (string, string) {
	t.Helper()
	token := createTestUser(t, env, email, name, password)
	userID := getUserID(t, env, token)
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/"+spaceSlug+"/members",
		`{"userId":"`+userID+`","role":"`+role+`"}`), http.StatusCreated)
	return token, userID
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
func createSpace(t *testing.T, env *testEnv, slug, name string) map[string]any {
	t.Helper()
	body := `{"slug":"` + slug + `","name":"` + name + `"}`
	resp := doRequest(t, env, "POST", "/spaces", body)
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
func createTask(t *testing.T, env *testEnv, spaceSlug, jsonBody string) map[string]any {
	t.Helper()
	resp := doRequest(t, env, "POST", "/spaces/"+spaceSlug+"/tasks", jsonBody)
	if resp.StatusCode != http.StatusCreated {
		data, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("createTask: got status %d; body: %s", resp.StatusCode, data)
	}
	var result map[string]any
	readJSON(t, resp, &result)
	return result
}

// --- Auth Tests ---

func TestAuthLoginSuccess(t *testing.T) {
	env := setupTestServer(t)

	// Login doesn't need auth header, but doRequest always adds one. Use a raw request.
	req, _ := http.NewRequest("POST", env.Server.URL+"/auth/login", strings.NewReader(`{"email":"test@example.com","password":"password"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)

	var result map[string]any
	readJSON(t, resp, &result)
	if result["token"] == nil || result["token"] == "" {
		t.Error("expected a token in response")
	}
	user := result["user"].(map[string]any)
	if user["email"] != "test@example.com" {
		t.Errorf("email = %v, want test@example.com", user["email"])
	}
}

func TestAuthLoginBadPassword(t *testing.T) {
	env := setupTestServer(t)

	req, _ := http.NewRequest("POST", env.Server.URL+"/auth/login", strings.NewReader(`{"email":"test@example.com","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestUnauthenticatedRequest(t *testing.T) {
	env := setupTestServer(t)

	// Request without auth header.
	req, _ := http.NewRequest("GET", env.Server.URL+"/spaces", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	assertStatusClose(t, resp, http.StatusUnauthorized)
}

func TestUsersMe(t *testing.T) {
	env := setupTestServer(t)

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
}

// --- Space Tests ---

func TestSpacesCreate(t *testing.T) {
	env := setupTestServer(t)

	resp := doRequest(t, env, "POST", "/spaces", `{"slug":"home","name":"Home"}`)
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
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "POST", "/spaces", `{"slug":"home","name":"Home 2"}`)
	assertStatusClose(t, resp, http.StatusConflict)
}

func TestSpacesRead(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "GET", "/spaces/home", "")
	assertStatus(t, resp, http.StatusOK)

	var space map[string]any
	readJSON(t, resp, &space)
	if space["slug"] != "home" {
		t.Errorf("slug = %v, want home", space["slug"])
	}
}

func TestSpacesReadNotFound(t *testing.T) {
	env := setupTestServer(t)

	resp := doRequest(t, env, "GET", "/spaces/nonexistent", "")
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestSpacesListEmpty(t *testing.T) {
	env := setupTestServer(t)

	resp := doRequest(t, env, "GET", "/spaces", "")
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
	env := setupTestServer(t)

	// Insert in non-alphabetical order to prove sort is applied.
	createSpace(t, env, "gamma", "Gamma")
	createSpace(t, env, "alpha", "Alpha")
	createSpace(t, env, "beta", "Beta")

	resp := doRequest(t, env, "GET", "/spaces", "")
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
	env := setupTestServer(t)

	// Create 4 items to test exact page boundary (4 items / limit 2 = 2 full pages).
	createSpace(t, env, "a", "A")
	createSpace(t, env, "b", "B")
	createSpace(t, env, "c", "C")
	createSpace(t, env, "d", "D")

	// Page 1: should return "a" and "b" with a cursor.
	resp := doRequest(t, env, "GET", "/spaces?limit=2", "")
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
	resp2 := doRequest(t, env, "GET", "/spaces?limit=2&cursor="+cursor.(string), "")
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
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	// First set a description so we can verify partial update preserves it.
	resp := doRequest(t, env, "PATCH", "/spaces/home", `{"description":"My house"}`)
	assertStatusClose(t, resp, http.StatusOK)

	// Now update only the name — description should be preserved.
	resp2 := doRequest(t, env, "PATCH", "/spaces/home", `{"name":"House"}`)
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
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "DELETE", "/spaces/home", "")
	assertStatusClose(t, resp, http.StatusNoContent)

	// Verify it's gone.
	resp2 := doRequest(t, env, "GET", "/spaces/home", "")
	assertStatusClose(t, resp2, http.StatusNotFound)
}

// --- Task Tests ---

func TestTasksCreate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "POST", "/spaces/home/tasks", `{"title":"Wash dishes"}`)
	assertStatus(t, resp, http.StatusCreated)

	var task map[string]any
	readJSON(t, resp, &task)
	if task["title"] != "Wash dishes" {
		t.Errorf("title = %v, want Wash dishes", task["title"])
	}
	// Should default to the initial status.
	if task["status"] != "todo" {
		t.Errorf("status = %v, want todo", task["status"])
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
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	body := `{"title":"Clean","description":"Deep clean","status":"done","dueDate":"2025-06-15"}`
	resp := doRequest(t, env, "POST", "/spaces/home/tasks", body)
	assertStatus(t, resp, http.StatusCreated)

	var task map[string]any
	readJSON(t, resp, &task)
	if task["description"] != "Deep clean" {
		t.Errorf("description = %v, want Deep clean", task["description"])
	}
	if task["status"] != "done" {
		t.Errorf("status = %v, want done", task["status"])
	}
	if task["dueDate"] == nil {
		t.Error("dueDate should not be nil")
	}
}

func TestTasksCreateInNonexistentSpace(t *testing.T) {
	env := setupTestServer(t)

	resp := doRequest(t, env, "POST", "/spaces/nonexistent/tasks", `{"title":"Task"}`)
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestTasksCreateInvalidStatus(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "POST", "/spaces/home/tasks", `{"title":"Task","status":"bogus"}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTasksRead(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	created := createTask(t, env, "home", `{"title":"Mop floors"}`)
	id := created["id"].(string)

	resp := doRequest(t, env, "GET", "/spaces/home/tasks/"+id, "")
	assertStatus(t, resp, http.StatusOK)
	var task map[string]any
	readJSON(t, resp, &task)
	if task["title"] != "Mop floors" {
		t.Errorf("title = %v, want Mop floors", task["title"])
	}
}

func TestTasksReadNotFound(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "GET", "/spaces/home/tasks/T999", "")
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestTasksReadInvalidID(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "GET", "/spaces/home/tasks/invalid", "")
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTasksListEmpty(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "GET", "/spaces/home/tasks", "")
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
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	// Create 4 tasks with distinct titles for verification.
	var createdIDs []string
	for i := 1; i <= 4; i++ {
		task := createTask(t, env, "home", `{"title":"Task"}`)
		createdIDs = append(createdIDs, task["id"].(string))
	}

	// Page 1: should return 2 items with a cursor.
	resp := doRequest(t, env, "GET", "/spaces/home/tasks?limit=2", "")
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
	resp2 := doRequest(t, env, "GET", "/spaces/home/tasks?limit=2&cursor="+cursor.(string), "")
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
	env := setupTestServer(t)

	resp := doRequest(t, env, "GET", "/spaces/nonexistent/tasks", "")
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestTasksUpdate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	// Create with a non-empty description to test merge preservation.
	created := createTask(t, env, "home", `{"title":"Old title","description":"Keep me"}`)
	id := created["id"].(string)

	resp := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+id, `{"title":"New title"}`)
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
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	created := createTask(t, env, "home", `{"title":"Chore"}`)
	id := created["id"].(string)

	resp := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+id, `{"status":"done"}`)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp, &updated)
	if updated["status"] != "done" {
		t.Errorf("status = %v, want done", updated["status"])
	}
}

func TestTasksUpdateInvalidStatus(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	created := createTask(t, env, "home", `{"title":"Chore"}`)
	id := created["id"].(string)

	resp := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+id, `{"status":"nonexistent"}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTasksUpdateClearDueDate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	created := createTask(t, env, "home", `{"title":"Task","dueDate":"2025-06-15"}`)
	id := created["id"].(string)

	// Verify the due date was actually set.
	resp := doRequest(t, env, "GET", "/spaces/home/tasks/"+id, "")
	assertStatus(t, resp, http.StatusOK)
	var fetched map[string]any
	readJSON(t, resp, &fetched)
	if fetched["dueDate"] == nil {
		t.Fatal("dueDate should be set after creation")
	}

	// Clear due date by sending null.
	resp2 := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+id, `{"dueDate":null}`)
	assertStatus(t, resp2, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp2, &updated)
	if updated["dueDate"] != nil {
		t.Errorf("dueDate = %v, want nil", updated["dueDate"])
	}
}

func TestTasksDelete(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	created := createTask(t, env, "home", `{"title":"Temp"}`)
	id := created["id"].(string)

	resp := doRequest(t, env, "DELETE", "/spaces/home/tasks/"+id, "")
	assertStatusClose(t, resp, http.StatusNoContent)

	// Verify it's gone.
	resp2 := doRequest(t, env, "GET", "/spaces/home/tasks/"+id, "")
	assertStatusClose(t, resp2, http.StatusNotFound)
}

func TestSpaceDeleteCascadesTasks(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	created := createTask(t, env, "home", `{"title":"Chore"}`)
	id := created["id"].(string)

	// Delete the space - tasks should cascade.
	resp := doRequest(t, env, "DELETE", "/spaces/home", "")
	assertStatusClose(t, resp, http.StatusNoContent)

	// Task should be gone.
	resp2 := doRequest(t, env, "GET", "/spaces/home/tasks/"+id, "")
	assertStatusClose(t, resp2, http.StatusNotFound)
}

// --- Authorization Tests ---

func TestNonMemberCannotAccessSpace(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "secret", "Secret")
	userToken := createTestUser(t, env, "bob@example.com", "Bob", "pass123")

	// Non-member cannot read the space.
	assertStatusClose(t, doRequestAs(t, env, userToken, "GET", "/spaces/secret", ""), http.StatusNotFound)
	// Non-member cannot update the space.
	assertStatusClose(t, doRequestAs(t, env, userToken, "PATCH", "/spaces/secret", `{"name":"X"}`), http.StatusNotFound)
	// Non-member cannot delete the space.
	assertStatusClose(t, doRequestAs(t, env, userToken, "DELETE", "/spaces/secret", ""), http.StatusNotFound)
	// Non-member cannot list tasks.
	assertStatusClose(t, doRequestAs(t, env, userToken, "GET", "/spaces/secret/tasks", ""), http.StatusNotFound)
	// Non-member cannot create tasks.
	assertStatusClose(t, doRequestAs(t, env, userToken, "POST", "/spaces/secret/tasks", `{"title":"Task"}`), http.StatusNotFound)
}

func TestViewerCannotWriteToSpace(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	userToken, _ := createAndAddMember(t, env, "home", "viewer@example.com", "Viewer", "pass123", "viewer")

	// Viewer can read the space.
	assertStatusClose(t, doRequestAs(t, env, userToken, "GET", "/spaces/home", ""), http.StatusOK)
	// Viewer can list tasks.
	assertStatusClose(t, doRequestAs(t, env, userToken, "GET", "/spaces/home/tasks", ""), http.StatusOK)
	// Viewer cannot create tasks.
	assertStatusClose(t, doRequestAs(t, env, userToken, "POST", "/spaces/home/tasks", `{"title":"X"}`), http.StatusBadRequest)
	// Viewer cannot update space.
	assertStatusClose(t, doRequestAs(t, env, userToken, "PATCH", "/spaces/home", `{"name":"X"}`), http.StatusBadRequest)
	// Viewer cannot delete space.
	assertStatusClose(t, doRequestAs(t, env, userToken, "DELETE", "/spaces/home", ""), http.StatusBadRequest)
}

func TestMemberCannotManageSpace(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	userToken, userID := createAndAddMember(t, env, "home", "member@example.com", "Member", "pass123", "member")

	// Member can create tasks.
	resp2 := doRequestAs(t, env, userToken, "POST", "/spaces/home/tasks", `{"title":"My task"}`)
	assertStatusClose(t, resp2, http.StatusCreated)
	// Member cannot update space settings.
	assertStatusClose(t, doRequestAs(t, env, userToken, "PATCH", "/spaces/home", `{"name":"X"}`), http.StatusBadRequest)
	// Member cannot delete space.
	assertStatusClose(t, doRequestAs(t, env, userToken, "DELETE", "/spaces/home", ""), http.StatusBadRequest)
	// Member cannot manage members.
	assertStatusClose(t, doRequestAs(t, env, userToken, "POST", "/spaces/home/members", `{"userId":"`+userID+`","role":"viewer"}`), http.StatusBadRequest)
}

func TestNonOwnerSpacesListFiltered(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "alpha", "Alpha")
	createSpace(t, env, "beta", "Beta")
	createSpace(t, env, "gamma", "Gamma")

	userToken, _ := createAndAddMember(t, env, "beta", "user@example.com", "User", "pass123", "member")

	// Non-owner should only see "beta".
	resp2 := doRequestAs(t, env, userToken, "GET", "/spaces", "")
	assertStatus(t, resp2, http.StatusOK)
	var page map[string]any
	readJSON(t, resp2, &page)
	items := page["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].(map[string]any)["slug"] != "beta" {
		t.Errorf("slug = %v, want beta", items[0].(map[string]any)["slug"])
	}
}

// --- Auth Token CRUD Tests ---

func TestAuthTokenCreate(t *testing.T) {
	env := setupTestServer(t)

	resp := doRequest(t, env, "POST", "/auth/tokens", `{"name":"my-token"}`)
	assertStatus(t, resp, http.StatusCreated)

	var result map[string]any
	readJSON(t, resp, &result)
	if result["token"] == nil || result["token"] == "" {
		t.Error("expected token in response")
	}
	authToken := result["authToken"].(map[string]any)
	if authToken["name"] != "my-token" {
		t.Errorf("name = %v, want my-token", authToken["name"])
	}
	if authToken["kind"] != "api" {
		t.Errorf("kind = %v, want api", authToken["kind"])
	}
}

func TestAuthTokenList(t *testing.T) {
	env := setupTestServer(t)

	doRequest(t, env, "POST", "/auth/tokens", `{"name":"token-a"}`)
	doRequest(t, env, "POST", "/auth/tokens", `{"name":"token-b"}`)

	resp := doRequest(t, env, "GET", "/auth/tokens", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := page["items"].([]any)
	// Should include the setup session token + 2 API tokens.
	if len(items) < 2 {
		t.Fatalf("got %d items, want at least 2", len(items))
	}
}

func TestAuthTokenDelete(t *testing.T) {
	env := setupTestServer(t)

	// Create a token.
	resp := doRequest(t, env, "POST", "/auth/tokens", `{"name":"disposable"}`)
	assertStatus(t, resp, http.StatusCreated)
	var result map[string]any
	readJSON(t, resp, &result)
	rawToken := result["token"].(string)
	tokenID := result["authToken"].(map[string]any)["id"].(string)

	// Verify the new token works.
	assertStatusClose(t, doRequestAs(t, env, rawToken, "GET", "/users/me", ""), http.StatusOK)

	// Delete it.
	assertStatusClose(t, doRequest(t, env, "DELETE", "/auth/tokens/"+tokenID, ""), http.StatusNoContent)

	// Using the deleted token should fail.
	assertStatusClose(t, doRequestAs(t, env, rawToken, "GET", "/users/me", ""), http.StatusUnauthorized)
}

func TestAuthTokenDeleteOtherUser(t *testing.T) {
	env := setupTestServer(t)

	// Create a token as owner.
	resp := doRequest(t, env, "POST", "/auth/tokens", `{"name":"owner-token"}`)
	assertStatus(t, resp, http.StatusCreated)
	var result map[string]any
	readJSON(t, resp, &result)
	tokenID := result["authToken"].(map[string]any)["id"].(string)

	// Second user tries to delete owner's token.
	userToken := createTestUser(t, env, "other@example.com", "Other", "pass123")
	assertStatusClose(t, doRequestAs(t, env, userToken, "DELETE", "/auth/tokens/"+tokenID, ""), http.StatusNotFound)
}

// --- Space Member CRUD Tests ---

func TestSpaceMembersCreate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	aliceToken := createTestUser(t, env, "alice@example.com", "Alice", "pass123")
	userID := getUserID(t, env, aliceToken)

	resp2 := doRequest(t, env, "POST", "/spaces/home/members", `{"userId":"`+userID+`","role":"member"}`)
	assertStatus(t, resp2, http.StatusCreated)
	var member map[string]any
	readJSON(t, resp2, &member)
	if member["userId"] != userID {
		t.Errorf("userId = %v, want %v", member["userId"], userID)
	}
	if member["role"] != "member" {
		t.Errorf("role = %v, want member", member["role"])
	}
	if member["userName"] != "Alice" {
		t.Errorf("userName = %v, want Alice", member["userName"])
	}
}

func TestSpaceMembersList(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "GET", "/spaces/home/members", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := page["items"].([]any)
	// Creator is auto-added as admin.
	if len(items) != 1 {
		t.Fatalf("got %d members, want 1", len(items))
	}
	if items[0].(map[string]any)["role"] != "admin" {
		t.Errorf("role = %v, want admin", items[0].(map[string]any)["role"])
	}
}

func TestSpaceMembersUpdate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	_, userID := createAndAddMember(t, env, "home", "bob@example.com", "Bob", "pass123", "member")

	// Promote to admin.
	resp2 := doRequest(t, env, "PATCH", "/spaces/home/members/"+userID, `{"role":"admin"}`)
	assertStatus(t, resp2, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp2, &updated)
	if updated["role"] != "admin" {
		t.Errorf("role = %v, want admin", updated["role"])
	}
}

func TestSpaceMembersDelete(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	userToken, userID := createAndAddMember(t, env, "home", "charlie@example.com", "Charlie", "pass123", "member")

	// Remove.
	assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/home/members/"+userID, ""), http.StatusNoContent)

	// User can no longer access the space.
	assertStatusClose(t, doRequestAs(t, env, userToken, "GET", "/spaces/home", ""), http.StatusNotFound)
}

func TestSpaceMembersLastAdminGuard(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	// Get owner's user ID.
	resp := doRequest(t, env, "GET", "/users/me", "")
	assertStatus(t, resp, http.StatusOK)
	var me map[string]any
	readJSON(t, resp, &me)
	ownerID := me["id"].(string)

	// Try to downgrade the only admin to viewer.
	assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/home/members/"+ownerID, `{"role":"viewer"}`), http.StatusBadRequest)
	// Try to remove the only admin.
	assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/home/members/"+ownerID, ""), http.StatusBadRequest)
}

// --- Task Assignee Tests ---

func TestTaskAssigneesOnCreate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	// Get owner's user ID.
	resp := doRequest(t, env, "GET", "/users/me", "")
	assertStatus(t, resp, http.StatusOK)
	var me map[string]any
	readJSON(t, resp, &me)
	ownerID := me["id"].(string)

	// Create task with assignees.
	task := createTask(t, env, "home", `{"title":"With assignees","assigneeIds":["`+ownerID+`"]}`)
	assigneeIds := task["assigneeIds"].([]any)
	if len(assigneeIds) != 1 {
		t.Fatalf("got %d assignees, want 1", len(assigneeIds))
	}
	if assigneeIds[0].(string) != ownerID {
		t.Errorf("assigneeIds[0] = %v, want %v", assigneeIds[0], ownerID)
	}
}

func TestTaskAssigneesEmptyByDefault(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	task := createTask(t, env, "home", `{"title":"No assignees"}`)
	assigneeIds := task["assigneeIds"].([]any)
	if len(assigneeIds) != 0 {
		t.Fatalf("got %d assignees, want 0", len(assigneeIds))
	}
}

func TestTaskAssigneesUpdate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	// Get owner's user ID.
	resp := doRequest(t, env, "GET", "/users/me", "")
	assertStatus(t, resp, http.StatusOK)
	var me map[string]any
	readJSON(t, resp, &me)
	ownerID := me["id"].(string)

	// Create a second user and add as member.
	_, bobID := createAndAddMember(t, env, "home", "bob@example.com", "Bob", "pass123", "member")

	// Create task without assignees.
	task := createTask(t, env, "home", `{"title":"Test"}`)
	taskID := task["id"].(string)

	// Update: set assignees.
	resp3 := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+taskID, `{"assigneeIds":["`+ownerID+`","`+bobID+`"]}`)
	assertStatus(t, resp3, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp3, &updated)
	assigneeIds := updated["assigneeIds"].([]any)
	if len(assigneeIds) != 2 {
		t.Fatalf("got %d assignees, want 2", len(assigneeIds))
	}
	// Verify actual IDs are present (order is by user_id).
	ids := make(map[string]bool)
	for _, id := range assigneeIds {
		ids[id.(string)] = true
	}
	if !ids[ownerID] || !ids[bobID] {
		t.Errorf("assigneeIds = %v, want both %s and %s", assigneeIds, ownerID, bobID)
	}

	// Update: clear assignees.
	resp4 := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+taskID, `{"assigneeIds":[]}`)
	assertStatus(t, resp4, http.StatusOK)
	var cleared map[string]any
	readJSON(t, resp4, &cleared)
	if len(cleared["assigneeIds"].([]any)) != 0 {
		t.Error("expected empty assignees after clearing")
	}
}

func TestTaskAssigneesPreservedOnUnrelatedUpdate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "GET", "/users/me", "")
	assertStatus(t, resp, http.StatusOK)
	var me map[string]any
	readJSON(t, resp, &me)
	ownerID := me["id"].(string)

	// Create task with assignee.
	task := createTask(t, env, "home", `{"title":"Test","assigneeIds":["`+ownerID+`"]}`)
	taskID := task["id"].(string)

	// Update title only — assignees should be preserved.
	resp2 := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+taskID, `{"title":"Updated"}`)
	assertStatus(t, resp2, http.StatusOK)
	var updated map[string]any
	readJSON(t, resp2, &updated)
	if updated["title"] != "Updated" {
		t.Errorf("title = %v, want Updated", updated["title"])
	}
	assigneeIds := updated["assigneeIds"].([]any)
	if len(assigneeIds) != 1 {
		t.Fatalf("got %d assignees, want 1 (preserved)", len(assigneeIds))
	}
	if assigneeIds[0].(string) != ownerID {
		t.Errorf("assigneeIds[0] = %v, want %v", assigneeIds[0], ownerID)
	}
}

func TestTaskAssigneesNonMemberRejected(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	// Create a user but don't add them as a member.
	outsiderToken := createTestUser(t, env, "outsider@example.com", "Outsider", "pass123")
	outsiderID := getUserID(t, env, outsiderToken)

	// Try to assign the non-member.
	task := createTask(t, env, "home", `{"title":"Test"}`)
	taskID := task["id"].(string)
	resp2 := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+taskID, `{"assigneeIds":["`+outsiderID+`"]}`)
	assertStatusClose(t, resp2, http.StatusBadRequest)
}

func TestTaskAssigneesCrossSpaceRejected(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "alpha", "Alpha")
	createSpace(t, env, "beta", "Beta")

	// Create user and add to beta only.
	_, bobID := createAndAddMember(t, env, "beta", "bob@example.com", "Bob", "pass123", "member")

	// Try to assign bob to a task in alpha (where he's not a member).
	task := createTask(t, env, "alpha", `{"title":"Test"}`)
	taskID := task["id"].(string)
	resp2 := doRequest(t, env, "PATCH", "/spaces/alpha/tasks/"+taskID, `{"assigneeIds":["`+bobID+`"]}`)
	assertStatusClose(t, resp2, http.StatusBadRequest)
}

func TestTaskAssigneesDeduplicated(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "GET", "/users/me", "")
	assertStatus(t, resp, http.StatusOK)
	var me map[string]any
	readJSON(t, resp, &me)
	ownerID := me["id"].(string)

	// Create task with duplicate assignee IDs.
	task := createTask(t, env, "home", `{"title":"Dedup","assigneeIds":["`+ownerID+`","`+ownerID+`"]}`)
	assigneeIds := task["assigneeIds"].([]any)
	if len(assigneeIds) != 1 {
		t.Fatalf("got %d assignees, want 1 (deduplicated)", len(assigneeIds))
	}
}

func TestTaskDeleteCascadesAssignees(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "GET", "/users/me", "")
	assertStatus(t, resp, http.StatusOK)
	var me map[string]any
	readJSON(t, resp, &me)
	ownerID := me["id"].(string)

	task := createTask(t, env, "home", `{"title":"Delete me","assigneeIds":["`+ownerID+`"]}`)
	taskID := task["id"].(string)

	// Delete the task.
	assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/home/tasks/"+taskID, ""), http.StatusNoContent)

	// Task is gone.
	assertStatusClose(t, doRequest(t, env, "GET", "/spaces/home/tasks/"+taskID, ""), http.StatusNotFound)
}

func TestTaskAssigneesInListResponse(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	resp := doRequest(t, env, "GET", "/users/me", "")
	assertStatus(t, resp, http.StatusOK)
	var me map[string]any
	readJSON(t, resp, &me)
	ownerID := me["id"].(string)

	// Create two tasks: one with assignees, one without.
	createTask(t, env, "home", `{"title":"Assigned","assigneeIds":["`+ownerID+`"]}`)
	createTask(t, env, "home", `{"title":"Unassigned"}`)

	// List tasks and check assigneeIds are present in both items.
	resp2 := doRequest(t, env, "GET", "/spaces/home/tasks", "")
	assertStatus(t, resp2, http.StatusOK)
	var page map[string]any
	readJSON(t, resp2, &page)
	items := page["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}

	for _, item := range items {
		task := item.(map[string]any)
		title := task["title"].(string)
		assignees := task["assigneeIds"].([]any)
		if title == "Assigned" {
			if len(assignees) != 1 || assignees[0].(string) != ownerID {
				t.Errorf("Assigned task: assigneeIds = %v, want [%s]", assignees, ownerID)
			}
		} else {
			if len(assignees) != 0 {
				t.Errorf("Unassigned task: assigneeIds = %v, want []", assignees)
			}
		}
	}
}

func TestMemberRemovalClearsAssignments(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")

	// Create a user and add as member.
	_, bobID := createAndAddMember(t, env, "home", "bob@example.com", "Bob", "pass123", "member")

	// Assign bob to a task.
	task := createTask(t, env, "home", `{"title":"Bob's task","assigneeIds":["`+bobID+`"]}`)
	taskID := task["id"].(string)

	// Verify bob is assigned.
	resp2 := doRequest(t, env, "GET", "/spaces/home/tasks/"+taskID, "")
	assertStatus(t, resp2, http.StatusOK)
	var before map[string]any
	readJSON(t, resp2, &before)
	if len(before["assigneeIds"].([]any)) != 1 {
		t.Fatalf("expected 1 assignee before removal")
	}

	// Remove bob from the space.
	assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/home/members/"+bobID, ""), http.StatusNoContent)

	// Bob's assignment should be cleared.
	resp3 := doRequest(t, env, "GET", "/spaces/home/tasks/"+taskID, "")
	assertStatus(t, resp3, http.StatusOK)
	var after map[string]any
	readJSON(t, resp3, &after)
	if len(after["assigneeIds"].([]any)) != 0 {
		t.Fatalf("expected 0 assignees after member removal, got %d", len(after["assigneeIds"].([]any)))
	}
}

func TestTaskAssigneesInvalidIDFormat(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	task := createTask(t, env, "home", `{"title":"Test"}`)
	taskID := task["id"].(string)

	// Invalid format (no U prefix).
	resp := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+taskID, `{"assigneeIds":["invalid"]}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTaskAssigneesNonExistentUser(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	task := createTask(t, env, "home", `{"title":"Test"}`)
	taskID := task["id"].(string)

	// Valid format but user doesn't exist.
	resp := doRequest(t, env, "PATCH", "/spaces/home/tasks/"+taskID, `{"assigneeIds":["U99999"]}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestMemberRemovalIsolation(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "alpha", "Alpha")
	createSpace(t, env, "beta", "Beta")

	// Create bob and add to both spaces.
	userToken := createTestUser(t, env, "bob@example.com", "Bob", "pass123")
	resp := doRequestAs(t, env, userToken, "GET", "/users/me", "")
	assertStatus(t, resp, http.StatusOK)
	var me map[string]any
	readJSON(t, resp, &me)
	bobID := me["id"].(string)
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/alpha/members", `{"userId":"`+bobID+`","role":"member"}`), http.StatusCreated)
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/beta/members", `{"userId":"`+bobID+`","role":"member"}`), http.StatusCreated)

	// Assign bob to a task in each space.
	taskAlpha := createTask(t, env, "alpha", `{"title":"Alpha task","assigneeIds":["`+bobID+`"]}`)
	taskBeta := createTask(t, env, "beta", `{"title":"Beta task","assigneeIds":["`+bobID+`"]}`)

	// Remove bob from alpha only.
	assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/alpha/members/"+bobID, ""), http.StatusNoContent)

	// Alpha task should have no assignees.
	resp2 := doRequest(t, env, "GET", "/spaces/alpha/tasks/"+taskAlpha["id"].(string), "")
	assertStatus(t, resp2, http.StatusOK)
	var alpha map[string]any
	readJSON(t, resp2, &alpha)
	if len(alpha["assigneeIds"].([]any)) != 0 {
		t.Fatalf("alpha task: expected 0 assignees, got %d", len(alpha["assigneeIds"].([]any)))
	}

	// Beta task should still have bob assigned.
	resp3 := doRequest(t, env, "GET", "/spaces/beta/tasks/"+taskBeta["id"].(string), "")
	assertStatus(t, resp3, http.StatusOK)
	var beta map[string]any
	readJSON(t, resp3, &beta)
	if len(beta["assigneeIds"].([]any)) != 1 {
		t.Fatalf("beta task: expected 1 assignee, got %d", len(beta["assigneeIds"].([]any)))
	}
}

// --- Edge Case Tests ---

func TestAuthLoginUnknownEmail(t *testing.T) {
	env := setupTestServer(t)

	resp := doRequestAs(t, env, "", "POST", "/auth/login", `{"email":"nobody@example.com","password":"anything"}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestExpiredTokenRejected(t *testing.T) {
	env := setupTestServer(t)

	// Create a token that's already expired directly in the DB.
	rawToken := "expired-test-token"
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])
	pastTime := types.EpochSeconds(time.Now().Add(-1 * time.Hour))

	q := dbgen.New(env.db)
	_, err := q.CreateAuthToken(context.Background(), dbgen.CreateAuthTokenParams{
		UserID:    1, // owner user
		TokenHash: tokenHash,
		Name:      "expired",
		Kind:      "session",
		ExpiresAt: &pastTime,
		CreatedAt: types.Now(),
	})
	if err != nil {
		t.Fatalf("create expired token: %v", err)
	}

	assertStatusClose(t, doRequestAs(t, env, rawToken, "GET", "/users/me", ""), http.StatusUnauthorized)
}

// --- Tags ---

func TestSpaceTagsCreate(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-test", "Tag Test")

	resp := doRequest(t, env, "POST", "/spaces/tag-test/tags", `{"name":"Bug"}`)
	assertStatus(t, resp, http.StatusCreated)
	var tag map[string]any
	readJSON(t, resp, &tag)
	if tag["name"] != "Bug" {
		t.Fatalf("got name %v, want Bug", tag["name"])
	}
	if tag["createdAt"] == nil {
		t.Fatal("expected createdAt to be set")
	}
}

func TestSpaceTagsCreateDuplicate(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-dup", "Tag Dup")

	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-dup/tags", `{"name":"Bug"}`), http.StatusCreated)

	// Same name (exact case) should 409.
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-dup/tags", `{"name":"Bug"}`), http.StatusConflict)
}

func TestSpaceTagsCreateCaseFoldDuplicate(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-fold", "Tag Fold")

	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-fold/tags", `{"name":"Bug"}`), http.StatusCreated)

	// Different case but same folded name should also 409.
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-fold/tags", `{"name":"bug"}`), http.StatusConflict)
}

func TestSpaceTagsCreateEmpty(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-empty", "Tag Empty")

	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-empty/tags", `{"name":""}`), http.StatusBadRequest)
}

func TestSpaceTagsList(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-list", "Tag List")

	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-list/tags", `{"name":"Alpha"}`), http.StatusCreated)
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-list/tags", `{"name":"Beta"}`), http.StatusCreated)

	resp := doRequest(t, env, "GET", "/spaces/tag-list/tags", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := page["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("got %d tags, want 2", len(items))
	}
}

func TestSpaceTagsListEmpty(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-le", "Tag List Empty")

	resp := doRequest(t, env, "GET", "/spaces/tag-le/tags", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := page["items"].([]any)
	if len(items) != 0 {
		t.Fatalf("got %d tags, want 0", len(items))
	}
}

func TestSpaceTagsUpdate(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-upd", "Tag Update")

	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-upd/tags", `{"name":"Bug"}`), http.StatusCreated)

	resp := doRequest(t, env, "PATCH", "/spaces/tag-upd/tags/Bug", `{"name":"Defect"}`)
	assertStatus(t, resp, http.StatusOK)
	var tag map[string]any
	readJSON(t, resp, &tag)
	if tag["name"] != "Defect" {
		t.Fatalf("got name %v, want Defect", tag["name"])
	}
}

func TestSpaceTagsUpdateCaseFoldCollision(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-coll", "Tag Collision")

	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-coll/tags", `{"name":"Bug"}`), http.StatusCreated)
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-coll/tags", `{"name":"Feature"}`), http.StatusCreated)

	// Renaming "Feature" to "bug" should conflict with "Bug".
	assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/tag-coll/tags/Feature", `{"name":"bug"}`), http.StatusConflict)
}

func TestSpaceTagsUpdateNotFound(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-unf", "Tag Update NF")

	assertStatusClose(t, doRequest(t, env, "PATCH", "/spaces/tag-unf/tags/nonexistent", `{"name":"X"}`), http.StatusNotFound)
}

func TestSpaceTagsDelete(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-del", "Tag Delete")

	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-del/tags", `{"name":"Bug"}`), http.StatusCreated)

	assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/tag-del/tags/Bug", ""), http.StatusNoContent)

	// List should be empty now.
	resp := doRequest(t, env, "GET", "/spaces/tag-del/tags", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	if len(page["items"].([]any)) != 0 {
		t.Fatal("expected no tags after delete")
	}
}

func TestSpaceTagsDeleteNotFound(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-dnf", "Tag Delete NF")

	assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/tag-dnf/tags/nonexistent", ""), http.StatusNotFound)
}

func TestSpaceTagsCrossSpaceIsolation(t *testing.T) {
	env := setupTestServer(t)
	createSpace(t, env, "tag-a", "Tag A")
	createSpace(t, env, "tag-b", "Tag B")

	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/tag-a/tags", `{"name":"Bug"}`), http.StatusCreated)

	// Tag in space A should not appear in space B.
	resp := doRequest(t, env, "GET", "/spaces/tag-b/tags", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	if len(page["items"].([]any)) != 0 {
		t.Fatal("space B should have no tags")
	}
}

// --- Task Relations ---

// assertTaskRelation is a helper that GETs a task and asserts it has exactly
// the expected number of relations, returning the relations slice.
func assertTaskRelations(t *testing.T, env *testEnv, spaceSlug, taskID string, wantCount int) []any {
	t.Helper()
	resp := doRequest(t, env, "GET", "/spaces/"+spaceSlug+"/tasks/"+taskID, "")
	assertStatus(t, resp, http.StatusOK)
	var task map[string]any
	readJSON(t, resp, &task)
	rels, ok := task["relations"].([]any)
	if !ok {
		t.Fatalf("relations field missing or wrong type on task %s: %T", taskID, task["relations"])
	}
	if len(rels) != wantCount {
		t.Fatalf("task %s: got %d relations, want %d", taskID, len(rels), wantCount)
	}
	return rels
}

// assertRelationKind asserts that rel has the expected kind and taskId.
func assertRelationKind(t *testing.T, rel any, wantKind, wantTaskID string) {
	t.Helper()
	r, ok := rel.(map[string]any)
	if !ok {
		t.Fatalf("relation wrong type: %T", rel)
	}
	if r["kind"] != wantKind {
		t.Fatalf("got kind %v, want %s", r["kind"], wantKind)
	}
	if r["taskId"] != wantTaskID {
		t.Fatalf("got taskId %v, want %s", r["taskId"], wantTaskID)
	}
}

func createRelation(t *testing.T, env *testEnv, spaceSlug, taskID, kind, relatedTaskID string) {
	t.Helper()
	resp := doRequest(t, env, "POST", "/spaces/"+spaceSlug+"/tasks/"+taskID+"/relations",
		`{"kind":"`+kind+`","taskId":"`+relatedTaskID+`"}`)
	assertStatusClose(t, resp, http.StatusCreated)
}

func TestTaskRelationsCreate(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "rel-test", "Relation Test")
	t1 := createTask(t, env, "rel-test", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "rel-test", `{"title":"Task 2"}`)
	t1id, t2id := t1["id"].(string), t2["id"].(string)

	// Create a "blocks" relation: T1 blocks T2.
	resp := doRequest(t, env, "POST", "/spaces/rel-test/tasks/"+t1id+"/relations",
		`{"kind":"blocks","taskId":"`+t2id+`"}`)
	assertStatus(t, resp, http.StatusCreated)
	var rel map[string]any
	readJSON(t, resp, &rel)
	if rel["kind"] != "blocks" {
		t.Fatalf("got kind %v, want blocks", rel["kind"])
	}
	if rel["taskId"] != t2id {
		t.Fatalf("got taskId %v, want %v", rel["taskId"], t2id)
	}

	// Verify relation appears on T1 as "blocks" and T2 as "blocked_by".
	rels1 := assertTaskRelations(t, env, "rel-test", t1id, 1)
	assertRelationKind(t, rels1[0], "blocks", t2id)

	rels2 := assertTaskRelations(t, env, "rel-test", t2id, 1)
	assertRelationKind(t, rels2[0], "blocked_by", t1id)
}

func TestTaskRelationsSymmetric(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "sym-test", "Symmetric Test")
	t1 := createTask(t, env, "sym-test", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "sym-test", `{"title":"Task 2"}`)
	t1id, t2id := t1["id"].(string), t2["id"].(string)

	createRelation(t, env, "sym-test", t1id, "relates_to", t2id)

	// Both tasks should show "relates_to" pointing at the other.
	rels1 := assertTaskRelations(t, env, "sym-test", t1id, 1)
	assertRelationKind(t, rels1[0], "relates_to", t2id)

	rels2 := assertTaskRelations(t, env, "sym-test", t2id, 1)
	assertRelationKind(t, rels2[0], "relates_to", t1id)
}

func TestTaskRelationsDelete(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "del-test", "Delete Test")
	t1 := createTask(t, env, "del-test", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "del-test", `{"title":"Task 2"}`)
	t1id, t2id := t1["id"].(string), t2["id"].(string)

	createRelation(t, env, "del-test", t1id, "blocks", t2id)

	assertStatusClose(t, doRequest(t, env, "DELETE",
		"/spaces/del-test/tasks/"+t1id+"/relations/blocks/"+t2id, ""),
		http.StatusNoContent)

	// Verify relation is gone from both tasks (T1).
	assertTaskRelations(t, env, "del-test", t1id, 0)
	assertTaskRelations(t, env, "del-test", t2id, 0)
}

func TestTaskRelationsDeleteFromOtherSide(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "del2-test", "Delete Other Side Test")
	t1 := createTask(t, env, "del2-test", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "del2-test", `{"title":"Task 2"}`)
	t1id, t2id := t1["id"].(string), t2["id"].(string)

	createRelation(t, env, "del2-test", t1id, "blocks", t2id)

	// Delete from T2's perspective as "blocked_by".
	assertStatusClose(t, doRequest(t, env, "DELETE",
		"/spaces/del2-test/tasks/"+t2id+"/relations/blocked_by/"+t1id, ""),
		http.StatusNoContent)

	// Verify it's gone from both sides.
	assertTaskRelations(t, env, "del2-test", t1id, 0)
	assertTaskRelations(t, env, "del2-test", t2id, 0)
}

func TestTaskRelationsDeleteNonExistent(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "del-ne", "Delete Non-Existent")
	t1 := createTask(t, env, "del-ne", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "del-ne", `{"title":"Task 2"}`)

	// Delete a relation that was never created.
	assertStatusClose(t, doRequest(t, env, "DELETE",
		"/spaces/del-ne/tasks/"+t1["id"].(string)+"/relations/blocks/"+t2["id"].(string), ""),
		http.StatusNotFound)
}

func TestTaskRelationsSelfRejected(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "self-test", "Self Test")
	t1 := createTask(t, env, "self-test", `{"title":"Task 1"}`)

	resp := doRequest(t, env, "POST", "/spaces/self-test/tasks/"+t1["id"].(string)+"/relations",
		`{"kind":"blocks","taskId":"`+t1["id"].(string)+`"}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTaskRelationsCrossSpaceRejected(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "space-a", "Space A")
	createSpace(t, env, "space-b", "Space B")
	t1 := createTask(t, env, "space-a", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "space-b", `{"title":"Task 2"}`)

	resp := doRequest(t, env, "POST", "/spaces/space-a/tasks/"+t1["id"].(string)+"/relations",
		`{"kind":"blocks","taskId":"`+t2["id"].(string)+`"}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestTaskRelationsNonExistentTask(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "ne-test", "Non-Existent Test")
	t1 := createTask(t, env, "ne-test", `{"title":"Task 1"}`)

	resp := doRequest(t, env, "POST", "/spaces/ne-test/tasks/"+t1["id"].(string)+"/relations",
		`{"kind":"blocks","taskId":"T99999"}`)
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestTaskRelationsDuplicateRejected(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "dup-test", "Dup Test")
	t1 := createTask(t, env, "dup-test", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "dup-test", `{"title":"Task 2"}`)
	t1id, t2id := t1["id"].(string), t2["id"].(string)

	createRelation(t, env, "dup-test", t1id, "blocks", t2id)

	// Same relation again should 409.
	resp := doRequest(t, env, "POST", "/spaces/dup-test/tasks/"+t1id+"/relations",
		`{"kind":"blocks","taskId":"`+t2id+`"}`)
	assertStatusClose(t, resp, http.StatusConflict)
}

func TestTaskRelationsDuplicateViaInverse(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "dup-inv", "Dup Inverse Test")
	t1 := createTask(t, env, "dup-inv", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "dup-inv", `{"title":"Task 2"}`)
	t1id, t2id := t1["id"].(string), t2["id"].(string)

	createRelation(t, env, "dup-inv", t1id, "blocks", t2id)

	// T2 blocked_by T1 should also 409 (same canonical row).
	resp := doRequest(t, env, "POST", "/spaces/dup-inv/tasks/"+t2id+"/relations",
		`{"kind":"blocked_by","taskId":"`+t1id+`"}`)
	assertStatusClose(t, resp, http.StatusConflict)
}

func TestTaskRelationsCreateViaBlockedBy(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "blkby-test", "Blocked By Test")
	t1 := createTask(t, env, "blkby-test", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "blkby-test", `{"title":"Task 2"}`)
	t1id, t2id := t1["id"].(string), t2["id"].(string)

	createRelation(t, env, "blkby-test", t2id, "blocked_by", t1id)

	rels1 := assertTaskRelations(t, env, "blkby-test", t1id, 1)
	assertRelationKind(t, rels1[0], "blocks", t2id)

	rels2 := assertTaskRelations(t, env, "blkby-test", t2id, 1)
	assertRelationKind(t, rels2[0], "blocked_by", t1id)
}

func TestTaskRelationsParentChild(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "pc-test", "Parent Child Test")
	parent := createTask(t, env, "pc-test", `{"title":"Parent"}`)
	child := createTask(t, env, "pc-test", `{"title":"Child"}`)
	parentID, childID := parent["id"].(string), child["id"].(string)

	createRelation(t, env, "pc-test", childID, "child_of", parentID)

	pRels := assertTaskRelations(t, env, "pc-test", parentID, 1)
	assertRelationKind(t, pRels[0], "parent_of", childID)

	cRels := assertTaskRelations(t, env, "pc-test", childID, 1)
	assertRelationKind(t, cRels[0], "child_of", parentID)
}

func TestTaskRelationsDuplicatesKind(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "dup-kind", "Duplicates Kind Test")
	t1 := createTask(t, env, "dup-kind", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "dup-kind", `{"title":"Task 2"}`)
	t1id, t2id := t1["id"].(string), t2["id"].(string)

	createRelation(t, env, "dup-kind", t1id, "duplicates", t2id)

	// Both sides should show "duplicates" (symmetric).
	rels1 := assertTaskRelations(t, env, "dup-kind", t1id, 1)
	assertRelationKind(t, rels1[0], "duplicates", t2id)

	rels2 := assertTaskRelations(t, env, "dup-kind", t2id, 1)
	assertRelationKind(t, rels2[0], "duplicates", t1id)
}

func TestTaskRelationsInListResponse(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "list-rel", "List Rel Test")
	t1 := createTask(t, env, "list-rel", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "list-rel", `{"title":"Task 2"}`)
	t1id, t2id := t1["id"].(string), t2["id"].(string)

	createRelation(t, env, "list-rel", t1id, "blocks", t2id)

	// List tasks and check relations are correctly included.
	resp := doRequest(t, env, "GET", "/spaces/list-rel/tasks", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := page["items"].([]any)

	for _, item := range items {
		task := item.(map[string]any)
		rels := task["relations"].([]any)
		switch task["id"] {
		case t1id:
			if len(rels) != 1 {
				t.Fatalf("T1 in list: got %d relations, want 1", len(rels))
			}
			assertRelationKind(t, rels[0], "blocks", t2id)
		case t2id:
			if len(rels) != 1 {
				t.Fatalf("T2 in list: got %d relations, want 1", len(rels))
			}
			assertRelationKind(t, rels[0], "blocked_by", t1id)
		}
	}
}

func TestTaskRelationsEmptyByDefault(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "empty-rel", "Empty Rel Test")
	task := createTask(t, env, "empty-rel", `{"title":"Task 1"}`)

	// Check both create response and GET response.
	rels := task["relations"].([]any)
	if len(rels) != 0 {
		t.Fatalf("create response: got %d relations, want 0", len(rels))
	}
	assertTaskRelations(t, env, "empty-rel", task["id"].(string), 0)
}

func TestTaskRelationsNonMemberRejected(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "perm-test", "Permission Test")
	t1 := createTask(t, env, "perm-test", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "perm-test", `{"title":"Task 2"}`)

	// Create a non-member user.
	nonMemberToken := createTestUser(t, env, "outsider@example.com", "Outsider", "password")

	// Non-member should not be able to create relations.
	resp := doRequestAs(t, env, nonMemberToken, "POST", "/spaces/perm-test/tasks/"+t1["id"].(string)+"/relations",
		`{"kind":"blocks","taskId":"`+t2["id"].(string)+`"}`)
	assertStatusClose(t, resp, http.StatusNotFound)
}

func TestTaskRelationsCascadeOnTaskDelete(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "cascade-rel", "Cascade Test")
	t1 := createTask(t, env, "cascade-rel", `{"title":"Task 1"}`)
	t2 := createTask(t, env, "cascade-rel", `{"title":"Task 2"}`)
	t2id := t2["id"].(string)

	createRelation(t, env, "cascade-rel", t1["id"].(string), "blocks", t2id)

	// Delete T1.
	assertStatusClose(t, doRequest(t, env, "DELETE", "/spaces/cascade-rel/tasks/"+t1["id"].(string), ""), http.StatusNoContent)

	// T2 should have no relations.
	assertTaskRelations(t, env, "cascade-rel", t2id, 0)
}

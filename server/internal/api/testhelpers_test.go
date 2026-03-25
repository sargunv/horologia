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

	"github.com/sargunv/tend/server/internal/api"
	"github.com/sargunv/tend/server/internal/database"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/taskengine"
	"github.com/sargunv/tend/server/internal/types"
)

type testEnv struct {
	Server  *httptest.Server
	Token   string
	Handler *api.Handler
	db      *sql.DB
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
	user, err := taskengine.CreateUserWithPassword(context.Background(), db, "test@example.com", "Test User", "password", true)
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

	log := slog.New(slog.DiscardHandler)
	engine := &taskengine.Engine{
		CopyOnSpawnKinds: api.StoredKindCopyOnSpawn(),
	}
	handler := &api.Handler{DB: db, Log: log, Engine: engine}
	h, err := api.NewServer(handler, log)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	return &testEnv{
		Server:  srv,
		Token:   rawToken,
		Handler: handler,
		db:      db,
	}
}

func doRequestAs(t *testing.T, env *testEnv, token, method, path, body string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(t.Context(), method, env.Server.URL+path, bodyReader)
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
	_, err := taskengine.CreateUserWithPassword(context.Background(), env.db, email, name, password, false)
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
	return jsonAs[string](t, result["token"])
}

// getUserID calls GET /users/me with the given token and returns the user ID.
func getUserID(t *testing.T, env *testEnv, token string) string {
	t.Helper()
	resp := doRequestAs(t, env, token, "GET", "/users/me", "")
	assertStatus(t, resp, http.StatusOK)
	var me map[string]any
	readJSON(t, resp, &me)
	return jsonAs[string](t, me["id"])
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

// jsonAs is a generic helper that performs a checked type assertion in tests.
func jsonAs[T any](t *testing.T, v any) T {
	t.Helper()
	result, ok := v.(T)
	if !ok {
		t.Fatalf("expected %T, got %T", result, v)
	}
	return result
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

package api_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/sargunv/horologia/server/internal/database/gen"
	"github.com/sargunv/horologia/server/internal/mcp"
	"github.com/sargunv/horologia/server/internal/types"
)

// mcpSession represents an initialized MCP session for testing.
type mcpSession struct {
	handler   http.Handler
	token     string
	sessionID string
}

// newMCPSession creates an MCP transport, performs the initialize handshake,
// and returns a session ready for tool calls.
func newMCPSession(t *testing.T, env *testEnv) *mcpSession {
	t.Helper()
	return newMCPSessionWithToken(t, env, env.Token)
}

func newMCPSessionWithToken(t *testing.T, env *testEnv, token string) *mcpSession {
	t.Helper()
	handler := mcp.NewTransport(env.pool, env.Handler)

	// Send initialize request.
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test-client", "version": "0.1.0"},
		},
	})
	if err != nil {
		t.Fatalf("marshal initialize body: %v", err)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("MCP initialize: status %d, want 200", resp.StatusCode)
	}

	sessionID := resp.Header.Get("Mcp-Session-Id")

	return &mcpSession{
		handler:   handler,
		token:     token,
		sessionID: sessionID,
	}
}

// request sends an initialized JSON-RPC request and returns the parsed response.
func (s *mcpSession) request(t *testing.T, method string, params map[string]any) map[string]any {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		t.Fatalf("marshal %s body: %v", method, err)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+s.token)
	if s.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", s.sessionID)
	}
	w := httptest.NewRecorder()
	s.handler.ServeHTTP(w, req)
	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("MCP request %s: status %d, want 200", method, resp.StatusCode)
	}

	// The response may be SSE (text/event-stream) or plain JSON.
	var result map[string]any
	ct := resp.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "text/event-stream") {
		raw := w.Body.String()
		for line := range strings.SplitSeq(raw, "\n") {
			data, ok := strings.CutPrefix(strings.TrimSpace(line), "data:")
			if !ok {
				continue
			}
			data = strings.TrimSpace(data)
			if err := json.Unmarshal([]byte(data), &result); err == nil && result["id"] != nil {
				break
			}
		}
		if result == nil {
			t.Fatalf("MCP request %s: no JSON-RPC response in SSE body: %s", method, raw)
		}
	} else {
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("MCP request %s: decode JSON: %v", method, err)
		}
	}

	return result
}

// call sends a JSON-RPC tools/call request and returns the parsed JSON-RPC response.
func (s *mcpSession) call(t *testing.T, toolName string, args map[string]any) map[string]any {
	t.Helper()
	return s.request(t, "tools/call", map[string]any{
		"name":      toolName,
		"arguments": args,
	})
}

// mcpResultText extracts the raw text and isError flag from an MCP JSON-RPC response.
func mcpResultText(t *testing.T, rpcResp map[string]any) (text string, isError bool) {
	t.Helper()
	if rpcResp["error"] != nil {
		t.Fatalf("unexpected JSON-RPC error: %v", rpcResp["error"])
	}
	result := jsonAs[map[string]any](t, rpcResp["result"])
	items := jsonAs[[]any](t, result["content"])
	if len(items) == 0 {
		t.Fatal("MCP result has no content items")
	}
	item := jsonAs[map[string]any](t, items[0])
	errFlag, _ := result["isError"].(bool)
	return jsonAs[string](t, item["text"]), errFlag
}

// toolResult extracts the tool result content from a JSON-RPC response.
// Returns parsed JSON content and whether the result was an error.
func toolResult(t *testing.T, rpcResp map[string]any) (content map[string]any, isError bool) {
	t.Helper()
	text, isErr := mcpResultText(t, rpcResp)
	if isErr {
		return nil, true
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("unmarshal tool result text: %v (text: %s)", err, text)
	}
	return parsed, false
}

// toolErrorText extracts the error message text from an MCP error result.
func toolErrorText(t *testing.T, rpcResp map[string]any) string {
	t.Helper()
	text, _ := mcpResultText(t, rpcResp)
	return text
}

// toolResultOk checks that the MCP tool call returned a non-error text result.
// Returns the text content. Use for void-return tools (delete operations).
func toolResultOk(t *testing.T, rpcResp map[string]any) string {
	t.Helper()
	text, isErr := mcpResultText(t, rpcResp)
	if isErr {
		t.Fatalf("expected success, got error: %s", text)
	}
	return text
}

// toolResultJSON extracts a successful JSON result, failing the test on error.
func toolResultJSON(t *testing.T, rpcResp map[string]any) map[string]any {
	t.Helper()
	content, isErr := toolResult(t, rpcResp)
	if isErr {
		t.Fatalf("expected success, got error: %s", toolErrorText(t, rpcResp))
	}
	return content
}

// toolResultList extracts items from a list/page response.
func toolResultList(t *testing.T, rpcResp map[string]any) []any {
	t.Helper()
	content := toolResultJSON(t, rpcResp)
	items, ok := content["items"].([]any)
	if !ok {
		t.Fatalf("expected items array, got %T", content["items"])
	}
	return items
}

// --- Tests ---

func TestMCPTaskCreate(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "task_create", map[string]any{
		"spaceSlug": "home",
		"title":     "MCP task",
	})
	task := toolResultJSON(t, rpcResp)
	if task["title"] != "MCP task" {
		t.Errorf("title = %v, want MCP task", task["title"])
	}
	if task["status"] != "todo" {
		t.Errorf("status = %v, want todo", task["status"])
	}
	id := jsonAs[string](t, task["id"])
	if !strings.HasPrefix(id, "T") {
		t.Errorf("id = %v, want T-prefixed", id)
	}
}

func TestMCPTaskCreateWithFields(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "task_create", map[string]any{
		"spaceSlug":   "home",
		"title":       "Detailed task",
		"description": "A description",
		"status":      "done",
	})
	task := toolResultJSON(t, rpcResp)
	if task["description"] != "A description" {
		t.Errorf("description = %v, want A description", task["description"])
	}
	if task["status"] != "done" {
		t.Errorf("status = %v, want done", task["status"])
	}
}

func TestMCPTaskCreateMissingTitle(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "task_create", map[string]any{
		"spaceSlug": "home",
	})
	errText := toolErrorText(t, rpcResp)
	if errText != "title is required" {
		t.Errorf("error = %q, want 'title is required'", errText)
	}
}

func TestMCPTaskList(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")
	createTask(t, env, "home", `{"title":"Task A"}`)
	createTask(t, env, "home", `{"title":"Task B"}`)

	rpcResp := s.call(t, "task_list", map[string]any{
		"spaceSlug": "home",
	})
	page := toolResultJSON(t, rpcResp)
	items := jsonAs[[]any](t, page["items"])
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
}

func TestMCPTaskGet(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")
	created := createTask(t, env, "home", `{"title":"Fetch me"}`)
	taskID := jsonAs[string](t, created["id"])

	rpcResp := s.call(t, "task_get", map[string]any{
		"spaceSlug": "home",
		"taskId":    taskID,
	})
	task := toolResultJSON(t, rpcResp)
	if task["title"] != "Fetch me" {
		t.Errorf("title = %v, want Fetch me", task["title"])
	}
	if task["id"] != taskID {
		t.Errorf("id = %v, want %s", task["id"], taskID)
	}
}

func TestMCPTaskSearch(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")
	task := createTask(t, env, "home", `{"title":"Task planning"}`)

	rpcResp := s.call(t, "task_search", map[string]any{
		"q": "Task",
	})
	items := toolResultList(t, rpcResp)
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}

	row := jsonAs[map[string]any](t, items[0])
	if row["id"] != task["id"] {
		t.Fatalf("id = %v, want %v", row["id"], task["id"])
	}
	if row["spaceSlug"] != "home" {
		t.Fatalf("spaceSlug = %v, want home", row["spaceSlug"])
	}
}

func TestMCPTaskUpdate(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")
	created := createTask(t, env, "home", `{"title":"Original"}`)
	taskID := jsonAs[string](t, created["id"])

	rpcResp := s.call(t, "task_update", map[string]any{
		"spaceSlug": "home",
		"taskId":    taskID,
		"title":     "Updated",
		"status":    "done",
	})
	task := toolResultJSON(t, rpcResp)
	if task["title"] != "Updated" {
		t.Errorf("title = %v, want Updated", task["title"])
	}
	if task["status"] != "done" {
		t.Errorf("status = %v, want done", task["status"])
	}
}

func TestMCPTaskCreateInNonexistentSpace(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	rpcResp := s.call(t, "task_create", map[string]any{
		"spaceSlug": "nonexistent",
		"title":     "Should fail",
	})
	_, isErr := toolResult(t, rpcResp)
	if !isErr {
		t.Error("expected error result for nonexistent space")
	}
	errText := toolErrorText(t, rpcResp)
	if errText != "resource not found" {
		t.Errorf("error = %q, want 'resource not found'", errText)
	}
}

func TestMCPTaskGetNonexistent(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "task_get", map[string]any{
		"spaceSlug": "home",
		"taskId":    "T999999",
	})
	_, isErr := toolResult(t, rpcResp)
	if !isErr {
		t.Error("expected error result for nonexistent task")
	}
	errText := toolErrorText(t, rpcResp)
	if errText != "resource not found" {
		t.Errorf("error = %q, want 'resource not found'", errText)
	}
}

func TestMCPTaskDelete(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")
	created := createTask(t, env, "home", `{"title":"Delete me"}`)
	taskID := jsonAs[string](t, created["id"])

	rpcResp := s.call(t, "task_delete", map[string]any{
		"spaceSlug": "home",
		"taskId":    taskID,
	})
	text := toolResultOk(t, rpcResp)
	if text != "ok" {
		t.Errorf("expected 'ok', got %q", text)
	}

	// Verify task is gone.
	rpcResp = s.call(t, "task_get", map[string]any{
		"spaceSlug": "home",
		"taskId":    taskID,
	})
	_, isErr := toolResult(t, rpcResp)
	if !isErr {
		t.Error("expected error for deleted task")
	}
}

func TestMCPRecipeCreateAndUpdateNestedSections(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)
	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "recipe_create", map[string]any{
		"spaceSlug": "home",
		"name":      "MCP soup",
		"yield": map[string]any{
			"amount": float64(4),
			"unit":   "bowls",
		},
		"ingredientSections": []any{
			map[string]any{
				"title": "Soup",
				"ingredients": []any{
					map[string]any{"quantity": float64(2), "unit": "cups", "item": "stock"},
				},
			},
		},
		"instructionSections": []any{
			map[string]any{
				"steps": []any{map[string]any{"body": "Simmer."}},
			},
		},
	})
	recipe := toolResultJSON(t, rpcResp)
	id := jsonAs[string](t, recipe["id"])
	if !strings.HasPrefix(id, "R") {
		t.Fatalf("id = %v, want R-prefixed", id)
	}
	sections := jsonAs[[]any](t, recipe["ingredientSections"])
	if len(sections) != 1 {
		t.Fatalf("unexpected ingredient sections: %v", sections)
	}

	rpcResp = s.call(t, "recipe_update", map[string]any{
		"spaceSlug":          "home",
		"recipeId":           id,
		"ingredientSections": []any{},
	})
	recipe = toolResultJSON(t, rpcResp)
	if len(jsonAs[[]any](t, recipe["ingredientSections"])) != 0 {
		t.Fatalf("nested replacement did not clear ingredients: %v", recipe)
	}
}

func TestMCPRecipeToolsRegistered(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)
	rpcResp := s.request(t, "tools/list", map[string]any{})
	result := jsonAs[map[string]any](t, rpcResp["result"])
	tools := jsonAs[[]any](t, result["tools"])
	found := make(map[string]bool, len(tools))
	for _, value := range tools {
		tool := jsonAs[map[string]any](t, value)
		found[jsonAs[string](t, tool["name"])] = true
	}
	for _, name := range []string{
		"recipe_list",
		"recipe_create",
		"recipe_get",
		"recipe_update",
		"recipe_delete",
		"recipe_search",
		"recipe_activity_list",
	} {
		if !found[name] {
			t.Errorf("MCP tool %q is not registered", name)
		}
	}
}

// --- Space tools ---

func TestMCPSpaceList(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "alpha", "Alpha")
	createSpace(t, env, "beta", "Beta")

	rpcResp := s.call(t, "space_list", map[string]any{})
	items := toolResultList(t, rpcResp)
	if len(items) != 2 {
		t.Fatalf("got %d spaces, want 2", len(items))
	}
}

func TestMCPSpaceListRequiresSpacesReadScope(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	ownerID := getUserID(t, env, env.Token)
	ownerNumericID, err := types.ParseUserID(ownerID)
	if err != nil {
		t.Fatalf("parse owner ID %q: %v", ownerID, err)
	}

	rawToken := "oauth-mcp-profile-only" //nolint:gosec // test token fixture
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	q := dbgen.New(env.pool)
	_, err = q.CreateAuthToken(t.Context(), dbgen.CreateAuthTokenParams{
		UserID:        ownerNumericID,
		TokenHash:     tokenHash,
		Name:          "Horologia MCP",
		Kind:          dbgen.AuthTokenKindOauthAccess,
		ExpiresAt:     pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
		CreatedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
		OauthClientID: pgtype.Text{String: "remote-mcp", Valid: true},
		OauthScopes:   []string{"profile:read"},
		OauthResource: pgtype.Text{String: env.Server.URL + "/mcp", Valid: true},
	})
	if err != nil {
		t.Fatalf("create oauth access token: %v", err)
	}

	s := newMCPSessionWithToken(t, env, rawToken)
	rpcResp := s.call(t, "space_list", map[string]any{})
	text, isErr := mcpResultText(t, rpcResp)
	if !isErr {
		t.Fatal("expected scope error for underscoped MCP token")
	}
	if text != "insufficient scope" {
		t.Fatalf("text = %q, want insufficient scope", text)
	}
}

func TestMCPSpaceCreate(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	rpcResp := s.call(t, "space_create", map[string]any{
		"slug": "new-space",
		"name": "New Space",
	})
	space := toolResultJSON(t, rpcResp)
	if space["slug"] != "new-space" {
		t.Errorf("slug = %v, want new-space", space["slug"])
	}
	if space["name"] != "New Space" {
		t.Errorf("name = %v, want New Space", space["name"])
	}
}

func TestMCPSpaceGet(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "space_get", map[string]any{
		"spaceSlug": "home",
	})
	space := toolResultJSON(t, rpcResp)
	if space["slug"] != "home" {
		t.Errorf("slug = %v, want home", space["slug"])
	}
}

func TestMCPSpaceUpdate(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "space_update", map[string]any{
		"spaceSlug": "home",
		"name":      "Updated Home",
	})
	space := toolResultJSON(t, rpcResp)
	if space["name"] != "Updated Home" {
		t.Errorf("name = %v, want Updated Home", space["name"])
	}
}

func TestMCPSpaceDelete(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "doomed", "Doomed")

	rpcResp := s.call(t, "space_delete", map[string]any{
		"spaceSlug": "doomed",
	})
	toolResultOk(t, rpcResp)

	// Verify space is gone.
	rpcResp = s.call(t, "space_get", map[string]any{
		"spaceSlug": "doomed",
	})
	_, isErr := toolResult(t, rpcResp)
	if !isErr {
		t.Error("expected error for deleted space")
	}
}

// --- Tag tools ---

func TestMCPTagList(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/home/tags", `{"name":"urgent"}`), http.StatusCreated)
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/home/tags", `{"name":"bug"}`), http.StatusCreated)

	rpcResp := s.call(t, "tag_list", map[string]any{
		"spaceSlug": "home",
	})
	items := toolResultList(t, rpcResp)
	if len(items) != 2 {
		t.Fatalf("got %d tags, want 2", len(items))
	}
}

func TestMCPTagCreate(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "tag_create", map[string]any{
		"spaceSlug": "home",
		"name":      "urgent",
	})
	tag := toolResultJSON(t, rpcResp)
	if tag["name"] != "urgent" {
		t.Errorf("name = %v, want urgent", tag["name"])
	}
}

func TestMCPTagUpdate(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/home/tags", `{"name":"urgent"}`), http.StatusCreated)

	rpcResp := s.call(t, "tag_update", map[string]any{
		"spaceSlug": "home",
		"tagName":   "urgent",
		"name":      "critical",
	})
	tag := toolResultJSON(t, rpcResp)
	if tag["name"] != "critical" {
		t.Errorf("name = %v, want critical", tag["name"])
	}
}

func TestMCPTagDelete(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")
	assertStatusClose(t, doRequest(t, env, "POST", "/spaces/home/tags", `{"name":"urgent"}`), http.StatusCreated)

	rpcResp := s.call(t, "tag_delete", map[string]any{
		"spaceSlug": "home",
		"tagName":   "urgent",
	})
	toolResultOk(t, rpcResp)

	// Verify tag is gone.
	rpcResp = s.call(t, "tag_list", map[string]any{
		"spaceSlug": "home",
	})
	items := toolResultList(t, rpcResp)
	if len(items) != 0 {
		t.Fatalf("got %d tags, want 0", len(items))
	}
}

// --- Member tools ---

func TestMCPMemberList(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "member_list", map[string]any{
		"spaceSlug": "home",
	})
	items := toolResultList(t, rpcResp)
	// Space creator is added as admin member by default.
	if len(items) != 1 {
		t.Fatalf("got %d members, want 1 (creator)", len(items))
	}
}

func TestMCPMemberCreate(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")
	_, userID := createAndAddMember(t, env, "home", "bob@test.com", "Bob", "password", "member")

	// Verify via MCP member_list.
	rpcResp := s.call(t, "member_list", map[string]any{
		"spaceSlug": "home",
	})
	items := toolResultList(t, rpcResp)
	found := false
	for _, item := range items {
		m := jsonAs[map[string]any](t, item)
		if m["userId"] == userID {
			found = true
			if m["role"] != "member" {
				t.Errorf("role = %v, want member", m["role"])
			}
		}
	}
	if !found {
		t.Error("added member not found in member_list")
	}
}

func TestMCPMemberCreateViaMCP(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")
	token := createTestUser(t, env, "charlie@test.com", "Charlie", "password")
	userID := getUserID(t, env, token)

	rpcResp := s.call(t, "member_create", map[string]any{
		"spaceSlug": "home",
		"userId":    userID,
		"role":      "viewer",
	})
	member := toolResultJSON(t, rpcResp)
	if member["userId"] != userID {
		t.Errorf("userId = %v, want %s", member["userId"], userID)
	}
	if member["role"] != "viewer" {
		t.Errorf("role = %v, want viewer", member["role"])
	}
}

func TestMCPMemberUpdate(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")
	_, userID := createAndAddMember(t, env, "home", "bob@test.com", "Bob", "password", "member")

	rpcResp := s.call(t, "member_update", map[string]any{
		"spaceSlug": "home",
		"userId":    userID,
		"role":      "admin",
	})
	member := toolResultJSON(t, rpcResp)
	if member["role"] != "admin" {
		t.Errorf("role = %v, want admin", member["role"])
	}
}

func TestMCPMemberDelete(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")
	_, userID := createAndAddMember(t, env, "home", "bob@test.com", "Bob", "password", "member")

	rpcResp := s.call(t, "member_delete", map[string]any{
		"spaceSlug": "home",
		"userId":    userID,
	})
	toolResultOk(t, rpcResp)

	// Verify member is gone.
	rpcResp = s.call(t, "member_list", map[string]any{
		"spaceSlug": "home",
	})
	items := toolResultList(t, rpcResp)
	for _, item := range items {
		m := jsonAs[map[string]any](t, item)
		if m["userId"] == userID {
			t.Error("deleted member still in member_list")
		}
	}
}

// --- Level tools ---

func TestMCPStatusList(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "status_list", map[string]any{
		"spaceSlug": "home",
	})
	items := toolResultList(t, rpcResp)
	if len(items) < 2 {
		t.Fatalf("got %d statuses, want at least 2 (default)", len(items))
	}
}

func TestMCPEffortLevelList(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "effort_level_list", map[string]any{
		"spaceSlug": "home",
	})
	items := toolResultList(t, rpcResp)
	if len(items) != 3 {
		t.Fatalf("got %d effort levels, want 3 (default)", len(items))
	}
}

func TestMCPPriorityLevelList(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "priority_level_list", map[string]any{
		"spaceSlug": "home",
	})
	items := toolResultList(t, rpcResp)
	if len(items) != 3 {
		t.Fatalf("got %d priority levels, want 3 (default)", len(items))
	}
}

func TestMCPStatusReplace(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "status_replace", map[string]any{
		"spaceSlug": "home",
		"items": []any{
			map[string]any{"name": "Open", "category": "initial"},
			map[string]any{"name": "In Progress", "category": "intermediate"},
			map[string]any{"name": "Done", "category": "completion"},
		},
	})
	items := toolResultList(t, rpcResp)
	if len(items) != 3 {
		t.Fatalf("got %d statuses, want 3", len(items))
	}
	wantStatuses := []struct{ name, category string }{
		{"Open", "initial"},
		{"In Progress", "intermediate"},
		{"Done", "completion"},
	}
	for i, want := range wantStatuses {
		m := jsonAs[map[string]any](t, items[i])
		if got := jsonAs[string](t, m["name"]); got != want.name {
			t.Errorf("items[%d].name = %q, want %q", i, got, want.name)
		}
		if got := jsonAs[string](t, m["category"]); got != want.category {
			t.Errorf("items[%d].category = %q, want %q", i, got, want.category)
		}
	}

	// Verify via list
	listResp := s.call(t, "status_list", map[string]any{"spaceSlug": "home"})
	listItems := toolResultList(t, listResp)
	if len(listItems) != 3 {
		t.Fatalf("list returned %d statuses, want 3", len(listItems))
	}
}

func TestMCPEffortLevelReplace(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "effort_level_replace", map[string]any{
		"spaceSlug": "home",
		"items": []any{
			map[string]any{"name": "Low"},
			map[string]any{"name": "High"},
		},
	})
	items := toolResultList(t, rpcResp)
	if len(items) != 2 {
		t.Fatalf("got %d effort levels, want 2", len(items))
	}
	for i, wantName := range []string{"Low", "High"} {
		m := jsonAs[map[string]any](t, items[i])
		if got := jsonAs[string](t, m["name"]); got != wantName {
			t.Errorf("items[%d].name = %q, want %q", i, got, wantName)
		}
	}

	// Verify via list
	listResp := s.call(t, "effort_level_list", map[string]any{"spaceSlug": "home"})
	listItems := toolResultList(t, listResp)
	if len(listItems) != 2 {
		t.Fatalf("list returned %d effort levels, want 2", len(listItems))
	}
}

func TestMCPPriorityLevelReplace(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "priority_level_replace", map[string]any{
		"spaceSlug": "home",
		"items": []any{
			map[string]any{"name": "Low"},
			map[string]any{"name": "Medium"},
			map[string]any{"name": "High"},
		},
	})
	items := toolResultList(t, rpcResp)
	if len(items) != 3 {
		t.Fatalf("got %d priority levels, want 3", len(items))
	}
	for i, wantName := range []string{"Low", "Medium", "High"} {
		m := jsonAs[map[string]any](t, items[i])
		if got := jsonAs[string](t, m["name"]); got != wantName {
			t.Errorf("items[%d].name = %q, want %q", i, got, wantName)
		}
	}

	// Verify via list
	listResp := s.call(t, "priority_level_list", map[string]any{"spaceSlug": "home"})
	listItems := toolResultList(t, listResp)
	if len(listItems) != 3 {
		t.Fatalf("list returned %d priority levels, want 3", len(listItems))
	}
}

func TestMCPStatusReplaceMissingCategory(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "status_replace", map[string]any{
		"spaceSlug": "home",
		"items": []any{
			map[string]any{"name": "Open"},
		},
	})
	errText := toolErrorText(t, rpcResp)
	if errText != "items[0].category is required" {
		t.Errorf("error = %q, want 'items[0].category is required'", errText)
	}
}

func TestMCPStatusReplaceMissingName(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "status_replace", map[string]any{
		"spaceSlug": "home",
		"items": []any{
			map[string]any{"category": "initial"},
		},
	})
	errText := toolErrorText(t, rpcResp)
	if errText != "items[0].name is required" {
		t.Errorf("error = %q, want 'items[0].name is required'", errText)
	}
}

func TestMCPEffortLevelReplaceMissingItems(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "effort_level_replace", map[string]any{
		"spaceSlug": "home",
	})
	errText := toolErrorText(t, rpcResp)
	if errText != "items is required" {
		t.Errorf("error = %q, want 'items is required'", errText)
	}
}

// --- Activity tools ---

func TestMCPSpaceActivityList(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")
	createTask(t, env, "home", `{"title":"Activity trigger"}`)

	rpcResp := s.call(t, "space_activity_list", map[string]any{
		"spaceSlug": "home",
	})
	items := toolResultList(t, rpcResp)
	if len(items) == 0 {
		t.Fatal("expected at least 1 activity entry")
	}
}

func TestMCPTaskActivityList(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")
	created := createTask(t, env, "home", `{"title":"Activity task"}`)
	taskID := jsonAs[string](t, created["id"])

	rpcResp := s.call(t, "task_activity_list", map[string]any{
		"spaceSlug": "home",
		"taskId":    taskID,
	})
	items := toolResultList(t, rpcResp)
	if len(items) == 0 {
		t.Fatal("expected at least 1 activity entry for task")
	}
}

func TestMCPUserActivityList(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	userID := getUserID(t, env, env.Token)
	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "user_activity_list", map[string]any{
		"userId": userID,
	})
	items := toolResultList(t, rpcResp)
	if len(items) == 0 {
		t.Fatal("expected at least 1 activity entry for user")
	}
}

// --- User tasks tool ---

func TestMCPUserTaskList(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	userID := getUserID(t, env, env.Token)
	createSpace(t, env, "home", "Home")
	createTask(t, env, "home", `{"title":"Assigned task","assigneeIds":["`+userID+`"]}`)

	rpcResp := s.call(t, "user_task_list", map[string]any{
		"userId": userID,
	})
	items := toolResultList(t, rpcResp)
	if len(items) != 1 {
		t.Fatalf("got %d tasks, want 1", len(items))
	}
}

// --- Relation tools ---

func TestMCPRelationCreate(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")
	taskA := createTask(t, env, "home", `{"title":"Task A"}`)
	taskB := createTask(t, env, "home", `{"title":"Task B"}`)
	taskAID := jsonAs[string](t, taskA["id"])
	taskBID := jsonAs[string](t, taskB["id"])

	rpcResp := s.call(t, "relation_create", map[string]any{
		"spaceSlug":     "home",
		"taskId":        taskAID,
		"kind":          "blocks",
		"relatedTaskId": taskBID,
	})
	rel := toolResultJSON(t, rpcResp)
	if rel["kind"] != "blocks" {
		t.Errorf("kind = %v, want blocks", rel["kind"])
	}
}

func TestMCPRelationDelete(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")
	taskA := createTask(t, env, "home", `{"title":"Task A"}`)
	taskB := createTask(t, env, "home", `{"title":"Task B"}`)
	taskAID := jsonAs[string](t, taskA["id"])
	taskBID := jsonAs[string](t, taskB["id"])

	createRelation(t, env, "home", taskAID, "blocks", taskBID)

	rpcResp := s.call(t, "relation_delete", map[string]any{
		"spaceSlug":     "home",
		"taskId":        taskAID,
		"kind":          "blocks",
		"relatedTaskId": taskBID,
	})
	toolResultOk(t, rpcResp)

	// Verify relation is gone.
	assertTaskRelations(t, env, "home", taskAID, 0)
}

// --- Error path tests ---

func TestMCPSpaceGetNonexistent(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	rpcResp := s.call(t, "space_get", map[string]any{
		"spaceSlug": "nonexistent",
	})
	_, isErr := toolResult(t, rpcResp)
	if !isErr {
		t.Error("expected error for nonexistent space")
	}
	if errText := toolErrorText(t, rpcResp); errText != "resource not found" {
		t.Errorf("error = %q, want 'resource not found'", errText)
	}
}

func TestMCPTaskDeleteNonexistent(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "task_delete", map[string]any{
		"spaceSlug": "home",
		"taskId":    "T999999",
	})
	_, isErr := toolResult(t, rpcResp)
	if !isErr {
		t.Error("expected error for nonexistent task")
	}
	if errText := toolErrorText(t, rpcResp); errText != "resource not found" {
		t.Errorf("error = %q, want 'resource not found'", errText)
	}
}

func TestMCPSpaceDeleteNonexistent(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	rpcResp := s.call(t, "space_delete", map[string]any{
		"spaceSlug": "nonexistent",
	})
	_, isErr := toolResult(t, rpcResp)
	if !isErr {
		t.Error("expected error for nonexistent space")
	}
	if errText := toolErrorText(t, rpcResp); errText != "resource not found" {
		t.Errorf("error = %q, want 'resource not found'", errText)
	}
}

func TestMCPSpaceCreateDuplicate(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	s := newMCPSession(t, env)

	createSpace(t, env, "home", "Home")

	rpcResp := s.call(t, "space_create", map[string]any{
		"slug": "home",
		"name": "Home Again",
	})
	_, isErr := toolResult(t, rpcResp)
	if !isErr {
		t.Error("expected error for duplicate space")
	}
	if errText := toolErrorText(t, rpcResp); errText != "resource already exists" {
		t.Errorf("error = %q, want 'resource already exists'", errText)
	}
}

func TestMCPMemberUpdateForbidden(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	createSpace(t, env, "home", "Home")
	viewerToken, viewerID := createAndAddMember(t, env, "home", "viewer@test.com", "Viewer", "password", "viewer")

	// Create a session as the viewer.
	viewerEnv := *env
	viewerEnv.Token = viewerToken
	s := newMCPSession(t, &viewerEnv)

	// Viewer should not be able to update their own role.
	rpcResp := s.call(t, "member_update", map[string]any{
		"spaceSlug": "home",
		"userId":    viewerID,
		"role":      "admin",
	})
	_, isErr := toolResult(t, rpcResp)
	if !isErr {
		t.Error("expected error for forbidden member update")
	}
}

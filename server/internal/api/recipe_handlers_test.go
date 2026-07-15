package api_test

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/sargunv/horologia/server/internal/types"
)

func createRecipe(t *testing.T, env *testEnv, spaceSlug, body string) map[string]any {
	t.Helper()
	resp := doRequest(t, env, http.MethodPost, "/spaces/"+spaceSlug+"/recipes", body)
	assertStatus(t, resp, http.StatusCreated)
	var recipe map[string]any
	readJSON(t, resp, &recipe)
	return recipe
}

func TestRecipeCRUDSearchAndActivity(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	space := testSlug(t, "home")
	createSpace(t, env, space, "Home")

	created := createRecipe(t, env, space, `{
        "name":"Tomato Pasta",
        "description":"A weeknight dinner.",
        "yield":{"amount":4,"unit":"servings"},
        "prepMinutes":10,
        "cookMinutes":20,
        "tags":["Dinner","dinner","Quick"],
        "ingredientSections":[{
            "title":"Pasta",
            "ingredients":[
                {"quantity":1,"unit":"lb","item":"spaghetti"},
                {"quantity":2,"quantityMax":3,"unit":"cups","item":"tomatoes"}
            ]
        }],
        "instructionSections":[{
            "title":"Cook",
            "steps":[{"body":"Boil the pasta."},{"body":"Add the tomatoes."}]
        }]
    }`)

	id := jsonAs[string](t, created["id"])
	if !strings.HasPrefix(id, "R") || created["spaceSlug"] != space {
		t.Fatalf("unexpected recipe identity: %v", created)
	}
	if created["name"] != "Tomato Pasta" || created["description"] != "A weeknight dinner." {
		t.Fatalf("unexpected recipe body: %v", created)
	}
	yield := jsonAs[map[string]any](t, created["yield"])
	if yield["amount"] != float64(4) || yield["unit"] != "servings" {
		t.Fatalf("unexpected yield: %v", yield)
	}
	tagValues := jsonAs[[]any](t, created["tags"])
	if len(tagValues) != 2 || tagValues[0] != "Dinner" || tagValues[1] != "Quick" {
		t.Fatalf("unexpected deduplicated tags: %v", tagValues)
	}
	ingredientSections := jsonAs[[]any](t, created["ingredientSections"])
	ingredients := jsonAs[[]any](t, jsonAs[map[string]any](t, ingredientSections[0])["ingredients"])
	if len(ingredients) != 2 || jsonAs[map[string]any](t, ingredients[1])["quantityMax"] != float64(3) {
		t.Fatalf("unexpected ingredients: %v", ingredientSections)
	}

	resp := doRequest(t, env, http.MethodGet, "/spaces/"+space+"/recipes/"+id, "")
	assertStatus(t, resp, http.StatusOK)
	var fetched map[string]any
	readJSON(t, resp, &fetched)
	if fetched["id"] != id || len(jsonAs[[]any](t, fetched["instructionSections"])) != 1 {
		t.Fatalf("unexpected fetched recipe: %v", fetched)
	}

	resp = doRequest(t, env, http.MethodGet, "/spaces/"+space+"/recipes", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := jsonAs[[]any](t, page["items"])
	if len(items) != 1 {
		t.Fatalf("unexpected recipe page: %v", page)
	}
	summary := jsonAs[map[string]any](t, items[0])
	if summary["id"] != id || summary["ingredientSections"] != nil {
		t.Fatalf("list should return a summary: %v", summary)
	}

	resp = doRequest(t, env, http.MethodPatch, "/spaces/"+space+"/recipes/"+id, `{"name":"Fresh Tomato Pasta"}`)
	assertStatus(t, resp, http.StatusOK)
	var renamed map[string]any
	readJSON(t, resp, &renamed)
	if renamed["name"] != "Fresh Tomato Pasta" || len(jsonAs[[]any](t, renamed["ingredientSections"])) != 1 {
		t.Fatalf("scalar patch did not preserve collections: %v", renamed)
	}

	resp = doRequest(t, env, http.MethodPatch, "/spaces/"+space+"/recipes/"+id, `{"yield":null,"tags":[],"ingredientSections":[]}`)
	assertStatus(t, resp, http.StatusOK)
	var cleared map[string]any
	readJSON(t, resp, &cleared)
	if cleared["yield"] != nil || len(jsonAs[[]any](t, cleared["tags"])) != 0 || len(jsonAs[[]any](t, cleared["ingredientSections"])) != 0 {
		t.Fatalf("collection/null clearing failed: %v", cleared)
	}
	if len(jsonAs[[]any](t, cleared["instructionSections"])) != 1 {
		t.Fatalf("omitted instructions were not preserved: %v", cleared)
	}
	updatedAtBeforeNoOp := jsonAs[string](t, cleared["updatedAt"])

	resp = doRequest(t, env, http.MethodPatch, "/spaces/"+space+"/recipes/"+id, `{
        "name":"Fresh Tomato Pasta",
        "yield":null,
        "prepMinutes":10,
        "cookMinutes":20,
        "tags":[],
        "ingredientSections":[],
        "instructionSections":[{
            "title":"Cook",
            "steps":[{"body":"Boil the pasta."},{"body":"Add the tomatoes."}]
        }]
    }`)
	assertStatus(t, resp, http.StatusOK)
	var unchanged map[string]any
	readJSON(t, resp, &unchanged)
	if unchanged["updatedAt"] != updatedAtBeforeNoOp {
		t.Fatalf("no-op patch changed updatedAt from %v to %v", updatedAtBeforeNoOp, unchanged["updatedAt"])
	}

	resp = doRequest(t, env, http.MethodGet, "/recipes/search?q="+url.QueryEscape(id), "")
	assertStatus(t, resp, http.StatusOK)
	var search map[string]any
	readJSON(t, resp, &search)
	searchItems := jsonAs[[]any](t, search["items"])
	if len(searchItems) != 1 || jsonAs[map[string]any](t, searchItems[0])["id"] != id {
		t.Fatalf("exact recipe search failed: %v", search)
	}

	resp = doRequest(t, env, http.MethodGet, "/spaces/"+space+"/recipes/"+id+"/activity", "")
	assertStatus(t, resp, http.StatusOK)
	var activity map[string]any
	readJSON(t, resp, &activity)
	activityItems := jsonAs[[]any](t, activity["items"])
	if len(activityItems) != 3 {
		t.Fatalf("got %d activity entries, want 3: %v", len(activityItems), activity)
	}
	if jsonAs[map[string]any](t, activityItems[0])["entityType"] != "recipe" {
		t.Fatalf("unexpected activity entity: %v", activityItems[0])
	}

	assertStatusClose(t, doRequest(t, env, http.MethodDelete, "/spaces/"+space+"/recipes/"+id, ""), http.StatusNoContent)
	assertStatusClose(t, doRequest(t, env, http.MethodGet, "/spaces/"+space+"/recipes/"+id, ""), http.StatusNotFound)
}

func TestRecipeValidationAndTransactionRollback(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	space := testSlug(t, "home")
	createSpace(t, env, space, "Home")

	resp := doRequest(t, env, http.MethodPost, "/spaces/"+space+"/recipes", `{
        "name":"Invalid range",
        "ingredientSections":[{"ingredients":[{"quantity":2,"quantityMax":1,"item":"flour"}]}]
    }`)
	assertStatusClose(t, resp, http.StatusBadRequest)

	resp = doRequest(t, env, http.MethodGet, "/spaces/"+space+"/recipes", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	if len(jsonAs[[]any](t, page["items"])) != 0 {
		t.Fatalf("invalid recipe was persisted: %v", page)
	}

	// Tag validation happens after the recipe and nested collections have been inserted. The
	// invalid tag must roll the whole transaction back rather than leaving a partial recipe.
	rollbackName := "Rollback " + space
	resp = doRequest(t, env, http.MethodPost, "/spaces/"+space+"/recipes", `{
		"name":"`+rollbackName+`",
		"tags":[""],
		"ingredientSections":[{"ingredients":[{"quantity":1,"unit":"cup","item":"stock"}]}]
	}`)
	assertStatusClose(t, resp, http.StatusBadRequest)

	var count int
	if err := env.pool.QueryRow(t.Context(), `SELECT count(*) FROM recipes WHERE space_slug = $1 AND name = $2`, space, rollbackName).Scan(&count); err != nil {
		t.Fatalf("count rolled-back recipes: %v", err)
	}
	if count != 0 {
		t.Fatalf("transaction left %d partial recipes, want 0", count)
	}
}

func TestRecipePaginationVisibilityAndScopes(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	visibleSpace := testSlug(t, "visible")
	hiddenSpace := testSlug(t, "hidden")
	createSpace(t, env, visibleSpace, "Visible")
	createSpace(t, env, hiddenSpace, "Hidden")

	createdIDs := make([]string, 0, 3)
	for _, name := range []string{"First soup", "Second soup", "Third soup"} {
		recipe := createRecipe(t, env, visibleSpace, `{"name":"`+name+`"}`)
		createdIDs = append(createdIDs, jsonAs[string](t, recipe["id"]))
	}
	createRecipe(t, env, hiddenSpace, `{"name":"Hidden soup"}`)

	resp := doRequest(t, env, http.MethodGet, "/spaces/"+visibleSpace+"/recipes?limit=2", "")
	assertStatus(t, resp, http.StatusOK)
	var firstPage map[string]any
	readJSON(t, resp, &firstPage)
	firstItems := jsonAs[[]any](t, firstPage["items"])
	if len(firstItems) != 2 {
		t.Fatalf("unexpected first page: %v", firstPage)
	}
	cursor := jsonAs[string](t, firstPage["nextCursor"])
	resp = doRequest(t, env, http.MethodGet, "/spaces/"+visibleSpace+"/recipes?limit=2&cursor="+url.QueryEscape(cursor), "")
	assertStatus(t, resp, http.StatusOK)
	var secondPage map[string]any
	readJSON(t, resp, &secondPage)
	secondItems := jsonAs[[]any](t, secondPage["items"])
	if len(secondItems) != 1 || secondPage["nextCursor"] != nil {
		t.Fatalf("unexpected second page: %v", secondPage)
	}
	seen := map[string]bool{}
	for _, item := range append(firstItems, secondItems...) {
		seen[jsonAs[string](t, jsonAs[map[string]any](t, item)["id"])] = true
	}
	for _, id := range createdIDs {
		if !seen[id] {
			t.Fatalf("pagination omitted %s: %v", id, seen)
		}
	}

	viewerToken, _ := createAndAddMember(t, env, visibleSpace, testEmail(t, "recipe-viewer"), "Viewer", "pass1234", "viewer")
	resp = doRequestAs(t, env, viewerToken, http.MethodGet, "/recipes?limit=2", "")
	assertStatus(t, resp, http.StatusOK)
	var visiblePage map[string]any
	readJSON(t, resp, &visiblePage)
	if len(jsonAs[[]any](t, visiblePage["items"])) != 2 || visiblePage["nextCursor"] == nil {
		t.Fatalf("unexpected visible recipe page: %v", visiblePage)
	}
	resp = doRequestAs(t, env, viewerToken, http.MethodGet, "/recipes?spaceSlug="+visibleSpace, "")
	assertStatus(t, resp, http.StatusOK)
	readJSON(t, resp, &visiblePage)
	if len(jsonAs[[]any](t, visiblePage["items"])) != 3 {
		t.Fatalf("global list leaked or omitted recipes: %v", visiblePage)
	}

	resp = doRequestAs(t, env, viewerToken, http.MethodGet, "/recipes/search?q=soup", "")
	assertStatus(t, resp, http.StatusOK)
	var search map[string]any
	readJSON(t, resp, &search)
	if len(jsonAs[[]any](t, search["items"])) != 3 {
		t.Fatalf("search leaked or omitted recipes: %v", search)
	}
	resp = doRequest(t, env, http.MethodGet, "/recipes/search?q=soup&spaceSlug="+visibleSpace, "")
	assertStatus(t, resp, http.StatusOK)
	readJSON(t, resp, &search)
	if len(jsonAs[[]any](t, search["items"])) != 3 {
		t.Fatalf("space-filtered search returned the wrong recipes: %v", search)
	}
	assertStatusClose(t, doRequestAs(t, env, viewerToken, http.MethodPost, "/spaces/"+visibleSpace+"/recipes", `{"name":"Nope"}`), http.StatusForbidden)

	oauthReadToken := createOAuthAccessTokenForBearerUser(t, env, viewerToken, "recipes:read")
	assertStatusClose(t, doRequestAs(t, env, oauthReadToken, http.MethodGet, "/spaces/"+visibleSpace+"/recipes", ""), http.StatusOK)
	assertStatusClose(t, doRequestAs(t, env, oauthReadToken, http.MethodPost, "/spaces/"+visibleSpace+"/recipes", `{"name":"Nope"}`), http.StatusForbidden)
	oauthWriteToken := createOAuthAccessTokenForBearerUser(t, env, env.Token, "recipes:write")
	assertStatusClose(t, doRequestAs(t, env, oauthWriteToken, http.MethodPost, "/spaces/"+visibleSpace+"/recipes", `{"name":"OAuth recipe"}`), http.StatusCreated)
}

func TestRecipeSpaceIsolationAndCascade(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	spaceA := testSlug(t, "space-a")
	spaceB := testSlug(t, "space-b")
	createSpace(t, env, spaceA, "Space A")
	createSpace(t, env, spaceB, "Space B")
	recipe := createRecipe(t, env, spaceB, `{"name":"Secret soup"}`)
	id := jsonAs[string](t, recipe["id"])

	assertStatusClose(t, doRequest(t, env, http.MethodGet, "/spaces/"+spaceA+"/recipes/"+id, ""), http.StatusNotFound)
	assertStatusClose(t, doRequest(t, env, http.MethodPatch, "/spaces/"+spaceA+"/recipes/"+id, `{"name":"Hacked"}`), http.StatusNotFound)
	assertStatusClose(t, doRequest(t, env, http.MethodDelete, "/spaces/"+spaceA+"/recipes/"+id, ""), http.StatusNotFound)
	assertStatusClose(t, doRequest(t, env, http.MethodGet, "/spaces/"+spaceB+"/recipes/"+id, ""), http.StatusOK)

	dbID, err := types.ParseRecipeID(id)
	if err != nil {
		t.Fatalf("parse recipe ID: %v", err)
	}
	assertStatusClose(t, doRequest(t, env, http.MethodDelete, "/spaces/"+spaceB, ""), http.StatusNoContent)
	var count int
	if err := env.pool.QueryRow(t.Context(), `SELECT count(*) FROM recipes WHERE id = $1`, dbID).Scan(&count); err != nil {
		t.Fatalf("count recipes after space deletion: %v", err)
	}
	if count != 0 {
		t.Fatalf("space deletion left %d recipes, want 0", count)
	}
}

func TestConcurrentRecipePatchPreservesDisjointFields(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	space := testSlug(t, "concurrent")
	createSpace(t, env, space, "Concurrent")
	created := createRecipe(t, env, space, `{"name":"Old name","description":"Old description"}`)
	id := jsonAs[string](t, created["id"])
	dbID, err := types.ParseRecipeID(id)
	if err != nil {
		t.Fatalf("parse recipe ID: %v", err)
	}

	tx, err := env.pool.Begin(t.Context())
	if err != nil {
		t.Fatalf("begin lock transaction: %v", err)
	}
	defer func() { _ = tx.Rollback(t.Context()) }()
	if _, err := tx.Exec(t.Context(), `SELECT id FROM recipes WHERE id = $1 AND space_slug = $2 FOR UPDATE`, dbID, space); err != nil {
		t.Fatalf("lock recipe row: %v", err)
	}

	type patchResult struct {
		status int
		body   string
		err    error
	}
	patchDone := make(chan patchResult, 1)
	go func() {
		req, requestErr := http.NewRequestWithContext(t.Context(), http.MethodPatch, env.Server.URL+"/spaces/"+space+"/recipes/"+id, strings.NewReader(`{"description":"Concurrent description"}`))
		if requestErr != nil {
			patchDone <- patchResult{err: requestErr}
			return
		}
		req.Header.Set("Authorization", "Bearer "+env.Token)
		req.Header.Set("Content-Type", "application/json")
		resp, requestErr := http.DefaultClient.Do(req)
		if requestErr != nil {
			patchDone <- patchResult{err: requestErr}
			return
		}
		defer func() { _ = resp.Body.Close() }()
		body, requestErr := io.ReadAll(resp.Body)
		patchDone <- patchResult{status: resp.StatusCode, body: string(body), err: requestErr}
	}()
	time.Sleep(100 * time.Millisecond)

	if _, err := tx.Exec(t.Context(), `UPDATE recipes SET name = $1 WHERE id = $2 AND space_slug = $3`, "Concurrent name", dbID, space); err != nil {
		t.Fatalf("update name while patch is waiting: %v", err)
	}
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatalf("commit lock transaction: %v", err)
	}

	patched := <-patchDone
	if patched.err != nil {
		t.Fatalf("patch: %v", patched.err)
	}
	if patched.status != http.StatusOK {
		t.Fatalf("patch status = %d, want %d; body: %s", patched.status, http.StatusOK, patched.body)
	}
	resp := doRequest(t, env, http.MethodGet, "/spaces/"+space+"/recipes/"+id, "")
	assertStatus(t, resp, http.StatusOK)
	var fetched map[string]any
	readJSON(t, resp, &fetched)
	if fetched["name"] != "Concurrent name" || fetched["description"] != "Concurrent description" {
		t.Fatalf("concurrent patches were not preserved: %v", fetched)
	}
}

func TestRecipeActivityDetailsAndDeletion(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	space := testSlug(t, "activity")
	createSpace(t, env, space, "Activity")
	created := createRecipe(t, env, space, `{
		"name":"Old soup",
		"tags":["Dinner"],
		"ingredientSections":[{"ingredients":[{"quantity":1,"unit":"cup","item":"stock"}]}],
		"instructionSections":[{"steps":[{"body":"Simmer."}]}]
	}`)
	id := jsonAs[string](t, created["id"])

	assertStatusClose(t, doRequest(t, env, http.MethodPatch, "/spaces/"+space+"/recipes/"+id, `{
		"name":"New soup",
		"prepMinutes":10,
		"tags":["Dinner","Quick"],
		"ingredientSections":[{"ingredients":[{"quantity":2,"unit":"cups","item":"stock"}]}],
		"instructionSections":[{"steps":[{"body":"Boil."}]}]
	}`), http.StatusOK)
	assertStatusClose(t, doRequest(t, env, http.MethodPatch, "/spaces/"+space+"/recipes/"+id, `{"name":"New soup"}`), http.StatusOK)

	items := activityItems(t, env, "/spaces/"+space+"/recipes/"+id+"/activity")
	if len(items) != 2 {
		t.Fatalf("got %d recipe activity entries, want 2", len(items))
	}
	updateDetails := entryDetails(t, entryMap(t, items[0]))
	nameDetail := findDetail(t, updateDetails, "name")
	if nameDetail["from"] != "Old soup" || nameDetail["to"] != "New soup" {
		t.Fatalf("name activity = %v, want Old soup -> New soup", nameDetail)
	}
	for _, field := range []string{"prep_minutes", "ingredients", "instructions"} {
		if detail := findDetail(t, updateDetails, field); detail["to"] != "updated" {
			t.Fatalf("%s activity = %v, want updated", field, detail)
		}
	}
	tagDetail := findDetail(t, updateDetails, "tag")
	if tagDetail["from"] != nil || tagDetail["to"] != "quick" {
		t.Fatalf("tag activity = %v, want added quick", tagDetail)
	}
	createName := findDetail(t, entryDetails(t, entryMap(t, items[1])), "name")
	if createName["from"] != nil || createName["to"] != "Old soup" {
		t.Fatalf("create name activity = %v", createName)
	}

	assertStatusClose(t, doRequest(t, env, http.MethodDelete, "/spaces/"+space+"/recipes/"+id, ""), http.StatusNoContent)
	items = activityItems(t, env, "/spaces/"+space+"/activity")
	var deleteEntry map[string]any
	for _, item := range items {
		entry := entryMap(t, item)
		if entry["entityType"] == "recipe" && entry["entityId"] == id && entry["action"] == "deleted" {
			deleteEntry = entry
			break
		}
	}
	if deleteEntry == nil {
		t.Fatal("space activity did not retain the deleted recipe entry")
	}
	deleteName := findDetail(t, entryDetails(t, deleteEntry), "name")
	if deleteName["from"] != "New soup" || deleteName["to"] != nil {
		t.Fatalf("delete name activity = %v", deleteName)
	}
}

func TestRecipeTagLifecycle(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)
	space := testSlug(t, "tags")
	createSpace(t, env, space, "Tags")
	assertStatusClose(t, doRequest(t, env, http.MethodPost, "/spaces/"+space+"/tags", `{"name":"Dinner"}`), http.StatusCreated)
	created := createRecipe(t, env, space, `{"name":"Soup","tags":["dinner","DINNER"]}`)
	id := jsonAs[string](t, created["id"])
	tags := jsonAs[[]any](t, created["tags"])
	if len(tags) != 1 || tags[0] != "Dinner" {
		t.Fatalf("created recipe tags = %v, want Dinner", tags)
	}

	assertStatusClose(t, doRequest(t, env, http.MethodPatch, "/spaces/"+space+"/tags/Dinner", `{"name":"Supper"}`), http.StatusOK)
	resp := doRequest(t, env, http.MethodGet, "/spaces/"+space+"/recipes/"+id, "")
	assertStatus(t, resp, http.StatusOK)
	var fetched map[string]any
	readJSON(t, resp, &fetched)
	tags = jsonAs[[]any](t, fetched["tags"])
	if len(tags) != 1 || tags[0] != "Supper" {
		t.Fatalf("renamed recipe tags = %v, want Supper", tags)
	}

	assertStatusClose(t, doRequest(t, env, http.MethodDelete, "/spaces/"+space+"/tags/Supper", ""), http.StatusNoContent)
	resp = doRequest(t, env, http.MethodGet, "/spaces/"+space+"/recipes/"+id, "")
	assertStatus(t, resp, http.StatusOK)
	readJSON(t, resp, &fetched)
	if len(jsonAs[[]any](t, fetched["tags"])) != 0 {
		t.Fatalf("deleted tag remained attached to recipe: %v", fetched["tags"])
	}
}

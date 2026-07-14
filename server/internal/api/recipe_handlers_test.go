package api_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
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
        "source":"Family cookbook",
        "sourceUrl":"https://example.com/pasta",
        "tags":["Dinner","dinner","Quick"],
        "ingredientSections":[{
            "title":"Pasta",
            "ingredients":[
                {"quantity":1,"unit":"lb","item":"spaghetti"},
                {"quantity":2,"quantityMax":3,"unit":"cups","item":"tomatoes","preparation":"chopped","optional":true}
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

func TestRecipeValidationIsTransactional(t *testing.T) {
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
	resp = doRequestAs(t, env, viewerToken, http.MethodGet, "/recipes/search?q=soup", "")
	assertStatus(t, resp, http.StatusOK)
	var search map[string]any
	readJSON(t, resp, &search)
	if len(jsonAs[[]any](t, search["items"])) != 3 {
		t.Fatalf("search leaked or omitted recipes: %v", search)
	}
	assertStatusClose(t, doRequestAs(t, env, viewerToken, http.MethodPost, "/spaces/"+visibleSpace+"/recipes", `{"name":"Nope"}`), http.StatusForbidden)

	oauthReadToken := createOAuthAccessTokenForBearerUser(t, env, viewerToken, "recipes:read")
	assertStatusClose(t, doRequestAs(t, env, oauthReadToken, http.MethodGet, "/spaces/"+visibleSpace+"/recipes", ""), http.StatusOK)
	assertStatusClose(t, doRequestAs(t, env, oauthReadToken, http.MethodPost, "/spaces/"+visibleSpace+"/recipes", `{"name":"Nope"}`), http.StatusForbidden)
}

package recipecmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/sargunv/horologia/clients/cli/internal/cmd/support"
)

func TestIngredientAddCreatesFirstSection(t *testing.T) {
	var patch map[string]any
	srv := newRecipeServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeRecipeJSON(t, w, recipeResponse(nil, nil))
		case http.MethodPatch:
			decodeJSONBody(t, r, &patch)
			writeRecipeJSON(t, w, recipeResponse([]map[string]any{{
				"title": "",
				"ingredients": []map[string]any{{
					"quantity": 1.5, "quantityMax": nil, "unit": "cups", "item": "bread flour",
				}},
			}}, nil))
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	})
	defer srv.Close()

	executeRecipe(t, srv.URL, "ingredient", "add", "home", "R1", "--ingredient", "bread flour", "--quantity", "1 1/2 cups")

	sections := patch["ingredientSections"].([]any)
	section := sections[0].(map[string]any)
	ingredient := section["ingredients"].([]any)[0].(map[string]any)
	if got, want := ingredient["quantity"], 1.5; got != want {
		t.Fatalf("quantity = %#v, want %#v", got, want)
	}
	if got, want := ingredient["unit"], "cups"; got != want {
		t.Fatalf("unit = %#v, want %#v", got, want)
	}
	if got, want := ingredient["item"], "bread flour"; got != want {
		t.Fatalf("item = %#v, want %#v", got, want)
	}
}

func TestIngredientMoveBetweenSections(t *testing.T) {
	ingredients := []map[string]any{
		{
			"title": "Dough",
			"ingredients": []map[string]any{
				{"quantity": 1.0, "quantityMax": nil, "unit": "cup", "item": "flour"},
				{"quantity": 1.0, "quantityMax": nil, "unit": "tsp", "item": "salt"},
			},
		},
		{
			"title": "Topping",
			"ingredients": []map[string]any{
				{"quantity": 2.0, "quantityMax": nil, "unit": "tbsp", "item": "oil"},
			},
		},
	}
	var patch struct {
		Sections []struct {
			Title       string `json:"title"`
			Ingredients []struct {
				Item string `json:"item"`
			} `json:"ingredients"`
		} `json:"ingredientSections"`
	}
	srv := newRecipeServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			writeRecipeJSON(t, w, recipeResponse(ingredients, nil))
			return
		}
		decodeJSONBody(t, r, &patch)
		writeRecipeJSON(t, w, recipeResponse(ingredients, nil))
	})
	defer srv.Close()

	executeRecipe(t, srv.URL, "ingredient", "move", "home", "R1", "2.1", "--section", "1", "--position", "2")

	got := []string{
		patch.Sections[0].Ingredients[0].Item,
		patch.Sections[0].Ingredients[1].Item,
		patch.Sections[0].Ingredients[2].Item,
	}
	if want := []string{"flour", "oil", "salt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("first section = %#v, want %#v", got, want)
	}
	if len(patch.Sections[1].Ingredients) != 0 {
		t.Fatalf("second section still has ingredients: %#v", patch.Sections[1].Ingredients)
	}
}

func TestInstructionMoveWithinSection(t *testing.T) {
	instructions := []map[string]any{{
		"title": "",
		"steps": []map[string]any{{"body": "A"}, {"body": "B"}, {"body": "C"}},
	}}
	var patch struct {
		Sections []struct {
			Steps []struct {
				Body string `json:"body"`
			} `json:"steps"`
		} `json:"instructionSections"`
	}
	srv := newRecipeServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			writeRecipeJSON(t, w, recipeResponse(nil, instructions))
			return
		}
		decodeJSONBody(t, r, &patch)
		writeRecipeJSON(t, w, recipeResponse(nil, instructions))
	})
	defer srv.Close()

	executeRecipe(t, srv.URL, "instruction", "move", "home", "R1", "1.3", "--position", "1")

	got := []string{patch.Sections[0].Steps[0].Body, patch.Sections[0].Steps[1].Body, patch.Sections[0].Steps[2].Body}
	if want := []string{"C", "A", "B"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("steps = %#v, want %#v", got, want)
	}
}

func TestSectionRemoveRequiresForceWhenNonEmpty(t *testing.T) {
	patches := 0
	srv := newRecipeServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			patches++
		}
		writeRecipeJSON(t, w, recipeResponse([]map[string]any{{
			"title": "Dough",
			"ingredients": []map[string]any{{
				"quantity": nil, "quantityMax": nil, "unit": "", "item": "flour",
			}},
		}}, nil))
	})
	defer srv.Close()

	_, err := executeRecipeError(t, srv.URL, "ingredient", "section", "remove", "home", "R1", "1")
	if err == nil || !strings.Contains(err.Error(), "section is not empty") {
		t.Fatalf("error = %v, want non-empty section error", err)
	}
	if patches != 0 {
		t.Fatalf("PATCH count = %d, want 0", patches)
	}
}

func TestRecipeCreateParsesConvenienceValues(t *testing.T) {
	var body map[string]any
	srv := newRecipeServer(t, func(w http.ResponseWriter, r *http.Request) {
		decodeJSONBody(t, r, &body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(mustJSON(t, recipeResponse(nil, nil)))
	})
	defer srv.Close()

	executeRecipe(t, srv.URL, "create", "home", "--name", "Bread", "--yield", "2 loaves", "--prep", "1h 15m", "--cook", "45m")

	if got, want := body["prepMinutes"], float64(75); got != want {
		t.Fatalf("prepMinutes = %#v, want %#v", got, want)
	}
	if got, want := body["cookMinutes"], float64(45); got != want {
		t.Fatalf("cookMinutes = %#v, want %#v", got, want)
	}
	if want := map[string]any{"amount": float64(2), "unit": "loaves"}; !reflect.DeepEqual(body["yield"], want) {
		t.Fatalf("yield = %#v, want %#v", body["yield"], want)
	}
}

func TestRecipeReadCommandsPassQueryParameters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/spaces/home/recipes":
			if got, want := r.URL.Query().Get("cursor"), "next-1"; got != want {
				t.Fatalf("list cursor = %q, want %q", got, want)
			}
			if got, want := r.URL.Query().Get("limit"), "5"; got != want {
				t.Fatalf("list limit = %q, want %q", got, want)
			}
			writeRecipeJSON(t, w, map[string]any{"items": []any{}, "nextCursor": "next-2"})
		case "/api/recipes/search":
			if got, want := r.URL.Query().Get("q"), "tomato soup"; got != want {
				t.Fatalf("search query = %q, want %q", got, want)
			}
			if got, want := r.URL.Query().Get("spaceSlug"), "home"; got != want {
				t.Fatalf("search space = %q, want %q", got, want)
			}
			if got, want := r.URL.Query().Get("limit"), "7"; got != want {
				t.Fatalf("search limit = %q, want %q", got, want)
			}
			writeRecipeJSON(t, w, map[string]any{"items": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	stdout := executeRecipe(t, srv.URL, "list", "home", "--cursor", "next-1", "--limit", "5")
	var page map[string]any
	if err := json.Unmarshal([]byte(stdout), &page); err != nil {
		t.Fatalf("decode list output: %v", err)
	}
	if got, want := page["nextCursor"], "next-2"; got != want {
		t.Fatalf("next cursor = %#v, want %#v", got, want)
	}
	executeRecipe(t, srv.URL, "search", " tomato soup ", "--space", "home", "--limit", "7")
}

func TestRecipeUpdateClearsNullableFields(t *testing.T) {
	var patch map[string]any
	srv := newRecipeServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("method = %s, want PATCH", r.Method)
		}
		decodeJSONBody(t, r, &patch)
		writeRecipeJSON(t, w, recipeResponse(nil, nil))
	})
	defer srv.Close()

	executeRecipe(t, srv.URL, "update", "home", "R1", "--clear-yield", "--clear-prep", "--clear-cook")
	for _, field := range []string{"yield", "prepMinutes", "cookMinutes"} {
		value, present := patch[field]
		if !present || value != nil {
			t.Fatalf("%s = %#v (present %v), want explicit null", field, value, present)
		}
	}
}

func newRecipeServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/spaces/home/recipes" && r.URL.Path != "/api/spaces/home/recipes/R1" {
			http.NotFound(w, r)
			return
		}
		handler(w, r)
	}))
}

func executeRecipe(t *testing.T, serverURL string, args ...string) string {
	t.Helper()
	stdout, err := executeRecipeError(t, serverURL, args...)
	if err != nil {
		t.Fatalf("execute recipe command: %v", err)
	}
	return stdout
}

func executeRecipeError(t *testing.T, serverURL string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("HOROLOGIA_SERVER", serverURL)
	t.Setenv("HOROLOGIA_TOKEN", "test-token")
	flags := &support.RootFlags{JSON: true}
	cmd := New(flags)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.ExecuteContext(context.Background())
	return stdout.String(), err
}

func recipeResponse(ingredients []map[string]any, instructions []map[string]any) map[string]any {
	if ingredients == nil {
		ingredients = []map[string]any{}
	}
	if instructions == nil {
		instructions = []map[string]any{}
	}
	return map[string]any{
		"id":                  "R1",
		"spaceSlug":           "home",
		"name":                "Bread",
		"description":         "",
		"yield":               nil,
		"prepMinutes":         nil,
		"cookMinutes":         nil,
		"tags":                []string{},
		"ingredientSections":  ingredients,
		"instructionSections": instructions,
		"createdAt":           "2026-07-14T12:00:00Z",
		"updatedAt":           "2026-07-14T12:00:00Z",
	}
}

func decodeJSONBody(t *testing.T, r *http.Request, target any) {
	t.Helper()
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
}

func writeRecipeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(mustJSON(t, value))
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode JSON: %v", err)
	}
	return data
}

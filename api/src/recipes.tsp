import "@typespec/http";
import "@horologia/typespec-mcp-go";

using Http;
using MCP;

namespace Horologia;

model RecipeYield {
  @minValue(0.0001) amount: float64;
  @minLength(1) @maxLength(100) unit: string;
}

model RecipeIngredient {
  quantity: float64 | null;
  quantityMax: float64 | null;
  unit: string;
  item: string;
  preparation: string;
  optional: boolean;
}

model RecipeIngredientInput {
  @minValue(0.0001) quantity?: float64;
  @minValue(0.0001) quantityMax?: float64;
  @maxLength(100) unit?: string;
  @minLength(1) @maxLength(500) item: string;
  @maxLength(500) preparation?: string;
  optional?: boolean;
}

model RecipeIngredientSection {
  title: string;
  ingredients: RecipeIngredient[];
}

model RecipeIngredientSectionInput {
  @maxLength(200) title?: string;
  @maxItems(100) ingredients: RecipeIngredientInput[];
}

model RecipeStep {
  body: string;
}

model RecipeStepInput {
  @minLength(1) @maxLength(10000) body: string;
}

model RecipeInstructionSection {
  title: string;
  steps: RecipeStep[];
}

model RecipeInstructionSectionInput {
  @maxLength(200) title?: string;
  @maxItems(100) steps: RecipeStepInput[];
}

model Recipe {
  id: string;
  spaceSlug: string;
  name: string;
  description: string;
  yield: RecipeYield | null;
  prepMinutes: int32 | null;
  cookMinutes: int32 | null;
  source: string;
  sourceUrl: string | null;
  tags: string[];
  ingredientSections: RecipeIngredientSection[];
  instructionSections: RecipeInstructionSection[];
  createdAt: utcDateTime;
  updatedAt: utcDateTime;
}

model RecipeSummary {
  id: string;
  spaceSlug: string;
  name: string;
  yield: RecipeYield | null;
  prepMinutes: int32 | null;
  cookMinutes: int32 | null;
  tags: string[];
  updatedAt: utcDateTime;
}

model RecipeCreate {
  @minLength(1) @maxLength(500) name: string;
  @maxLength(10000) description?: string;
  yield?: RecipeYield;
  @minValue(0) prepMinutes?: int32;
  @minValue(0) cookMinutes?: int32;
  @maxLength(1000) source?: string;
  @maxLength(2000) sourceUrl?: string;
  @maxItems(100) tags?: string[];
  @maxItems(20) ingredientSections?: RecipeIngredientSectionInput[];
  @maxItems(20) instructionSections?: RecipeInstructionSectionInput[];
}

model RecipeUpdate {
  @minLength(1) @maxLength(500) name?: string;
  @maxLength(10000) description?: string;
  yield?: RecipeYield | null;
  @minValue(0) prepMinutes?: int32 | null;
  @minValue(0) cookMinutes?: int32 | null;
  @maxLength(1000) source?: string;
  @maxLength(2000) sourceUrl?: string | null;
  @maxItems(100) tags?: string[];
  @maxItems(20) ingredientSections?: RecipeIngredientSectionInput[];
  @maxItems(20) instructionSections?: RecipeInstructionSectionInput[];
}

model RecipePage is Page<RecipeSummary>;

model RecipeSearchResultList {
  items: RecipeSummary[];
}

@route("/spaces/{spaceSlug}/recipes")
@tag("Recipes")
interface SpaceRecipes {
  @mcpTool(
    "recipe_list",
    "List recipe summaries in a space. Returns recipe IDs needed by recipe_get and recipe_update."
  )
  @get
  list(@path spaceSlug: string, ...PageParams): RecipePage | ApiError;

  @mcpTool("recipe_create", "Create a recipe in a space.")
  @post
  create(@path spaceSlug: string, @body body: RecipeCreate):
    | {
        @statusCode statusCode: 201;
        @body body: Recipe;
      }
    | ApiError;

  @mcpTool(
    "recipe_get",
    "Get a recipe by ID. Obtain recipe IDs from recipe_list or recipe_search."
  )
  @get
  read(@path spaceSlug: string, @path recipeId: string): Recipe | ApiError;

  @mcpTool(
    "recipe_update",
    "Update a recipe. Nested collections replace their existing values when provided."
  )
  @patch
  update(
    @path spaceSlug: string,
    @path recipeId: string,
    @body body: RecipeUpdate,
  ): Recipe | ApiError;

  @mcpTool("recipe_delete", "Delete a recipe by ID.")
  @delete
  delete(@path spaceSlug: string, @path recipeId: string): void | ApiError;
}

@route("/recipes")
@tag("Recipes")
interface Recipes {
  @mcpTool(
    "recipe_search",
    "Search recipes visible to the current user across all spaces."
  )
  @get
  @route("/search")
  search(
    @query @minLength(1) q: string,
    @query spaceSlug?: string,
    @query limit?: int32,
  ): RecipeSearchResultList | ApiError;
}

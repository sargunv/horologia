import { describe, expect, it } from "vitest";

import {
  recipeCreateFromDraft,
  recipeDraftFromRecipe,
  validateRecipeDraft,
} from "./recipeDraft.ts";

describe("portable recipe editor state", () => {
  it("parses ordered Markdown-like sections without changing description Markdown", () => {
    const draft = recipeDraftFromRecipe();
    Object.assign(draft, {
      name: "Flatbread",
      description: "A **fast** dinner.",
      yield: "4 servings",
      prepTime: "20 min",
      cookTime: "1h",
      tags: "bread, dinner",
      ingredients: "## Dough\n2 cups | flour\n1 tsp | salt\n\n## Finish\n2 tbsp | olive oil",
      instructions: "## Mix\n- Combine ingredients\n- Knead\n\n## Bake\n- Bake until golden",
    });
    expect(validateRecipeDraft(draft)).toEqual({
      errors: [],
      body: {
        name: "Flatbread",
        description: "A **fast** dinner.",
        yield: { amount: 4, unit: "servings" },
        prepMinutes: 20,
        cookMinutes: 60,
        tags: ["bread", "dinner"],
        ingredientSections: [
          {
            title: "Dough",
            ingredients: [
              { quantity: 2, unit: "cups", item: "flour" },
              { quantity: 1, unit: "tsp", item: "salt" },
            ],
          },
          {
            title: "Finish",
            ingredients: [{ quantity: 2, unit: "tbsp", item: "olive oil" }],
          },
        ],
        instructionSections: [
          { title: "Mix", steps: [{ body: "Combine ingredients" }, { body: "Knead" }] },
          { title: "Bake", steps: [{ body: "Bake until golden" }] },
        ],
      },
    });
  });

  it("reports malformed ingredient lines and omits nullable create-only fields", () => {
    const draft = recipeDraftFromRecipe();
    draft.name = "Toast";
    draft.ingredients = "bread without separator";
    expect(validateRecipeDraft(draft).errors).toEqual([
      "Ingredient line 1 needs a 'quantity | item' separator.",
    ]);
    draft.ingredients = "";
    expect(recipeCreateFromDraft(draft).body).not.toHaveProperty("yield");
  });
});

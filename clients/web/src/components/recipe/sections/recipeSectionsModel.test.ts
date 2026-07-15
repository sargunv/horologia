import { describe, expect, it } from "vitest";
import {
  ingredientSectionsInput,
  insertIngredient,
  instructionSectionsInput,
  recipeSectionsDraft,
  removeIngredient,
  removeStep,
  type IngredientSectionDraft,
  type InstructionSectionDraft,
  type Recipe,
} from "./recipeSectionsModel.ts";

function recipe(): Recipe {
  return {
    id: "R1",
    spaceSlug: "home",
    name: "Soup",
    description: "",
    yield: null,
    prepMinutes: null,
    cookMinutes: null,
    tags: [],
    ingredientSections: [
      {
        title: "Soup",
        ingredients: [{ quantity: 1, quantityMax: null, unit: "cup", item: "stock" }],
      },
    ],
    instructionSections: [{ title: "", steps: [{ body: "Simmer." }] }],
    createdAt: "2026-07-14T00:00:00Z",
    updatedAt: "2026-07-14T00:00:00Z",
  };
}

describe("recipeSectionsDraft", () => {
  it("retains local keys while applying refreshed server values", () => {
    const initial = recipeSectionsDraft(recipe());
    const refreshed = recipeSectionsDraft(
      {
        ...recipe(),
        ingredientSections: [
          {
            title: "Broth",
            ingredients: [{ quantity: 2, quantityMax: null, unit: "cups", item: "stock" }],
          },
        ],
      },
      initial,
    );

    expect(refreshed.ingredientSections[0]!.key).toBe(initial.ingredientSections[0]!.key);
    expect(refreshed.ingredientSections[0]!.ingredients[0]!.key).toBe(
      initial.ingredientSections[0]!.ingredients[0]!.key,
    );
    expect(refreshed.ingredientSections[0]!.title).toBe("Broth");
    expect(refreshed.ingredientSections[0]!.ingredients[0]!.quantity).toBe("2 cups");
  });
});

describe("ingredient section operations", () => {
  const sections = (): IngredientSectionDraft[] => [
    {
      key: "first",
      title: "",
      ingredients: [{ key: "salt", quantity: "", item: "salt" }],
    },
    {
      key: "second",
      title: "Sauce",
      ingredients: [{ key: "tomato", quantity: "2 cups", item: "tomatoes" }],
    },
  ];

  it("removes an empty untitled section with its last ingredient", () => {
    expect(removeIngredient(sections(), "first", "salt").map((section) => section.key)).toEqual([
      "second",
    ]);
  });

  it("creates a default section when adding the first ingredient", () => {
    const added = insertIngredient([]);
    expect(added.sections).toHaveLength(1);
    expect(added.sections[0]!.key).toBe(added.sectionKey);
    expect(added.sections[0]!.ingredients[0]!.key).toBe(added.itemKey);
  });

  it("serializes trimmed titles, quantities, and names", () => {
    expect(
      ingredientSectionsInput([
        {
          key: "section",
          title: " Sauce ",
          ingredients: [{ key: "item", quantity: "1–2 tbsp", item: " oil " }],
        },
      ]),
    ).toEqual([
      {
        title: "Sauce",
        ingredients: [{ quantity: 1, quantityMax: 2, unit: "tbsp", item: "oil" }],
      },
    ]);
  });
});

describe("instruction section operations", () => {
  const sections = (): InstructionSectionDraft[] => [
    { key: "default", title: "", steps: [{ key: "simmer", body: "Simmer." }] },
    { key: "finish", title: "Finish", steps: [] },
  ];

  it("removes an empty untitled section with its last step", () => {
    expect(removeStep(sections(), "default", "simmer").map((section) => section.key)).toEqual([
      "finish",
    ]);
  });

  it("trims instruction input", () => {
    expect(
      instructionSectionsInput([
        { key: "section", title: " Finish ", steps: [{ key: "step", body: " Serve. " }] },
      ]),
    ).toEqual([{ title: "Finish", steps: [{ body: "Serve." }] }]);
  });
});

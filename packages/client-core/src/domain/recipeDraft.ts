import type { components } from "../api/schema.d.ts";
import {
  formatIngredientQuantity,
  parseDurationInput,
  parseIngredientQuantity,
  parseYieldInput,
} from "./recipeInputs";

type Recipe = components["schemas"]["Recipe"];

export interface RecipeDraft {
  name: string;
  description: string;
  yield: string;
  prepTime: string;
  cookTime: string;
  tags: string;
  ingredients: string;
  instructions: string;
}

export interface RecipeDraftValidation {
  body: components["schemas"]["RecipeUpdate"] | null;
  errors: string[];
}

export function recipeDraftFromRecipe(recipe?: Recipe): RecipeDraft {
  return {
    name: recipe?.name ?? "",
    description: recipe?.description ?? "",
    yield: recipe?.yield ? `${recipe.yield.amount} ${recipe.yield.unit}` : "",
    prepTime:
      recipe?.prepMinutes === null || recipe?.prepMinutes === undefined
        ? ""
        : `${recipe.prepMinutes} min`,
    cookTime:
      recipe?.cookMinutes === null || recipe?.cookMinutes === undefined
        ? ""
        : `${recipe.cookMinutes} min`,
    tags: recipe?.tags.join(", ") ?? "",
    ingredients: formatIngredients(recipe?.ingredientSections ?? []),
    instructions: formatInstructions(recipe?.instructionSections ?? []),
  };
}

export function validateRecipeDraft(draft: RecipeDraft): RecipeDraftValidation {
  const errors: string[] = [];
  const parsedYield = draft.yield.trim() ? parseYieldInput(draft.yield) : null;
  const prep = draft.prepTime.trim() ? parseDurationInput(draft.prepTime) : null;
  const cook = draft.cookTime.trim() ? parseDurationInput(draft.cookTime) : null;
  if (!draft.name.trim()) errors.push("Name is required.");
  if (draft.yield.trim() && !parsedYield) errors.push("Yield should look like '4 servings'.");
  if (draft.prepTime.trim() && !prep) errors.push("Prep time should look like '20 min'.");
  if (draft.cookTime.trim() && !cook) errors.push("Cook time should look like '1h 15m'.");
  const ingredientResult = parseIngredients(draft.ingredients);
  errors.push(...ingredientResult.errors);
  if (errors.length > 0) return { body: null, errors };
  return {
    body: {
      name: draft.name.trim(),
      description: draft.description,
      yield: parsedYield ? { amount: parsedYield.amount, unit: parsedYield.unit } : null,
      prepMinutes: prep?.minutes ?? null,
      cookMinutes: cook?.minutes ?? null,
      tags: splitValues(draft.tags),
      ingredientSections: ingredientResult.sections,
      instructionSections: parseInstructions(draft.instructions),
    },
    errors: [],
  };
}

export function recipeCreateFromDraft(draft: RecipeDraft): {
  body: components["schemas"]["RecipeCreate"] | null;
  errors: string[];
} {
  const result = validateRecipeDraft(draft);
  if (!result.body) return { body: null, errors: result.errors };
  return {
    body: {
      name: result.body.name ?? "",
      description: draft.description,
      ...(result.body.yield ? { yield: result.body.yield } : {}),
      ...(result.body.prepMinutes === null || result.body.prepMinutes === undefined
        ? {}
        : { prepMinutes: result.body.prepMinutes }),
      ...(result.body.cookMinutes === null || result.body.cookMinutes === undefined
        ? {}
        : { cookMinutes: result.body.cookMinutes }),
      tags: splitValues(draft.tags),
      ingredientSections: result.body.ingredientSections ?? [],
      instructionSections: result.body.instructionSections ?? [],
    },
    errors: [],
  };
}

function formatIngredients(sections: Recipe["ingredientSections"]): string {
  return sections
    .flatMap((section) => [
      ...(section.title ? [`## ${section.title}`] : []),
      ...section.ingredients.map(
        (ingredient) =>
          `${formatIngredientQuantity(ingredient.quantity, ingredient.quantityMax, ingredient.unit)} | ${ingredient.item}`,
      ),
      "",
    ])
    .join("\n")
    .trim();
}

function formatInstructions(sections: Recipe["instructionSections"]): string {
  return sections
    .flatMap((section) => [
      ...(section.title ? [`## ${section.title}`] : []),
      ...section.steps.map((step) => `- ${step.body}`),
      "",
    ])
    .join("\n")
    .trim();
}

function parseIngredients(value: string): {
  sections: components["schemas"]["RecipeIngredientSectionInput"][];
  errors: string[];
} {
  const sections: components["schemas"]["RecipeIngredientSectionInput"][] = [];
  const errors: string[] = [];
  let current: components["schemas"]["RecipeIngredientSectionInput"] = { ingredients: [] };
  for (const [index, source] of value.split("\n").entries()) {
    const line = source.trim();
    if (!line) continue;
    if (line.startsWith("## ")) {
      if (current.ingredients.length > 0 || current.title) sections.push(current);
      current = { title: line.slice(3).trim(), ingredients: [] };
      continue;
    }
    const separator = line.indexOf("|");
    if (separator < 0) {
      errors.push(`Ingredient line ${index + 1} needs a 'quantity | item' separator.`);
      continue;
    }
    const quantity = parseIngredientQuantity(line.slice(0, separator).trim());
    const item = line.slice(separator + 1).trim();
    if (!quantity || !item) {
      errors.push(`Ingredient line ${index + 1} is invalid.`);
      continue;
    }
    current.ingredients.push({ ...quantity, item });
  }
  if (current.ingredients.length > 0 || current.title) sections.push(current);
  return { sections, errors };
}

function parseInstructions(
  value: string,
): components["schemas"]["RecipeInstructionSectionInput"][] {
  const sections: components["schemas"]["RecipeInstructionSectionInput"][] = [];
  let current: components["schemas"]["RecipeInstructionSectionInput"] = { steps: [] };
  for (const source of value.split("\n")) {
    const line = source.trim();
    if (!line) continue;
    if (line.startsWith("## ")) {
      if (current.steps.length > 0 || current.title) sections.push(current);
      current = { title: line.slice(3).trim(), steps: [] };
      continue;
    }
    current.steps.push({ body: line.replace(/^[-*]\s*/u, "") });
  }
  if (current.steps.length > 0 || current.title) sections.push(current);
  return sections;
}

function splitValues(value: string): string[] {
  return [
    ...new Set(
      value
        .split(",")
        .map((part) => part.trim())
        .filter(Boolean),
    ),
  ];
}

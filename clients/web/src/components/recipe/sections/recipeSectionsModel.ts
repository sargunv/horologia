import { moveKeyedCollectionItem } from "@horologia/client-core/domain/keyed-collections";
import {
  formatIngredientQuantity,
  parseIngredientQuantity,
} from "@horologia/client-core/domain/recipe-inputs";
import type { components } from "@horologia/client-core/schema";

export type Recipe = components["schemas"]["Recipe"];
export type IngredientInput = components["schemas"]["RecipeIngredientInput"];
export type IngredientSectionInput = components["schemas"]["RecipeIngredientSectionInput"];
export type InstructionSectionInput = components["schemas"]["RecipeInstructionSectionInput"];

export interface IngredientDraft {
  key: string;
  quantity: string;
  item: string;
}

export interface IngredientSectionDraft {
  key: string;
  title: string;
  ingredients: IngredientDraft[];
}

export interface StepDraft {
  key: string;
  body: string;
}

export interface InstructionSectionDraft {
  key: string;
  title: string;
  steps: StepDraft[];
}

export interface SectionsDraft {
  ingredientSections: IngredientSectionDraft[];
  instructionSections: InstructionSectionDraft[];
}

export type EditingTarget =
  | { kind: "ingredient"; sectionKey: string; itemKey: string }
  | { kind: "ingredient-section"; sectionKey: string }
  | { kind: "step"; sectionKey: string; itemKey: string }
  | { kind: "instruction-section"; sectionKey: string }
  | null;

let draftKeySequence = 0;

function draftKey(prefix: string): string {
  draftKeySequence += 1;
  return `${prefix}-${draftKeySequence}`;
}

function newIngredientSection(): IngredientSectionDraft {
  return { key: draftKey("ingredient-section"), title: "", ingredients: [] };
}

function newInstructionSection(): InstructionSectionDraft {
  return { key: draftKey("instruction-section"), title: "", steps: [] };
}

export function recipeSectionsDraft(recipe: Recipe, previous?: SectionsDraft): SectionsDraft {
  return {
    ingredientSections: recipe.ingredientSections.map((section, sectionIndex) => ({
      key: previous?.ingredientSections[sectionIndex]?.key ?? draftKey("ingredient-section"),
      title: section.title,
      ingredients: section.ingredients.map((ingredient, ingredientIndex) => ({
        key:
          previous?.ingredientSections[sectionIndex]?.ingredients[ingredientIndex]?.key ??
          draftKey("ingredient"),
        quantity: formatIngredientQuantity(
          ingredient.quantity,
          ingredient.quantityMax,
          ingredient.unit,
        ),
        item: ingredient.item,
      })),
    })),
    instructionSections: recipe.instructionSections.map((section, sectionIndex) => ({
      key: previous?.instructionSections[sectionIndex]?.key ?? draftKey("instruction-section"),
      title: section.title,
      steps: section.steps.map((step, stepIndex) => ({
        key: previous?.instructionSections[sectionIndex]?.steps[stepIndex]?.key ?? draftKey("step"),
        body: step.body,
      })),
    })),
  };
}

function ingredientInput(ingredient: IngredientDraft): IngredientInput {
  const parsed = parseIngredientQuantity(ingredient.quantity)!;
  return {
    ...(parsed.quantity == null ? {} : { quantity: parsed.quantity }),
    ...(parsed.quantityMax == null ? {} : { quantityMax: parsed.quantityMax }),
    unit: parsed.unit,
    item: ingredient.item.trim(),
  };
}

export function ingredientSectionsInput(
  sections: IngredientSectionDraft[],
): IngredientSectionInput[] {
  return sections.map((section) => ({
    title: section.title.trim(),
    ingredients: section.ingredients.map(ingredientInput),
  }));
}

export function instructionSectionsInput(
  sections: InstructionSectionDraft[],
): InstructionSectionInput[] {
  return sections.map((section) => ({
    title: section.title.trim(),
    steps: section.steps.map((step) => ({ body: step.body.trim() })),
  }));
}

export function findIngredient(
  sections: IngredientSectionDraft[],
  sectionKey: string,
  itemKey: string,
): IngredientDraft | undefined {
  return sections
    .find((section) => section.key === sectionKey)
    ?.ingredients.find((ingredient) => ingredient.key === itemKey);
}

export function updateIngredient(
  sections: IngredientSectionDraft[],
  sectionKey: string,
  itemKey: string,
  update: Partial<IngredientDraft>,
): IngredientSectionDraft[] {
  return sections.map((section) =>
    section.key === sectionKey
      ? {
          ...section,
          ingredients: section.ingredients.map((ingredient) =>
            ingredient.key === itemKey ? { ...ingredient, ...update } : ingredient,
          ),
        }
      : section,
  );
}

export function updateIngredientSectionTitle(
  sections: IngredientSectionDraft[],
  sectionKey: string,
  title: string,
): IngredientSectionDraft[] {
  return sections.map((section) => (section.key === sectionKey ? { ...section, title } : section));
}

export function removeIngredient(
  sections: IngredientSectionDraft[],
  sectionKey: string,
  itemKey: string,
): IngredientSectionDraft[] {
  return sections
    .map((section) =>
      section.key === sectionKey
        ? {
            ...section,
            ingredients: section.ingredients.filter((ingredient) => ingredient.key !== itemKey),
          }
        : section,
    )
    .filter((section) => section.title.trim() || section.ingredients.length > 0);
}

export function removeIngredientSection(
  sections: IngredientSectionDraft[],
  sectionKey: string,
): IngredientSectionDraft[] {
  return sections.filter((section) => section.key !== sectionKey);
}

export function insertIngredient(
  sections: IngredientSectionDraft[],
  targetSectionKey?: string,
): { sections: IngredientSectionDraft[]; sectionKey: string; itemKey: string } {
  const next = sections.length === 0 ? [newIngredientSection()] : [...sections];
  const sectionIndex = targetSectionKey
    ? next.findIndex((section) => section.key === targetSectionKey)
    : next.length - 1;
  const targetIndex = sectionIndex < 0 ? next.length - 1 : sectionIndex;
  const ingredient: IngredientDraft = {
    key: draftKey("ingredient"),
    quantity: "",
    item: "",
  };
  const section = next[targetIndex]!;
  next[targetIndex] = { ...section, ingredients: [...section.ingredients, ingredient] };
  return { sections: next, sectionKey: section.key, itemKey: ingredient.key };
}

export function insertIngredientSection(sections: IngredientSectionDraft[]): {
  sections: IngredientSectionDraft[];
  sectionKey: string;
} {
  const section = newIngredientSection();
  return { sections: [...sections, section], sectionKey: section.key };
}

export function moveIngredient(
  sections: IngredientSectionDraft[],
  activeKey: string,
  sourceSectionKey: string,
  targetSectionKey: string,
  targetKey?: string,
): IngredientSectionDraft[] {
  return moveKeyedCollectionItem(
    sections.map((section) => ({
      key: section.key,
      title: section.title,
      items: section.ingredients,
    })),
    activeKey,
    sourceSectionKey,
    targetSectionKey,
    targetKey,
  ).map((section) => ({
    key: section.key,
    title: section.title,
    ingredients: section.items,
  }));
}

export function findStep(
  sections: InstructionSectionDraft[],
  sectionKey: string,
  itemKey: string,
): StepDraft | undefined {
  return sections
    .find((section) => section.key === sectionKey)
    ?.steps.find((step) => step.key === itemKey);
}

export function updateStep(
  sections: InstructionSectionDraft[],
  sectionKey: string,
  itemKey: string,
  body: string,
): InstructionSectionDraft[] {
  return sections.map((section) =>
    section.key === sectionKey
      ? {
          ...section,
          steps: section.steps.map((step) => (step.key === itemKey ? { ...step, body } : step)),
        }
      : section,
  );
}

export function updateInstructionSectionTitle(
  sections: InstructionSectionDraft[],
  sectionKey: string,
  title: string,
): InstructionSectionDraft[] {
  return sections.map((section) => (section.key === sectionKey ? { ...section, title } : section));
}

export function removeStep(
  sections: InstructionSectionDraft[],
  sectionKey: string,
  itemKey: string,
): InstructionSectionDraft[] {
  return sections
    .map((section) =>
      section.key === sectionKey
        ? { ...section, steps: section.steps.filter((step) => step.key !== itemKey) }
        : section,
    )
    .filter((section) => section.title.trim() || section.steps.length > 0);
}

export function removeInstructionSection(
  sections: InstructionSectionDraft[],
  sectionKey: string,
): InstructionSectionDraft[] {
  return sections.filter((section) => section.key !== sectionKey);
}

export function insertStep(
  sections: InstructionSectionDraft[],
  targetSectionKey?: string,
): { sections: InstructionSectionDraft[]; sectionKey: string; itemKey: string } {
  const next = sections.length === 0 ? [newInstructionSection()] : [...sections];
  const sectionIndex = targetSectionKey
    ? next.findIndex((section) => section.key === targetSectionKey)
    : next.length - 1;
  const targetIndex = sectionIndex < 0 ? next.length - 1 : sectionIndex;
  const step: StepDraft = { key: draftKey("step"), body: "" };
  const section = next[targetIndex]!;
  next[targetIndex] = { ...section, steps: [...section.steps, step] };
  return { sections: next, sectionKey: section.key, itemKey: step.key };
}

export function insertInstructionSection(sections: InstructionSectionDraft[]): {
  sections: InstructionSectionDraft[];
  sectionKey: string;
} {
  const section = newInstructionSection();
  return { sections: [...sections, section], sectionKey: section.key };
}

export function moveStep(
  sections: InstructionSectionDraft[],
  activeKey: string,
  sourceSectionKey: string,
  targetSectionKey: string,
  targetKey?: string,
): InstructionSectionDraft[] {
  return moveKeyedCollectionItem(
    sections.map((section) => ({ key: section.key, title: section.title, items: section.steps })),
    activeKey,
    sourceSectionKey,
    targetSectionKey,
    targetKey,
  ).map((section) => ({ key: section.key, title: section.title, steps: section.items }));
}

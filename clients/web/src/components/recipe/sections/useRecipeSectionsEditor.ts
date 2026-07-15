import type { DragEndEvent } from "@dnd-kit/core";
import { type KeyboardEvent, useEffect, useState } from "react";
import { moveKeyed } from "../../../lib/keyedCollections.ts";
import { useRecipePatch } from "../../../lib/mutations.ts";
import { parseIngredientQuantity } from "../../../lib/recipeInputs.ts";
import { readSortableData } from "./recipeSectionDnd.ts";
import {
  findIngredient,
  findStep,
  ingredientSectionsInput,
  insertIngredient,
  insertIngredientSection,
  insertInstructionSection,
  insertStep,
  instructionSectionsInput,
  moveIngredient,
  moveStep,
  recipeSectionsDraft,
  removeIngredient,
  removeIngredientSection,
  removeInstructionSection,
  removeStep,
  updateIngredient,
  updateIngredientSectionTitle,
  updateInstructionSectionTitle,
  updateStep,
  type EditingTarget,
  type IngredientDraft,
  type IngredientSectionDraft,
  type InstructionSectionDraft,
  type Recipe,
} from "./recipeSectionsModel.ts";

interface PersistOptions {
  rollbackOnError?: boolean;
}

export interface IngredientSectionsController {
  sections: IngredientSectionDraft[];
  editing: EditingTarget;
  pending: boolean;
  controlsDisabled: boolean;
  beginIngredient: (sectionKey: string, itemKey: string) => void;
  beginSection: (sectionKey: string) => void;
  cancel: () => void;
  handleEscape: (event: KeyboardEvent<HTMLElement>) => void;
  changeIngredient: (sectionKey: string, itemKey: string, update: Partial<IngredientDraft>) => void;
  changeSectionTitle: (sectionKey: string, title: string) => void;
  finishIngredient: (sectionKey: string, itemKey: string) => void;
  saveSections: () => void;
  deleteIngredient: (sectionKey: string, itemKey: string) => void;
  deleteSection: (sectionKey: string) => void;
  addIngredient: (sectionKey?: string) => void;
  addSection: () => void;
  handleDragEnd: (event: DragEndEvent) => void;
}

export interface InstructionSectionsController {
  sections: InstructionSectionDraft[];
  editing: EditingTarget;
  pending: boolean;
  controlsDisabled: boolean;
  beginStep: (sectionKey: string, itemKey: string) => void;
  beginSection: (sectionKey: string) => void;
  cancel: () => void;
  handleEscape: (event: KeyboardEvent<HTMLElement>) => void;
  changeStep: (sectionKey: string, itemKey: string, body: string) => void;
  changeSectionTitle: (sectionKey: string, title: string) => void;
  finishStep: (sectionKey: string, itemKey: string) => void;
  saveSections: () => void;
  deleteStep: (sectionKey: string, itemKey: string) => void;
  deleteSection: (sectionKey: string) => void;
  addStep: (sectionKey?: string) => void;
  addSection: () => void;
  handleDragEnd: (event: DragEndEvent) => void;
}

export function useRecipeSectionsEditor(recipe: Recipe): {
  ingredients: IngredientSectionsController;
  instructions: InstructionSectionsController;
  errorMessage: string | undefined;
} {
  const mutation = useRecipePatch(recipe.spaceSlug, recipe.id);
  const [draft, setDraft] = useState(() => recipeSectionsDraft(recipe));
  const [editing, setEditing] = useState<EditingTarget>(null);
  const [validationError, setValidationError] = useState<string | null>(null);

  useEffect(() => {
    if (editing === null && !mutation.isPending) {
      setDraft((current) => recipeSectionsDraft(recipe, current));
    }
  }, [editing, mutation.isPending, recipe]);

  function begin(target: Exclude<EditingTarget, null>) {
    if (mutation.isPending) return;
    mutation.reset();
    setValidationError(null);
    setEditing(target);
  }

  function cancel() {
    setDraft((current) => recipeSectionsDraft(recipe, current));
    setValidationError(null);
    mutation.reset();
    setEditing(null);
  }

  function handleEscape(event: KeyboardEvent<HTMLElement>) {
    if (event.key !== "Escape") return;
    event.preventDefault();
    event.stopPropagation();
    cancel();
  }

  function persistIngredients(
    sections: IngredientSectionDraft[],
    { rollbackOnError = false }: PersistOptions = {},
  ) {
    setValidationError(null);
    const input = ingredientSectionsInput(sections);
    const persisted = ingredientSectionsInput(recipeSectionsDraft(recipe).ingredientSections);
    if (JSON.stringify(input) === JSON.stringify(persisted)) {
      setDraft((current) => recipeSectionsDraft(recipe, current));
      mutation.reset();
      setEditing(null);
      return;
    }
    setDraft((current) => ({ ...current, ingredientSections: sections }));
    mutation.mutate(
      { ingredientSections: input },
      {
        onSuccess: (updated) => {
          setDraft((current) => recipeSectionsDraft(updated, current));
          setEditing(null);
        },
        onError: () => {
          if (rollbackOnError) {
            setDraft((current) => recipeSectionsDraft(recipe, current));
            setEditing(null);
          }
        },
      },
    );
  }

  function persistInstructions(
    sections: InstructionSectionDraft[],
    { rollbackOnError = false }: PersistOptions = {},
  ) {
    setValidationError(null);
    const input = instructionSectionsInput(sections);
    const persisted = instructionSectionsInput(recipeSectionsDraft(recipe).instructionSections);
    if (JSON.stringify(input) === JSON.stringify(persisted)) {
      setDraft((current) => recipeSectionsDraft(recipe, current));
      mutation.reset();
      setEditing(null);
      return;
    }
    setDraft((current) => ({ ...current, instructionSections: sections }));
    mutation.mutate(
      { instructionSections: input },
      {
        onSuccess: (updated) => {
          setDraft((current) => recipeSectionsDraft(updated, current));
          setEditing(null);
        },
        onError: () => {
          if (rollbackOnError) {
            setDraft((current) => recipeSectionsDraft(recipe, current));
            setEditing(null);
          }
        },
      },
    );
  }

  function handleIngredientDragEnd(event: DragEndEvent) {
    if (!event.over || event.active.id === event.over.id || controlsDisabled) return;
    const activeData = readSortableData(event.active.data.current);
    const overData = readSortableData(event.over.data.current);
    if (!activeData || !overData) return;

    if (activeData.type === "ingredient-section") {
      const targetKey =
        overData.type === "ingredient-section"
          ? String(event.over.id)
          : overData.type === "ingredient"
            ? overData.sectionKey
            : undefined;
      if (!targetKey) return;
      persistIngredients(moveKeyed(draft.ingredientSections, String(event.active.id), targetKey), {
        rollbackOnError: true,
      });
      return;
    }

    if (activeData.type !== "ingredient") return;
    const targetSectionKey =
      overData.type === "ingredient"
        ? overData.sectionKey
        : overData.type === "ingredient-section"
          ? String(event.over.id)
          : undefined;
    if (!targetSectionKey) return;
    persistIngredients(
      moveIngredient(
        draft.ingredientSections,
        String(event.active.id),
        activeData.sectionKey,
        targetSectionKey,
        overData.type === "ingredient" ? String(event.over.id) : undefined,
      ),
      { rollbackOnError: true },
    );
  }

  function handleInstructionDragEnd(event: DragEndEvent) {
    if (!event.over || event.active.id === event.over.id || controlsDisabled) return;
    const activeData = readSortableData(event.active.data.current);
    const overData = readSortableData(event.over.data.current);
    if (!activeData || !overData) return;

    if (activeData.type === "instruction-section") {
      const targetKey =
        overData.type === "instruction-section"
          ? String(event.over.id)
          : overData.type === "step"
            ? overData.sectionKey
            : undefined;
      if (!targetKey) return;
      persistInstructions(
        moveKeyed(draft.instructionSections, String(event.active.id), targetKey),
        { rollbackOnError: true },
      );
      return;
    }

    if (activeData.type !== "step") return;
    const targetSectionKey =
      overData.type === "step"
        ? overData.sectionKey
        : overData.type === "instruction-section"
          ? String(event.over.id)
          : undefined;
    if (!targetSectionKey) return;
    persistInstructions(
      moveStep(
        draft.instructionSections,
        String(event.active.id),
        activeData.sectionKey,
        targetSectionKey,
        overData.type === "step" ? String(event.over.id) : undefined,
      ),
      { rollbackOnError: true },
    );
  }

  function changeIngredient(sectionKey: string, itemKey: string, update: Partial<IngredientDraft>) {
    setDraft((current) => ({
      ...current,
      ingredientSections: updateIngredient(current.ingredientSections, sectionKey, itemKey, update),
    }));
  }

  function finishIngredient(sectionKey: string, itemKey: string) {
    const ingredient = findIngredient(draft.ingredientSections, sectionKey, itemKey);
    if (!ingredient?.item.trim()) {
      cancel();
      return;
    }
    if (!parseIngredientQuantity(ingredient.quantity)) {
      setValidationError('Use a quantity like "1 cup", "1–2 tbsp", or "to taste".');
      return;
    }
    persistIngredients(draft.ingredientSections);
  }

  function addIngredient(sectionKey?: string) {
    const added = insertIngredient(draft.ingredientSections, sectionKey);
    setDraft((current) => ({ ...current, ingredientSections: added.sections }));
    begin({ kind: "ingredient", sectionKey: added.sectionKey, itemKey: added.itemKey });
  }

  function addIngredientSection() {
    const added = insertIngredientSection(draft.ingredientSections);
    setDraft((current) => ({ ...current, ingredientSections: added.sections }));
    begin({ kind: "ingredient-section", sectionKey: added.sectionKey });
  }

  function changeStep(sectionKey: string, itemKey: string, body: string) {
    setDraft((current) => ({
      ...current,
      instructionSections: updateStep(current.instructionSections, sectionKey, itemKey, body),
    }));
  }

  function finishStep(sectionKey: string, itemKey: string) {
    const step = findStep(draft.instructionSections, sectionKey, itemKey);
    if (!step?.body.trim()) {
      cancel();
      return;
    }
    persistInstructions(draft.instructionSections);
  }

  function addStep(sectionKey?: string) {
    const added = insertStep(draft.instructionSections, sectionKey);
    setDraft((current) => ({ ...current, instructionSections: added.sections }));
    begin({ kind: "step", sectionKey: added.sectionKey, itemKey: added.itemKey });
  }

  function addInstructionSection() {
    const added = insertInstructionSection(draft.instructionSections);
    setDraft((current) => ({ ...current, instructionSections: added.sections }));
    begin({ kind: "instruction-section", sectionKey: added.sectionKey });
  }

  const controlsDisabled = mutation.isPending || editing !== null;
  return {
    ingredients: {
      sections: draft.ingredientSections,
      editing,
      pending: mutation.isPending,
      controlsDisabled,
      beginIngredient: (sectionKey, itemKey) => begin({ kind: "ingredient", sectionKey, itemKey }),
      beginSection: (sectionKey) => begin({ kind: "ingredient-section", sectionKey }),
      cancel,
      handleEscape,
      changeIngredient,
      changeSectionTitle: (sectionKey, title) =>
        setDraft((current) => ({
          ...current,
          ingredientSections: updateIngredientSectionTitle(
            current.ingredientSections,
            sectionKey,
            title,
          ),
        })),
      finishIngredient,
      saveSections: () => persistIngredients(draft.ingredientSections),
      deleteIngredient: (sectionKey, itemKey) =>
        persistIngredients(removeIngredient(draft.ingredientSections, sectionKey, itemKey), {
          rollbackOnError: true,
        }),
      deleteSection: (sectionKey) =>
        persistIngredients(removeIngredientSection(draft.ingredientSections, sectionKey), {
          rollbackOnError: true,
        }),
      addIngredient,
      addSection: addIngredientSection,
      handleDragEnd: handleIngredientDragEnd,
    },
    instructions: {
      sections: draft.instructionSections,
      editing,
      pending: mutation.isPending,
      controlsDisabled,
      beginStep: (sectionKey, itemKey) => begin({ kind: "step", sectionKey, itemKey }),
      beginSection: (sectionKey) => begin({ kind: "instruction-section", sectionKey }),
      cancel,
      handleEscape,
      changeStep,
      changeSectionTitle: (sectionKey, title) =>
        setDraft((current) => ({
          ...current,
          instructionSections: updateInstructionSectionTitle(
            current.instructionSections,
            sectionKey,
            title,
          ),
        })),
      finishStep,
      saveSections: () => persistInstructions(draft.instructionSections),
      deleteStep: (sectionKey, itemKey) =>
        persistInstructions(removeStep(draft.instructionSections, sectionKey, itemKey), {
          rollbackOnError: true,
        }),
      deleteSection: (sectionKey) =>
        persistInstructions(removeInstructionSection(draft.instructionSections, sectionKey), {
          rollbackOnError: true,
        }),
      addStep,
      addSection: addInstructionSection,
      handleDragEnd: handleInstructionDragEnd,
    },
    errorMessage: validationError ?? mutation.error?.message,
  };
}

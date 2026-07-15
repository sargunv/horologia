import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import {
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { GripVertical, Plus, Trash2 } from "lucide-react";
import {
  type CSSProperties,
  type FocusEvent,
  type KeyboardEvent,
  type ReactNode,
  useEffect,
  useState,
} from "react";
import type { components } from "../../api/schema.d.ts";
import { moveKeyed, moveKeyedCollectionItem } from "../../lib/keyedCollections.ts";
import { useRecipePatch } from "../../lib/mutations.ts";
import { formatIngredientQuantity, parseIngredientQuantity } from "../../lib/recipeInputs.ts";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";

type Recipe = components["schemas"]["Recipe"];
type IngredientInput = components["schemas"]["RecipeIngredientInput"];
type IngredientSectionInput = components["schemas"]["RecipeIngredientSectionInput"];
type InstructionSectionInput = components["schemas"]["RecipeInstructionSectionInput"];

interface IngredientDraft {
  key: string;
  quantity: string;
  item: string;
}

interface IngredientSectionDraft {
  key: string;
  title: string;
  ingredients: IngredientDraft[];
}

interface StepDraft {
  key: string;
  body: string;
}

interface InstructionSectionDraft {
  key: string;
  title: string;
  steps: StepDraft[];
}

interface SectionsDraft {
  ingredientSections: IngredientSectionDraft[];
  instructionSections: InstructionSectionDraft[];
}

type Editing =
  | { kind: "ingredient"; sectionIndex: number; itemIndex: number }
  | { kind: "ingredient-section"; sectionIndex: number }
  | { kind: "step"; sectionIndex: number; itemIndex: number }
  | { kind: "instruction-section"; sectionIndex: number }
  | null;

type SortableData =
  | { type: "ingredient-section" }
  | { type: "ingredient"; sectionKey: string }
  | { type: "instruction-section" }
  | { type: "step"; sectionKey: string };

function sortableData(data: Record<string, unknown> | undefined): SortableData | undefined {
  if (!data || typeof data["type"] !== "string") return undefined;
  if (data["type"] === "ingredient-section" || data["type"] === "instruction-section") {
    return { type: data["type"] };
  }
  if (
    (data["type"] === "ingredient" || data["type"] === "step") &&
    typeof data["sectionKey"] === "string"
  ) {
    return { type: data["type"], sectionKey: data["sectionKey"] };
  }
  return undefined;
}

let draftKeySequence = 0;

function draftKey(prefix: string): string {
  draftKeySequence += 1;
  return `${prefix}-${draftKeySequence}`;
}

const emptyIngredient = (): IngredientDraft => ({
  key: draftKey("ingredient"),
  quantity: "",
  item: "",
});

function recipeDraft(recipe: Recipe, previous?: SectionsDraft): SectionsDraft {
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

function ingredientSectionsInput(sections: IngredientSectionDraft[]): IngredientSectionInput[] {
  return sections.map((section) => ({
    title: section.title.trim(),
    ingredients: section.ingredients.map(ingredientInput),
  }));
}

function instructionSectionsInput(sections: InstructionSectionDraft[]): InstructionSectionInput[] {
  return sections.map((section) => ({
    title: section.title.trim(),
    steps: section.steps.map((step) => ({ body: step.body.trim() })),
  }));
}

function moveIngredient(
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

function moveStep(
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

function AddRow({
  label,
  onClick,
  disabled,
  variant = "empty",
}: {
  label: string;
  onClick: () => void;
  disabled: boolean;
  variant?: "empty" | "inline";
}) {
  return (
    <button
      type="button"
      className={
        variant === "inline"
          ? "flex w-full items-center justify-center gap-2 border-t border-base-300 px-3 py-2 text-sm text-base-content/50 transition-colors hover:bg-base-200 hover:text-base-content/80 disabled:opacity-50"
          : "flex w-full items-center justify-center gap-2 rounded-box border-2 border-dashed border-base-300 p-3 text-sm text-base-content/60 transition-colors hover:border-base-content/40 hover:text-base-content/80 disabled:opacity-50"
      }
      onClick={onClick}
      disabled={disabled}
    >
      <Plus className="size-4" aria-hidden="true" />
      {label}
    </button>
  );
}

function AddSectionButton({ onClick, disabled }: { onClick: () => void; disabled: boolean }) {
  return (
    <button
      type="button"
      className="btn btn-ghost btn-sm gap-1.5 px-2 font-normal text-base-content/60"
      onClick={onClick}
      disabled={disabled}
    >
      <Plus className="size-3.5" aria-hidden="true" />
      Add section
    </button>
  );
}

function DeleteButton({
  label,
  pending,
  onDelete,
  visible = false,
}: {
  label: string;
  pending: boolean;
  onDelete: () => void;
  visible?: boolean;
}) {
  return (
    <button
      type="button"
      className={`btn btn-ghost btn-square btn-xs shrink-0 text-base-content/35 hover:text-error ${
        visible ? "opacity-100" : "opacity-0 group-hover:opacity-100 focus:opacity-100"
      }`}
      aria-label={`Delete ${label}`}
      disabled={pending}
      onMouseDown={(event) => event.preventDefault()}
      onClick={onDelete}
    >
      <Trash2 className="size-3.5" aria-hidden="true" />
    </button>
  );
}

function SortableRoot({
  onDragEnd,
  children,
}: {
  onDragEnd: (event: DragEndEvent) => void;
  children: ReactNode;
}) {
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  return (
    <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
      {children}
    </DndContext>
  );
}

function SortableItem({
  id,
  label,
  disabled,
  data,
  canMoveBetweenSections = false,
  reserveHandleSpace = false,
  children,
}: {
  id: string;
  label: string;
  disabled: boolean;
  data: SortableData;
  canMoveBetweenSections?: boolean;
  reserveHandleSpace?: boolean;
  children: (props: {
    setNodeRef: (node: HTMLElement | null) => void;
    style: CSSProperties;
    className: string;
    handle: ReactNode;
  }) => ReactNode;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging, isOver } =
    useSortable({
      id,
      data,
      disabled: { draggable: disabled, droppable: false },
    });
  return children({
    setNodeRef,
    style: { transform: CSS.Transform.toString(transform), transition },
    className: isDragging
      ? "relative z-10 opacity-60"
      : isOver
        ? "outline-2 outline-offset-2 outline-primary/40"
        : "",
    handle: disabled ? (
      reserveHandleSpace ? (
        <span className="size-6 shrink-0" aria-hidden="true" />
      ) : null
    ) : (
      <button
        type="button"
        className="btn btn-ghost btn-square btn-xs shrink-0 cursor-grab touch-none text-base-content/45 hover:text-base-content/75 focus:text-base-content/75 active:cursor-grabbing"
        aria-label={
          canMoveBetweenSections
            ? `Drag to reorder or move ${label} between sections`
            : `Drag to reorder ${label}`
        }
        title={
          canMoveBetweenSections ? "Drag to reorder or move between sections" : "Drag to reorder"
        }
        {...attributes}
        {...listeners}
      >
        <GripVertical className="size-3.5" aria-hidden="true" />
      </button>
    ),
  });
}

function RecipeSectionShell({
  id,
  kind,
  title,
  editing,
  sortableDisabled,
  reserveHandleSpace,
  pending,
  controlsDisabled,
  hasItems,
  addItemLabel,
  onTitleChange,
  onSaveTitle,
  onCancel,
  onBeginEditing,
  onDelete,
  onAddItem,
  children,
}: {
  id: string;
  kind: "ingredient" | "instruction";
  title: string;
  editing: boolean;
  sortableDisabled: boolean;
  reserveHandleSpace: boolean;
  pending: boolean;
  controlsDisabled: boolean;
  hasItems: boolean;
  addItemLabel: string;
  onTitleChange: (title: string) => void;
  onSaveTitle: () => void;
  onCancel: () => void;
  onBeginEditing: () => void;
  onDelete: () => void;
  onAddItem: () => void;
  children: ReactNode;
}) {
  const label = `${kind} section`;
  const showHeader = editing || Boolean(title);
  return (
    <SortableItem
      id={id}
      label={title || label}
      data={{ type: `${kind}-section` }}
      disabled={sortableDisabled}
      reserveHandleSpace={reserveHandleSpace}
    >
      {({ setNodeRef, style, className, handle }) => (
        <div ref={setNodeRef} style={style} className={`space-y-2 ${className}`}>
          {showHeader && (
            <div className="group flex items-center gap-1">
              {handle}
              {editing ? (
                <input
                  className="min-w-0 flex-1 border-b-2 border-primary bg-transparent px-1 py-1 text-sm font-semibold outline-none"
                  aria-label={`${kind === "ingredient" ? "Ingredient" : "Instruction"} section title`}
                  placeholder="Section title"
                  value={title}
                  maxLength={200}
                  autoFocus
                  onChange={(event) => onTitleChange(event.target.value)}
                  onBlur={() => {
                    if (title.trim()) onSaveTitle();
                    else onCancel();
                  }}
                  onKeyDown={(event) => {
                    if (event.key === "Escape") {
                      event.preventDefault();
                      event.stopPropagation();
                      onCancel();
                    }
                    if (event.key === "Enter") event.currentTarget.blur();
                  }}
                />
              ) : (
                <button
                  type="button"
                  className="min-w-0 flex-1 rounded-field px-1 text-left text-sm font-semibold text-base-content/70 transition-colors hover:bg-base-200 hover:text-base-content"
                  onClick={onBeginEditing}
                >
                  {title}
                </button>
              )}
              <DeleteButton
                label={title || label}
                pending={pending}
                visible={editing}
                onDelete={onDelete}
              />
            </div>
          )}

          {hasItems ? (
            <div className="overflow-hidden rounded-box border border-base-300">
              {children}
              <AddRow
                label={addItemLabel}
                onClick={onAddItem}
                disabled={controlsDisabled}
                variant="inline"
              />
            </div>
          ) : (
            <AddRow label={addItemLabel} onClick={onAddItem} disabled={controlsDisabled} />
          )}
        </div>
      )}
    </SortableItem>
  );
}

function focusStayedInside(event: FocusEvent<HTMLElement>): boolean {
  return event.relatedTarget instanceof Node && event.currentTarget.contains(event.relatedTarget);
}

export function RecipeSectionsEditor({ recipe }: { recipe: Recipe }) {
  const mutation = useRecipePatch(recipe.spaceSlug, recipe.id);
  const [draft, setDraft] = useState(() => recipeDraft(recipe));
  const [editing, setEditing] = useState<Editing>(null);
  const [validationError, setValidationError] = useState<string | null>(null);

  useEffect(() => {
    if (editing === null && !mutation.isPending) {
      setDraft((current) => recipeDraft(recipe, current));
    }
  }, [editing, mutation.isPending, recipe]);

  function begin(target: Exclude<Editing, null>) {
    if (mutation.isPending) return;
    mutation.reset();
    setValidationError(null);
    setEditing(target);
  }

  function cancel() {
    setDraft((current) => recipeDraft(recipe, current));
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
    { rollbackOnError = false }: { rollbackOnError?: boolean } = {},
  ) {
    setValidationError(null);
    const input = ingredientSectionsInput(sections);
    const persisted = ingredientSectionsInput(recipeDraft(recipe).ingredientSections);
    if (JSON.stringify(input) === JSON.stringify(persisted)) {
      setDraft((current) => recipeDraft(recipe, current));
      mutation.reset();
      setEditing(null);
      return;
    }
    setDraft((current) => ({ ...current, ingredientSections: sections }));
    mutation.mutate(
      { ingredientSections: input },
      {
        onSuccess: (updated) => {
          setDraft((current) => recipeDraft(updated, current));
          setEditing(null);
        },
        onError: () => {
          if (rollbackOnError) {
            setDraft((current) => recipeDraft(recipe, current));
            setEditing(null);
          }
        },
      },
    );
  }

  function persistInstructions(
    sections: InstructionSectionDraft[],
    { rollbackOnError = false }: { rollbackOnError?: boolean } = {},
  ) {
    setValidationError(null);
    const input = instructionSectionsInput(sections);
    const persisted = instructionSectionsInput(recipeDraft(recipe).instructionSections);
    if (JSON.stringify(input) === JSON.stringify(persisted)) {
      setDraft((current) => recipeDraft(recipe, current));
      mutation.reset();
      setEditing(null);
      return;
    }
    setDraft((current) => ({ ...current, instructionSections: sections }));
    mutation.mutate(
      { instructionSections: input },
      {
        onSuccess: (updated) => {
          setDraft((current) => recipeDraft(updated, current));
          setEditing(null);
        },
        onError: () => {
          if (rollbackOnError) {
            setDraft((current) => recipeDraft(recipe, current));
            setEditing(null);
          }
        },
      },
    );
  }

  function handleIngredientDragEnd(event: DragEndEvent) {
    if (!event.over || event.active.id === event.over.id || controlsDisabled) return;

    const activeData = sortableData(event.active.data.current);
    const overData = sortableData(event.over.data.current);
    if (!activeData || !overData) return;

    if (activeData.type === "ingredient-section") {
      const sourceIndex = draft.ingredientSections.findIndex(
        (section) => section.key === event.active.id,
      );
      const targetKey =
        overData.type === "ingredient-section"
          ? String(event.over.id)
          : overData.type === "ingredient"
            ? overData.sectionKey
            : undefined;
      if (!targetKey) return;
      const targetIndex = draft.ingredientSections.findIndex(
        (section) => section.key === targetKey,
      );
      if (sourceIndex >= 0 && targetIndex >= 0 && sourceIndex !== targetIndex) {
        persistIngredients(
          moveKeyed(draft.ingredientSections, String(event.active.id), targetKey),
          { rollbackOnError: true },
        );
      }
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

    const activeData = sortableData(event.active.data.current);
    const overData = sortableData(event.over.data.current);
    if (!activeData || !overData) return;

    if (activeData.type === "instruction-section") {
      const sourceIndex = draft.instructionSections.findIndex(
        (section) => section.key === event.active.id,
      );
      const targetKey =
        overData.type === "instruction-section"
          ? String(event.over.id)
          : overData.type === "step"
            ? overData.sectionKey
            : undefined;
      if (!targetKey) return;
      const targetIndex = draft.instructionSections.findIndex(
        (section) => section.key === targetKey,
      );
      if (sourceIndex >= 0 && targetIndex >= 0 && sourceIndex !== targetIndex) {
        persistInstructions(
          moveKeyed(draft.instructionSections, String(event.active.id), targetKey),
          { rollbackOnError: true },
        );
      }
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

  function updateIngredient(
    sectionIndex: number,
    itemIndex: number,
    update: Partial<IngredientDraft>,
  ) {
    setDraft((current) => ({
      ...current,
      ingredientSections: current.ingredientSections.map((section, index) =>
        index === sectionIndex
          ? {
              ...section,
              ingredients: section.ingredients.map((ingredient, ingredientIndex) =>
                ingredientIndex === itemIndex ? { ...ingredient, ...update } : ingredient,
              ),
            }
          : section,
      ),
    }));
  }

  function finishIngredient(sectionIndex: number, itemIndex: number) {
    const ingredient = draft.ingredientSections[sectionIndex]?.ingredients[itemIndex];
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

  function deleteIngredient(sectionIndex: number, itemIndex: number) {
    const next = draft.ingredientSections
      .map((section, index) =>
        index === sectionIndex
          ? {
              ...section,
              ingredients: section.ingredients.filter(
                (_, ingredientIndex) => ingredientIndex !== itemIndex,
              ),
            }
          : section,
      )
      .filter((section) => section.title.trim() || section.ingredients.length > 0);
    persistIngredients(next, { rollbackOnError: true });
  }

  function addIngredient(targetSectionIndex?: number) {
    const next = [...draft.ingredientSections];
    if (next.length === 0) {
      next.push({ key: draftKey("ingredient-section"), title: "", ingredients: [] });
    }
    const sectionIndex = targetSectionIndex ?? next.length - 1;
    next[sectionIndex] = {
      ...next[sectionIndex]!,
      ingredients: [...next[sectionIndex]!.ingredients, emptyIngredient()],
    };
    const itemIndex = next[sectionIndex]!.ingredients.length - 1;
    setDraft((current) => ({ ...current, ingredientSections: next }));
    begin({ kind: "ingredient", sectionIndex, itemIndex });
  }

  function addIngredientSection() {
    const next = [
      ...draft.ingredientSections,
      { key: draftKey("ingredient-section"), title: "", ingredients: [] },
    ];
    const sectionIndex = next.length - 1;
    setDraft((current) => ({ ...current, ingredientSections: next }));
    begin({ kind: "ingredient-section", sectionIndex });
  }

  function addStep(targetSectionIndex?: number) {
    const next = [...draft.instructionSections];
    if (next.length === 0) {
      next.push({ key: draftKey("instruction-section"), title: "", steps: [] });
    }
    const sectionIndex = targetSectionIndex ?? next.length - 1;
    next[sectionIndex] = {
      ...next[sectionIndex]!,
      steps: [...next[sectionIndex]!.steps, { key: draftKey("step"), body: "" }],
    };
    const itemIndex = next[sectionIndex]!.steps.length - 1;
    setDraft((current) => ({ ...current, instructionSections: next }));
    begin({ kind: "step", sectionIndex, itemIndex });
  }

  function addInstructionSection() {
    const next = [
      ...draft.instructionSections,
      { key: draftKey("instruction-section"), title: "", steps: [] },
    ];
    const sectionIndex = next.length - 1;
    setDraft((current) => ({ ...current, instructionSections: next }));
    begin({ kind: "instruction-section", sectionIndex });
  }

  const controlsDisabled = mutation.isPending || editing !== null;
  const errorMessage = validationError ?? mutation.error?.message;

  return (
    <div className="space-y-8">
      <section className="space-y-3">
        <h2 className="text-lg font-semibold">Ingredients</h2>

        {draft.ingredientSections.length === 0 ? (
          <AddRow
            label="Add the first ingredient"
            onClick={() => addIngredient()}
            disabled={controlsDisabled}
          />
        ) : (
          <SortableRoot onDragEnd={handleIngredientDragEnd}>
            <SortableContext
              items={draft.ingredientSections.map((section) => section.key)}
              strategy={verticalListSortingStrategy}
            >
              <div className="space-y-5">
                {draft.ingredientSections.map((section, sectionIndex) => {
                  const editingSection =
                    editing?.kind === "ingredient-section" && editing.sectionIndex === sectionIndex;
                  return (
                    <RecipeSectionShell
                      key={section.key}
                      id={section.key}
                      kind="ingredient"
                      title={section.title}
                      editing={editingSection}
                      sortableDisabled={
                        controlsDisabled ||
                        draft.ingredientSections.length < 2 ||
                        !section.title.trim()
                      }
                      reserveHandleSpace={
                        draft.ingredientSections.length >= 2 && Boolean(section.title.trim())
                      }
                      pending={mutation.isPending}
                      controlsDisabled={controlsDisabled}
                      hasItems={section.ingredients.length > 0}
                      addItemLabel="Add ingredient"
                      onTitleChange={(title) =>
                        setDraft((current) => ({
                          ...current,
                          ingredientSections: current.ingredientSections.map((value, index) =>
                            index === sectionIndex ? { ...value, title } : value,
                          ),
                        }))
                      }
                      onSaveTitle={() => persistIngredients(draft.ingredientSections)}
                      onCancel={cancel}
                      onBeginEditing={() => begin({ kind: "ingredient-section", sectionIndex })}
                      onDelete={() =>
                        persistIngredients(
                          draft.ingredientSections.filter((_, index) => index !== sectionIndex),
                          { rollbackOnError: true },
                        )
                      }
                      onAddItem={() => addIngredient(sectionIndex)}
                    >
                      <SortableContext
                        items={section.ingredients.map((ingredient) => ingredient.key)}
                        strategy={verticalListSortingStrategy}
                      >
                        <ul className="divide-y divide-base-300">
                          {section.ingredients.map((ingredient, itemIndex) => {
                            const editingIngredient =
                              editing?.kind === "ingredient" &&
                              editing.sectionIndex === sectionIndex &&
                              editing.itemIndex === itemIndex;
                            return (
                              <SortableItem
                                key={ingredient.key}
                                id={ingredient.key}
                                label={ingredient.item || "ingredient"}
                                data={{ type: "ingredient", sectionKey: section.key }}
                                disabled={controlsDisabled}
                                canMoveBetweenSections
                                reserveHandleSpace
                              >
                                {({ setNodeRef, style, className, handle }) => (
                                  <li
                                    ref={setNodeRef}
                                    style={style}
                                    className={`group flex items-center gap-1 px-2 py-2 ${className}`}
                                  >
                                    {handle}
                                    {editingIngredient ? (
                                      <div
                                        data-inline-editor
                                        className="grid min-w-0 flex-1 grid-cols-[4rem_minmax(0,1fr)] items-center gap-3"
                                        onBlur={(event) => {
                                          if (!focusStayedInside(event)) {
                                            finishIngredient(sectionIndex, itemIndex);
                                          }
                                        }}
                                        onKeyDown={handleEscape}
                                      >
                                        <input
                                          className="min-w-0 border-b border-base-content/20 bg-transparent py-1 text-base leading-6 tabular-nums outline-none placeholder:text-base-content/40 focus:border-primary"
                                          aria-label="Quantity"
                                          placeholder="1 cup"
                                          value={ingredient.quantity}
                                          maxLength={120}
                                          onChange={(event) =>
                                            updateIngredient(sectionIndex, itemIndex, {
                                              quantity: event.target.value,
                                            })
                                          }
                                          onKeyDown={(event) => {
                                            if (event.key === "Enter") event.currentTarget.blur();
                                          }}
                                        />
                                        <input
                                          className="min-w-0 border-b border-base-content/20 bg-transparent py-1 text-base leading-6 outline-none placeholder:text-base-content/40 focus:border-primary"
                                          aria-label="Ingredient"
                                          placeholder="Ingredient"
                                          value={ingredient.item}
                                          maxLength={500}
                                          autoFocus
                                          onChange={(event) =>
                                            updateIngredient(sectionIndex, itemIndex, {
                                              item: event.target.value,
                                            })
                                          }
                                          onKeyDown={(event) => {
                                            if (event.key === "Enter") {
                                              event.currentTarget.blur();
                                            }
                                          }}
                                        />
                                      </div>
                                    ) : (
                                      <button
                                        type="button"
                                        className="grid min-w-0 flex-1 grid-cols-[4rem_minmax(0,1fr)] gap-3 rounded-field text-left transition-colors hover:bg-base-200"
                                        onClick={() =>
                                          begin({
                                            kind: "ingredient",
                                            sectionIndex,
                                            itemIndex,
                                          })
                                        }
                                      >
                                        <span className="text-base leading-6 tabular-nums text-base-content/70">
                                          {ingredient.quantity}
                                        </span>
                                        <span className="min-w-0 text-base leading-6">
                                          {ingredient.item}
                                        </span>
                                      </button>
                                    )}
                                    <DeleteButton
                                      label={ingredient.item || "ingredient"}
                                      pending={mutation.isPending}
                                      visible={editingIngredient}
                                      onDelete={() => deleteIngredient(sectionIndex, itemIndex)}
                                    />
                                  </li>
                                )}
                              </SortableItem>
                            );
                          })}
                        </ul>
                      </SortableContext>
                    </RecipeSectionShell>
                  );
                })}
              </div>
            </SortableContext>
          </SortableRoot>
        )}
        <AddSectionButton onClick={addIngredientSection} disabled={controlsDisabled} />
      </section>

      <section className="space-y-3">
        <h2 className="text-lg font-semibold">Instructions</h2>

        {draft.instructionSections.length === 0 ? (
          <AddRow
            label="Add the first instruction"
            onClick={() => addStep()}
            disabled={controlsDisabled}
          />
        ) : (
          <SortableRoot onDragEnd={handleInstructionDragEnd}>
            <SortableContext
              items={draft.instructionSections.map((section) => section.key)}
              strategy={verticalListSortingStrategy}
            >
              <div className="space-y-5">
                {draft.instructionSections.map((section, sectionIndex) => {
                  const editingSection =
                    editing?.kind === "instruction-section" &&
                    editing.sectionIndex === sectionIndex;
                  return (
                    <RecipeSectionShell
                      key={section.key}
                      id={section.key}
                      kind="instruction"
                      title={section.title}
                      editing={editingSection}
                      sortableDisabled={
                        controlsDisabled ||
                        draft.instructionSections.length < 2 ||
                        !section.title.trim()
                      }
                      reserveHandleSpace={
                        draft.instructionSections.length >= 2 && Boolean(section.title.trim())
                      }
                      pending={mutation.isPending}
                      controlsDisabled={controlsDisabled}
                      hasItems={section.steps.length > 0}
                      addItemLabel="Add step"
                      onTitleChange={(title) =>
                        setDraft((current) => ({
                          ...current,
                          instructionSections: current.instructionSections.map((value, index) =>
                            index === sectionIndex ? { ...value, title } : value,
                          ),
                        }))
                      }
                      onSaveTitle={() => persistInstructions(draft.instructionSections)}
                      onCancel={cancel}
                      onBeginEditing={() => begin({ kind: "instruction-section", sectionIndex })}
                      onDelete={() =>
                        persistInstructions(
                          draft.instructionSections.filter((_, index) => index !== sectionIndex),
                          { rollbackOnError: true },
                        )
                      }
                      onAddItem={() => addStep(sectionIndex)}
                    >
                      <SortableContext
                        items={section.steps.map((step) => step.key)}
                        strategy={verticalListSortingStrategy}
                      >
                        <ol className="divide-y divide-base-300">
                          {section.steps.map((step, itemIndex) => {
                            const editingStep =
                              editing?.kind === "step" &&
                              editing.sectionIndex === sectionIndex &&
                              editing.itemIndex === itemIndex;
                            return (
                              <SortableItem
                                key={step.key}
                                id={step.key}
                                label={`step ${itemIndex + 1}`}
                                data={{ type: "step", sectionKey: section.key }}
                                disabled={controlsDisabled}
                                canMoveBetweenSections
                                reserveHandleSpace
                              >
                                {({ setNodeRef, style, className, handle }) => (
                                  <li
                                    ref={setNodeRef}
                                    style={style}
                                    className={`group flex items-start gap-1 px-2 py-2.5 ${className}`}
                                  >
                                    {handle}
                                    <span className="flex size-7 shrink-0 items-center justify-center text-sm font-semibold tabular-nums text-base-content/70">
                                      {itemIndex + 1}
                                    </span>
                                    {editingStep ? (
                                      <textarea
                                        className="field-sizing-content min-h-7 min-w-0 flex-1 resize-none bg-transparent pt-0.5 leading-relaxed outline-none placeholder:text-base-content/40 focus:border-b focus:border-primary"
                                        aria-label={`Step ${itemIndex + 1}`}
                                        value={step.body}
                                        maxLength={10000}
                                        autoFocus
                                        placeholder="Describe this step"
                                        onChange={(event) =>
                                          setDraft((current) => ({
                                            ...current,
                                            instructionSections: current.instructionSections.map(
                                              (value, index) =>
                                                index === sectionIndex
                                                  ? {
                                                      ...value,
                                                      steps: value.steps.map(
                                                        (valueStep, stepIndex) =>
                                                          stepIndex === itemIndex
                                                            ? {
                                                                ...valueStep,
                                                                body: event.target.value,
                                                              }
                                                            : valueStep,
                                                      ),
                                                    }
                                                  : value,
                                            ),
                                          }))
                                        }
                                        onBlur={() => {
                                          if (!step.body.trim()) cancel();
                                          else persistInstructions(draft.instructionSections);
                                        }}
                                        onKeyDown={handleEscape}
                                      />
                                    ) : (
                                      <button
                                        type="button"
                                        className="min-w-0 flex-1 rounded-field pt-0.5 text-left leading-relaxed transition-colors hover:bg-base-200"
                                        onClick={() =>
                                          begin({
                                            kind: "step",
                                            sectionIndex,
                                            itemIndex,
                                          })
                                        }
                                      >
                                        {step.body}
                                      </button>
                                    )}
                                    <DeleteButton
                                      label={`step ${itemIndex + 1}`}
                                      pending={mutation.isPending}
                                      visible={editingStep}
                                      onDelete={() => {
                                        const next = draft.instructionSections
                                          .map((value, index) =>
                                            index === sectionIndex
                                              ? {
                                                  ...value,
                                                  steps: value.steps.filter(
                                                    (_, stepIndex) => stepIndex !== itemIndex,
                                                  ),
                                                }
                                              : value,
                                          )
                                          .filter(
                                            (value) => value.title.trim() || value.steps.length > 0,
                                          );
                                        persistInstructions(next, {
                                          rollbackOnError: true,
                                        });
                                      }}
                                    />
                                  </li>
                                )}
                              </SortableItem>
                            );
                          })}
                        </ol>
                      </SortableContext>
                    </RecipeSectionShell>
                  );
                })}
              </div>
            </SortableContext>
          </SortableRoot>
        )}
        <AddSectionButton onClick={addInstructionSection} disabled={controlsDisabled} />
      </section>

      {errorMessage && <ErrorAlert message={errorMessage} />}
    </div>
  );
}

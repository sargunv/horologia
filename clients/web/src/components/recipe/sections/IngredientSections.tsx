import { SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import {
  AddRow,
  AddSectionButton,
  DeleteButton,
  focusStayedInside,
  SectionBody,
  SectionHeader,
  SortableRoot,
  SortableRow,
  SortableSection,
} from "./RecipeSectionPrimitives.tsx";
import type { IngredientSectionsController } from "./useRecipeSectionsEditor.ts";

export function IngredientSections({ editor }: { editor: IngredientSectionsController }) {
  const { sections, editing, pending, controlsDisabled } = editor;
  return (
    <section className="space-y-3">
      <h2 className="text-lg font-semibold">Ingredients</h2>

      {sections.length === 0 ? (
        <AddRow
          label="Add the first ingredient"
          onClick={() => editor.addIngredient()}
          disabled={controlsDisabled}
        />
      ) : (
        <SortableRoot onDragEnd={editor.handleDragEnd}>
          <SortableContext
            items={sections.map((section) => section.key)}
            strategy={verticalListSortingStrategy}
          >
            <div className="space-y-5">
              {sections.map((section) => {
                const editingSection =
                  editing?.kind === "ingredient-section" && editing.sectionKey === section.key;
                return (
                  <SortableSection
                    key={section.key}
                    id={section.key}
                    label={section.title || "ingredient section"}
                    data={{ type: "ingredient-section" }}
                    disabled={controlsDisabled || sections.length < 2 || !section.title.trim()}
                    reserveHandleSpace={sections.length >= 2 && Boolean(section.title.trim())}
                  >
                    {(handle) => (
                      <>
                        <SectionHeader
                          kind="ingredient"
                          title={section.title}
                          editing={editingSection}
                          handle={handle}
                          pending={pending}
                          onTitleChange={(title) => editor.changeSectionTitle(section.key, title)}
                          onSave={editor.saveSections}
                          onCancel={editor.cancel}
                          onBeginEditing={() => editor.beginSection(section.key)}
                          onDelete={() => editor.deleteSection(section.key)}
                        />
                        <SectionBody
                          hasItems={section.ingredients.length > 0}
                          addItemLabel="Add ingredient"
                          controlsDisabled={controlsDisabled}
                          onAddItem={() => editor.addIngredient(section.key)}
                        >
                          <SortableContext
                            items={section.ingredients.map((ingredient) => ingredient.key)}
                            strategy={verticalListSortingStrategy}
                          >
                            <ul className="divide-y divide-base-300">
                              {section.ingredients.map((ingredient) => {
                                const editingIngredient =
                                  editing?.kind === "ingredient" &&
                                  editing.sectionKey === section.key &&
                                  editing.itemKey === ingredient.key;
                                return (
                                  <SortableRow
                                    key={ingredient.key}
                                    id={ingredient.key}
                                    label={ingredient.item || "ingredient"}
                                    data={{ type: "ingredient", sectionKey: section.key }}
                                    disabled={controlsDisabled}
                                    className="items-center py-2"
                                  >
                                    {(rowHandle) => (
                                      <>
                                        {rowHandle}
                                        {editingIngredient ? (
                                          <div
                                            data-inline-editor
                                            className="grid min-w-0 flex-1 grid-cols-[4rem_minmax(0,1fr)] items-center gap-3"
                                            onBlur={(event) => {
                                              if (!focusStayedInside(event)) {
                                                editor.finishIngredient(
                                                  section.key,
                                                  ingredient.key,
                                                );
                                              }
                                            }}
                                            onKeyDown={editor.handleEscape}
                                          >
                                            <input
                                              className="min-w-0 border-b border-base-content/20 bg-transparent py-1 text-base leading-6 tabular-nums outline-none placeholder:text-base-content/40 focus:border-primary"
                                              aria-label="Quantity"
                                              placeholder="1 cup"
                                              value={ingredient.quantity}
                                              maxLength={120}
                                              onChange={(event) =>
                                                editor.changeIngredient(
                                                  section.key,
                                                  ingredient.key,
                                                  { quantity: event.target.value },
                                                )
                                              }
                                              onKeyDown={(event) => {
                                                if (event.key === "Enter") {
                                                  event.currentTarget.blur();
                                                }
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
                                                editor.changeIngredient(
                                                  section.key,
                                                  ingredient.key,
                                                  { item: event.target.value },
                                                )
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
                                              editor.beginIngredient(section.key, ingredient.key)
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
                                          pending={pending}
                                          visible={editingIngredient}
                                          onDelete={() =>
                                            editor.deleteIngredient(section.key, ingredient.key)
                                          }
                                        />
                                      </>
                                    )}
                                  </SortableRow>
                                );
                              })}
                            </ul>
                          </SortableContext>
                        </SectionBody>
                      </>
                    )}
                  </SortableSection>
                );
              })}
            </div>
          </SortableContext>
        </SortableRoot>
      )}
      <AddSectionButton onClick={editor.addSection} disabled={controlsDisabled} />
    </section>
  );
}

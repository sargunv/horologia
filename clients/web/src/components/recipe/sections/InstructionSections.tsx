import { SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import {
  AddRow,
  AddSectionButton,
  DeleteButton,
  SectionBody,
  SectionHeader,
  SortableRoot,
  SortableRow,
  SortableSection,
} from "./RecipeSectionPrimitives.tsx";
import type { InstructionSectionsController } from "./useRecipeSectionsEditor.ts";

export function InstructionSections({ editor }: { editor: InstructionSectionsController }) {
  const { sections, editing, pending, controlsDisabled } = editor;
  return (
    <section className="space-y-3">
      <h2 className="text-lg font-semibold">Instructions</h2>

      {sections.length === 0 ? (
        <AddRow
          label="Add the first instruction"
          onClick={() => editor.addStep()}
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
                  editing?.kind === "instruction-section" && editing.sectionKey === section.key;
                return (
                  <SortableSection
                    key={section.key}
                    id={section.key}
                    label={section.title || "instruction section"}
                    data={{ type: "instruction-section" }}
                    disabled={controlsDisabled || sections.length < 2 || !section.title.trim()}
                    reserveHandleSpace={sections.length >= 2 && Boolean(section.title.trim())}
                  >
                    {(handle) => (
                      <>
                        <SectionHeader
                          kind="instruction"
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
                          hasItems={section.steps.length > 0}
                          addItemLabel="Add step"
                          controlsDisabled={controlsDisabled}
                          onAddItem={() => editor.addStep(section.key)}
                        >
                          <SortableContext
                            items={section.steps.map((step) => step.key)}
                            strategy={verticalListSortingStrategy}
                          >
                            <ol className="divide-y divide-base-300">
                              {section.steps.map((step, itemIndex) => {
                                const editingStep =
                                  editing?.kind === "step" &&
                                  editing.sectionKey === section.key &&
                                  editing.itemKey === step.key;
                                return (
                                  <SortableRow
                                    key={step.key}
                                    id={step.key}
                                    label={`step ${itemIndex + 1}`}
                                    data={{ type: "step", sectionKey: section.key }}
                                    disabled={controlsDisabled}
                                    className="items-start py-2.5"
                                  >
                                    {(rowHandle) => (
                                      <>
                                        {rowHandle}
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
                                              editor.changeStep(
                                                section.key,
                                                step.key,
                                                event.target.value,
                                              )
                                            }
                                            onBlur={() => editor.finishStep(section.key, step.key)}
                                            onKeyDown={editor.handleEscape}
                                          />
                                        ) : (
                                          <button
                                            type="button"
                                            className="min-w-0 flex-1 rounded-field pt-0.5 text-left leading-relaxed transition-colors hover:bg-base-200"
                                            onClick={() => editor.beginStep(section.key, step.key)}
                                          >
                                            {step.body}
                                          </button>
                                        )}
                                        <DeleteButton
                                          label={`step ${itemIndex + 1}`}
                                          pending={pending}
                                          visible={editingStep}
                                          onDelete={() => editor.deleteStep(section.key, step.key)}
                                        />
                                      </>
                                    )}
                                  </SortableRow>
                                );
                              })}
                            </ol>
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

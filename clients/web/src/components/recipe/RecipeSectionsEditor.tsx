import type { components } from "../../api/schema.d.ts";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";
import { IngredientSections } from "./sections/IngredientSections.tsx";
import { InstructionSections } from "./sections/InstructionSections.tsx";
import { useRecipeSectionsEditor } from "./sections/useRecipeSectionsEditor.ts";

type Recipe = components["schemas"]["Recipe"];

export function RecipeSectionsEditor({ recipe }: { recipe: Recipe }) {
  const editor = useRecipeSectionsEditor(recipe);
  return (
    <div className="space-y-8">
      <IngredientSections editor={editor.ingredients} />
      <InstructionSections editor={editor.instructions} />
      {editor.errorMessage && <ErrorAlert message={editor.errorMessage} />}
    </div>
  );
}

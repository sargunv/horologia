import { useRecipePatch } from "../../lib/mutations.ts";
import { QueuedMarkdownEditor } from "../QueuedMarkdownEditor.tsx";

export function RecipeDescriptionEditor({
  spaceSlug,
  recipeId,
  value,
}: {
  spaceSlug: string;
  recipeId: string;
  value: string;
}) {
  const mutation = useRecipePatch(spaceSlug, recipeId);
  return (
    <QueuedMarkdownEditor
      identity={`${spaceSlug}\u0000${recipeId}`}
      value={value}
      save={async (description) => {
        const recipe = await mutation.mutateAsync({ description });
        return recipe?.description ?? description;
      }}
      resetError={mutation.reset}
      error={mutation.error}
    />
  );
}

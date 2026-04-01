import { ErrorAlert } from "./space-settings/ErrorAlert.tsx";
import { MarkdownEditor } from "./MarkdownEditor.tsx";
import { useTaskPatch } from "../lib/mutations.ts";

interface TaskDescriptionEditorProps {
  spaceSlug: string;
  taskId: string;
  value: string;
}

export function TaskDescriptionEditor({ spaceSlug, taskId, value }: TaskDescriptionEditorProps) {
  const mutation = useTaskPatch(spaceSlug, taskId);

  return (
    <div className="mt-4">
      <MarkdownEditor
        value={value}
        onChange={(markdown) => {
          mutation.reset();
          mutation.mutate({ description: markdown });
        }}
        disabled={mutation.isPending}
      />
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </div>
  );
}

import { useTaskPatch } from "../lib/mutations.ts";
import { QueuedMarkdownEditor } from "./QueuedMarkdownEditor.tsx";

interface TaskDescriptionEditorProps {
  spaceSlug: string;
  taskId: string;
  value: string;
}

export function TaskDescriptionEditor({ spaceSlug, taskId, value }: TaskDescriptionEditorProps) {
  const mutation = useTaskPatch(spaceSlug, taskId);
  return (
    <QueuedMarkdownEditor
      identity={`${spaceSlug}\u0000${taskId}`}
      value={value}
      save={async (description) => {
        const task = await mutation.mutateAsync({ description });
        return task?.description ?? description;
      }}
      resetError={mutation.reset}
      error={mutation.error}
    />
  );
}

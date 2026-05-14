import { useEffect, useReducer, useRef } from "react";
import { ErrorAlert } from "./space-settings/ErrorAlert.tsx";
import { MarkdownEditor } from "./MarkdownEditor.tsx";
import { QueuedAutosave } from "../lib/queuedAutosave.ts";
import { useTaskPatch } from "../lib/mutations.ts";

interface TaskDescriptionEditorProps {
  spaceSlug: string;
  taskId: string;
  value: string;
}

export function TaskDescriptionEditor({ spaceSlug, taskId, value }: TaskDescriptionEditorProps) {
  const mutation = useTaskPatch(spaceSlug, taskId);
  const mountedRef = useRef(false);
  const [, forceUpdate] = useReducer((x: number) => x + 1, 0);
  const controllerKey = `${spaceSlug}\u0000${taskId}`;
  const controllerRef = useRef<
    | {
        key: string;
        mutationRef: { current: typeof mutation };
        controller: QueuedAutosave<string>;
      }
    | undefined
  >(undefined);
  if (controllerRef.current?.key !== controllerKey) {
    const mutationRef = { current: mutation };
    controllerRef.current = {
      key: controllerKey,
      mutationRef,
      controller: new QueuedAutosave(value, async (description) => {
        const task = await mutationRef.current.mutateAsync({ description });
        return task?.description ?? description;
      }),
    };
  } else {
    controllerRef.current.mutationRef.current = mutation;
  }
  const controller = controllerRef.current.controller;

  useEffect(() => {
    mountedRef.current = true;
    controller.setChangeListener(() => {
      if (mountedRef.current) forceUpdate();
    });

    return () => {
      mountedRef.current = false;
      controller.setChangeListener(undefined);
      controller.requestSave(controller.localValue);
    };
  }, [controller]);

  useEffect(() => {
    controller.receiveExternalValue(value);
  }, [controller, value]);

  return (
    <div className="mt-4">
      <MarkdownEditor
        value={value}
        onLocalChange={(markdown) => {
          if (mountedRef.current) mutation.reset();
          controller.setLocalValue(markdown);
        }}
        onChange={(markdown) => {
          if (mountedRef.current) mutation.reset();
          controller.requestSave(markdown);
        }}
        syncExternalValue={controller.canSyncExternalValue(value)}
      />
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </div>
  );
}

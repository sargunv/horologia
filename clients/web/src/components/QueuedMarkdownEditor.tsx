import { useEffect, useReducer, useRef } from "react";
import { QueuedAutosave } from "../lib/queuedAutosave.ts";
import { MarkdownEditor } from "./MarkdownEditor.tsx";
import { ErrorAlert } from "./space-settings/ErrorAlert.tsx";

export function QueuedMarkdownEditor({
  identity,
  value,
  save,
  resetError,
  error,
}: {
  identity: string;
  value: string;
  save: (value: string) => Promise<string | undefined>;
  resetError: () => void;
  error: Error | null;
}) {
  const mountedRef = useRef(false);
  const saveRef = useRef(save);
  saveRef.current = save;
  const [, forceUpdate] = useReducer((current: number) => current + 1, 0);
  const controllerRef = useRef<
    { identity: string; controller: QueuedAutosave<string> } | undefined
  >(undefined);

  if (controllerRef.current?.identity !== identity) {
    controllerRef.current = {
      identity,
      controller: new QueuedAutosave(value, (nextValue) => saveRef.current(nextValue)),
    };
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
        onLocalChange={(nextValue) => {
          resetError();
          controller.setLocalValue(nextValue);
        }}
        onChange={(nextValue) => {
          resetError();
          controller.requestSave(nextValue);
        }}
        syncExternalValue={controller.canSyncExternalValue(value)}
      />
      {error && <ErrorAlert message={error.message} />}
    </div>
  );
}

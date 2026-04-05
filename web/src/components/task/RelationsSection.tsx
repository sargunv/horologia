import { Link } from "@tanstack/react-router";
import { Link2, X } from "lucide-react";
import { useState } from "react";
import type { components } from "../../api/schema.d.ts";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";
import { useAddRelation, useDeleteRelation } from "../../lib/mutations.ts";

type TaskRelation = components["schemas"]["TaskRelation"];
type TaskRelationKind = components["schemas"]["TaskRelationKind"];

const KIND_LABELS: Record<TaskRelationKind, string> = {
  parent_of: "Parent of",
  child_of: "Child of",
  blocks: "Blocks",
  blocked_by: "Blocked by",
  relates_to: "Relates to",
  duplicates: "Duplicates",
  triggers: "Triggers",
  triggered_by: "Triggered by",
  spawns: "Spawns",
  spawned_by: "Spawned by",
};

const USER_MANAGEABLE_KINDS: TaskRelationKind[] = [
  "parent_of",
  "child_of",
  "blocks",
  "blocked_by",
  "relates_to",
  "duplicates",
];

function RelationItem({
  spaceSlug,
  currentTaskId,
  relation,
}: {
  spaceSlug: string;
  currentTaskId: string;
  relation: TaskRelation;
}) {
  const deleteMutation = useDeleteRelation(spaceSlug, currentTaskId);
  const isSystemManaged = !USER_MANAGEABLE_KINDS.includes(relation.kind);

  return (
    <div className="py-2">
      <div className="flex items-center gap-2">
        <span
          className={`chip text-xs ${isSystemManaged ? "preset-tonal-warning" : "preset-tonal-surface"}`}
        >
          {KIND_LABELS[relation.kind]}
        </span>
        <Link
          to="/spaces/$spaceSlug/tasks/$taskId"
          params={{ spaceSlug, taskId: relation.taskId }}
          className="font-mono text-sm hover:underline"
        >
          {relation.taskId}
        </Link>
        {!isSystemManaged && (
          <button
            type="button"
            className="btn btn-sm preset-outlined-surface-200-800 ml-auto"
            aria-label={`Remove ${KIND_LABELS[relation.kind]} relation to ${relation.taskId}`}
            disabled={deleteMutation.isPending}
            onClick={() => {
              deleteMutation.reset();
              deleteMutation.mutate({ kind: relation.kind, relatedTaskId: relation.taskId });
            }}
          >
            <X className="size-3.5" aria-hidden="true" />
          </button>
        )}
      </div>
      {deleteMutation.error && <ErrorAlert message={deleteMutation.error.message} />}
    </div>
  );
}

function AddRelationForm({
  spaceSlug,
  taskId,
}: {
  spaceSlug: string;
  taskId: string;
}) {
  const [taskIdInput, setTaskIdInput] = useState("");
  const [kindInput, setKindInput] = useState<TaskRelationKind>("relates_to");
  const [validationError, setValidationError] = useState<string | null>(null);
  const addMutation = useAddRelation(spaceSlug, taskId);

  function handleKindChange(e: React.ChangeEvent<HTMLSelectElement>) {
    const kind = USER_MANAGEABLE_KINDS.find((k) => k === e.target.value);
    if (kind !== undefined) setKindInput(kind);
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = taskIdInput.trim();
    if (!/^T\d+$/.test(trimmed)) {
      setValidationError("Task ID must be in the format T42");
      return;
    }
    if (trimmed === taskId) {
      setValidationError("Cannot relate a task to itself");
      return;
    }
    addMutation.reset();
    addMutation.mutate(
      { kind: kindInput, taskId: trimmed },
      {
        onSuccess: () => {
          setTaskIdInput("");
          setValidationError(null);
        },
      },
    );
  }

  return (
    <>
      <form onSubmit={handleSubmit} className="flex flex-wrap gap-2 mt-3">
        <select
          value={kindInput}
          onChange={handleKindChange}
          className="select preset-outlined-surface-200-800 text-sm"
        >
          {USER_MANAGEABLE_KINDS.map((k) => (
            <option key={k} value={k}>
              {KIND_LABELS[k]}
            </option>
          ))}
        </select>
        <input
          type="text"
          placeholder="T42"
          value={taskIdInput}
          onChange={(e) => {
            setTaskIdInput(e.target.value);
            setValidationError(null);
          }}
          className="input preset-outlined-surface-200-800 w-24 font-mono text-sm"
          aria-label="Related task ID"
        />
        <button
          type="submit"
          disabled={!taskIdInput.trim() || addMutation.isPending}
          className="btn preset-tonal-surface text-sm"
        >
          Add
        </button>
      </form>
      {validationError && <ErrorAlert message={validationError} />}
      {addMutation.error && <ErrorAlert message={addMutation.error.message} />}
    </>
  );
}

export function RelationsSection({
  spaceSlug,
  taskId,
  relations,
}: {
  spaceSlug: string;
  taskId: string;
  relations: TaskRelation[];
}) {
  return (
    <div className="card preset-outlined-surface-200-800 mt-6 p-4">
      <h2 className="text-sm font-medium text-surface-600-400 mb-3 flex items-center gap-2">
        <Link2 className="size-4" aria-hidden="true" /> Relations
      </h2>
      {relations.length > 0 && (
        <div className="divide-y divide-surface-200-800 mb-2">
          {relations.map((rel) => (
            <RelationItem
              key={`${rel.kind}:${rel.taskId}`}
              spaceSlug={spaceSlug}
              currentTaskId={taskId}
              relation={rel}
            />
          ))}
        </div>
      )}
      <AddRelationForm spaceSlug={spaceSlug} taskId={taskId} />
    </div>
  );
}

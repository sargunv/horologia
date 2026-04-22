import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { ChevronRight, Link2, X } from "lucide-react";
import { type ReactNode, useDeferredValue, useMemo } from "react";
import { apiClient } from "../../api/client.ts";
import type { components } from "../../api/schema.d.ts";
import { useAddRelation, useDeleteRelation } from "../../lib/mutations.ts";
import { taskSearchQueryOptions } from "../../lib/queries.ts";
import { useMenuSearch } from "../../lib/useMenuSearch.ts";
import {
  DropdownMenuItem,
  DropdownMenuRoot,
  DropdownMenuSub,
  DropdownMenuSubTrigger,
} from "../../ui/DropdownMenu.tsx";
import { FieldPill } from "../FieldPill.tsx";
import { SearchableMenuContent, SearchableSubMenuContent } from "../SearchableMenuContent.tsx";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";

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

function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}

function TaskPickerSubMenu({
  spaceSlug,
  currentTaskId,
  onSelect,
  triggerContent,
}: {
  spaceSlug: string;
  currentTaskId: string;
  onSelect: (relatedTaskId: string) => void;
  triggerContent: ReactNode;
}) {
  const search = useMenuSearch();
  const deferredQuery = useDeferredValue(search.query.trim());
  const {
    data: searchResults = [],
    isFetching: isSearchFetching,
    error: searchError,
  } = useQuery({
    ...taskSearchQueryOptions({
      query: deferredQuery,
      spaceSlug,
      excludeTaskId: currentTaskId,
      limit: 10,
    }),
    enabled: deferredQuery.length > 0,
  });
  const {
    data: initialResults = [],
    isFetching: isInitialFetching,
    error: initialError,
  } = useQuery({
    queryKey: ["spaces", spaceSlug, "tasks", "relation-picker", currentTaskId],
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/spaces/{spaceSlug}/tasks", {
        params: {
          path: { spaceSlug },
          query: { limit: 10 },
        },
      });
      if (error) throw error;
      return data.items
        .filter((task) => task.id !== currentTaskId)
        .map((task) => ({ id: task.id, title: task.title, status: task.status }));
    },
    enabled: deferredQuery.length === 0,
    staleTime: 10_000,
  });

  const results = deferredQuery.length === 0 ? initialResults : searchResults;
  const isFetching = deferredQuery.length === 0 ? isInitialFetching : isSearchFetching;
  const error = deferredQuery.length === 0 ? initialError : searchError;

  return (
    <DropdownMenuSub {...search.menuProps}>
      <DropdownMenuSubTrigger>{triggerContent}</DropdownMenuSubTrigger>
      <SearchableSubMenuContent
        search={search}
        placeholder="Search tasks..."
        inputLabel="Search tasks"
      >
        {isFetching ? (
          <div className="flex items-center gap-2 px-3 py-2 text-sm text-base-content/60">
            <span className="loading loading-spinner loading-xs" />
            {deferredQuery.length === 0 ? "Loading tasks…" : "Searching tasks…"}
          </div>
        ) : error ? (
          <div className="px-3 py-2 text-sm text-error">
            {getErrorMessage(error, "Task search failed")}
          </div>
        ) : results.length === 0 ? (
          <div className="px-3 py-2 text-sm text-base-content/60">No matching tasks</div>
        ) : (
          results.map((result) => (
            <DropdownMenuItem
              key={result.id}
              className="flex flex-col items-start gap-1 px-3 py-2 text-left"
              onSelect={() => onSelect(result.id)}
            >
              <div className="flex w-full items-center gap-2">
                <span>{result.title}</span>
                <span className="ml-auto text-xs text-base-content/60">{result.status}</span>
              </div>
              <div className="font-mono text-xs text-base-content/60">{result.id}</div>
            </DropdownMenuItem>
          ))
        )}
      </SearchableSubMenuContent>
    </DropdownMenuSub>
  );
}

export function TaskRelationMenuField({
  spaceSlug,
  taskId,
  relations,
}: {
  spaceSlug: string;
  taskId: string;
  relations: TaskRelation[];
}) {
  const addMutation = useAddRelation(spaceSlug, taskId);
  const search = useMenuSearch();

  const kindItems = useMemo(() => {
    if (!search.query) return USER_MANAGEABLE_KINDS;
    return USER_MANAGEABLE_KINDS.filter((kind) =>
      KIND_LABELS[kind].toLowerCase().includes(search.query.toLowerCase()),
    );
  }, [search.query]);

  const displayValue =
    relations.length === 0
      ? null
      : `${relations.length} relation${relations.length === 1 ? "" : "s"}`;

  return (
    <div className="flex flex-col gap-1">
      <DropdownMenuRoot {...search.menuProps}>
        <FieldPill
          icon={<Link2 className="size-3.5" aria-hidden="true" />}
          label="Relations"
          value={displayValue}
        />
        <SearchableMenuContent
          search={search}
          placeholder="Search relation types..."
          inputLabel="Search relation types"
        >
          {kindItems.length === 0 ? (
            <div className="px-3 py-2 text-sm text-base-content/60">No matching relation types</div>
          ) : (
            kindItems.map((kind) => (
              <TaskPickerSubMenu
                key={kind}
                spaceSlug={spaceSlug}
                currentTaskId={taskId}
                onSelect={(relatedTaskId) => {
                  addMutation.reset();
                  addMutation.mutate({ kind, relatedTaskId });
                }}
                triggerContent={
                  <>
                    <span className="size-4" />
                    <span>{KIND_LABELS[kind]}</span>
                    <ChevronRight className="ml-auto size-4" aria-hidden="true" />
                  </>
                }
              />
            ))
          )}
        </SearchableMenuContent>
      </DropdownMenuRoot>
      {addMutation.error && <ErrorAlert message={addMutation.error.message} />}
    </div>
  );
}

function RelationChip({
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
    <div className="flex flex-col gap-1">
      <div
        className={`badge gap-1.5 pr-1 ${
          isSystemManaged ? "badge-warning badge-soft" : "badge-soft"
        }`}
      >
        <span className="text-xs opacity-70">{KIND_LABELS[relation.kind]}</span>
        <Link
          to="/spaces/$spaceSlug/tasks/$taskId"
          params={{ spaceSlug, taskId: relation.relatedTaskId }}
          className="font-mono hover:underline"
        >
          {relation.relatedTaskId}
        </Link>
        {!isSystemManaged && (
          <button
            type="button"
            className="rounded-full p-0.5 opacity-60 transition-opacity hover:opacity-100 hover:bg-base-content/10"
            aria-label={`Remove ${KIND_LABELS[relation.kind]} relation to ${relation.relatedTaskId}`}
            disabled={deleteMutation.isPending}
            onClick={() => {
              deleteMutation.reset();
              deleteMutation.mutate({
                kind: relation.kind,
                relatedTaskId: relation.relatedTaskId,
              });
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

export function TaskRelationChipRow({
  spaceSlug,
  taskId,
  relations,
}: {
  spaceSlug: string;
  taskId: string;
  relations: TaskRelation[];
}) {
  if (relations.length === 0) return null;

  return (
    <div className="flex flex-wrap gap-2">
      {relations.map((relation) => (
        <RelationChip
          key={`${relation.kind}:${relation.relatedTaskId}`}
          spaceSlug={spaceSlug}
          currentTaskId={taskId}
          relation={relation}
        />
      ))}
    </div>
  );
}

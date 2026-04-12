import { Menu, Portal } from "@skeletonlabs/skeleton-react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { ChevronRight, Link2, LoaderCircle, X } from "lucide-react";
import { useDeferredValue, useMemo } from "react";
import { apiClient } from "../../api/client.ts";
import type { components } from "../../api/schema.d.ts";
import { useAddRelation, useDeleteRelation } from "../../lib/mutations.ts";
import { taskSearchQueryOptions } from "../../lib/queries.ts";
import { useMenuSearch } from "../../lib/useMenuSearch.ts";
import { FieldPill } from "../FieldPill.tsx";
import { SearchableMenuContent } from "../SearchableMenuContent.tsx";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";

type TaskRelation = components["schemas"]["TaskRelation"];
type TaskRelationKind = components["schemas"]["TaskRelationKind"];

const Z_SUBMENU = "z-10";

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
}: {
  spaceSlug: string;
  currentTaskId: string;
  onSelect: (relatedTaskId: string) => void;
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
    <SearchableMenuContent
      inputProps={search.inputProps}
      placeholder={`Search tasks...`}
      className={Z_SUBMENU}
    >
      {isFetching ? (
        <div className="flex items-center gap-2 px-3 py-2 text-sm text-surface-500">
          <LoaderCircle className="size-4 animate-spin" aria-hidden="true" />
          {deferredQuery.length === 0 ? "Loading tasks…" : "Searching tasks…"}
        </div>
      ) : error ? (
        <div className="px-3 py-2 text-sm text-error-500">
          {getErrorMessage(error, "Task search failed")}
        </div>
      ) : results.length === 0 ? (
        <div className="px-3 py-2 text-sm text-surface-500">No matching tasks</div>
      ) : (
        results.map((result) => (
          <Menu.Item
            key={result.id}
            value={`task-${result.id}`}
            className="flex flex-col items-start gap-1 px-3 py-2 text-left"
            onClick={() => onSelect(result.id)}
          >
            <div className="flex w-full items-center gap-2">
              <Menu.ItemText>{result.title}</Menu.ItemText>
              <span className="ml-auto text-xs text-surface-500">{result.status}</span>
            </div>
            <div className="font-mono text-xs text-surface-500">{result.id}</div>
          </Menu.Item>
        ))
      )}
    </SearchableMenuContent>
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
      <Menu {...search.menuProps} closeOnSelect={false}>
        <FieldPill
          icon={<Link2 className="size-3.5" aria-hidden="true" />}
          label="Relations"
          value={displayValue}
        />
        <Portal>
          <Menu.Positioner>
            <SearchableMenuContent
              inputProps={search.inputProps}
              placeholder="Search relation types..."
            >
              {kindItems.length === 0 ? (
                <div className="px-3 py-2 text-sm text-surface-500">No matching relation types</div>
              ) : (
                kindItems.map((kind) => (
                  <Menu key={kind} typeahead={false}>
                    <Menu.TriggerItem value={kind} className="justify-start gap-2 text-sm">
                      <span className="size-4" />
                      <Menu.ItemText>{KIND_LABELS[kind]}</Menu.ItemText>
                      <Menu.ItemIndicator className="ml-auto">
                        <ChevronRight className="size-4" />
                      </Menu.ItemIndicator>
                    </Menu.TriggerItem>
                    <Portal>
                      <Menu.Positioner>
                        <TaskPickerSubMenu
                          spaceSlug={spaceSlug}
                          currentTaskId={taskId}
                          onSelect={(relatedTaskId) => {
                            addMutation.reset();
                            addMutation.mutate({ kind, relatedTaskId });
                          }}
                        />
                      </Menu.Positioner>
                    </Portal>
                  </Menu>
                ))
              )}
            </SearchableMenuContent>
          </Menu.Positioner>
        </Portal>
      </Menu>
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
        className={`chip gap-1.5 pr-1 text-sm ${
          isSystemManaged ? "preset-tonal-warning" : "preset-tonal-surface"
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
            className="rounded-base p-0.5 opacity-60 transition-opacity hover:opacity-100"
            aria-label={`Remove ${KIND_LABELS[relation.kind]} relation to ${relation.relatedTaskId}`}
            disabled={deleteMutation.isPending}
            onClick={() => {
              deleteMutation.reset();
              deleteMutation.mutate({ kind: relation.kind, relatedTaskId: relation.relatedTaskId });
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

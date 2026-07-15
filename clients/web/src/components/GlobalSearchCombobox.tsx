import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { CookingPot, ListChecks, Search } from "lucide-react";
import {
  type KeyboardEvent,
  useDeferredValue,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
} from "react";
import { recipeSearchQueryOptions, taskSearchQueryOptions } from "../lib/queries.ts";

interface SearchResult {
  kind: "task" | "recipe";
  id: string;
  spaceSlug: string;
  title: string;
  meta: string;
}

function errorMessage(error: unknown): string {
  if (error instanceof Error && error.message) return error.message;
  return "Search failed";
}

export function GlobalSearchCombobox({ page = false }: { page?: boolean }) {
  const navigate = useNavigate();
  const listboxId = useId();
  const wrapperRef = useRef<HTMLDivElement>(null);
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(page);
  const [activeIndex, setActiveIndex] = useState(-1);
  const deferredQuery = useDeferredValue(query.trim());
  const enabled = (page || open) && deferredQuery.length > 0;
  const taskQuery = useQuery({
    ...taskSearchQueryOptions({ query: deferredQuery, limit: 6 }),
    enabled,
  });
  const recipeQuery = useQuery({
    ...recipeSearchQueryOptions({ query: deferredQuery, limit: 6 }),
    enabled,
  });
  const results = useMemo<SearchResult[]>(
    () => [
      ...(taskQuery.data ?? []).map((task) => ({
        kind: "task" as const,
        id: task.id,
        spaceSlug: task.spaceSlug,
        title: task.title,
        meta: task.status,
      })),
      ...(recipeQuery.data ?? []).map((recipe) => ({
        kind: "recipe" as const,
        id: recipe.id,
        spaceSlug: recipe.spaceSlug,
        title: recipe.name,
        meta: recipe.tags.slice(0, 2).join(" · ") || "Recipe",
      })),
    ],
    [recipeQuery.data, taskQuery.data],
  );
  const isFetching = taskQuery.isFetching || recipeQuery.isFetching;
  const error = taskQuery.error ?? recipeQuery.error;

  useEffect(() => {
    if (page) return;
    function handlePointerDown(event: MouseEvent) {
      if (!(event.target instanceof Node)) return;
      if (!wrapperRef.current?.contains(event.target)) {
        setOpen(false);
        setActiveIndex(-1);
      }
    }
    document.addEventListener("mousedown", handlePointerDown);
    return () => document.removeEventListener("mousedown", handlePointerDown);
  }, [page]);

  useEffect(() => {
    if (!open && !page) return;
    setActiveIndex(results.length > 0 ? 0 : -1);
  }, [open, page, results]);

  function selectResult(result: SearchResult) {
    setQuery("");
    setOpen(page);
    setActiveIndex(-1);
    if (result.kind === "task") {
      void navigate({
        to: "/tasks/$spaceSlug/$taskId",
        params: { spaceSlug: result.spaceSlug, taskId: result.id },
      });
    } else {
      void navigate({
        to: "/recipes/$spaceSlug/$recipeId",
        params: { spaceSlug: result.spaceSlug, recipeId: result.id },
      });
    }
  }

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (!open && (event.key === "ArrowDown" || event.key === "Enter")) setOpen(true);
    if (event.key === "Escape") {
      setOpen(false);
      setActiveIndex(-1);
      return;
    }
    if (results.length === 0) return;
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setActiveIndex((current) => (current + 1) % results.length);
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setActiveIndex((current) => (current <= 0 ? results.length - 1 : current - 1));
    } else if (event.key === "Enter" && activeIndex >= 0) {
      event.preventDefault();
      const result = results[activeIndex];
      if (result) selectResult(result);
    }
  }

  const showPanel = page || (open && (query.length > 0 || isFetching || !!error));
  return (
    <div ref={wrapperRef} className={page ? "space-y-3" : "relative"}>
      <label className={`input w-full ${page ? "" : "input-sm"}`}>
        <Search className="size-4 opacity-60" aria-hidden="true" />
        <input
          type="search"
          role="combobox"
          aria-label="Search everything"
          aria-expanded={showPanel}
          aria-controls={listboxId}
          aria-autocomplete="list"
          aria-activedescendant={
            activeIndex >= 0 && results[activeIndex]
              ? `${listboxId}-${results[activeIndex].kind}-${results[activeIndex].spaceSlug}-${results[activeIndex].id}`
              : undefined
          }
          value={query}
          onFocus={() => setOpen(true)}
          onChange={(event) => {
            setQuery(event.target.value);
            setOpen(true);
          }}
          onKeyDown={handleKeyDown}
          placeholder="Search everything…"
          autoFocus={page}
        />
      </label>
      {showPanel && (
        <div
          id={listboxId}
          role="listbox"
          className={
            page
              ? "overflow-hidden rounded-box border border-base-300 bg-base-100"
              : "absolute inset-x-0 top-[calc(100%+0.5rem)] z-50 max-h-96 overflow-y-auto rounded-box border border-base-300 bg-base-100 p-1 shadow-lg"
          }
        >
          {deferredQuery.length === 0 ? (
            <div className="px-3 py-8 text-center text-sm text-base-content/60">
              Search this space
            </div>
          ) : isFetching ? (
            <div className="flex items-center gap-2 px-3 py-3 text-sm text-base-content/60">
              <span className="loading loading-spinner loading-xs" />
              Searching…
            </div>
          ) : error ? (
            <div className="px-3 py-3 text-sm text-error">{errorMessage(error)}</div>
          ) : results.length === 0 ? (
            <div className="px-3 py-8 text-center text-sm text-base-content/60">No matches</div>
          ) : (
            results.map((result, index) => {
              const Icon = result.kind === "task" ? ListChecks : CookingPot;
              return (
                <button
                  key={`${result.kind}/${result.spaceSlug}/${result.id}`}
                  id={`${listboxId}-${result.kind}-${result.spaceSlug}-${result.id}`}
                  type="button"
                  role="option"
                  aria-selected={activeIndex === index}
                  className={`flex w-full items-start gap-2 border-b border-base-300 px-3 py-2 text-left transition-colors last:border-b-0 ${
                    activeIndex === index ? "bg-base-200" : "hover:bg-base-200"
                  }`}
                  onMouseDown={(event) => event.preventDefault()}
                  onMouseEnter={() => setActiveIndex(index)}
                  onClick={() => selectResult(result)}
                >
                  <Icon
                    className="mt-0.5 size-4 shrink-0 text-base-content/60"
                    aria-hidden="true"
                  />
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-sm font-medium">{result.title}</span>
                    <span className="flex items-center gap-2 text-xs text-base-content/60">
                      <span className="capitalize">{result.kind}</span>
                      <span className="font-mono">{result.id}</span>
                      <span>{result.meta}</span>
                    </span>
                  </span>
                </button>
              );
            })
          )}
        </div>
      )}
    </div>
  );
}

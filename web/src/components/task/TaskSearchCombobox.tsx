import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { Search } from "lucide-react";
import { useDeferredValue, useEffect, useId, useRef, useState, type KeyboardEvent } from "react";
import type { components } from "../../api/schema.d.ts";
import { taskSearchQueryOptions } from "../../lib/queries.ts";

type TaskSearchResult = components["schemas"]["TaskSearchResult"];

function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}

export function TaskSearchCombobox() {
  const navigate = useNavigate();
  const listboxId = useId();
  const wrapperRef = useRef<HTMLDivElement>(null);
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(-1);
  const deferredQuery = useDeferredValue(query.trim());

  const {
    data: results = [],
    isFetching,
    error,
  } = useQuery({
    ...taskSearchQueryOptions({ query: deferredQuery, limit: 8 }),
    enabled: open && deferredQuery.length > 0,
  });

  useEffect(() => {
    function handlePointerDown(event: MouseEvent) {
      if (!(event.target instanceof Node)) return;
      if (!wrapperRef.current?.contains(event.target)) {
        setOpen(false);
        setActiveIndex(-1);
      }
    }

    document.addEventListener("mousedown", handlePointerDown);
    return () => document.removeEventListener("mousedown", handlePointerDown);
  }, []);

  useEffect(() => {
    if (!open) return;
    setActiveIndex(results.length > 0 ? 0 : -1);
  }, [open, results]);

  function handleSelect(result: TaskSearchResult) {
    setQuery("");
    setOpen(false);
    setActiveIndex(-1);
    void navigate({
      to: "/spaces/$spaceSlug/tasks/$taskId",
      params: { spaceSlug: result.spaceSlug, taskId: result.id },
    });
  }

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (!open && (event.key === "ArrowDown" || event.key === "Enter")) {
      setOpen(true);
    }

    if (event.key === "Escape") {
      setOpen(false);
      setActiveIndex(-1);
      return;
    }

    if (results.length === 0) return;

    if (event.key === "ArrowDown") {
      event.preventDefault();
      setActiveIndex((current) => (current + 1) % results.length);
      return;
    }

    if (event.key === "ArrowUp") {
      event.preventDefault();
      setActiveIndex((current) => (current <= 0 ? results.length - 1 : current - 1));
      return;
    }

    if (event.key === "Enter" && activeIndex >= 0) {
      event.preventDefault();
      const activeResult = results[activeIndex];
      if (activeResult) handleSelect(activeResult);
    }
  }

  const showPanel = open && (query.length > 0 || isFetching || !!error);

  return (
    <div ref={wrapperRef} className="relative">
      <label className="input input-sm w-full">
        <Search className="size-4 opacity-60" aria-hidden="true" />
        <input
          type="search"
          role="combobox"
          aria-label="Search tasks"
          aria-expanded={showPanel}
          aria-controls={listboxId}
          aria-autocomplete="list"
          aria-activedescendant={
            activeIndex >= 0 && results[activeIndex]
              ? `${listboxId}-${results[activeIndex].spaceSlug}-${results[activeIndex].id}`
              : undefined
          }
          value={query}
          onFocus={() => setOpen(true)}
          onChange={(event) => {
            setQuery(event.target.value);
            setOpen(true);
          }}
          onKeyDown={handleKeyDown}
          placeholder="Search tasks…"
        />
      </label>
      {showPanel && (
        <div
          id={listboxId}
          role="listbox"
          className="absolute inset-x-0 top-[calc(100%+0.5rem)] z-50 max-h-96 overflow-y-auto rounded-box border border-base-300 bg-base-100 p-1 shadow-lg"
        >
          {deferredQuery.length === 0 ? (
            <div className="px-3 py-2 text-sm text-base-content/60">
              Type to search visible tasks
            </div>
          ) : isFetching ? (
            <div className="flex items-center gap-2 px-3 py-2 text-sm text-base-content/60">
              <span className="loading loading-spinner loading-xs" />
              Searching tasks…
            </div>
          ) : error ? (
            <div className="px-3 py-2 text-sm text-error">
              {getErrorMessage(error, "Task search failed")}
            </div>
          ) : results.length === 0 ? (
            <div className="px-3 py-2 text-sm text-base-content/60">No matching tasks</div>
          ) : (
            results.map((result, index) => (
              <button
                key={`${result.spaceSlug}/${result.id}`}
                id={`${listboxId}-${result.spaceSlug}-${result.id}`}
                type="button"
                role="option"
                aria-selected={activeIndex === index}
                className={`flex w-full flex-col items-start gap-0.5 rounded-field px-2 py-1.5 text-left transition-colors ${
                  activeIndex === index ? "bg-base-200" : "hover:bg-base-200"
                }`}
                onMouseDown={(event) => event.preventDefault()}
                onMouseEnter={() => setActiveIndex(index)}
                onClick={() => handleSelect(result)}
              >
                <span className="truncate text-sm font-medium">{result.title}</span>
                <div className="flex items-center gap-2 text-xs text-base-content/60">
                  <span className="font-mono">{result.id}</span>
                  <span>{result.status}</span>
                </div>
              </button>
            ))
          )}
        </div>
      )}
    </div>
  );
}

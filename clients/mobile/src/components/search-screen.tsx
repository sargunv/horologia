import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { Stack, useRouter } from "expo-router";
import { useEffect, useMemo, useState } from "react";

import { createSearchResults, searchResultRoute } from "@/components/search-controller";
import { SearchView } from "@/components/native-views";
import type { SearchResultItem } from "@/components/native-views.types";
import { errorMessage } from "@/components/read-model";
import { useAppRuntime } from "@/runtime/app-runtime";

const SEARCH_DEBOUNCE_MS = 300;

export function GlobalSearchScreen() {
  const runtime = useAppRuntime();
  const router = useRouter();
  const [query, setQuery] = useState("");
  const normalizedQuery = query.trim();
  const [searchQuery, setSearchQuery] = useState(normalizedQuery);
  useEffect(() => {
    if (!normalizedQuery) {
      setSearchQuery("");
      return;
    }
    const timeout = setTimeout(() => setSearchQuery(normalizedQuery), SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(timeout);
  }, [normalizedQuery]);
  const enabled = searchQuery.length > 0;
  const taskQuery = useQuery({
    ...runtime.queries.taskSearchQueryOptions({ query: searchQuery, limit: 12 }),
    enabled,
    placeholderData: keepPreviousData,
  });
  const recipeQuery = useQuery({
    ...runtime.queries.recipeSearchQueryOptions({ query: searchQuery, limit: 12 }),
    enabled,
    placeholderData: keepPreviousData,
  });
  const results = useMemo(
    () =>
      createSearchResults({
        tasks: taskQuery.data ?? [],
        recipes: recipeQuery.data ?? [],
      }),
    [recipeQuery.data, taskQuery.data],
  );
  const error = taskQuery.error ?? recipeQuery.error;

  function open(result: SearchResultItem) {
    router.push(searchResultRoute(runtime.scope, result));
  }

  return (
    <>
      <Stack.SearchBar
        autoFocus
        placement="automatic"
        placeholder="Search tasks and recipes"
        onChangeText={(event) => setQuery(event.nativeEvent.text)}
      />
      <SearchView
        query={query}
        results={normalizedQuery ? results : []}
        isSearching={
          normalizedQuery !== searchQuery ||
          (enabled && (taskQuery.isFetching || recipeQuery.isFetching))
        }
        error={error ? errorMessage(error, "Search failed.") : null}
        onOpen={open}
      />
    </>
  );
}

import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { Stack, useRouter } from "expo-router";
import { useMemo } from "react";

import {
  LibraryHubView,
  RecipeDetailView,
  RecipeListView,
  SpacesView,
  SpaceWorkspaceView,
  TaskListView,
} from "@/components/native-views";
import { errorMessage, formatTimestamp, isStaleTimestamp } from "@/components/read-model";
import type { RecipeSummary, Task } from "@/components/native-views.types";
import { routes } from "@/navigation/routes";
import { useAppRuntime } from "@/runtime/app-runtime";

export function LibraryHubScreen() {
  const runtime = useAppRuntime();
  const router = useRouter();
  const spacesQuery = useQuery(runtime.queries.spacesQueryOptions);
  const recipesQuery = useInfiniteQuery(runtime.queries.recipesInfiniteQueryOptions());
  const recipes = recipesQuery.data?.pages.flatMap((page) => page.items) ?? [];
  const error = spacesQuery.error ?? recipesQuery.error;
  return (
    <LibraryHubView
      spaces={spacesQuery.data ?? []}
      recipePreview={recipes.slice(0, 4)}
      isLoading={spacesQuery.isLoading || recipesQuery.isLoading}
      error={error ? errorMessage(error, "Could not load the library.") : null}
      onOpenSpaces={() => router.push(routes.librarySpaces(runtime.scope))}
      onOpenRecipes={() => router.push(routes.recipes(runtime.scope))}
      onOpenRecipe={(recipe) =>
        router.push(routes.recipeDetail(runtime.scope, recipe.spaceSlug, recipe.id))
      }
      onRetry={() => {
        void Promise.all([spacesQuery.refetch(), recipesQuery.refetch()]);
      }}
    />
  );
}

export function SpacesScreen() {
  const runtime = useAppRuntime();
  const router = useRouter();
  const query = useQuery(runtime.queries.spacesQueryOptions);
  return (
    <SpacesView
      spaces={query.data ?? []}
      isLoading={query.isLoading}
      error={query.error ? errorMessage(query.error, "Could not load spaces.") : null}
      onOpen={(space) => router.push(routes.librarySpace(runtime.scope, space.slug))}
      onRetry={() => {
        void query.refetch();
      }}
    />
  );
}

export function SpaceWorkspaceScreen({ spaceSlug }: { spaceSlug: string }) {
  const runtime = useAppRuntime();
  const router = useRouter();
  const query = useQuery(runtime.queries.spaceQueryOptions(spaceSlug));
  return (
    <>
      <Stack.Screen options={{ title: query.data?.name ?? "" }} />
      <SpaceWorkspaceView
        space={query.data ?? null}
        isLoading={query.isLoading}
        error={query.error ? errorMessage(query.error, "Could not load space.") : null}
        onOpenTasks={() => router.push(routes.spaceTasks(runtime.scope, spaceSlug))}
        onOpenRecipes={() => router.push(routes.spaceRecipes(runtime.scope, spaceSlug))}
        onRetry={() => {
          void query.refetch();
        }}
      />
    </>
  );
}

export function SpaceTasksScreen({ spaceSlug }: { spaceSlug: string }) {
  const runtime = useAppRuntime();
  const router = useRouter();
  const spaceQuery = useQuery(runtime.queries.spaceQueryOptions(spaceSlug));
  const query = useInfiniteQuery(runtime.queries.spaceTasksInfiniteQueryOptions(spaceSlug));
  const tasks = query.data?.pages.flatMap((page) => page.items) ?? [];
  const initialError = !query.isFetchNextPageError && query.error ? query.error : null;
  return (
    <>
      <Stack.Screen options={{ title: spaceQuery.data?.name ?? "" }} />
      <TaskListView
        emptyTitle="No tasks in this space"
        emptyDetail="Tasks added to this space will appear here."
        tasks={tasks}
        source="network"
        timestamp={query.dataUpdatedAt ? formatTimestamp(query.dataUpdatedAt) : null}
        isStale={query.dataUpdatedAt ? isStaleTimestamp(query.dataUpdatedAt) : false}
        isInitialLoading={query.isLoading}
        isRefreshing={query.isRefetching && !query.isFetchingNextPage}
        isLoadingMore={query.isFetchingNextPage}
        initialError={initialError ? errorMessage(initialError, "Could not load tasks.") : null}
        loadMoreError={
          query.isFetchNextPageError
            ? errorMessage(query.error, "Could not load more tasks.")
            : null
        }
        canLoadMore={query.hasNextPage}
        cachedHasMore={false}
        onSelect={(task: Task) =>
          router.push(routes.spaceTaskDetail(runtime.scope, spaceSlug, task.id))
        }
        onRefresh={async () => {
          await query.refetch();
        }}
        onLoadMore={() => {
          void query.fetchNextPage();
        }}
        onRetry={() => {
          void query.refetch();
        }}
      />
    </>
  );
}

export function RecipesScreen({ spaceSlug }: { spaceSlug?: string }) {
  const runtime = useAppRuntime();
  const router = useRouter();
  const spacesQuery = useQuery(runtime.queries.spacesQueryOptions);
  const query = useInfiniteQuery(runtime.queries.recipesInfiniteQueryOptions(spaceSlug));
  const recipes = useMemo(
    () => query.data?.pages.flatMap((page) => page.items) ?? [],
    [query.data],
  );
  const spacesBySlug = useMemo(
    () => new Map((spacesQuery.data ?? []).map((space) => [space.slug, space.name])),
    [spacesQuery.data],
  );
  const initialError = !query.isFetchNextPageError && query.error ? query.error : spacesQuery.error;
  const scopedSpaceName = spaceSlug ? spacesBySlug.get(spaceSlug) : undefined;
  return (
    <>
      <Stack.Screen options={{ title: scopedSpaceName ?? "Recipes" }} />
      <RecipeListView
        recipes={recipes}
        spacesBySlug={spacesBySlug}
        scopedSpaceName={scopedSpaceName}
        isLoading={query.isLoading || spacesQuery.isLoading}
        isRefreshing={query.isRefetching && !query.isFetchingNextPage}
        isLoadingMore={query.isFetchingNextPage}
        error={initialError ? errorMessage(initialError, "Could not load recipes.") : null}
        loadMoreError={
          query.isFetchNextPageError
            ? errorMessage(query.error, "Could not load more recipes.")
            : null
        }
        canLoadMore={query.hasNextPage}
        onOpen={(recipe: RecipeSummary) => {
          router.push(
            spaceSlug
              ? routes.spaceRecipeDetail(runtime.scope, recipe.spaceSlug, recipe.id)
              : routes.recipeDetail(runtime.scope, recipe.spaceSlug, recipe.id),
          );
        }}
        onRefresh={async () => {
          await Promise.all([query.refetch(), spacesQuery.refetch()]);
        }}
        onLoadMore={() => {
          void query.fetchNextPage();
        }}
        onRetry={() => {
          void Promise.all([query.refetch(), spacesQuery.refetch()]);
        }}
      />
    </>
  );
}

export function RecipeDetailController({
  spaceSlug,
  recipeId,
}: {
  spaceSlug: string;
  recipeId: string;
}) {
  const runtime = useAppRuntime();
  const recipeQuery = useQuery(runtime.queries.recipeQueryOptions(spaceSlug, recipeId));
  const spaceQuery = useQuery(runtime.queries.spaceQueryOptions(spaceSlug));
  const error = recipeQuery.error ?? spaceQuery.error;
  return (
    <>
      <Stack.Screen options={{ title: recipeQuery.data?.name ?? "" }} />
      <RecipeDetailView
        recipe={recipeQuery.data ?? null}
        spaceName={spaceQuery.data?.name ?? null}
        isLoading={recipeQuery.isLoading || spaceQuery.isLoading}
        error={error ? errorMessage(error, "Could not load recipe.") : null}
        showTitle={false}
        onRetry={() => {
          void Promise.all([recipeQuery.refetch(), spaceQuery.refetch()]);
        }}
      />
    </>
  );
}

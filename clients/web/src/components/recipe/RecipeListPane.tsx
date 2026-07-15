import { useSuspenseInfiniteQuery, useSuspenseQuery } from "@tanstack/react-query";
import { createLink } from "@tanstack/react-router";
import { ChevronDown, CookingPot, Plus } from "lucide-react";
import { useMemo, useState } from "react";
import { recipesInfiniteQueryOptions, spacesQueryOptions } from "../../lib/queries.ts";
import {
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRoot,
  DropdownMenuTrigger,
} from "../../ui/DropdownMenu.tsx";
import { SpaceListPaneHeader } from "../SpaceListPaneHeader.tsx";
import { RecipeRow } from "./RecipeRow.tsx";

const CreateRecipeItemLink = createLink(DropdownMenuItem);
const CreateRecipeLink = createLink("a");

function CreateRecipeAction({
  spaceSlug,
  scoped,
}: {
  spaceSlug?: string | undefined;
  scoped: boolean;
}) {
  const { data: spaces } = useSuspenseQuery(spacesQueryOptions);
  const [open, setOpen] = useState(false);
  if (spaceSlug) {
    if (!scoped) {
      return (
        <CreateRecipeLink
          to="/recipes/new/$spaceSlug"
          params={{ spaceSlug }}
          className="flex w-full items-center justify-center gap-2 rounded-box border-2 border-dashed border-base-300 p-3 text-sm text-base-content/60 transition-colors hover:border-base-content/40 hover:text-base-content/80"
        >
          <Plus className="size-4" aria-hidden="true" />
          Create recipe
        </CreateRecipeLink>
      );
    }
    return (
      <CreateRecipeLink
        to="/spaces/$spaceSlug/recipes/new"
        params={{ spaceSlug }}
        className="flex w-full items-center justify-center gap-2 rounded-box border-2 border-dashed border-base-300 p-3 text-sm text-base-content/60 transition-colors hover:border-base-content/40 hover:text-base-content/80"
      >
        <Plus className="size-4" aria-hidden="true" />
        Create recipe
      </CreateRecipeLink>
    );
  }
  return (
    <DropdownMenuRoot open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className="flex w-full items-center justify-center gap-2 rounded-box border-2 border-dashed border-base-300 p-3 text-sm text-base-content/60 transition-colors hover:border-base-content/40 hover:text-base-content/80 disabled:cursor-not-allowed disabled:opacity-50"
          disabled={spaces.length === 0}
        >
          <Plus className="size-4" aria-hidden="true" />
          Create recipe
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start">
        {spaces.map((space) => (
          <CreateRecipeItemLink
            key={space.slug}
            to="/recipes/new/$spaceSlug"
            params={{ spaceSlug: space.slug }}
            onClick={() => setOpen(false)}
            onSelect={() => setOpen(false)}
          >
            {space.name}
          </CreateRecipeItemLink>
        ))}
      </DropdownMenuContent>
    </DropdownMenuRoot>
  );
}

function RecipeListResults({
  spaceSlug,
  scoped,
  spaceNames,
}: {
  spaceSlug?: string | undefined;
  scoped: boolean;
  spaceNames: Map<string, string>;
}) {
  const {
    data: pages,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    error,
    isError,
  } = useSuspenseInfiniteQuery(recipesInfiniteQueryOptions(spaceSlug));
  const recipes = useMemo(() => pages.pages.flatMap((page) => page.items), [pages]);

  return (
    <>
      {recipes.length > 0 ? (
        <div className="overflow-hidden rounded-box border border-base-300">
          {recipes.map((recipe) => (
            <RecipeRow
              key={`${recipe.spaceSlug}/${recipe.id}`}
              recipe={recipe}
              spaceName={
                spaceSlug ? undefined : (spaceNames.get(recipe.spaceSlug) ?? recipe.spaceSlug)
              }
              scoped={scoped}
            />
          ))}
        </div>
      ) : (
        <div className="flex flex-col items-center gap-3 rounded-box border border-base-300 p-12 text-center">
          <CookingPot className="size-12 text-base-content/40" aria-hidden="true" />
          <div>
            <p className="font-medium">No recipes yet</p>
            <p className="mt-1 text-sm text-base-content/70">
              Recipes {spaceSlug ? "in this space" : "across your spaces"} will appear here.
            </p>
          </div>
        </div>
      )}

      {isError && (
        <p className="text-center text-sm text-error">
          Failed to load more recipes: {error?.message ?? "Unknown error"}
        </p>
      )}

      {hasNextPage && (
        <div className="flex justify-center">
          <button
            type="button"
            className="btn btn-soft"
            onClick={() => fetchNextPage()}
            disabled={isFetchingNextPage}
          >
            {isFetchingNextPage ? (
              "Loading…"
            ) : (
              <>
                <ChevronDown className="size-4" aria-hidden="true" />
                Load more
              </>
            )}
          </button>
        </div>
      )}
    </>
  );
}

export function RecipeListPane({ lockedSpaceSlug }: { lockedSpaceSlug?: string | undefined }) {
  const { data: spaces } = useSuspenseQuery(spacesQueryOptions);
  const [selectedSpaceSlug, setSelectedSpaceSlug] = useState<string | undefined>(lockedSpaceSlug);
  const spaceSlug = lockedSpaceSlug ?? selectedSpaceSlug;
  const spaceNames = useMemo(
    () => new Map(spaces.map((space) => [space.slug, space.name])),
    [spaces],
  );
  const heading = lockedSpaceSlug
    ? (spaceNames.get(lockedSpaceSlug) ?? lockedSpaceSlug)
    : "Recipes";

  return (
    <div className="flex flex-col gap-3">
      {lockedSpaceSlug ? (
        <SpaceListPaneHeader spaceSlug={lockedSpaceSlug} name={heading} />
      ) : (
        <div className="flex flex-wrap items-center gap-2">
          <h2 className="truncate text-lg font-semibold">{heading}</h2>
          <select
            className="select select-sm w-auto"
            aria-label="Recipe space"
            value={selectedSpaceSlug ?? ""}
            onChange={(event) => setSelectedSpaceSlug(event.target.value || undefined)}
          >
            <option value="">All spaces</option>
            {spaces.map((space) => (
              <option key={space.slug} value={space.slug}>
                {space.name}
              </option>
            ))}
          </select>
        </div>
      )}

      <CreateRecipeAction
        spaceSlug={lockedSpaceSlug ?? selectedSpaceSlug}
        scoped={!!lockedSpaceSlug}
      />

      <RecipeListResults
        key={spaceSlug ?? "all"}
        spaceSlug={spaceSlug}
        scoped={!!lockedSpaceSlug}
        spaceNames={spaceNames}
      />
    </div>
  );
}

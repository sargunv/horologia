import {
  useMutation,
  useQueryClient,
  useSuspenseInfiniteQuery,
  useSuspenseQuery,
} from "@tanstack/react-query";
import { Activity, Clock3, CookingPot, Copy, Ellipsis, Trash2, X } from "lucide-react";
import { type ReactNode, Suspense, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { apiClient } from "../../api/client.ts";
import type { components } from "../../api/schema.d.ts";
import { useSpaceMemberMap } from "../../lib/hooks.ts";
import { invalidateRecipeLists, useRecipePatch } from "../../lib/mutations.ts";
import { recipeActivityInfiniteQueryOptions, recipeQueryOptions } from "../../lib/queries.ts";
import {
  formatDuration,
  formatYield,
  parseDurationInput,
  parseYieldInput,
} from "../../lib/recipeInputs.ts";
import { notifyStaleData } from "../../lib/toaster.ts";
import { useMenuSearch } from "../../lib/useMenuSearch.ts";
import {
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogRoot,
} from "../../ui/AlertDialog.tsx";
import {
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRoot,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../../ui/DropdownMenu.tsx";
import { TagsInput } from "../../ui/TagsInput.tsx";
import { ActivityFeed } from "../ActivityFeed.tsx";
import { DetailPaneHeader, DETAIL_PANE_TITLE_CLASS } from "../DetailPaneHeader.tsx";
import { FieldPill } from "../FieldPill.tsx";
import { SearchableMenuContent } from "../SearchableMenuContent.tsx";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";
import { RecipeDescriptionEditor } from "./RecipeDescriptionEditor.tsx";
import { RecipeSectionsEditor } from "./RecipeSectionsEditor.tsx";

type Recipe = components["schemas"]["Recipe"];

const DURATION_PRESETS = [10, 15, 30, 45, 60, 90, 120];
const SERVING_PRESETS = [2, 4, 6, 8];

function RecipeActions({
  spaceSlug,
  recipeId,
  onDeleteSuccess,
}: {
  spaceSlug: string;
  recipeId: string;
  onDeleteSuccess: () => void;
}) {
  const queryClient = useQueryClient();
  const [deleteOpen, setDeleteOpen] = useState(false);
  const deleteMutation = useMutation({
    mutationFn: async () => {
      const { error } = await apiClient.DELETE("/spaces/{spaceSlug}/recipes/{recipeId}", {
        params: { path: { spaceSlug, recipeId } },
      });
      if (error) throw new Error(error.message ?? "Failed to delete recipe");
    },
    onSuccess: async () => {
      queryClient.removeQueries({ queryKey: ["spaces", spaceSlug, "recipes", recipeId] });
      try {
        await invalidateRecipeLists(queryClient);
      } catch (err) {
        console.error("Cache invalidation failed after mutation:", err);
        notifyStaleData();
      }
      onDeleteSuccess();
    },
  });

  function copy(text: string, failure: string) {
    void navigator.clipboard.writeText(text).catch(() => toast.error(failure));
  }

  return (
    <>
      <DropdownMenuRoot>
        <DropdownMenuTrigger className="btn btn-soft btn-square btn-sm" aria-label="Recipe actions">
          <Ellipsis className="size-3.5" aria-hidden="true" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem onSelect={() => copy(recipeId, "Failed to copy recipe ID")}>
            <Copy className="size-4" aria-hidden="true" />
            Copy recipe ID
          </DropdownMenuItem>
          <DropdownMenuItem onSelect={() => copy(window.location.href, "Failed to copy URL")}>
            <Copy className="size-4" aria-hidden="true" />
            Copy URL
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem className="text-error" onSelect={() => setDeleteOpen(true)}>
            <Trash2 className="size-4" aria-hidden="true" />
            Delete recipe
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenuRoot>

      <AlertDialogRoot
        open={deleteOpen}
        onOpenChange={(open) => {
          setDeleteOpen(open);
          if (!open) deleteMutation.reset();
        }}
      >
        <AlertDialogContent className="max-w-md space-y-4">
          <AlertDialogHeader title="Delete recipe" />
          <AlertDialogDescription>
            Are you sure you want to delete this recipe? This action cannot be undone.
          </AlertDialogDescription>
          <div role="alert" aria-live="assertive">
            {deleteMutation.error && <ErrorAlert message={deleteMutation.error.message} />}
          </div>
          <AlertDialogFooter>
            <AlertDialogAction asChild>
              <button
                type="button"
                className="btn btn-error flex-1"
                disabled={deleteMutation.isPending}
                onClick={(event) => {
                  event.preventDefault();
                  deleteMutation.mutate();
                }}
              >
                {deleteMutation.isPending ? "Deleting..." : "Delete"}
              </button>
            </AlertDialogAction>
            <AlertDialogCancel className="btn btn-soft">Cancel</AlertDialogCancel>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialogRoot>
    </>
  );
}

function EditableTitle({ recipe }: { recipe: Recipe }) {
  const mutation = useRecipePatch(recipe.spaceSlug, recipe.id);
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(recipe.name);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!editing) setDraft(recipe.name);
  }, [editing, recipe.name]);

  useEffect(() => {
    if (editing) inputRef.current?.focus();
  }, [editing]);

  function save() {
    setEditing(false);
    const trimmed = draft.trim();
    if (trimmed && trimmed !== recipe.name) {
      mutation.mutate({ name: trimmed });
    } else {
      setDraft(recipe.name);
    }
  }

  function enterEditing() {
    mutation.reset();
    setEditing(true);
  }

  if (editing) {
    return (
      <div>
        <input
          ref={inputRef}
          type="text"
          aria-label="Recipe name"
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          onBlur={save}
          onKeyDown={(event) => {
            if (event.key === "Enter") save();
            if (event.key === "Escape") {
              setDraft(recipe.name);
              setEditing(false);
            }
          }}
          className={`w-full border-b-2 border-primary bg-transparent outline-none ${DETAIL_PANE_TITLE_CLASS}`}
          maxLength={500}
          disabled={mutation.isPending}
        />
        {mutation.error && <ErrorAlert message={mutation.error.message} />}
      </div>
    );
  }

  return (
    <div>
      <h1
        className={`w-fit max-w-full cursor-pointer rounded-field transition-colors hover:bg-base-200 ${DETAIL_PANE_TITLE_CLASS}`}
        onClick={enterEditing}
        onKeyDown={(event) => {
          if (event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            enterEditing();
          }
        }}
        role="button"
        tabIndex={0}
      >
        {recipe.name}
      </h1>
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </div>
  );
}

function MinutesField({
  recipe,
  field,
  label,
}: {
  recipe: Recipe;
  field: "prepMinutes" | "cookMinutes";
  label: string;
}) {
  const value = recipe[field];
  const mutation = useRecipePatch(recipe.spaceSlug, recipe.id);
  const search = useMenuSearch();
  const parsed = useMemo(() => parseDurationInput(search.query), [search.query]);

  function select(minutes: number | null) {
    if (minutes !== value) mutation.mutate({ [field]: minutes });
  }

  return (
    <>
      <DropdownMenuRoot
        onOpenChange={(open) => {
          search.menuProps.onOpenChange(open);
          if (open) mutation.reset();
        }}
      >
        <FieldPill
          icon={<Clock3 className="size-3.5" aria-hidden="true" />}
          label={label}
          value={value == null ? null : formatDuration(value)}
          showLabelWithValue
        />
        <SearchableMenuContent
          search={search}
          align="start"
          className="w-56"
          placeholder='e.g. "1h 15m"'
          inputLabel={`${label} duration`}
        >
          {value != null && (
            <DropdownMenuItem className="text-error" onSelect={() => select(null)}>
              <X className="size-4" aria-hidden="true" />
              Clear {label.toLowerCase()}
            </DropdownMenuItem>
          )}
          {search.query ? (
            parsed ? (
              <DropdownMenuItem onSelect={() => select(parsed.minutes)}>
                <Clock3 className="size-4" aria-hidden="true" />
                {parsed.label}
              </DropdownMenuItem>
            ) : (
              <div role="status" className="px-3 py-2 text-sm text-base-content/60">
                Enter minutes or a duration like 1h 15m
              </div>
            )
          ) : (
            DURATION_PRESETS.map((minutes) => (
              <DropdownMenuItem key={minutes} onSelect={() => select(minutes)}>
                {formatDuration(minutes)}
              </DropdownMenuItem>
            ))
          )}
        </SearchableMenuContent>
      </DropdownMenuRoot>
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </>
  );
}

function YieldField({ recipe }: { recipe: Recipe }) {
  const mutation = useRecipePatch(recipe.spaceSlug, recipe.id);
  const search = useMenuSearch();
  const parsed = useMemo(() => parseYieldInput(search.query), [search.query]);

  function select(amount: number, unit = "servings") {
    if (recipe.yield?.amount === amount && recipe.yield.unit === unit) return;
    mutation.mutate({ yield: { amount, unit } });
  }

  const value = recipe.yield ? formatYield(recipe.yield.amount, recipe.yield.unit) : null;

  return (
    <>
      <DropdownMenuRoot
        onOpenChange={(open) => {
          search.menuProps.onOpenChange(open);
          if (open) mutation.reset();
        }}
      >
        <FieldPill
          icon={<CookingPot className="size-3.5" aria-hidden="true" />}
          label="Yield"
          value={value}
        />
        <SearchableMenuContent
          search={search}
          align="start"
          className="w-56"
          placeholder='e.g. "4 servings"'
          inputLabel="Recipe yield"
        >
          {recipe.yield && (
            <DropdownMenuItem
              className="text-error"
              onSelect={() => mutation.mutate({ yield: null })}
            >
              <X className="size-4" aria-hidden="true" />
              Clear yield
            </DropdownMenuItem>
          )}
          {search.query ? (
            parsed ? (
              <DropdownMenuItem onSelect={() => select(parsed.amount, parsed.unit)}>
                <CookingPot className="size-4" aria-hidden="true" />
                {parsed.label}
              </DropdownMenuItem>
            ) : (
              <div role="status" className="px-3 py-2 text-sm text-base-content/60">
                Enter an amount and optional unit
              </div>
            )
          ) : (
            SERVING_PRESETS.map((servings) => (
              <DropdownMenuItem key={servings} onSelect={() => select(servings)}>
                {servings} servings
              </DropdownMenuItem>
            ))
          )}
        </SearchableMenuContent>
      </DropdownMenuRoot>
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </>
  );
}

function TagsField({ recipe }: { recipe: Recipe }) {
  const mutation = useRecipePatch(recipe.spaceSlug, recipe.id);
  return (
    <div className="flex flex-col gap-1">
      <TagsInput
        label="Tags"
        value={recipe.tags}
        onValueChange={(tags) => mutation.mutate({ tags })}
        placeholder={recipe.tags.length === 0 ? "Add tags..." : ""}
        disabled={mutation.isPending}
      />
      {mutation.error && <ErrorAlert message={mutation.error.message} />}
    </div>
  );
}

function RecipeActivity({ spaceSlug, recipeId }: { spaceSlug: string; recipeId: string }) {
  const memberMap = useSpaceMemberMap(spaceSlug);
  const {
    data: pages,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
  } = useSuspenseInfiniteQuery(recipeActivityInfiniteQueryOptions(spaceSlug, recipeId));
  const entries = useMemo(() => pages.pages.flatMap((page) => page.items), [pages]);
  return (
    <ActivityFeed
      entries={entries}
      hasNextPage={hasNextPage}
      fetchNextPage={fetchNextPage}
      isFetchingNextPage={isFetchingNextPage}
      memberMap={memberMap}
      variant="recipe"
    />
  );
}

function RecipeDetailContent({
  recipe,
  backLink,
  breadcrumb,
  onDeleteSuccess,
}: {
  recipe: Recipe;
  backLink: ReactNode;
  breadcrumb: ReactNode;
  onDeleteSuccess: () => void;
}) {
  return (
    <div className="space-y-4">
      <DetailPaneHeader
        backLink={backLink}
        breadcrumb={breadcrumb}
        actions={
          <RecipeActions
            spaceSlug={recipe.spaceSlug}
            recipeId={recipe.id}
            onDeleteSuccess={onDeleteSuccess}
          />
        }
        title={<EditableTitle recipe={recipe} />}
      />

      <div className="flex flex-wrap items-center gap-2">
        <YieldField recipe={recipe} />
        <MinutesField recipe={recipe} field="prepMinutes" label="Prep" />
        <MinutesField recipe={recipe} field="cookMinutes" label="Cook" />
      </div>

      <TagsField recipe={recipe} />

      <RecipeDescriptionEditor
        spaceSlug={recipe.spaceSlug}
        recipeId={recipe.id}
        value={recipe.description}
      />

      <RecipeSectionsEditor recipe={recipe} />

      <div>
        <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold">
          <Activity className="size-4" aria-hidden="true" />
          Activity
        </h2>
        <Suspense
          fallback={
            <div className="py-6 text-center text-sm text-base-content/60">Loading activity…</div>
          }
        >
          <RecipeActivity spaceSlug={recipe.spaceSlug} recipeId={recipe.id} />
        </Suspense>
      </div>
    </div>
  );
}

export function RecipeDetailView({
  spaceSlug,
  recipeId,
  backLink,
  breadcrumb,
  onDeleteSuccess,
}: {
  spaceSlug: string;
  recipeId: string;
  backLink: ReactNode;
  breadcrumb: ReactNode;
  onDeleteSuccess: () => void;
}) {
  const { data: recipe } = useSuspenseQuery(recipeQueryOptions(spaceSlug, recipeId));

  return (
    <RecipeDetailContent
      key={`${recipe.spaceSlug}/${recipe.id}`}
      recipe={recipe}
      backLink={backLink}
      breadcrumb={breadcrumb}
      onDeleteSuccess={onDeleteSuccess}
    />
  );
}

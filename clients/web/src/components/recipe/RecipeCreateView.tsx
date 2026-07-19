import { useMutation, useSuspenseQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { ArrowLeft, ChevronRight } from "lucide-react";
import { type FormEvent, useState } from "react";
import { AnchorLink } from "../../lib/links.ts";
import { useLibraryCommands } from "../../lib/mutations.ts";
import { spaceQueryOptions } from "../../lib/queries.ts";
import { DetailPaneHeader, DETAIL_PANE_TITLE_CLASS } from "../DetailPaneHeader.tsx";
import { ErrorAlert } from "../space-settings/ErrorAlert.tsx";

export function RecipeCreateView({ spaceSlug, scoped }: { spaceSlug: string; scoped: boolean }) {
  const { data: space } = useSuspenseQuery(spaceQueryOptions(spaceSlug));
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const commands = useLibraryCommands();
  const createMutation = useMutation({
    mutationFn: (recipeName: string) => commands.createRecipe(spaceSlug, { name: recipeName }),
    onSuccess: async (recipe) => {
      if (scoped) {
        await navigate({
          to: "/spaces/$spaceSlug/recipes/$recipeId",
          params: { spaceSlug, recipeId: recipe.id },
        });
      } else {
        await navigate({
          to: "/recipes/$spaceSlug/$recipeId",
          params: { spaceSlug, recipeId: recipe.id },
        });
      }
    },
  });

  function handleSubmit(event: FormEvent) {
    event.preventDefault();
    const trimmed = name.trim();
    if (!trimmed || createMutation.isPending) return;
    createMutation.mutate(trimmed);
  }

  const listLink = scoped ? (
    <AnchorLink
      to="/spaces/$spaceSlug/recipes"
      params={{ spaceSlug }}
      className="truncate text-base-content/70 hover:underline"
    >
      {space.name} recipes
    </AnchorLink>
  ) : (
    <AnchorLink to="/recipes" className="truncate text-base-content/70 hover:underline">
      Recipes
    </AnchorLink>
  );

  return (
    <div className="space-y-4">
      <DetailPaneHeader
        backLink={
          scoped ? (
            <AnchorLink
              to="/spaces/$spaceSlug/recipes"
              params={{ spaceSlug }}
              className="inline-flex items-center gap-1 text-sm text-base-content/70 transition-colors hover:text-base-content lg:hidden"
            >
              <ArrowLeft className="size-4" aria-hidden="true" />
              Back to {space.name} recipes
            </AnchorLink>
          ) : (
            <AnchorLink
              to="/recipes"
              className="inline-flex items-center gap-1 text-sm text-base-content/70 transition-colors hover:text-base-content lg:hidden"
            >
              <ArrowLeft className="size-4" aria-hidden="true" />
              Back to recipes
            </AnchorLink>
          )
        }
        breadcrumb={
          <ol className="flex min-w-0 items-center gap-1 text-sm">
            <li>{listLink}</li>
            <li className="text-base-content/60" aria-hidden="true">
              <ChevronRight className="size-3" />
            </li>
            <li className="shrink-0" aria-current="page">
              New recipe
            </li>
          </ol>
        }
        title={<h1 className={DETAIL_PANE_TITLE_CLASS}>Create recipe</h1>}
      />

      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium text-base-content/70">Name</span>
          <input
            type="text"
            required
            value={name}
            onChange={(event) => setName(event.target.value)}
            className="input w-full"
            placeholder="What are you making?"
            maxLength={500}
            autoFocus
            disabled={createMutation.isPending}
          />
        </label>

        <div role="alert" aria-live="assertive">
          {createMutation.error && <ErrorAlert message={createMutation.error.message} />}
        </div>

        <div className="flex gap-3">
          <button
            type="submit"
            className="btn btn-primary"
            disabled={createMutation.isPending || !name.trim()}
          >
            {createMutation.isPending ? "Creating..." : "Create recipe"}
          </button>
        </div>
      </form>
    </div>
  );
}

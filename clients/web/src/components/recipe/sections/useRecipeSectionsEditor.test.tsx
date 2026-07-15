// @vitest-environment jsdom

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, type ReactNode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { apiClient } from "../../../api/client.ts";
import type { Recipe } from "./recipeSectionsModel.ts";
import {
  type IngredientSectionsController,
  type InstructionSectionsController,
  useRecipeSectionsEditor,
} from "./useRecipeSectionsEditor.ts";

const patchMock = vi.spyOn(apiClient, "PATCH");
Object.defineProperty(globalThis, "IS_REACT_ACT_ENVIRONMENT", {
  configurable: true,
  value: true,
});

interface EditorState {
  ingredients: IngredientSectionsController;
  instructions: InstructionSectionsController;
  errorMessage: string | undefined;
}

interface RenderedEditor {
  current: () => EditorState;
  root: Root;
}

function recipe(overrides: Partial<Recipe> = {}): Recipe {
  return {
    id: "R1",
    spaceSlug: "home",
    name: "Soup",
    description: "",
    yield: null,
    prepMinutes: null,
    cookMinutes: null,
    tags: [],
    ingredientSections: [
      {
        title: "",
        ingredients: [{ quantity: 1, quantityMax: null, unit: "cup", item: "stock" }],
      },
    ],
    instructionSections: [{ title: "", steps: [{ body: "Simmer." }] }],
    createdAt: "2026-07-14T00:00:00Z",
    updatedAt: "2026-07-14T00:00:00Z",
    ...overrides,
  };
}

function jsonResponse(value: unknown, status = 200): Response {
  return new Response(JSON.stringify(value), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function renderEditor(value: Recipe): RenderedEditor {
  let state: EditorState | undefined;
  function Harness(): ReactNode {
    state = useRecipeSectionsEditor(value);
    return null;
  }

  const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  const root = createRoot(document.createElement("div"));
  act(() => {
    root.render(
      <QueryClientProvider client={queryClient}>
        <Harness />
      </QueryClientProvider>,
    );
  });
  return {
    current: () => {
      if (!state) throw new Error("editor did not render");
      return state;
    },
    root,
  };
}

async function settleMutation(): Promise<void> {
  for (let attempt = 0; attempt < 3; attempt += 1) {
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });
  }
}

beforeEach(() => {
  patchMock.mockReset();
});

describe("useRecipeSectionsEditor persistence", () => {
  it("does not send a patch when editing finishes without a change", () => {
    const editor = renderEditor(recipe());
    const section = editor.current().ingredients.sections[0]!;
    const ingredient = section.ingredients[0]!;

    act(() => editor.current().ingredients.beginIngredient(section.key, ingredient.key));
    act(() => editor.current().ingredients.finishIngredient(section.key, ingredient.key));

    expect(patchMock).not.toHaveBeenCalled();
    expect(editor.current().ingredients.editing).toBeNull();
    act(() => editor.root.unmount());
  });

  it("keeps a successful mutation response while the parent prop is stale", async () => {
    const updated = recipe({
      ingredientSections: [
        {
          title: "",
          ingredients: [{ quantity: 1, quantityMax: null, unit: "cup", item: "broth" }],
        },
      ],
      updatedAt: "2026-07-14T01:00:00Z",
    });
    patchMock.mockResolvedValue({ data: updated, response: jsonResponse(updated) });
    const editor = renderEditor(recipe());
    const section = editor.current().ingredients.sections[0]!;
    const ingredient = section.ingredients[0]!;

    act(() => editor.current().ingredients.beginIngredient(section.key, ingredient.key));
    act(() =>
      editor.current().ingredients.changeIngredient(section.key, ingredient.key, { item: "broth" }),
    );
    act(() => editor.current().ingredients.finishIngredient(section.key, ingredient.key));
    await settleMutation();

    expect(patchMock).toHaveBeenCalledTimes(1);
    expect(editor.current().ingredients.sections[0]!.ingredients[0]!.item).toBe("broth");
    expect(editor.current().ingredients.editing).toBeNull();
    act(() => editor.root.unmount());
  });

  it("persists edits through the independent instruction collection path", async () => {
    const updated = recipe({
      instructionSections: [{ title: "", steps: [{ body: "Boil." }] }],
      updatedAt: "2026-07-14T01:00:00Z",
    });
    patchMock.mockResolvedValue({ data: updated, response: jsonResponse(updated) });
    const editor = renderEditor(recipe());
    const section = editor.current().instructions.sections[0]!;
    const step = section.steps[0]!;

    act(() => editor.current().instructions.beginStep(section.key, step.key));
    act(() => editor.current().instructions.changeStep(section.key, step.key, "Boil."));
    act(() => editor.current().instructions.finishStep(section.key, step.key));
    await settleMutation();

    expect(patchMock).toHaveBeenCalledTimes(1);
    expect(editor.current().instructions.sections[0]!.steps[0]!.body).toBe("Boil.");
    expect(editor.current().instructions.editing).toBeNull();
    act(() => editor.root.unmount());
  });

  it("rolls a destructive edit back when its patch fails", async () => {
    patchMock.mockResolvedValue({
      error: { code: "internal_error", message: "network down" },
      response: jsonResponse({ message: "network down" }, 500),
    });
    const editor = renderEditor(recipe());
    const section = editor.current().ingredients.sections[0]!;
    const ingredient = section.ingredients[0]!;

    act(() => editor.current().ingredients.deleteIngredient(section.key, ingredient.key));
    await settleMutation();

    expect(editor.current().ingredients.sections[0]!.ingredients[0]!.item).toBe("stock");
    expect(editor.current().ingredients.editing).toBeNull();
    expect(editor.current().errorMessage).toBe("network down");
    act(() => editor.root.unmount());
  });

  it("keeps a failed field edit retryable", async () => {
    const updated = recipe({
      ingredientSections: [
        {
          title: "",
          ingredients: [{ quantity: 1, quantityMax: null, unit: "cup", item: "broth" }],
        },
      ],
      updatedAt: "2026-07-14T01:00:00Z",
    });
    patchMock
      .mockResolvedValueOnce({
        error: { code: "internal_error", message: "network down" },
        response: jsonResponse({ message: "network down" }, 500),
      })
      .mockResolvedValueOnce({ data: updated, response: jsonResponse(updated) });
    const editor = renderEditor(recipe());
    const section = editor.current().ingredients.sections[0]!;
    const ingredient = section.ingredients[0]!;

    act(() => editor.current().ingredients.beginIngredient(section.key, ingredient.key));
    act(() =>
      editor.current().ingredients.changeIngredient(section.key, ingredient.key, { item: "broth" }),
    );
    act(() => editor.current().ingredients.finishIngredient(section.key, ingredient.key));
    await settleMutation();

    expect(editor.current().ingredients.sections[0]!.ingredients[0]!.item).toBe("broth");
    expect(editor.current().ingredients.editing).not.toBeNull();

    act(() => editor.current().ingredients.finishIngredient(section.key, ingredient.key));
    await settleMutation();

    expect(patchMock).toHaveBeenCalledTimes(2);
    expect(editor.current().ingredients.editing).toBeNull();
    act(() => editor.root.unmount());
  });
});

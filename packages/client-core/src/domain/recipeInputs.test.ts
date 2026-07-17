import { describe, expect, it } from "vitest";
import { parseDurationInput, parseIngredientQuantity, parseYieldInput } from "./recipeInputs.ts";

describe("parseDurationInput", () => {
  it.each([
    ["20", 20, "20 min"],
    ["20m", 20, "20 min"],
    ["1h", 60, "1h"],
    ["1h 15m", 75, "1h 15m"],
    ["1.5 hours", 90, "1h 30m"],
  ])("parses %s", (input, minutes, label) => {
    expect(parseDurationInput(input)).toEqual({ minutes, label });
  });

  it.each(["", "soon", "1h later", "1.2m"])("rejects %s", (input) => {
    expect(parseDurationInput(input)).toBeNull();
  });
});

describe("parseYieldInput", () => {
  it("defaults a bare amount to servings", () => {
    expect(parseYieldInput("4")).toEqual({ amount: 4, unit: "servings", label: "4 servings" });
  });

  it("accepts a custom unit", () => {
    expect(parseYieldInput("2 loaves")).toEqual({ amount: 2, unit: "loaves", label: "2 loaves" });
  });

  it("uses a singular serving label", () => {
    expect(parseYieldInput("1 servings")?.label).toBe("1 serving");
  });

  it.each(["", "many servings", "0 servings"])("rejects %s", (input) => {
    expect(parseYieldInput(input)).toBeNull();
  });
});

describe("parseIngredientQuantity", () => {
  it.each([
    ["1 cup", { quantity: 1, unit: "cup" }],
    ["1–2 tbsp", { quantity: 1, quantityMax: 2, unit: "tbsp" }],
    ["1/2 tsp", { quantity: 0.5, unit: "tsp" }],
    ["1 1/2 cups", { quantity: 1.5, unit: "cups" }],
    ["to taste", { unit: "to taste" }],
    ["", { unit: "" }],
  ])("parses %s", (input, expected) => {
    expect(parseIngredientQuantity(input)).toEqual(expected);
  });

  it.each(["2–1 cups", "1– cups", "1/0 cup"])("rejects %s", (input) => {
    expect(parseIngredientQuantity(input)).toBeNull();
  });
});

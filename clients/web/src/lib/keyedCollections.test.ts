import { describe, expect, it } from "vitest";
import { moveKeyed, moveKeyedCollectionItem, type KeyedCollection } from "./keyedCollections.ts";

interface Item {
  key: string;
  value: string;
}

const item = (key: string): Item => ({ key, value: key.toUpperCase() });

function collections(): KeyedCollection<Item>[] {
  return [
    { key: "first", title: "", items: [item("a"), item("b")] },
    { key: "second", title: "Second", items: [item("c")] },
    { key: "empty", title: "Empty", items: [] },
  ];
}

describe("moveKeyed", () => {
  it("moves a value to the target position", () => {
    expect(moveKeyed([item("a"), item("b")], "b", "a").map((value) => value.key)).toEqual([
      "b",
      "a",
    ]);
  });

  it("preserves the original array when the move is invalid or unchanged", () => {
    const values = [item("a"), item("b")];
    expect(moveKeyed(values, "a", "a")).toBe(values);
    expect(moveKeyed(values, "missing", "b")).toBe(values);
  });
});

describe("moveKeyedCollectionItem", () => {
  it("reorders within a collection", () => {
    const result = moveKeyedCollectionItem(collections(), "b", "first", "first", "a");
    expect(result[0]!.items.map((value) => value.key)).toEqual(["b", "a"]);
  });

  it("moves to the end when its current collection is the target", () => {
    const result = moveKeyedCollectionItem(collections(), "a", "first", "first");
    expect(result[0]!.items.map((value) => value.key)).toEqual(["b", "a"]);
  });

  it("moves between collections at the target position", () => {
    const result = moveKeyedCollectionItem(collections(), "a", "first", "second", "c");
    expect(result[0]!.items.map((value) => value.key)).toEqual(["b"]);
    expect(result[1]!.items.map((value) => value.key)).toEqual(["a", "c"]);
  });

  it("appends when the collection itself is the target", () => {
    const result = moveKeyedCollectionItem(collections(), "a", "first", "empty");
    expect(result[2]!.items.map((value) => value.key)).toEqual(["a"]);
  });

  it("removes an untitled source collection after moving its last item", () => {
    const values: KeyedCollection<Item>[] = [
      { key: "default", title: "", items: [item("a")] },
      { key: "named", title: "Named", items: [] },
    ];
    const result = moveKeyedCollectionItem(values, "a", "default", "named");
    expect(result.map((collection) => collection.key)).toEqual(["named"]);
    expect(result[0]!.items.map((value) => value.key)).toEqual(["a"]);
  });

  it("preserves the original array when identifiers are invalid", () => {
    const values = collections();
    expect(moveKeyedCollectionItem(values, "missing", "first", "second")).toBe(values);
    expect(moveKeyedCollectionItem(values, "a", "missing", "second")).toBe(values);
  });
});

import { describe, expect, it } from "vitest";
import { QueuedAutosave } from "./queuedAutosave.ts";

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

async function flushMicrotasks() {
  await Promise.resolve();
  await Promise.resolve();
}

describe("QueuedAutosave", () => {
  it("keeps one save in flight and drains the latest queued value", async () => {
    const first = deferred<string>();
    const second = deferred<string>();
    const saves: string[] = [];
    const autosave = new QueuedAutosave("", (value) => {
      saves.push(value);
      return saves.length === 1 ? first.promise : second.promise;
    });

    autosave.requestSave("first");
    autosave.requestSave("second");
    autosave.requestSave("third");

    expect(saves).toEqual(["first"]);

    first.resolve("first");
    await flushMicrotasks();

    expect(saves).toEqual(["first", "third"]);

    second.resolve("third");
    await flushMicrotasks();

    expect(autosave.persistedValue).toBe("third");
    expect(autosave.isBlocked()).toBe(false);
  });

  it("does not drain a stale queued value after local content returns to the in-flight value", async () => {
    const first = deferred<string>();
    const saves: string[] = [];
    const autosave = new QueuedAutosave("", (value) => {
      saves.push(value);
      return first.promise;
    });

    autosave.requestSave("A");
    autosave.requestSave("B");
    autosave.requestSave("A");

    expect(saves).toEqual(["A"]);

    first.resolve("A");
    await flushMicrotasks();

    expect(saves).toEqual(["A"]);
    expect(autosave.persistedValue).toBe("A");
    expect(autosave.isBlocked()).toBe(false);
  });

  it("drains newer local content even when no save request fired while in flight", async () => {
    const first = deferred<string>();
    const second = deferred<string>();
    const saves: string[] = [];
    const autosave = new QueuedAutosave("", (value) => {
      saves.push(value);
      return saves.length === 1 ? first.promise : second.promise;
    });

    autosave.requestSave("first");
    autosave.setLocalValue("typed while saving");

    expect(saves).toEqual(["first"]);

    first.resolve("first");
    await flushMicrotasks();

    expect(saves).toEqual(["first", "typed while saving"]);

    second.resolve("typed while saving");
    await flushMicrotasks();

    expect(autosave.persistedValue).toBe("typed while saving");
    expect(autosave.isBlocked()).toBe(false);
  });

  it("does not re-accept a stale external prop after save success", async () => {
    const saveDone = deferred<string>();
    const autosave = new QueuedAutosave("old", () => saveDone.promise);

    autosave.requestSave("new");
    saveDone.resolve("new");
    await flushMicrotasks();

    expect(autosave.canSyncExternalValue("old")).toBe(false);

    autosave.receiveExternalValue("new");

    expect(autosave.canSyncExternalValue("new")).toBe(true);
  });

  it("ignores external values that arrive while local content is unsaved", () => {
    const autosave = new QueuedAutosave("old", async (value) => value);

    autosave.setLocalValue("local draft");
    autosave.receiveExternalValue("refetched old value");

    expect(autosave.localValue).toBe("local draft");
    expect(autosave.canSyncExternalValue("refetched old value")).toBe(false);
  });

  it("keeps dirty local content retryable after a save failure", async () => {
    const failed = deferred<string>();
    const retried = deferred<string>();
    const saves: string[] = [];
    const autosave = new QueuedAutosave("old", (value) => {
      saves.push(value);
      return saves.length === 1 ? failed.promise : retried.promise;
    });

    autosave.requestSave("draft");
    failed.reject(new Error("network down"));
    await flushMicrotasks();

    expect(autosave.localValue).toBe("draft");
    expect(autosave.persistedValue).toBe("old");
    expect(autosave.isBlocked()).toBe(true);

    autosave.requestSave("draft");

    expect(saves).toEqual(["draft", "draft"]);

    retried.resolve("draft");
    await flushMicrotasks();

    expect(autosave.isBlocked()).toBe(false);
  });
});

import { describe, expect, it, vi } from "vitest";
import { createViewStateStore, stringCodec, jsonCodec, type StorageLike } from "./view-state-utils";

function fakeStorage(initial: Record<string, string> = {}): StorageLike & { data: Record<string, string> } {
  const data = { ...initial };
  return {
    data,
    getItem: (k) => (k in data ? data[k] : null),
    setItem: (k, v) => { data[k] = v; },
  };
}

const throwingStorage: StorageLike = {
  getItem: () => { throw new Error("denied"); },
  setItem: () => { throw new Error("denied"); },
};

describe("createViewStateStore", () => {
  it("returns the initial value for an unknown memory key", () => {
    const store = createViewStateStore(fakeStorage());
    expect(store.get("backlog.tab", () => "all")).toBe("all");
  });

  it("keeps memory values across reads without touching storage", () => {
    const storage = fakeStorage();
    const store = createViewStateStore(storage);
    store.set("backlog.tab", "done");
    expect(store.get("backlog.tab", () => "all")).toBe("done");
    expect(storage.data).toEqual({});
  });

  it("returns a stable value for repeated reads", () => {
    const store = createViewStateStore(fakeStorage());
    const init = vi.fn(() => ({ a: 1 }));
    const first = store.get("k", init);
    expect(store.get("k", init)).toBe(first);
    expect(init).toHaveBeenCalledTimes(1);
  });

  it("reads persisted values through the codec on first access", () => {
    const store = createViewStateStore(fakeStorage({ "exponential-sort": "priority" }));
    expect(store.get("exponential-sort", () => "manual", stringCodec("manual"))).toBe("priority");
  });

  it("falls back when nothing is persisted", () => {
    const store = createViewStateStore(fakeStorage());
    expect(store.get("exponential-sort", () => "manual", stringCodec("manual"))).toBe("manual");
  });

  it("writes persisted values through the codec", () => {
    const storage = fakeStorage();
    const store = createViewStateStore(storage);
    const codec = jsonCodec((raw) => (raw ? JSON.parse(raw) : []));
    store.set("exponential-x", ["a"], codec);
    expect(storage.data["exponential-x"]).toBe('["a"]');
    expect(store.get("exponential-x", () => [], codec)).toEqual(["a"]);
  });

  it("treats each key independently, so a dynamic key reads its own value", () => {
    const codec = jsonCodec((raw) => (raw ? JSON.parse(raw) : null));
    const store = createViewStateStore(fakeStorage({
      "f-all": '{"labels":["ui"]}',
      "f-done": '{"labels":["bug"]}',
    }));
    expect(store.get("f-all", () => null, codec)).toEqual({ labels: ["ui"] });
    expect(store.get("f-done", () => null, codec)).toEqual({ labels: ["bug"] });
  });

  it("notifies subscribers on set and stops after unsubscribe", () => {
    const store = createViewStateStore(fakeStorage());
    const listener = vi.fn();
    const unsubscribe = store.subscribe(listener);
    store.set("k", 1);
    expect(listener).toHaveBeenCalledTimes(1);
    unsubscribe();
    store.set("k", 2);
    expect(listener).toHaveBeenCalledTimes(1);
  });

  it("survives storage that throws", () => {
    const store = createViewStateStore(throwingStorage);
    expect(store.get("k", () => "init", stringCodec("init"))).toBe("init");
    store.set("k", "next", stringCodec("init"));
    expect(store.get("k", () => "init", stringCodec("init"))).toBe("next");
  });

  it("works without any storage", () => {
    const store = createViewStateStore(null);
    store.set("k", "v", stringCodec(""));
    expect(store.get("k", () => "", stringCodec(""))).toBe("v");
  });
});

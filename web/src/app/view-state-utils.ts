export interface StorageLike {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
}

/** How a persisted value is read from and written to storage. */
export interface Codec<T> {
  parse(raw: string | null): T;
  serialize(value: T): string;
}

export function stringCodec<T extends string>(fallback: T): Codec<T> {
  return { parse: (raw) => (raw as T) || fallback, serialize: (v) => v };
}

export function jsonCodec<T>(parse: (raw: string | null) => T): Codec<T> {
  return { parse, serialize: (v) => JSON.stringify(v) };
}

export interface ViewStateStore {
  get<T>(key: string, init: () => T, codec?: Codec<T>): T;
  set<T>(key: string, value: T, codec?: Codec<T>): void;
  subscribe(listener: () => void): () => void;
}

/**
 * App-scoped store for view-local state that must outlive the view component
 * (it unmounts while IssueDetail is open). Keys with a codec are also persisted
 * to storage under that same key; keys without one live for the session only.
 */
export function createViewStateStore(storage: StorageLike | null): ViewStateStore {
  const values = new Map<string, unknown>();
  const listeners = new Set<() => void>();

  const read = (key: string): string | null => {
    try { return storage?.getItem(key) ?? null; } catch { return null; }
  };
  const write = (key: string, raw: string) => {
    try { storage?.setItem(key, raw); } catch { /* storage unavailable */ }
  };

  return {
    get<T>(key: string, init: () => T, codec?: Codec<T>): T {
      if (!values.has(key)) {
        values.set(key, codec ? codec.parse(read(key)) : init());
      }
      return values.get(key) as T;
    },
    set<T>(key: string, value: T, codec?: Codec<T>) {
      values.set(key, value);
      if (codec) write(key, codec.serialize(value));
      listeners.forEach((l) => l());
    },
    subscribe(listener) {
      listeners.add(listener);
      return () => { listeners.delete(listener); };
    },
  };
}

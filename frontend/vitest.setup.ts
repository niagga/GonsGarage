import '@testing-library/jest-dom/vitest';
import { vi } from 'vitest';

globalThis.ResizeObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
}));

// Node 22+ exposes a global `localStorage` getter that stays `undefined` unless the
// process runs with --experimental-webstorage. Vitest's jsdom environment assigns
// `window = globalThis`, so that Node getter shadows jsdom's own Web Storage and
// every test sees `localStorage === undefined`. Browsers always expose it, so the
// suite gets a spec-shaped in-memory Storage whenever the real one is missing.
function createInMemoryStorage(): Storage {
  const store = new Map<string, string>();
  const storage = {
    get length() {
      return store.size;
    },
    key: (index: number) => Array.from(store.keys())[index] ?? null,
    getItem: (key: string) => (store.has(key) ? (store.get(key) as string) : null),
    setItem: (key: string, value: string) => {
      store.set(key, String(value));
    },
    removeItem: (key: string) => {
      store.delete(key);
    },
    clear: () => {
      store.clear();
    },
  };
  return storage as unknown as Storage;
}

if (typeof globalThis.localStorage === 'undefined') {
  Object.defineProperty(globalThis, 'localStorage', {
    value: createInMemoryStorage(),
    configurable: true,
  });
}

if (typeof globalThis.sessionStorage === 'undefined') {
  Object.defineProperty(globalThis, 'sessionStorage', {
    value: createInMemoryStorage(),
    configurable: true,
  });
}

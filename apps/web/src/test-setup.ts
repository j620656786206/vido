import '@testing-library/jest-dom/vitest';

// Mock URL.createObjectURL for jsdom environment
if (typeof URL.createObjectURL === 'undefined') {
  URL.createObjectURL = () => 'blob:mock-url';
}
if (typeof URL.revokeObjectURL === 'undefined') {
  URL.revokeObjectURL = () => {};
}

// Node.js v25+ exposes a native `localStorage` stub with no methods (clear/setItem/etc
// are undefined) unless --localstorage-file is configured. Override it with a proper
// in-memory Web Storage implementation so tests that call localStorage.clear() work.
if (typeof localStorage === 'undefined' || typeof localStorage.clear !== 'function') {
  const makeStorage = () => {
    let store: Record<string, string> = {};
    return {
      getItem: (key: string): string | null => store[key] ?? null,
      setItem: (key: string, value: string): void => {
        store[key] = String(value);
      },
      removeItem: (key: string): void => {
        delete store[key];
      },
      clear: (): void => {
        store = {};
      },
      get length(): number {
        return Object.keys(store).length;
      },
      key: (index: number): string | null => Object.keys(store)[index] ?? null,
    };
  };

  Object.defineProperty(globalThis, 'localStorage', {
    value: makeStorage(),
    writable: true,
    configurable: true,
  });
  Object.defineProperty(globalThis, 'sessionStorage', {
    value: makeStorage(),
    writable: true,
    configurable: true,
  });
}

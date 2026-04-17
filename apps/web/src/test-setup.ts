import '@testing-library/jest-dom/vitest';

// jsdom does not implement IntersectionObserver. Homepage explore blocks use it
// to lazy-load below-the-fold content (Story 10-5 Task 2.3). Provide an inert
// stub — tests that need to simulate intersection override it per-spec.
if (typeof globalThis.IntersectionObserver === 'undefined') {
  class MockIntersectionObserver {
    root = null;
    rootMargin = '';
    thresholds: number[] = [];
    observe(): void {
      /* no-op */
    }
    unobserve(): void {
      /* no-op */
    }
    disconnect(): void {
      /* no-op */
    }
    takeRecords(): IntersectionObserverEntry[] {
      return [];
    }
  }
  Object.defineProperty(globalThis, 'IntersectionObserver', {
    value: MockIntersectionObserver,
    writable: true,
    configurable: true,
  });
}

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

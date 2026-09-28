import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { useEnrichmentRefresh } from './useEnrichmentRefresh';
import { libraryKeys } from './useLibrary';

vi.mock('../services/scannerService', () => ({
  scannerService: { getSSEUrl: () => '/api/v1/events' },
}));

class MockEventSource {
  static instances: MockEventSource[] = [];
  listeners: Record<string, ((e: Event) => void)[]> = {};
  closed = false;
  constructor(public url: string) {
    MockEventSource.instances.push(this);
  }
  addEventListener(type: string, cb: (e: Event) => void) {
    (this.listeners[type] ??= []).push(cb);
  }
  removeEventListener(type: string, cb: (e: Event) => void) {
    this.listeners[type] = (this.listeners[type] ?? []).filter((l) => l !== cb);
  }
  close() {
    this.closed = true;
  }
  emit(type: string, data: unknown) {
    this.listeners[type]?.forEach((cb) =>
      cb(new MessageEvent(type, { data: JSON.stringify(data) }))
    );
  }
}

describe('useEnrichmentRefresh (disc-2026-09-batch-reparse-never-runs)', () => {
  let queryClient: QueryClient;
  const wrapper = ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);

  beforeEach(() => {
    MockEventSource.instances = [];
    (globalThis as Record<string, unknown>).EventSource = MockEventSource;
    queryClient = new QueryClient();
  });
  afterEach(() => {
    delete (globalThis as Record<string, unknown>).EventSource;
  });

  it('opens no SSE connection until a re-parse asks for one', () => {
    renderHook(() => useEnrichmentRefresh(false), { wrapper });
    expect(MockEventSource.instances).toHaveLength(0);
  });

  it('refetches the library when a matching pass completes, and closes on unmount', () => {
    const invalidate = vi.spyOn(queryClient, 'invalidateQueries');
    const { unmount } = renderHook(() => useEnrichmentRefresh(true), { wrapper });

    expect(MockEventSource.instances).toHaveLength(1);
    const es = MockEventSource.instances[0];
    expect(es.url).toBe('/api/v1/events');

    es.emit('enrich_complete', { total: 3, succeeded: 3 });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: libraryKeys.all });

    // A second pass (rows queued behind a running one) refreshes again.
    es.emit('enrich_complete', { total: 1, succeeded: 1 });
    expect(invalidate).toHaveBeenCalledTimes(2);

    unmount();
    expect(es.closed).toBe(true);
  });
});

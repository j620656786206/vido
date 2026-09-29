import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
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
      cb(new MessageEvent(type, { data: JSON.stringify({ type, data }) }))
    );
  }
}

describe('useEnrichmentRefresh (batch 重新解析 → background matching)', () => {
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

  it('opens no SSE connection and reports nothing until a re-parse asks for one', () => {
    const { result } = renderHook(() => useEnrichmentRefresh(false), { wrapper });
    expect(MockEventSource.instances).toHaveLength(0);
    expect(result.current).toBeNull();
  });

  it('queued → running (enrich_progress) → done (enrich_complete, refetch) → running again', () => {
    const invalidate = vi.spyOn(queryClient, 'invalidateQueries');
    const { result, unmount } = renderHook(() => useEnrichmentRefresh(true), { wrapper });

    expect(MockEventSource.instances).toHaveLength(1);
    const es = MockEventSource.instances[0];
    expect(es.url).toBe('/api/v1/events');
    // Nothing heard yet: the dialog says 等待開始.
    expect(result.current?.phase).toBe('queued');

    act(() => {
      es.emit('enrich_progress', {
        total: 20,
        processed: 3,
        succeeded: 2,
        failed: 1,
        skipped: 0,
        current_title: '你的名字',
        is_active: true,
      });
    });
    expect(result.current).toEqual({
      phase: 'running',
      total: 20,
      processed: 3,
      succeeded: 2,
      failed: 1,
      skipped: 0,
      currentTitle: '你的名字',
    });
    expect(invalidate).not.toHaveBeenCalled();

    act(() => {
      es.emit('enrich_complete', {
        total: 20,
        succeeded: 18,
        failed: 2,
        skipped: 0,
        duration: '1m3s',
      });
    });
    expect(result.current).toMatchObject({ phase: 'done', total: 20, succeeded: 18, failed: 2 });
    expect(result.current?.currentTitle).toBe('');
    expect(invalidate).toHaveBeenCalledWith({ queryKey: libraryKeys.all });

    // The rows were queued behind that pass: a second pass starts — back to 比對中.
    act(() => {
      es.emit('enrich_progress', {
        total: 5,
        processed: 5,
        succeeded: 5,
        failed: 0,
        skipped: 0,
        current_title: '瀑布',
      });
    });
    expect(result.current?.phase).toBe('running');
    act(() => {
      es.emit('enrich_complete', { total: 5, succeeded: 5, failed: 0, skipped: 0 });
    });
    expect(invalidate).toHaveBeenCalledTimes(2);

    unmount();
    expect(es.closed).toBe(true);
  });

  it('a new runId starts over from queued on a fresh connection', () => {
    const { result, rerender } = renderHook(({ run }) => useEnrichmentRefresh(true, run), {
      wrapper,
      initialProps: { run: 1 },
    });
    const first = MockEventSource.instances[0];
    act(() => {
      first.emit('enrich_complete', { total: 5, succeeded: 5, failed: 0, skipped: 0 });
    });
    expect(result.current?.phase).toBe('done');

    rerender({ run: 2 });
    expect(first.closed).toBe(true);
    expect(MockEventSource.instances).toHaveLength(2);
    expect(result.current?.phase).toBe('queued');
  });

  it('a malformed progress payload is ignored, not a crash', () => {
    const { result } = renderHook(() => useEnrichmentRefresh(true), { wrapper });
    const es = MockEventSource.instances[0];
    act(() => {
      es.listeners['enrich_progress']?.forEach((cb) =>
        cb(new MessageEvent('enrich_progress', { data: 'not json' }))
      );
    });
    expect(result.current?.phase).toBe('queued');
  });
});

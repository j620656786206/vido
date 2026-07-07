import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { requestKeys } from './useRequestedMedia';
import { useRequestProgress, applyRequestSnapshot, type LiveRequest } from './useRequestProgress';

// Minimal EventSource stub — records instances + lets a test emit a named event
// with a data payload (cloned from useDownloadProgress.spec.ts).
class MockEventSource {
  static instances: MockEventSource[] = [];
  url: string;
  readyState = 0;
  onerror: (() => void) | null = null;
  private listeners: Record<string, ((e: MessageEvent) => void)[]> = {};
  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
  }
  addEventListener(type: string, cb: (e: MessageEvent) => void) {
    (this.listeners[type] ||= []).push(cb);
  }
  close() {
    this.readyState = 2;
  }
  emit(type: string, payload: unknown) {
    (this.listeners[type] || []).forEach((cb) =>
      cb({ data: JSON.stringify(payload) } as MessageEvent)
    );
  }
}

const req = (over: Partial<LiveRequest> = {}): LiveRequest => ({
  id: 'r1',
  tmdbId: 550,
  mediaType: 'movie',
  title: '沙丘：第二部',
  status: 'searching',
  fulfilmentSource: null,
  externalId: null,
  seasons: null,
  episodes: null,
  errorMessage: null,
  requestedAt: '2026-06-28T10:00:00Z',
  updatedAt: '2026-06-28T10:00:00Z',
  ...over,
});

const LIST_KEY = requestKeys.list();

describe('applyRequestSnapshot (13-3b AC#2 — [@contract-v1] bare-array merge)', () => {
  it('replaces cached rows by id and rides the ephemeral progress in', () => {
    const qc = new QueryClient();
    qc.setQueryData<LiveRequest[]>(LIST_KEY, [req({ id: 'a', status: 'searching' })]);

    applyRequestSnapshot(qc, [req({ id: 'a', status: 'downloading', progress: 0.42 })]);

    const rows = qc.getQueryData<LiveRequest[]>(LIST_KEY)!;
    expect(rows).toHaveLength(1);
    expect(rows[0].status).toBe('downloading');
    expect(rows[0].progress).toBe(0.42); // ephemeral progress rides into the cached row
  });

  it('KEEPS cached terminal rows absent from the snapshot (completed/failed history)', () => {
    const qc = new QueryClient();
    qc.setQueryData<LiveRequest[]>(LIST_KEY, [
      req({ id: 'done', status: 'completed' }),
      req({ id: 'live', status: 'downloading', progress: 0.1 }),
    ]);

    // The snapshot carries only the live row — terminal history is dropped from the wire.
    applyRequestSnapshot(qc, [req({ id: 'live', status: 'downloading', progress: 0.8 })]);

    const rows = qc.getQueryData<LiveRequest[]>(LIST_KEY)!;
    expect(rows.map((r) => r.id).sort()).toEqual(['done', 'live']);
    expect(rows.find((r) => r.id === 'live')!.progress).toBe(0.8);
    expect(rows.find((r) => r.id === 'done')!.status).toBe('completed'); // history preserved
  });

  it('DROPS cached ACTIVE rows absent from the snapshot (phantom-row hazard — STALE-MARK / 13-7a)', () => {
    const qc = new QueryClient();
    qc.setQueryData<LiveRequest[]>(LIST_KEY, [
      req({ id: 'gone', status: 'pending' }), // cancelled in another tab (13-7a hard-DELETE)
      req({ id: 'live', status: 'searching' }),
    ]);

    applyRequestSnapshot(qc, [req({ id: 'live', status: 'searching' })]);

    const rows = qc.getQueryData<LiveRequest[]>(LIST_KEY)!;
    expect(rows.map((r) => r.id)).toEqual(['live']); // phantom 'gone' pending row dropped
  });

  it('APPENDS snapshot rows not yet in cache (created/activated in another tab)', () => {
    const qc = new QueryClient();
    qc.setQueryData<LiveRequest[]>(LIST_KEY, [req({ id: 'a', status: 'searching' })]);

    applyRequestSnapshot(qc, [
      req({ id: 'a', status: 'searching' }),
      req({ id: 'b', status: 'pending' }),
    ]);

    const rows = qc.getQueryData<LiveRequest[]>(LIST_KEY)!;
    expect(rows.map((r) => r.id)).toEqual(['a', 'b']);
  });

  it('appends a brand-new id at most ONCE even if the wire snapshot duplicates it (no dup React key)', () => {
    const qc = new QueryClient();
    qc.setQueryData<LiveRequest[]>(LIST_KEY, [req({ id: 'a', status: 'searching' })]);

    // Defensive: a malformed frame lists the same new id twice — must not double the row.
    applyRequestSnapshot(qc, [
      req({ id: 'a', status: 'searching' }),
      req({ id: 'dup', status: 'pending' }),
      req({ id: 'dup', status: 'pending' }),
    ]);

    const rows = qc.getQueryData<LiveRequest[]>(LIST_KEY)!;
    expect(rows.map((r) => r.id)).toEqual(['a', 'dup']); // 'dup' appears once
  });

  it('is a no-op when no requests list is cached (nothing to patch)', () => {
    const qc = new QueryClient();
    expect(() => applyRequestSnapshot(qc, [req({ id: 'a' })])).not.toThrow();
    expect(qc.getQueryData<LiveRequest[]>(LIST_KEY)).toBeUndefined();
  });
});

describe('useRequestProgress (13-3b AC#1 — lazy SSE clone)', () => {
  beforeEach(() => {
    MockEventSource.instances = [];
    vi.stubGlobal('EventSource', MockEventSource);
  });

  function renderProgress() {
    const qc = new QueryClient();
    qc.setQueryData<LiveRequest[]>(LIST_KEY, [req({ id: 'a', status: 'searching' })]);
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      React.createElement(QueryClientProvider, { client: qc }, children);
    const { result } = renderHook(() => useRequestProgress(), { wrapper });
    return { qc, result };
  }

  it('does NOT open an EventSource on mount (§8 lazy-SSE)', () => {
    renderProgress();
    expect(MockEventSource.instances).toHaveLength(0);
  });

  it('opens the SSE only after startTracking() and points at the events endpoint', () => {
    const { result } = renderProgress();
    act(() => result.current.startTracking());
    expect(MockEventSource.instances).toHaveLength(1);
    expect(MockEventSource.instances[0].url).toContain('/events');
  });

  it('is idempotent — startTracking twice keeps ONE open connection', () => {
    const { result } = renderProgress();
    act(() => result.current.startTracking());
    act(() => result.current.startTracking());
    expect(MockEventSource.instances).toHaveLength(1);
  });

  it('stopTracking closes the connection (readyState CLOSED)', () => {
    const { result } = renderProgress();
    act(() => result.current.startTracking());
    const es = MockEventSource.instances[0];
    act(() => result.current.stopTracking());
    expect(es.readyState).toBe(2);
  });

  it('patches the requests cache from a request_progress event (envelope + snake→camel incl. progress)', () => {
    const { qc, result } = renderProgress();
    act(() => result.current.startTracking());

    // The BE wraps the payload as the whole Event {id,type,data}; data is the snake_case snapshot.
    act(() =>
      MockEventSource.instances[0].emit('request_progress', {
        id: 'x',
        type: 'request_progress',
        data: [
          {
            id: 'a',
            tmdb_id: 550,
            media_type: 'movie',
            title: '沙丘：第二部',
            status: 'downloading',
            progress: 0.66,
            error_message: null,
            requested_at: '2026-06-28T10:00:00Z',
            updated_at: '2026-06-28T10:05:00Z',
          },
        ],
      })
    );

    const row = qc.getQueryData<LiveRequest[]>(LIST_KEY)![0];
    expect(row.status).toBe('downloading');
    expect(row.progress).toBe(0.66);
    expect(row.tmdbId).toBe(550); // snake_case tmdb_id → camelCase
  });

  it('reconnects after an SSE error via the latest-ref reconnect timer', () => {
    vi.useFakeTimers();
    try {
      const { result } = renderProgress();
      act(() => result.current.startTracking());
      expect(MockEventSource.instances).toHaveLength(1);

      act(() => MockEventSource.instances[0].onerror?.());
      act(() => vi.advanceTimersByTime(10000)); // SSE_RECONNECT_MS

      expect(MockEventSource.instances).toHaveLength(2); // a fresh connection was opened
    } finally {
      vi.useRealTimers();
    }
  });
});

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import type { Download, PaginatedDownloads } from '../services/downloadService';
import { downloadKeys } from './useDownloads';
import { useDownloadProgress, applyDownloadSnapshot } from './useDownloadProgress';

// Minimal EventSource stub — records instances + lets a test emit a named event with a data payload.
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

const item = (over: Partial<Download> = {}): Download => ({
  hash: 'a',
  name: 'A.mkv',
  size: 100,
  progress: 0.1,
  downloadSpeed: 5,
  uploadSpeed: 0,
  eta: 100,
  status: 'downloading',
  addedOn: '2026-07-01T00:00:00Z',
  seeds: 1,
  peers: 1,
  downloaded: 10,
  uploaded: 0,
  ratio: 0,
  savePath: '/dl',
  ...over,
});

const LIST_KEY = downloadKeys.list('all', 'added_on', 'desc', 1, 100);

describe('applyDownloadSnapshot (ux3-4-3b AC4 — [@contract-v1] merge)', () => {
  it('refreshes live fields by hash, PRESERVES parse_status, and drops removed torrents', () => {
    const qc = new QueryClient();
    qc.setQueryData<PaginatedDownloads>(LIST_KEY, {
      items: [
        item({ hash: 'a', progress: 0.1, parseStatus: { status: 'completed' } }),
        item({ hash: 'b', progress: 0.2 }),
      ],
      page: 1,
      pageSize: 100,
      totalItems: 2,
      totalPages: 1,
    });

    // Snapshot: 'a' advanced to 0.9 (no parse_status on the wire), 'b' gone (removed).
    applyDownloadSnapshot(qc, [item({ hash: 'a', progress: 0.9 })]);

    const data = qc.getQueryData<PaginatedDownloads>(LIST_KEY)!;
    expect(data.items).toHaveLength(1);
    const a = data.items[0];
    expect(a.hash).toBe('a');
    expect(a.progress).toBe(0.9); // fresh live field
    expect(a.parseStatus?.status).toBe('completed'); // preserved across the merge
    expect(data.totalItems).toBe(1); // decremented for the removed 'b'
  });
});

describe('useDownloadProgress (ux3-4-3b AC4 — lazy SSE)', () => {
  beforeEach(() => {
    MockEventSource.instances = [];
    vi.stubGlobal('EventSource', MockEventSource);
  });

  function renderProgress() {
    const qc = new QueryClient();
    qc.setQueryData<PaginatedDownloads>(LIST_KEY, {
      items: [item({ hash: 'a', progress: 0.1 })],
      page: 1,
      pageSize: 100,
      totalItems: 1,
      totalPages: 1,
    });
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      React.createElement(QueryClientProvider, { client: qc }, children);
    const { result } = renderHook(() => useDownloadProgress(), { wrapper });
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

  it('updates the downloads cache from a download_progress event (envelope + snake_case)', () => {
    const { qc, result } = renderProgress();
    act(() => result.current.startTracking());

    // The BE wraps the payload as the whole Event {id,type,data}; data is the snake_case Torrent[].
    act(() =>
      MockEventSource.instances[0].emit('download_progress', {
        id: 'x',
        type: 'download_progress',
        data: [
          { hash: 'a', name: 'A.mkv', progress: 0.75, download_speed: 999, status: 'downloading' },
        ],
      })
    );

    const a = qc.getQueryData<PaginatedDownloads>(LIST_KEY)!.items[0];
    expect(a.progress).toBe(0.75);
    expect(a.downloadSpeed).toBe(999); // snake_case download_speed → camelCase
  });
});

import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { GenerationBatchItemState } from '../../services/subtitleService';

/**
 * Container-level spec for `GenerationWorkspace` (dsr-6d-c-1). The presentational
 * half lives in GenerationWorkspaceV2.spec.tsx; this file covers the wiring that
 * had ZERO coverage before — and every regression it covers was a real bug found
 * by the dsr-6d-c-1 audit.
 */
const h = vi.hoisted(() => ({
  batchState: {
    batchId: '',
    totalItems: 0,
    currentIndex: 0,
    currentMediaId: null as string | null,
    currentItem: '',
    successCount: 0,
    failCount: 0,
    pausedCount: 0,
    status: 'idle' as string,
    spentUsd: 0,
    budgetUsd: 0,
    items: null as GenerationBatchItemState[] | null,
  },
  batchStartTracking: vi.fn(),
  batchAttachSnapshot: vi.fn(),
  batchReset: vi.fn(),
  batchEpoch: 0,
  itemStartTracking: vi.fn(),
  itemReset: vi.fn(),
  jobsStart: vi.fn(),
  jobsStop: vi.fn(),
  jobsSeed: vi.fn(),
  jobsConnected: true,
  singleJobs: {} as Record<string, unknown>,
  visible: true,
}));

vi.mock('../../hooks/useGenerationBatchProgress', () => ({
  useGenerationBatchProgress: () => ({
    progress: h.batchState,
    status: h.batchState.status,
    startTracking: h.batchStartTracking,
    attachSnapshot: h.batchAttachSnapshot,
    connectionEpoch: h.batchEpoch,
    reset: h.batchReset,
  }),
}));

vi.mock('../../hooks/useGenerationProgress', () => ({
  useGenerationProgress: () => ({
    progress: {
      phase: 'idle',
      failedPhase: null,
      percentage: null,
      message: '',
      jobId: null,
      error: null,
      srtPath: null,
      zhSrtPath: null,
      partial: false,
      englishKeptBlocks: null,
    },
    startTracking: h.itemStartTracking,
    reset: h.itemReset,
  }),
}));

vi.mock('../../hooks/useGenerationJobsFeed', () => ({
  useGenerationJobsFeed: () => ({
    feed: [],
    singleJobs: h.singleJobs,
    connected: h.jobsConnected,
    startTracking: h.jobsStart,
    stop: h.jobsStop,
    seedBatch: h.jobsSeed,
  }),
}));

// ⚠️ DownloadsBrowseV2.spec.tsx is the precedent that stubs usePageVisibility —
// DownloadPanel.spec.tsx does NOT, and copying that one breaks this file.
vi.mock('../../hooks/useDownloads', () => ({
  usePageVisibility: () => h.visible,
}));

vi.mock('../../services/subtitleService', () => ({
  subtitleService: {
    getGenerationBatchStatus: vi.fn(),
    previewGenerationBatch: vi.fn(),
    cancelGenerationBatch: vi.fn(),
    dismissGenerationBatch: vi.fn(),
  },
}));

import { GenerationWorkspace } from './GenerationWorkspaceV2';
import { subtitleService } from '../../services/subtitleService';

const mocked = vi.mocked(subtitleService);

const item = (
  mediaId: string,
  title: string,
  status: GenerationBatchItemState['status'],
  reason: GenerationBatchItemState['reason'] = ''
): GenerationBatchItemState => ({
  mediaId,
  title,
  mediaType: 'movie',
  seriesTitle: '',
  status,
  reason,
});

const RUNNING_SNAPSHOT = {
  batchId: 'gb-1',
  totalItems: 3,
  currentIndex: 1,
  currentMediaId: 'm1',
  currentItem: '沙丘：第二部',
  successCount: 0,
  failCount: 0,
  pausedCount: 0,
  status: 'running' as const,
  spentUsd: 0,
  budgetUsd: 5,
  items: [item('m1', '沙丘：第二部', 'running'), item('m2', '奧本海默', 'queued')],
};

const LAST_SNAPSHOT = {
  ...RUNNING_SNAPSHOT,
  batchId: 'gb-done',
  status: 'complete' as const,
  successCount: 1,
  failCount: 1,
  currentMediaId: '',
  items: [item('m1', '沙丘：第二部', 'done'), item('m2', '奧本海默', 'failed', 'busy_elsewhere')],
};

function renderWorkspace() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const invalidate = vi.spyOn(queryClient, 'invalidateQueries');
  const makeUi = () => (
    <QueryClientProvider client={queryClient}>
      <GenerationWorkspace active onLaunch={vi.fn()} />
    </QueryClientProvider>
  );
  const view = render(makeUi());
  return { queryClient, invalidate, rerender: () => view.rerender(makeUi()) };
}

beforeEach(() => {
  vi.clearAllMocks();
  h.batchState.batchId = '';
  h.batchState.status = 'idle';
  h.batchState.currentMediaId = null;
  h.batchState.currentItem = '';
  h.batchState.totalItems = 0;
  h.batchState.successCount = 0;
  h.batchState.failCount = 0;
  h.batchState.pausedCount = 0;
  h.batchState.items = null;
  h.batchEpoch = 0;
  h.singleJobs = {};
  h.jobsConnected = true;
  h.visible = true;
  mocked.getGenerationBatchStatus.mockResolvedValue({ running: false, progress: null, last: null });
  mocked.previewGenerationBatch.mockResolvedValue({ totalItems: 12 });
});

describe('GenerationWorkspace — the status probe (dsr-6d-c-1 AC #3)', () => {
  it('[P0] a RUNNING batch is tracked (stream attached)', async () => {
    mocked.getGenerationBatchStatus.mockResolvedValue({
      running: true,
      progress: RUNNING_SNAPSHOT,
      last: null,
    });

    renderWorkspace();

    await waitFor(() => expect(h.batchStartTracking).toHaveBeenCalledWith(RUNNING_SNAPSHOT));
    expect(h.batchAttachSnapshot).not.toHaveBeenCalled();
  });

  it('[P0] 🔴 #1 a FINISHED batch is ATTACHED from `last` — the queue survives the dialog removing its cache', async () => {
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: false, last: LAST_SNAPSHOT });

    renderWorkspace();

    await waitFor(() => expect(h.batchAttachSnapshot).toHaveBeenCalledWith(LAST_SNAPSHOT));
    expect(h.batchStartTracking).not.toHaveBeenCalled();
  });

  it('[P0] CR H1 leaving the tab and coming back RE-ATTACHES the result — it must not vanish', async () => {
    // Each probe returns a NEW object, exactly as a refetch does.
    mocked.getGenerationBatchStatus.mockImplementation(async () => ({
      running: false,
      progress: null,
      last: { ...LAST_SNAPSHOT },
    }));
    const { rerender } = renderWorkspace();
    await waitFor(() => expect(h.batchAttachSnapshot).toHaveBeenCalledTimes(1));

    // Tab hidden → the hook resets; tab visible again → the result must come back,
    // because the 關閉 button (the ONLY dismiss caller) lives on that screen.
    h.visible = false;
    h.batchState.status = 'idle';
    rerender();
    h.visible = true;
    rerender();

    await waitFor(() => expect(h.batchAttachSnapshot).toHaveBeenCalledTimes(2));
  });

  it('[P0] but a result the user CLOSED is not re-attached by a probe that still carries it', async () => {
    mocked.getGenerationBatchStatus.mockImplementation(async () => ({
      running: false,
      progress: null,
      last: { ...LAST_SNAPSHOT },
    }));
    mocked.dismissGenerationBatch.mockResolvedValue({ dismissed: true, running: false });
    h.batchState.status = 'complete';
    h.batchState.batchId = 'gb-done';
    h.batchState.items = LAST_SNAPSHOT.items;
    const { rerender } = renderWorkspace();

    await waitFor(() => expect(h.batchAttachSnapshot).toHaveBeenCalledTimes(1));
    fireEvent.click(await screen.findByTestId('workspace-close'));
    await waitFor(() => expect(mocked.dismissGenerationBatch).toHaveBeenCalled());

    h.batchState.status = 'idle';
    rerender();
    h.visible = false;
    rerender();
    h.visible = true;
    rerender();

    expect(h.batchAttachSnapshot).toHaveBeenCalledTimes(1);
  });

  it('[P0] CR H3 a probe that resolves AFTER the terminal must not force the batch back to running', async () => {
    let resolveProbe: (v: unknown) => void = () => {};
    mocked.getGenerationBatchStatus.mockReturnValue(
      new Promise((res) => {
        resolveProbe = res as (v: unknown) => void;
      }) as never
    );
    // The terminal SSE already landed while the probe was in flight.
    h.batchState.status = 'complete';
    h.batchState.batchId = 'gb-1';
    renderWorkspace();
    await waitFor(() => expect(mocked.getGenerationBatchStatus).toHaveBeenCalled());

    resolveProbe({ running: true, progress: RUNNING_SNAPSHOT, last: null });

    await waitFor(() => expect(screen.getByTestId('generation-workspace')).toBeInTheDocument());
    // Re-seeding would set status:'running' AND reopen a dead EventSource.
    expect(h.batchStartTracking).not.toHaveBeenCalled();
  });

  it('[P0] 🔴 #11 the probe is never served from a stale cache — staleTime 0 + refetch on focus', async () => {
    mocked.getGenerationBatchStatus.mockResolvedValue({
      running: true,
      progress: RUNNING_SNAPSHOT,
      last: null,
    });
    const { queryClient } = renderWorkspace();

    await waitFor(() => expect(mocked.getGenerationBatchStatus).toHaveBeenCalled());
    const q = queryClient
      .getQueryCache()
      .findAll()
      .find(
        (entry) =>
          JSON.stringify(entry.queryKey) ===
          JSON.stringify(['subtitles', 'generation-batch', 'status'])
      );
    expect(q).toBeDefined();
    expect(q!.options.staleTime).toBe(0);
    expect(q!.options.refetchOnWindowFocus).toBe('always');
  });

  it('[P0] a reconnect (connectionEpoch tick) re-reads status — the cure for a lost terminal event', async () => {
    mocked.getGenerationBatchStatus.mockResolvedValue({
      running: true,
      progress: RUNNING_SNAPSHOT,
      last: null,
    });
    const { rerender } = renderWorkspace();
    await waitFor(() => expect(mocked.getGenerationBatchStatus).toHaveBeenCalledTimes(1));

    h.batchEpoch = 1;
    rerender();

    await waitFor(() => expect(mocked.getGenerationBatchStatus).toHaveBeenCalledTimes(2));
  });
});

describe('GenerationWorkspace — per-item subscription (dsr-6d-c-1 🔴 #8)', () => {
  it('[P0] the stepper follows the batch to the NEXT title', async () => {
    mocked.getGenerationBatchStatus.mockResolvedValue({
      running: true,
      progress: RUNNING_SNAPSHOT,
      last: null,
    });
    h.batchState.status = 'running';
    h.batchState.currentMediaId = 'm1';
    h.batchState.items = RUNNING_SNAPSHOT.items;
    const { rerender } = renderWorkspace();

    await waitFor(() => expect(h.itemStartTracking).toHaveBeenCalledWith('m1'));

    // The batch moves on — the old code stayed subscribed to m1 for ever.
    h.batchState.currentMediaId = 'm2';
    rerender();

    await waitFor(() => expect(h.itemStartTracking).toHaveBeenCalledWith('m2'));
  });

  it('[P1] it does not subscribe while the batch is not running', async () => {
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: false, last: LAST_SNAPSHOT });
    h.batchState.status = 'complete';
    h.batchState.currentMediaId = 'm2';

    renderWorkspace();

    await waitFor(() => expect(h.batchAttachSnapshot).toHaveBeenCalled());
    expect(h.itemStartTracking).not.toHaveBeenCalled();
  });
});

describe('GenerationWorkspace — terminal refresh (dsr-6d-c-1 🔴 #18)', () => {
  const keysOf = (spy: ReturnType<typeof vi.spyOn>) =>
    spy.mock.calls.map((c) => JSON.stringify((c[0] as { queryKey: unknown }).queryKey));

  it('[P0] a terminal refreshes library, preview, details, activity and the per-title quote', async () => {
    h.batchState.status = 'complete';
    h.batchState.batchId = 'gb-t1';
    const { invalidate } = renderWorkspace();

    await waitFor(() => expect(invalidate).toHaveBeenCalled());
    const keys = keysOf(invalidate);
    expect(keys).toContain(JSON.stringify(['library']));
    expect(keys).toContain(JSON.stringify(['subtitles', 'generation-batch', 'preview']));
    expect(keys).toContain(JSON.stringify(['details']));
    expect(keys).toContain(JSON.stringify(['activity']));
    expect(keys).toContain(JSON.stringify(['subtitles', 'transcription-estimate']));
  });

  it('[P0] the same terminal batch refreshes ONCE, not on every render', async () => {
    h.batchState.status = 'complete';
    h.batchState.batchId = 'gb-t2';
    const { invalidate, rerender } = renderWorkspace();

    await waitFor(() => expect(invalidate).toHaveBeenCalled());
    const first = invalidate.mock.calls.length;

    h.batchState.status = 'idle';
    rerender();
    h.batchState.status = 'complete';
    rerender();
    await waitFor(() => expect(screen.getByTestId('generation-workspace')).toBeInTheDocument());

    expect(invalidate.mock.calls.length).toBe(first);
  });
});

describe('GenerationWorkspace — cancel + dismiss (dsr-6d-c-1 🔴 #13/#14)', () => {
  it('[P0] cancelling asks first, then calls the endpoint', async () => {
    mocked.getGenerationBatchStatus.mockResolvedValue({
      running: true,
      progress: RUNNING_SNAPSHOT,
      last: null,
    });
    mocked.cancelGenerationBatch.mockResolvedValue({ cancelled: true, running: false });
    h.batchState.status = 'running';
    h.batchState.items = RUNNING_SNAPSHOT.items;

    renderWorkspace();

    fireEvent.click(await screen.findByTestId('workspace-cancel-all'));
    expect(mocked.cancelGenerationBatch).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId('workspace-cancel-confirm-btn'));
    await waitFor(() => expect(mocked.cancelGenerationBatch).toHaveBeenCalledOnce());
  });

  it('[P0] a cancel that FAILS surfaces the alert instead of being swallowed', async () => {
    mocked.getGenerationBatchStatus.mockResolvedValue({
      running: true,
      progress: RUNNING_SNAPSHOT,
      last: null,
    });
    mocked.cancelGenerationBatch.mockRejectedValue(new Error('network'));
    h.batchState.status = 'running';
    h.batchState.items = RUNNING_SNAPSHOT.items;

    renderWorkspace();

    fireEvent.click(await screen.findByTestId('workspace-cancel-all'));
    fireEvent.click(screen.getByTestId('workspace-cancel-confirm-btn'));

    expect(await screen.findByTestId('workspace-cancel-error')).toHaveTextContent(
      '取消失敗，批次仍在進行。請再試一次。'
    );
  });

  it('[P0] 🔴 #14 關閉 calls dismiss — the first caller in the app', async () => {
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: false, last: LAST_SNAPSHOT });
    mocked.dismissGenerationBatch.mockResolvedValue({ dismissed: true, running: false });
    h.batchState.status = 'complete';
    h.batchState.batchId = 'gb-done';
    h.batchState.items = LAST_SNAPSHOT.items;

    renderWorkspace();

    fireEvent.click(await screen.findByTestId('workspace-close'));
    await waitFor(() => expect(mocked.dismissGenerationBatch).toHaveBeenCalledOnce());
    await waitFor(() => expect(h.batchReset).toHaveBeenCalled());
  });

  it('[P0] a dismiss refused because the batch is still running is reported, not pretended', async () => {
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: false, last: LAST_SNAPSHOT });
    mocked.dismissGenerationBatch.mockResolvedValue({ dismissed: false, running: true });
    h.batchState.status = 'complete';
    h.batchState.batchId = 'gb-done';
    h.batchState.items = LAST_SNAPSHOT.items;

    renderWorkspace();

    fireEvent.click(await screen.findByTestId('workspace-close'));
    expect(await screen.findByTestId('workspace-dismiss-error')).toHaveTextContent(
      '批次還在進行中，現在無法關閉。'
    );
  });
});

describe('GenerationWorkspace — the idle count (dsr-6d-c-1 🔴 #17)', () => {
  it('[P0] prefers the episode-inclusive count over the movies-only one', async () => {
    mocked.previewGenerationBatch.mockResolvedValue({
      totalItems: 12,
      totalItemsIncludingEpisodes: 38,
    });

    renderWorkspace();

    const idle = await screen.findByTestId('workspace-idle');
    await waitFor(() => expect(idle).toHaveTextContent('38'));
    expect(idle).not.toHaveTextContent('12');
  });

  it('[P1] falls back to total_items on a server that has no episode count', async () => {
    mocked.previewGenerationBatch.mockResolvedValue({ totalItems: 12 });

    renderWorkspace();

    const idle = await screen.findByTestId('workspace-idle');
    await waitFor(() => expect(idle).toHaveTextContent('12'));
  });
});

describe('GenerationWorkspace — the live log wiring (dsr-6d-c-2 AC #4)', () => {
  it('[P0] a RUNNING batch tells the log which films are in it (a mid-batch attach knows the titles)', async () => {
    mocked.getGenerationBatchStatus.mockResolvedValue({
      running: true,
      progress: RUNNING_SNAPSHOT,
      last: null,
    });

    renderWorkspace();

    await waitFor(() => expect(h.jobsSeed).toHaveBeenCalledWith('gb-1', RUNNING_SNAPSHOT.items));
  });

  it('[P0] a probe ignored by the CR H3 guard does not seed the log either', async () => {
    let resolveProbe: (v: unknown) => void = () => {};
    mocked.getGenerationBatchStatus.mockReturnValue(
      new Promise((res) => {
        resolveProbe = res as (v: unknown) => void;
      }) as never
    );
    h.batchState.status = 'complete';
    h.batchState.batchId = 'gb-1';
    renderWorkspace();
    await waitFor(() => expect(mocked.getGenerationBatchStatus).toHaveBeenCalled());

    resolveProbe({ running: true, progress: RUNNING_SNAPSHOT, last: null });

    await waitFor(() => expect(screen.getByTestId('generation-workspace')).toBeInTheDocument());
    expect(h.jobsSeed).not.toHaveBeenCalled();
  });

  it('[P1] a FINISHED batch (`last`) has no running members to seed', async () => {
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: false, last: LAST_SNAPSHOT });

    renderWorkspace();

    await waitFor(() => expect(h.batchAttachSnapshot).toHaveBeenCalled());
    expect(h.jobsSeed).not.toHaveBeenCalled();
  });

  it('[P0] a single job row shows the backend title — never the stage sentence', async () => {
    h.singleJobs = {
      m9: {
        mediaId: 'm9',
        phase: 'transcribing',
        title: '芭比',
        message: '正在轉錄音訊',
        percentage: null,
      },
    };

    renderWorkspace();

    const row = await screen.findByTestId('workspace-queue-row-m9');
    // The row's NAME — the stepper under it may still show the stage sentence.
    expect(row.querySelector('.font-semibold.text-base')).toHaveTextContent(/^芭比$/);
  });

  it('[P0] an untitled single job says 處理中的項目 — not the stage sentence, not a UUID', async () => {
    h.singleJobs = {
      'uuid-9': {
        mediaId: 'uuid-9',
        phase: 'transcribing',
        title: '',
        message: '正在轉錄音訊',
        percentage: null,
      },
    };

    renderWorkspace();

    const single = await screen.findByTestId('workspace-single');
    expect(single).toHaveTextContent('處理中的項目');
    expect(single).not.toHaveTextContent('uuid-9');
  });

  it('[P0] 🔴 #14 the log chip follows the jobs stream, not the batch', async () => {
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: false, last: LAST_SNAPSHOT });
    h.batchState.status = 'complete';
    h.batchState.batchId = 'gb-done';
    h.jobsConnected = false;

    const { rerender } = renderWorkspace();
    const log = await screen.findByTestId('workspace-event-log');
    expect(within(log).queryByTestId('workspace-sse-chip')).not.toBeInTheDocument();

    h.jobsConnected = true;
    rerender();
    expect(
      within(screen.getByTestId('workspace-event-log')).getByTestId('workspace-sse-chip')
    ).toBeInTheDocument();
  });
});

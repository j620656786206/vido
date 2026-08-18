import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

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
  },
  batchStartTracking: vi.fn(),
  batchReset: vi.fn(),
  itemState: {
    phase: 'idle' as string,
    failedPhase: null as string | null,
    percentage: null as number | null,
    message: '',
    jobId: null,
    error: null as string | null,
    srtPath: null,
    zhSrtPath: null,
  },
  itemStartTracking: vi.fn(),
  itemReset: vi.fn(),
}));

vi.mock('../../hooks/useGenerationBatchProgress', () => ({
  useGenerationBatchProgress: () => ({
    progress: h.batchState,
    status: h.batchState.status,
    startTracking: h.batchStartTracking,
    reset: h.batchReset,
  }),
}));

vi.mock('../../hooks/useGenerationProgress', () => ({
  useGenerationProgress: () => ({
    progress: h.itemState,
    startTracking: h.itemStartTracking,
    reset: h.itemReset,
  }),
}));

vi.mock('../../services/subtitleService', () => ({
  subtitleService: {
    startGenerationBatch: vi.fn(),
    getGenerationBatchStatus: vi.fn(),
    cancelGenerationBatch: vi.fn(),
    previewGenerationBatch: vi.fn(),
    getGenerationCandidates: vi.fn(),
    startCandidateAnalysis: vi.fn(),
    cancelCandidateAnalysis: vi.fn(),
  },
}));

// The consent flow has its own spec suite (consent/*.spec.tsx) — the container
// tests drive it through a shallow stub that exposes onStartBatch/onClose and
// surfaces the startError prop.
vi.mock('./consent/GenerationConsentView', () => ({
  GenerationConsentView: ({
    preselectedIds,
    forceAnalyze,
    startError,
    onStartBatch,
    onClose,
  }: {
    preselectedIds?: string[];
    forceAnalyze?: boolean;
    startError?: string | null;
    onStartBatch: (ids: string[], budgetUsd: number) => void;
    onClose: () => void;
  }) => (
    <div
      data-testid="consent-view-stub"
      data-preselected={(preselectedIds ?? []).join(',')}
      data-force-analyze={forceAnalyze ? 'true' : 'false'}
    >
      {startError && <p data-testid="consent-stub-error">{startError}</p>}
      <button
        type="button"
        data-testid="consent-stub-start"
        onClick={() =>
          onStartBatch(
            preselectedIds && preselectedIds.length > 0
              ? preselectedIds
              : ['4f8c2d1a-5b6e-4c7d-8e9f-0a1b2c3d4e51'],
            5
          )
        }
      >
        stub-start
      </button>
      <button type="button" data-testid="consent-stub-close" onClick={onClose}>
        stub-close
      </button>
    </div>
  ),
}));

import {
  GenerationBatchDialogV2,
  GenerationBatchPanelV2,
  deriveRowStates,
  failedRowIds,
  remainingIds,
  type GenerationBatchPanelV2Props,
} from './GenerationBatchDialogV2';
import { subtitleService, type GenerationBatchItem } from '../../services/subtitleService';
import type { GenerationBatchProgressState } from '../../hooks/useGenerationBatchProgress';

const mocked = vi.mocked(subtitleService);

// Media-id fixture convention (9R-18 AC 7): media ids are UUID STRINGS —
// mirror the prod creation path (uuid.New().String()); do NOT invent numeric ids.
const M1 = '4f8c2d1a-5b6e-4c7d-8e9f-0a1b2c3d4e51';
const M2 = '4f8c2d1a-5b6e-4c7d-8e9f-0a1b2c3d4e52';
const M3 = '4f8c2d1a-5b6e-4c7d-8e9f-0a1b2c3d4e53';
const M4 = '4f8c2d1a-5b6e-4c7d-8e9f-0a1b2c3d4e54';
const M5 = '4f8c2d1a-5b6e-4c7d-8e9f-0a1b2c3d4e55';
const E9 = '8fa9fed7-8fbc-4e8d-8edc-f6b7c8d9e006';

const ITEMS: GenerationBatchItem[] = [
  { mediaId: M1, title: '沙丘：第二部', mediaType: 'movie' },
  { mediaId: M2, title: '奧本海默', mediaType: 'movie' },
  { mediaId: M3, title: '怪奇物語 S04E07', mediaType: 'episode' },
  { mediaId: M4, title: '星際效應', mediaType: 'movie' },
  { mediaId: M5, title: '全面啟動', mediaType: 'movie' },
];

function progressOf(p: Partial<GenerationBatchProgressState>): GenerationBatchProgressState {
  return {
    batchId: 'gb-1',
    totalItems: 5,
    currentIndex: 3,
    currentMediaId: M3,
    currentItem: '怪奇物語 S04E07',
    successCount: 2,
    failCount: 0,
    pausedCount: 0,
    status: 'running',
    spentUsd: 0.42,
    budgetUsd: 5,
    ...p,
  };
}

// ---------------------------------------------------------------------------
// deriveRowStates — the batch-status-authoritative join (9R-16 CR caveat)
// ---------------------------------------------------------------------------

describe('deriveRowStates', () => {
  it('running: resolved before the active row, active on current_media_id, queued after', () => {
    expect(deriveRowStates(ITEMS, progressOf({}), new Set())).toEqual([
      'done',
      'done',
      'active',
      'queued',
      'queued',
    ]);
  });

  it('running: per-item failures mark resolved rows 失敗', () => {
    expect(deriveRowStates(ITEMS, progressOf({ failCount: 1 }), new Set([M2]))).toEqual([
      'done',
      'failed',
      'active',
      'queued',
      'queued',
    ]);
  });

  it('budget_ceiling: paused_count is AUTHORITATIVE — the interrupted in-flight item renders paused even when its per-item pipeline emitted a terminal failure', () => {
    const progress = progressOf({ status: 'budget_ceiling', pausedCount: 3, spentUsd: 5 });
    expect(deriveRowStates(ITEMS, progress, new Set([M3]))).toEqual([
      'done',
      'done',
      'paused',
      'paused',
      'paused',
    ]);
  });

  it('cancelled: rows from the in-flight item on render 已取消, earlier failures stay 失敗', () => {
    const progress = progressOf({ status: 'cancelled', failCount: 1 });
    expect(deriveRowStates(ITEMS, progress, new Set([M1, M3]))).toEqual([
      'failed',
      'done',
      'stopped',
      'stopped',
      'stopped',
    ]);
  });

  it('complete: all rows resolved via the failure set', () => {
    const progress = progressOf({ status: 'complete', successCount: 4, failCount: 1 });
    expect(deriveRowStates(ITEMS, progress, new Set([M4]))).toEqual([
      'done',
      'done',
      'done',
      'failed',
      'done',
    ]);
  });
});

// ---------------------------------------------------------------------------
// Panel — prop-driven state matrix (execution states only; idle belongs to the
// sub-4-3 consent flow and never reaches this panel)
// ---------------------------------------------------------------------------

function renderPanel(props: Partial<GenerationBatchPanelV2Props> = {}) {
  const merged: GenerationBatchPanelV2Props = {
    open: true,
    status: 'running',
    progress: progressOf({ status: 'running' }),
    items: [],
    onConfirmCancelAll: vi.fn(),
    onResume: vi.fn(),
    onClose: vi.fn(),
    ...props,
  };
  render(<GenerationBatchPanelV2 {...merged} />);
  return merged;
}

describe('GenerationBatchPanelV2', () => {
  it('renders the mobile bottom-sheet drag handle, hidden on the desktop dialog (F8-M-v2 H717g)', () => {
    renderPanel({ items: ITEMS });
    const handle = screen.getByTestId('gen-batch-drag-handle');
    expect(handle).toBeInTheDocument();
    expect(handle.className).toContain('sm:hidden');
  });

  it('running renders queue rows from items[], the counter, cost line and SSE chip', () => {
    renderPanel({
      status: 'running',
      progress: progressOf({ successCount: 2, failCount: 0 }),
      items: ITEMS,
    });

    expect(screen.getByText('產生字幕')).toBeInTheDocument();
    expect(screen.getByTestId('gen-batch-counter')).toHaveTextContent('2 / 5');
    expect(screen.getByTestId('gen-batch-item-list').children).toHaveLength(5);
    expect(screen.getByTestId(`gen-batch-row-${M3}`)).toHaveAttribute('data-state', 'active');
    expect(screen.getByTestId(`gen-batch-row-${M1}`)).toHaveTextContent('完成');
    expect(screen.getByTestId(`gen-batch-row-${M5}`)).toHaveTextContent('排隊中');
    expect(screen.getByTestId('generation-progress-v2')).toBeInTheDocument();
    expect(screen.getByTestId('gen-batch-cost-line')).toHaveTextContent(
      '本次用量：$0.42 / 上限 $5.00'
    );
    expect(screen.getByTestId('gen-batch-sse-chip')).toHaveTextContent('即時更新（SSE）');
    expect(screen.getByRole('progressbar', { name: '批次生成進度' })).toHaveAttribute(
      'aria-valuenow',
      '2'
    );
    // The consented batch is always scope=selected (sub-4-3).
    expect(screen.getByText(/範圍：已選項目（/)).toBeInTheDocument();
  });

  it('全部取消 uses an inline confirm before cancelling', () => {
    const props = renderPanel({ status: 'running', progress: progressOf({}), items: ITEMS });

    fireEvent.click(screen.getByTestId('gen-batch-cancel-all'));
    expect(screen.getByTestId('gen-batch-cancel-confirm')).toBeInTheDocument();
    expect(props.onConfirmCancelAll).not.toHaveBeenCalled();

    fireEvent.click(screen.getByTestId('gen-batch-cancel-confirm-btn'));
    expect(props.onConfirmCancelAll).toHaveBeenCalledOnce();
  });

  it('budget_ceiling renders the F9 banner, paused rows, 關閉 + 下次繼續', () => {
    const props = renderPanel({
      status: 'budget_ceiling',
      progress: progressOf({
        status: 'budget_ceiling',
        successCount: 2,
        pausedCount: 3,
        spentUsd: 5,
      }),
      items: ITEMS,
    });

    const banner = screen.getByTestId('gen-batch-budget-banner');
    expect(banner).toHaveTextContent('已達本次預算上限（$5.00）— 已完成2部，剩餘3部下次繼續');
    expect(screen.getByTestId(`gen-batch-row-${M4}`)).toHaveTextContent('已暫停 — 下次繼續');
    expect(screen.getByTestId('gen-batch-close-btn')).toHaveTextContent('關閉');

    fireEvent.click(screen.getByTestId('gen-batch-resume-btn'));
    expect(props.onResume).toHaveBeenCalledOnce();
    expect(banner.className).toContain('warning-tint');
    expect(banner.className).not.toContain('error-tint');
  });

  it('recover-attach fallback card honors terminal semantics — budget_ceiling renders 已暫停, not 已取消', () => {
    renderPanel({
      status: 'budget_ceiling',
      progress: progressOf({
        status: 'budget_ceiling',
        currentMediaId: M3,
        currentItem: '怪奇物語 S04E07',
        pausedCount: 3,
        spentUsd: 5,
      }),
      items: [], // 409/recover-attach: the status probe carries no items[]
    });

    const card = screen.getByTestId(`gen-batch-row-${M3}`);
    expect(card).toHaveAttribute('data-state', 'paused');
    expect(card).toHaveTextContent('已暫停 — 下次繼續');
    expect(card).not.toHaveTextContent('已取消');
  });

  it('recover-attach fallback card resolves 完成 on complete (not 已取消)', () => {
    renderPanel({
      status: 'complete',
      progress: progressOf({ status: 'complete', currentMediaId: M5, currentItem: '全面啟動' }),
      items: [],
    });

    expect(screen.getByTestId(`gen-batch-row-${M5}`)).toHaveAttribute('data-state', 'done');
    expect(screen.getByTestId(`gen-batch-row-${M5}`)).toHaveTextContent('完成');
  });

  it('running without items[] (409/recover-attach) falls back to the in-flight item card', () => {
    renderPanel({
      status: 'running',
      progress: progressOf({ currentMediaId: M3, currentItem: '怪奇物語 S04E07' }),
      items: [],
    });

    expect(screen.getByTestId(`gen-batch-row-${M3}`)).toHaveAttribute('data-state', 'active');
    expect(screen.getByTestId(`gen-batch-row-${M3}`)).toHaveTextContent('怪奇物語 S04E07');
  });

  it('Escape is gated while running', () => {
    const props = renderPanel({ status: 'running', progress: progressOf({}), items: ITEMS });

    fireEvent.keyDown(screen.getByTestId('generation-batch-dialog-v2'), { key: 'Escape' });
    expect(props.onClose).not.toHaveBeenCalled();
  });

  it('Escape closes on a terminal state', () => {
    const props = renderPanel({
      status: 'complete',
      progress: progressOf({ status: 'complete' }),
      items: ITEMS,
    });

    fireEvent.keyDown(screen.getByTestId('generation-batch-dialog-v2'), { key: 'Escape' });
    expect(props.onClose).toHaveBeenCalledOnce();
  });
});

// ---------------------------------------------------------------------------
// Container — consent flow (idle) ⇄ execution panel (running+) wiring
// ---------------------------------------------------------------------------

function renderDialog(props: Partial<React.ComponentProps<typeof GenerationBatchDialogV2>> = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const merged: React.ComponentProps<typeof GenerationBatchDialogV2> = {
    open: true,
    onOpenChange: vi.fn(),
    ...props,
  };
  render(
    <QueryClientProvider client={queryClient}>
      <GenerationBatchDialogV2 {...merged} />
    </QueryClientProvider>
  );
  return merged;
}

describe('GenerationBatchDialogV2 (container)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    h.batchState.status = 'idle';
    h.batchState.currentMediaId = null;
    h.batchState.pausedCount = 0;
    h.itemState.phase = 'idle';
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: false, progress: null });
  });

  it('idle renders the consent flow AFTER the recovery probe settles, not the execution panel (AC #9 + CR M4)', async () => {
    let resolveProbe: (v: { running: boolean; progress: null }) => void = () => {};
    mocked.getGenerationBatchStatus.mockReturnValue(
      new Promise((res) => {
        resolveProbe = res;
      })
    );
    renderDialog({ selectedMediaIds: [M1, E9] });

    // CR M4: before the probe settles NOTHING renders — the consent bootstrap
    // must not kick a library probe sweep while a batch might be running.
    expect(screen.queryByTestId('consent-view-stub')).not.toBeInTheDocument();

    resolveProbe({ running: false, progress: null });
    const stub = await screen.findByTestId('consent-view-stub');
    expect(stub).toHaveAttribute('data-preselected', `${M1},${E9}`);
    expect(stub).toHaveAttribute('data-force-analyze', 'false');
    expect(screen.queryByTestId('generation-batch-dialog-v2')).not.toBeInTheDocument();
  });

  it('[CR H2] forceAnalyze prop (F17 deep link) reaches the consent view', async () => {
    renderDialog({ forceAnalyze: true });
    const stub = await screen.findByTestId('consent-view-stub');
    expect(stub).toHaveAttribute('data-force-analyze', 'true');
  });

  it('recovers an already-running batch on open via the status probe (409-recover)', async () => {
    const snapshot = progressOf({ status: 'running' });
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: true, progress: snapshot });

    renderDialog();

    await waitFor(() => expect(h.batchStartTracking).toHaveBeenCalledWith(snapshot));
  });

  it('[P0 AC #4] consented start sends scope=selected + mixed media_ids + budget_usd and seeds tracking', async () => {
    mocked.startGenerationBatch.mockResolvedValue({
      conflict: false,
      result: { batchId: 'gb-9', totalItems: 2, items: ITEMS.slice(0, 2) },
    });

    renderDialog({ selectedMediaIds: [M1, E9] });

    fireEvent.click(await screen.findByTestId('consent-stub-start'));

    await waitFor(() =>
      expect(mocked.startGenerationBatch).toHaveBeenCalledWith({
        scope: 'selected',
        mediaIds: [M1, E9],
        budgetUsd: 5,
      })
    );
    expect(h.batchStartTracking).toHaveBeenCalledWith({ batchId: 'gb-9', totalItems: 2 });
  });

  it('attaches to the 409 in-progress snapshot instead of erroring', async () => {
    const snapshot = progressOf({ status: 'running' });
    mocked.startGenerationBatch.mockResolvedValue({ conflict: true, progress: snapshot });

    renderDialog();

    fireEvent.click(await screen.findByTestId('consent-stub-start'));

    await waitFor(() => expect(h.batchStartTracking).toHaveBeenCalledWith(snapshot));
    expect(screen.queryByTestId('consent-stub-error')).not.toBeInTheDocument();
  });

  it('start failure surfaces the error to the consent view (400 selection reject path)', async () => {
    mocked.startGenerationBatch.mockRejectedValue(
      new Error('media_ids 含無法生成字幕的項目（查無此電影或影集，或沒有媒體檔案）')
    );

    renderDialog();

    fireEvent.click(await screen.findByTestId('consent-stub-start'));

    await waitFor(() =>
      expect(screen.getByTestId('consent-stub-error')).toHaveTextContent(
        'media_ids 含無法生成字幕的項目'
      )
    );
  });

  it('joins the per-item stream on current_media_id while running — episode UUIDs included (AC #8)', async () => {
    h.batchState.status = 'running';
    h.batchState.currentMediaId = E9;

    renderDialog();

    await waitFor(() => expect(h.itemStartTracking).toHaveBeenCalledWith(E9));
  });

  it('cancel calls the cancel endpoint (terminal arrives via SSE)', async () => {
    h.batchState.status = 'running';
    h.batchState.currentMediaId = M3;
    h.batchState.currentItem = '怪奇物語 S04E07';
    mocked.cancelGenerationBatch.mockResolvedValue({ cancelled: true, running: false });

    renderDialog();

    fireEvent.click(screen.getByTestId('gen-batch-cancel-all'));
    fireEvent.click(screen.getByTestId('gen-batch-cancel-confirm-btn'));

    await waitFor(() => expect(mocked.cancelGenerationBatch).toHaveBeenCalledOnce());
  });

  // -------------------------------------------------------------------------
  // The 9R-16 CR race through the REAL component: on cancelled/budget_ceiling
  // the interrupted in-flight item ALSO emits a terminal per-item event —
  // whichever order the two events arrive, the row must end up 已暫停.
  // -------------------------------------------------------------------------

  function renderDialogRaw() {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const onOpenChange = vi.fn();
    const makeUi = () => (
      <QueryClientProvider client={queryClient}>
        <GenerationBatchDialogV2 open onOpenChange={onOpenChange} />
      </QueryClientProvider>
    );
    const view = render(makeUi());
    return { rerender: () => view.rerender(makeUi()) };
  }

  async function startBatchWithItems(rerender: () => void) {
    mocked.startGenerationBatch.mockResolvedValue({
      conflict: false,
      result: { batchId: 'gb-race', totalItems: 5, items: ITEMS },
    });
    fireEvent.click(await screen.findByTestId('consent-stub-start'));
    await waitFor(() => expect(h.batchStartTracking).toHaveBeenCalled());
    // The batch SSE stream reports item 3 in flight.
    h.batchState.status = 'running';
    h.batchState.totalItems = 5;
    h.batchState.currentMediaId = M3;
    h.batchState.currentItem = '怪奇物語 S04E07';
    rerender();
  }

  it('race order A — per-item failed arrives BEFORE the terminal batch event: paused wins at render', async () => {
    const { rerender } = renderDialogRaw();
    await startBatchWithItems(rerender);

    h.itemState.phase = 'failed';
    rerender();
    await waitFor(() =>
      expect(screen.getByTestId(`gen-batch-row-${M3}`)).toHaveAttribute('data-state', 'failed')
    );

    h.batchState.status = 'budget_ceiling';
    h.batchState.pausedCount = 3;
    h.batchState.spentUsd = 5;
    rerender();
    await waitFor(() =>
      expect(screen.getByTestId(`gen-batch-row-${M3}`)).toHaveAttribute('data-state', 'paused')
    );
    expect(screen.getByTestId(`gen-batch-row-${M3}`)).toHaveTextContent('已暫停 — 下次繼續');
    expect(screen.getByTestId(`gen-batch-row-${M3}`)).not.toHaveTextContent('失敗');
  });

  it('race order B — per-item failed arrives AFTER the terminal batch event: never recorded, row stays paused', async () => {
    const { rerender } = renderDialogRaw();
    await startBatchWithItems(rerender);

    h.batchState.status = 'budget_ceiling';
    h.batchState.pausedCount = 3;
    h.batchState.spentUsd = 5;
    rerender();
    await waitFor(() =>
      expect(screen.getByTestId(`gen-batch-row-${M3}`)).toHaveAttribute('data-state', 'paused')
    );

    h.itemState.phase = 'failed';
    rerender();
    await waitFor(() =>
      expect(screen.getByTestId(`gen-batch-row-${M3}`)).toHaveAttribute('data-state', 'paused')
    );
    expect(screen.getByTestId(`gen-batch-row-${M3}`)).not.toHaveTextContent('失敗');
  });

  it('下次繼續 returns to the consent flow — a resume is a NEW consent, never an auto-restart (sub-4-3)', async () => {
    h.batchState.status = 'budget_ceiling';
    h.batchState.pausedCount = 3;

    renderDialog();

    fireEvent.click(screen.getByTestId('gen-batch-resume-btn'));

    expect(h.batchReset).toHaveBeenCalled();
    expect(mocked.startGenerationBatch).not.toHaveBeenCalled();
  });

  it('[CR H2] after a batch terminal the next consent render forces a re-analysis (stale snapshot ban)', async () => {
    // Reach a terminal state → the container marks the snapshot stale.
    h.batchState.status = 'complete';
    const { rerender } = renderDialogRaw();
    await waitFor(() => expect(screen.getByTestId('gen-batch-close-btn')).toBeInTheDocument());

    // The batch hook resets to idle (e.g. 下次繼續) — consent view re-mounts
    // with forceAnalyze so the completed items cannot be re-quoted.
    h.batchState.status = 'idle';
    rerender();
    const stub = await screen.findByTestId('consent-view-stub');
    expect(stub).toHaveAttribute('data-force-analyze', 'true');
  });
});

// ---------------------------------------------------------------------------
// sub-5-3 — failed-retry + resume preselection
// ---------------------------------------------------------------------------

describe('failedRowIds / remainingIds (sub-5-3 AC #3/#4)', () => {
  it('failedRowIds = rows RENDERED failed at complete (matches what the user sees)', () => {
    const progress = progressOf({ status: 'complete', successCount: 3, failCount: 2 });
    expect(failedRowIds(ITEMS, progress, new Set([M2, M4]))).toEqual([M2, M4]);
  });

  it('budget_ceiling: an interrupted in-flight item renders 已暫停, so it is REMAINING, not failed', () => {
    // pausedCount=3 → the last 3 rows are paused; M3 (the interrupted item)
    // sits in that tail even though its per-item stream reported failed.
    const progress = progressOf({ status: 'budget_ceiling', pausedCount: 3 });
    expect(failedRowIds(ITEMS, progress, new Set([M3]))).toEqual([]);
    expect(remainingIds(ITEMS, progress, new Set([M3]))).toEqual([M3, M4, M5]);
  });

  it('remainingIds includes failed + paused + stopped rows, never the done ones', () => {
    const ceiling = progressOf({ status: 'budget_ceiling', pausedCount: 2 });
    expect(remainingIds(ITEMS, ceiling, new Set([M1]))).toEqual([M1, M4, M5]);

    const errored = progressOf({ status: 'error', currentMediaId: M3 });
    expect(remainingIds(ITEMS, errored, new Set([M2]))).toEqual([M2, M3, M4, M5]);
  });
});

describe('GenerationBatchPanelV2 — 重試失敗項目 (sub-5-3 AC #3)', () => {
  it('renders the retry button at a terminal when onRetryFailed is provided', () => {
    const onRetryFailed = vi.fn();
    renderPanel({
      status: 'complete',
      progress: progressOf({ status: 'complete', successCount: 4, failCount: 1 }),
      items: ITEMS,
      failedIds: new Set([M2]),
      onRetryFailed,
    });

    fireEvent.click(screen.getByTestId('gen-batch-retry-failed-btn'));
    expect(onRetryFailed).toHaveBeenCalledTimes(1);
  });

  it('does NOT render the button when onRetryFailed is absent (attach mode / zero failures)', () => {
    renderPanel({
      status: 'complete',
      progress: progressOf({ status: 'complete', successCount: 5 }),
      items: ITEMS,
    });
    expect(screen.queryByTestId('gen-batch-retry-failed-btn')).toBeNull();
  });

  it('[CR M1] never renders at budget_ceiling — 下次繼續 owns recovery there (failed ⊂ remaining)', () => {
    renderPanel({
      status: 'budget_ceiling',
      progress: progressOf({ status: 'budget_ceiling', pausedCount: 2 }),
      items: ITEMS,
      failedIds: new Set([M1]),
      onRetryFailed: vi.fn(),
    });
    expect(screen.queryByTestId('gen-batch-retry-failed-btn')).toBeNull();
    expect(screen.getByTestId('gen-batch-resume-btn')).toBeInTheDocument();
  });

  it('never renders the button while running — a retry only exists at a terminal', () => {
    renderPanel({
      status: 'running',
      progress: progressOf({ status: 'running' }),
      items: ITEMS,
      failedIds: new Set([M1]),
      onRetryFailed: vi.fn(),
    });
    expect(screen.queryByTestId('gen-batch-retry-failed-btn')).toBeNull();
  });
});

describe('GenerationBatchDialogV2 — retry/resume preselection (sub-5-3 AC #3/#4)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    h.batchState.status = 'idle';
    h.batchState.currentMediaId = null;
    h.batchState.pausedCount = 0;
    h.batchState.totalItems = 0;
    h.itemState.phase = 'idle';
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: false, progress: null });
  });

  function renderRetryDialog() {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const makeUi = () => (
      <QueryClientProvider client={queryClient}>
        <GenerationBatchDialogV2 open onOpenChange={vi.fn()} />
      </QueryClientProvider>
    );
    const view = render(makeUi());
    return { rerender: () => view.rerender(makeUi()) };
  }

  /** Drive a real consented start so the container owns items[]. */
  async function startOwnedBatch(rerender: () => void) {
    mocked.startGenerationBatch.mockResolvedValue({
      conflict: false,
      result: { batchId: 'gb-53', totalItems: 5, items: ITEMS },
    });
    fireEvent.click(await screen.findByTestId('consent-stub-start'));
    await waitFor(() => expect(h.batchStartTracking).toHaveBeenCalled());
    h.batchState.status = 'running';
    h.batchState.totalItems = 5;
    h.batchState.currentMediaId = M2;
    rerender();
  }

  it('重試失敗項目 returns to consent with the failed rows preselected + forceAnalyze — and NEVER starts a batch itself (同意紅線)', async () => {
    const { rerender } = renderRetryDialog();
    await startOwnedBatch(rerender);

    // M2 fails while running, then the batch completes.
    h.itemState.phase = 'failed';
    rerender();
    h.batchState.status = 'complete';
    h.batchState.currentMediaId = null;
    h.batchState.successCount = 4;
    h.batchState.failCount = 1;
    rerender();

    const callsBefore = mocked.startGenerationBatch.mock.calls.length;
    fireEvent.click(await screen.findByTestId('gen-batch-retry-failed-btn'));
    h.batchState.status = 'idle';
    rerender();

    const stub = await screen.findByTestId('consent-view-stub');
    expect(stub.getAttribute('data-preselected')).toBe(M2);
    expect(stub.getAttribute('data-force-analyze')).toBe('true');
    // 同意紅線: the retry itself started NOTHING — only F16 confirm can.
    expect(mocked.startGenerationBatch.mock.calls.length).toBe(callsBefore);
  });

  it('下次繼續 preselects the unfinished rows instead of dropping the consented picks', async () => {
    const { rerender } = renderRetryDialog();
    await startOwnedBatch(rerender);

    h.batchState.status = 'budget_ceiling';
    h.batchState.pausedCount = 2;
    rerender();

    fireEvent.click(await screen.findByTestId('gen-batch-resume-btn'));
    h.batchState.status = 'idle';
    rerender();

    const stub = await screen.findByTestId('consent-view-stub');
    expect(stub.getAttribute('data-preselected')).toBe([M4, M5].join(','));
    expect(stub.getAttribute('data-force-analyze')).toBe('true');
  });

  it('a consented start after a resume clears the carried preselection', async () => {
    const { rerender } = renderRetryDialog();
    await startOwnedBatch(rerender);

    h.batchState.status = 'budget_ceiling';
    h.batchState.pausedCount = 1;
    rerender();
    fireEvent.click(await screen.findByTestId('gen-batch-resume-btn'));
    h.batchState.status = 'idle';
    rerender();

    // Consent starts again — retryIds are consumed by the start.
    mocked.startGenerationBatch.mockResolvedValue({
      conflict: false,
      result: { batchId: 'gb-54', totalItems: 1, items: [ITEMS[4]] },
    });
    fireEvent.click(await screen.findByTestId('consent-stub-start'));
    await waitFor(() => expect(mocked.startGenerationBatch).toHaveBeenCalledTimes(2));
    h.batchState.status = 'running';
    h.batchState.currentMediaId = M5;
    rerender();

    // A clean ceiling (nothing unfinished) resumes with NO stale carryover.
    h.batchState.status = 'budget_ceiling';
    h.batchState.pausedCount = 0;
    h.batchState.totalItems = 1;
    rerender();
    fireEvent.click(await screen.findByTestId('gen-batch-resume-btn'));
    h.batchState.status = 'idle';
    rerender();
    const stub = await screen.findByTestId('consent-view-stub');
    expect(stub.getAttribute('data-preselected')).toBe('');
  });
});

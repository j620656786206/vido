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
  },
}));

import {
  GenerationBatchDialogV2,
  GenerationBatchPanelV2,
  deriveRowStates,
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
const M9 = '9ff0c000-dead-4bee-8f00-000000000999';

const ITEMS: GenerationBatchItem[] = [
  { mediaId: M1, title: '沙丘：第二部' },
  { mediaId: M2, title: '奧本海默' },
  { mediaId: M3, title: '怪奇物語' },
  { mediaId: M4, title: '星際效應' },
  { mediaId: M5, title: '全面啟動' },
];

function progressOf(p: Partial<GenerationBatchProgressState>): GenerationBatchProgressState {
  return {
    batchId: 'gb-1',
    totalItems: 5,
    currentIndex: 3,
    currentMediaId: M3,
    currentItem: '怪奇物語',
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

  it('budget_ceiling: paused_count is AUTHORITATIVE — the interrupted in-flight item renders paused even when its per-item pipeline emitted transcription_failed', () => {
    // Mid-item ceiling at item 3 (index 2): paused_count = 3 → rows 2..4 paused.
    const progress = progressOf({ status: 'budget_ceiling', pausedCount: 3, spentUsd: 5 });
    // The racing per-item failed event recorded media id 3 — must NOT paint 失敗.
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
// Panel — prop-driven state matrix (AC 1/2/7)
// ---------------------------------------------------------------------------

function renderPanel(props: Partial<GenerationBatchPanelV2Props> = {}) {
  const merged: GenerationBatchPanelV2Props = {
    open: true,
    status: 'idle',
    progress: progressOf({ status: 'running' }),
    items: [],
    scope: 'missing',
    onScopeChange: vi.fn(),
    onStart: vi.fn(),
    onConfirmCancelAll: vi.fn(),
    onResume: vi.fn(),
    onClose: vi.fn(),
    ...props,
  };
  render(<GenerationBatchPanelV2 {...merged} />);
  return merged;
}

describe('GenerationBatchPanelV2', () => {
  it('idle renders the 缺字幕的項目 segment with the Mono preview count', () => {
    renderPanel({ previewCount: 38 });
    expect(screen.getByText('批次生成字幕')).toBeInTheDocument();
    const missing = screen.getByTestId('gen-batch-scope-missing');
    expect(missing).toHaveTextContent('缺字幕的項目');
    expect(missing).toHaveTextContent('38');
    expect(missing).toHaveAttribute('aria-pressed', 'true');
  });

  it('已選項目 segment renders ONLY when opened with a non-empty selection (AC 1)', () => {
    renderPanel({ previewCount: 38 });
    expect(screen.queryByTestId('gen-batch-scope-selected')).not.toBeInTheDocument();
  });

  it('已選項目 segment renders with the selection count and the excluded-series note', () => {
    renderPanel({ previewCount: 38, scope: 'selected', selectedCount: 4, excludedSeriesCount: 2 });
    const selected = screen.getByTestId('gen-batch-scope-selected');
    expect(selected).toHaveTextContent('已選項目');
    expect(selected).toHaveTextContent('4');
    expect(selected).toHaveAttribute('aria-pressed', 'true');
    const note = screen.getByTestId('gen-batch-excluded-note');
    expect(note).toHaveTextContent('已排除');
    expect(note).toHaveTextContent('2');
    expect(note).toHaveTextContent('部影集（影集字幕生成即將推出）');
  });

  it('empty scope renders the friendly state, not an error (AC 7)', () => {
    renderPanel({ previewCount: 0 });
    expect(screen.getByTestId('gen-batch-empty-scope')).toHaveTextContent('目前沒有缺字幕的項目');
    expect(screen.getByTestId('gen-batch-start-btn')).toBeDisabled();
  });

  it('running renders queue rows from items[], the counter, cost line and SSE chip (AC 1)', () => {
    renderPanel({
      status: 'running',
      progress: progressOf({ successCount: 2, failCount: 0 }),
      items: ITEMS,
    });

    expect(screen.getByTestId('gen-batch-counter')).toHaveTextContent('2 / 5');
    expect(screen.getByTestId('gen-batch-item-list').children).toHaveLength(5);
    expect(screen.getByTestId(`gen-batch-row-${M3}`)).toHaveAttribute('data-state', 'active');
    expect(screen.getByTestId(`gen-batch-row-${M3}`)).toHaveTextContent('轉錄中');
    expect(screen.getByTestId(`gen-batch-row-${M1}`)).toHaveTextContent('完成');
    expect(screen.getByTestId(`gen-batch-row-${M5}`)).toHaveTextContent('排隊中');
    // Active row shows the frozen per-item stepper (slice-1 reuse).
    expect(screen.getByTestId('generation-progress-v2')).toBeInTheDocument();
    // Cost line: Mono numerals fed by SSE spent_usd/budget_usd.
    expect(screen.getByTestId('gen-batch-cost-line')).toHaveTextContent(
      '本次用量：$0.42 / 上限 $5.00'
    );
    expect(screen.getByTestId('gen-batch-sse-chip')).toHaveTextContent('即時更新（SSE）');
    expect(screen.getByRole('progressbar', { name: '批次生成進度' })).toHaveAttribute(
      'aria-valuenow',
      '2'
    );
  });

  it('全部取消 uses an inline confirm before cancelling (AC 1)', () => {
    const props = renderPanel({
      status: 'running',
      progress: progressOf({}),
      items: ITEMS,
    });

    fireEvent.click(screen.getByTestId('gen-batch-cancel-all'));
    expect(screen.getByTestId('gen-batch-cancel-confirm')).toBeInTheDocument();
    expect(props.onConfirmCancelAll).not.toHaveBeenCalled();

    fireEvent.click(screen.getByTestId('gen-batch-cancel-confirm-btn'));
    expect(props.onConfirmCancelAll).toHaveBeenCalledOnce();
  });

  it('budget_ceiling renders the F9 banner, paused rows, 關閉 + 下次繼續 (AC 2)', () => {
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
    // NOT error tokens — the banner uses the drawn warning-tint, not error-tint.
    expect(banner.className).toContain('warning-tint');
    expect(banner.className).not.toContain('error-tint');
  });

  it('exclusion note stays visible when EVERY selected item was excluded (scope falls back to missing, AC 5)', () => {
    // All-series selection (or ids the id→type map cannot classify): zero movie
    // ids → no 已選項目 segment, scope=missing — the note must STILL render,
    // otherwise the user's selection silently vanishes.
    renderPanel({ previewCount: 38, scope: 'missing', selectedCount: 0, excludedSeriesCount: 3 });
    expect(screen.queryByTestId('gen-batch-scope-selected')).not.toBeInTheDocument();
    const note = screen.getByTestId('gen-batch-excluded-note');
    expect(note).toHaveTextContent('已排除');
    expect(note).toHaveTextContent('3');
  });

  it('recover-attach fallback card honors terminal semantics — budget_ceiling renders 已暫停, not 已取消 (AC 2)', () => {
    renderPanel({
      status: 'budget_ceiling',
      progress: progressOf({
        status: 'budget_ceiling',
        currentMediaId: M3,
        currentItem: '怪奇物語',
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
      progress: progressOf({ currentMediaId: M3, currentItem: '怪奇物語' }),
      items: [],
    });

    expect(screen.getByTestId(`gen-batch-row-${M3}`)).toHaveAttribute('data-state', 'active');
    expect(screen.getByTestId(`gen-batch-row-${M3}`)).toHaveTextContent('怪奇物語');
  });

  it('static scope line replaces the segments outside idle', () => {
    renderPanel({
      status: 'running',
      progress: progressOf({ totalItems: 5 }),
      items: ITEMS,
    });

    expect(screen.queryByTestId('gen-batch-scope-missing')).not.toBeInTheDocument();
    expect(screen.getByText(/範圍：缺字幕的項目（/)).toBeInTheDocument();
  });

  it('Escape is gated while running (AC 7)', () => {
    const props = renderPanel({
      status: 'running',
      progress: progressOf({}),
      items: ITEMS,
    });

    fireEvent.keyDown(screen.getByTestId('generation-batch-dialog-v2'), { key: 'Escape' });
    expect(props.onClose).not.toHaveBeenCalled();
  });

  it('Escape closes when idle', () => {
    const props = renderPanel({ previewCount: 3 });

    fireEvent.keyDown(screen.getByTestId('generation-batch-dialog-v2'), { key: 'Escape' });
    expect(props.onClose).toHaveBeenCalledOnce();
  });
});

// ---------------------------------------------------------------------------
// Container — service wiring (AC 1/3/5)
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
    h.itemState.phase = 'idle';
    mocked.previewGenerationBatch.mockResolvedValue({ totalItems: 38 });
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: false, progress: null });
  });

  it('fetches the 缺字幕 count from the preview endpoint on open (AC 1)', async () => {
    renderDialog();

    await waitFor(() =>
      expect(screen.getByTestId('gen-batch-scope-missing')).toHaveTextContent('38')
    );
    expect(mocked.previewGenerationBatch).toHaveBeenCalled();
  });

  it('recovers an already-running batch on open via the status probe (AC 1 409-recover)', async () => {
    const snapshot = progressOf({ status: 'running' });
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: true, progress: snapshot });

    renderDialog();

    await waitFor(() => expect(h.batchStartTracking).toHaveBeenCalledWith(snapshot));
  });

  it('start sends scope=missing without media_ids and seeds tracking from the 202 (AC 1/3)', async () => {
    mocked.startGenerationBatch.mockResolvedValue({
      conflict: false,
      result: { batchId: 'gb-9', totalItems: 2, items: ITEMS.slice(0, 2) },
    });

    renderDialog();
    await waitFor(() =>
      expect(screen.getByTestId('gen-batch-scope-missing')).toHaveTextContent('38')
    );

    fireEvent.click(screen.getByTestId('gen-batch-start-btn'));

    await waitFor(() =>
      expect(mocked.startGenerationBatch).toHaveBeenCalledWith({ scope: 'missing' })
    );
    expect(h.batchStartTracking).toHaveBeenCalledWith({ batchId: 'gb-9', totalItems: 2 });
  });

  it('start passes the pre-filtered selected MOVIE ids (AC 5 — BE rejects any non-movie id)', async () => {
    mocked.startGenerationBatch.mockResolvedValue({
      conflict: false,
      result: { batchId: 'gb-9', totalItems: 2, items: ITEMS.slice(0, 2) },
    });

    renderDialog({ selectedMovieIds: [M1, M2], excludedSeriesCount: 1 });

    // Opened with a selection → 已選項目 preselected.
    const selected = await screen.findByTestId('gen-batch-scope-selected');
    expect(selected).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByTestId('gen-batch-excluded-note')).toHaveTextContent('已排除');

    fireEvent.click(screen.getByTestId('gen-batch-start-btn'));

    await waitFor(() =>
      expect(mocked.startGenerationBatch).toHaveBeenCalledWith({
        scope: 'selected',
        mediaIds: [M1, M2],
      })
    );
  });

  it('attaches to the 409 in-progress snapshot instead of erroring (AC 1)', async () => {
    const snapshot = progressOf({ status: 'running' });
    mocked.startGenerationBatch.mockResolvedValue({ conflict: true, progress: snapshot });

    renderDialog();
    await waitFor(() =>
      expect(screen.getByTestId('gen-batch-scope-missing')).toHaveTextContent('38')
    );

    fireEvent.click(screen.getByTestId('gen-batch-start-btn'));

    await waitFor(() => expect(h.batchStartTracking).toHaveBeenCalledWith(snapshot));
    expect(screen.queryByTestId('gen-batch-start-error')).not.toBeInTheDocument();
  });

  it('total_items:0 start renders the friendly empty state (AC 7)', async () => {
    mocked.startGenerationBatch.mockResolvedValue({
      conflict: false,
      result: { batchId: null, totalItems: 0, items: [] },
    });
    // Preview said 3 so start is clickable; the START response is the truth.
    mocked.previewGenerationBatch.mockResolvedValue({ totalItems: 3 });

    renderDialog();
    await waitFor(() =>
      expect(screen.getByTestId('gen-batch-scope-missing')).toHaveTextContent('3')
    );

    fireEvent.click(screen.getByTestId('gen-batch-start-btn'));

    await waitFor(() =>
      expect(screen.getByTestId('gen-batch-empty-scope')).toHaveTextContent('目前沒有缺字幕的項目')
    );
    expect(h.batchStartTracking).not.toHaveBeenCalled();
  });

  it('start failure surfaces the error inline (400 selection reject path)', async () => {
    mocked.startGenerationBatch.mockRejectedValue(
      new Error('media_ids 含無法生成字幕的項目（非電影或沒有媒體檔案）')
    );

    renderDialog();
    await waitFor(() =>
      expect(screen.getByTestId('gen-batch-scope-missing')).toHaveTextContent('38')
    );

    fireEvent.click(screen.getByTestId('gen-batch-start-btn'));

    await waitFor(() =>
      expect(screen.getByTestId('gen-batch-start-error')).toHaveTextContent(
        'media_ids 含無法生成字幕的項目'
      )
    );
  });

  it('joins the per-item stream on current_media_id while running (AC 1)', async () => {
    h.batchState.status = 'running';
    h.batchState.currentMediaId = M9;

    renderDialog();

    await waitFor(() => expect(h.itemStartTracking).toHaveBeenCalledWith(M9));
  });

  it('cancel calls the cancel endpoint (terminal arrives via SSE)', async () => {
    h.batchState.status = 'running';
    h.batchState.currentMediaId = M3;
    h.batchState.currentItem = '怪奇物語';
    mocked.cancelGenerationBatch.mockResolvedValue({ cancelled: true, running: false });

    renderDialog();

    fireEvent.click(screen.getByTestId('gen-batch-cancel-all'));
    fireEvent.click(screen.getByTestId('gen-batch-cancel-confirm-btn'));

    await waitFor(() => expect(mocked.cancelGenerationBatch).toHaveBeenCalledOnce());
  });

  // -------------------------------------------------------------------------
  // The 9R-16 CR race, exercised through the REAL component (container +
  // recording effect + deriveRowStates), not the helper in isolation: on
  // cancelled/budget_ceiling the interrupted in-flight item ALSO emits
  // transcription_failed — whichever order the two events arrive, the row must
  // end up 已暫停, never 失敗.
  // -------------------------------------------------------------------------

  function renderDialogRaw() {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const onOpenChange = vi.fn();
    // Fresh element per (re)render — reusing one element reference lets React
    // bail out on the referentially-equal props and skip re-reading the
    // (mutated) mock hook state.
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
    await waitFor(() =>
      expect(screen.getByTestId('gen-batch-scope-missing')).toHaveTextContent('38')
    );
    fireEvent.click(screen.getByTestId('gen-batch-start-btn'));
    await waitFor(() => expect(h.batchStartTracking).toHaveBeenCalled());
    // The batch SSE stream reports item 3 in flight.
    h.batchState.status = 'running';
    h.batchState.totalItems = 5;
    h.batchState.currentMediaId = M3;
    h.batchState.currentItem = '怪奇物語';
    rerender();
  }

  it('race order A — per-item failed arrives BEFORE the terminal batch event: paused wins at render', async () => {
    const { rerender } = renderDialogRaw();
    await startBatchWithItems(rerender);

    // transcription_failed for the in-flight item lands while status is still
    // running → the container records it (legitimately renders 失敗 for a beat).
    h.itemState.phase = 'failed';
    rerender();
    await waitFor(() =>
      expect(screen.getByTestId(`gen-batch-row-${M3}`)).toHaveAttribute('data-state', 'failed')
    );

    // THEN the terminal budget_ceiling batch event arrives — paused_count is
    // authoritative; the recorded failure must NOT paint the row 失敗.
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

    // Terminal batch event first…
    h.batchState.status = 'budget_ceiling';
    h.batchState.pausedCount = 3;
    h.batchState.spentUsd = 5;
    rerender();
    await waitFor(() =>
      expect(screen.getByTestId(`gen-batch-row-${M3}`)).toHaveAttribute('data-state', 'paused')
    );

    // …then the straggling per-item failed for the interrupted item: the
    // recording effect must ignore it (batch no longer running).
    h.itemState.phase = 'failed';
    rerender();
    await waitFor(() =>
      expect(screen.getByTestId(`gen-batch-row-${M3}`)).toHaveAttribute('data-state', 'paused')
    );
    expect(screen.getByTestId(`gen-batch-row-${M3}`)).not.toHaveTextContent('失敗');
  });

  it('下次繼續 starts a NEW scope=missing batch (resume-for-free, AC 2)', async () => {
    h.batchState.status = 'budget_ceiling';
    h.batchState.pausedCount = 3;
    mocked.startGenerationBatch.mockResolvedValue({
      conflict: false,
      result: { batchId: 'gb-next', totalItems: 3, items: ITEMS.slice(2) },
    });

    renderDialog();

    fireEvent.click(screen.getByTestId('gen-batch-resume-btn'));

    await waitFor(() =>
      expect(mocked.startGenerationBatch).toHaveBeenCalledWith({ scope: 'missing' })
    );
    expect(h.batchReset).toHaveBeenCalled();
  });
});

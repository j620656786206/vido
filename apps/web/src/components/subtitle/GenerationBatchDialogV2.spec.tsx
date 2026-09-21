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
    // dsr-6d-b: null keeps the fallback (deriveRowStates) path covered —
    // items-first tests override it explicitly.
    items: null as GenerationBatchItemState[] | null,
  },
  batchStartTracking: vi.fn(),
  batchAttachSnapshot: vi.fn(),
  batchEpoch: 0,
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
    attachSnapshot: h.batchAttachSnapshot,
    connectionEpoch: h.batchEpoch,
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
    dismissGenerationBatch: vi.fn(),
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
import {
  subtitleService,
  type GenerationBatchItem,
  type GenerationBatchItemState,
} from '../../services/subtitleService';
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
  { mediaId: M1, title: '沙丘：第二部', mediaType: 'movie', seriesTitle: '' },
  { mediaId: M2, title: '奧本海默', mediaType: 'movie', seriesTitle: '' },
  { mediaId: M3, title: '怪奇物語 S04E07', mediaType: 'episode', seriesTitle: '' },
  { mediaId: M4, title: '星際效應', mediaType: 'movie', seriesTitle: '' },
  { mediaId: M5, title: '全面啟動', mediaType: 'movie', seriesTitle: '' },
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
    // dsr-6d-a: the queue rides the snapshot; the SSE event leaves it null
    // while running, so the hook state defaults to null too.
    items: null,
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
    onConfirmCancelAll: vi.fn().mockResolvedValue(undefined),
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

  it("[dsr-6f-1] phone sheet shell: slides up, and the 44px ✕ sits on this dialog's title row (44 on a phone)", () => {
    renderPanel({ items: ITEMS });
    const shell = screen.getByTestId('generation-batch-dialog-v2');
    const t = (el: Element) => el.className.split(/\s+/);
    expect(t(shell)).toContain('max-sm:data-[state=open]:animate-sheet-enter');
    const close = screen.getByText('Close').closest('button')!;
    // dsr-6f-3: the phone title row is 44 high (F8-M/F15-M/F16-M sheet-header), so
    // grabber 16 + half of 44 (22) − half of the 44px ✕ (22) = 16 = top-4. It
    // moves with the header: change the row's height and this token changes too.
    expect(t(close)).toEqual(expect.arrayContaining(['max-sm:h-11', 'max-sm:top-4']));
    expect(t(close)).not.toContain('max-sm:top-[22px]');
    expect(t(screen.getByTestId('gen-batch-title-bar'))).toEqual(
      expect.arrayContaining(['h-14', 'max-sm:h-11', 'border-b'])
    );
    expect(t(screen.getByTestId('gen-batch-drag-handle'))).toContain('sm:hidden');
    expect(t(screen.getByTestId('gen-batch-title-bar'))).toContain('max-sm:pl-4');
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
    expect(screen.getByTestId(`gen-batch-row-${M3}`)).toHaveAttribute('data-state', 'running');
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

  it('全部取消 uses an inline confirm before cancelling', async () => {
    const props = renderPanel({ status: 'running', progress: progressOf({}), items: ITEMS });

    fireEvent.click(screen.getByTestId('gen-batch-cancel-all'));
    expect(screen.getByTestId('gen-batch-cancel-confirm')).toBeInTheDocument();
    expect(props.onConfirmCancelAll).not.toHaveBeenCalled();

    fireEvent.click(screen.getByTestId('gen-batch-cancel-confirm-btn'));
    expect(props.onConfirmCancelAll).toHaveBeenCalledOnce();
    // 🔴 #12 / M7: an ACCEPTED cancel is not a FINISHED cancel — the server
    // still has to stop the in-flight job. The confirm row stays up (取消中…)
    // so the focused control never vanishes and the panel tells no lies.
    await waitFor(() => expect(screen.getByText('取消中…')).toBeInTheDocument());
    expect(screen.getByTestId('gen-batch-cancel-confirm')).toBeInTheDocument();
  });

  it('[P0] M7 the 取消中… row only clears when the batch actually reports a terminal status', async () => {
    const onConfirmCancelAll = vi.fn().mockResolvedValue(undefined);
    const { rerender } = render(
      <GenerationBatchPanelV2
        open
        status="running"
        progress={progressOf({ status: 'running' })}
        items={ITEMS}
        onConfirmCancelAll={onConfirmCancelAll}
        onResume={vi.fn()}
        onClose={vi.fn()}
      />
    );

    fireEvent.click(screen.getByTestId('gen-batch-cancel-all'));
    fireEvent.click(screen.getByTestId('gen-batch-cancel-confirm-btn'));
    await waitFor(() => expect(screen.getByText('取消中…')).toBeInTheDocument());

    // The terminal `cancelled` event lands → confirm row goes, 關閉 takes focus.
    rerender(
      <GenerationBatchPanelV2
        open
        status="cancelled"
        progress={progressOf({ status: 'cancelled', successCount: 2 })}
        items={ITEMS}
        onConfirmCancelAll={onConfirmCancelAll}
        onResume={vi.fn()}
        onClose={vi.fn()}
      />
    );
    await waitFor(() =>
      expect(screen.queryByTestId('gen-batch-cancel-confirm')).not.toBeInTheDocument()
    );
    await waitFor(() => expect(screen.getByTestId('gen-batch-close-btn')).toHaveFocus());
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

    expect(screen.getByTestId(`gen-batch-row-${M3}`)).toHaveAttribute('data-state', 'running');
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
    h.batchState.batchId = '';
    h.batchState.currentMediaId = null;
    h.batchState.pausedCount = 0;
    h.batchState.successCount = 0;
    h.batchState.failCount = 0;
    h.batchState.items = null;
    h.batchEpoch = 0;
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
      result: { batchId: 'gb-9', totalItems: 2, items: ITEMS.slice(0, 2), progress: null },
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

  it('[P0] 🔴 #5 a cancel request that FAILS surfaces the alert instead of being swallowed', async () => {
    h.batchState.status = 'running';
    h.batchState.currentMediaId = M3;
    h.batchState.currentItem = '怪奇物語 S04E07';
    mocked.cancelGenerationBatch.mockRejectedValue(new Error('network'));

    renderDialog();

    fireEvent.click(screen.getByTestId('gen-batch-cancel-all'));
    fireEvent.click(screen.getByTestId('gen-batch-cancel-confirm-btn'));

    expect(await screen.findByTestId('gen-batch-cancel-error')).toHaveTextContent(
      '取消失敗，批次仍在進行。請再試一次。'
    );
    expect(screen.getByTestId('gen-batch-cancel-confirm')).toBeInTheDocument();
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
      result: { batchId: 'gb-race', totalItems: 5, items: ITEMS, progress: null },
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
    // Let the on-open status probe settle inside act().
    await waitFor(() => expect(mocked.getGenerationBatchStatus).toHaveBeenCalled());
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
    h.batchState.batchId = '';
    h.batchState.currentMediaId = null;
    h.batchState.pausedCount = 0;
    h.batchState.totalItems = 0;
    h.batchState.successCount = 0;
    h.batchState.failCount = 0;
    h.batchState.items = null;
    h.batchEpoch = 0;
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
      result: { batchId: 'gb-53', totalItems: 5, items: ITEMS, progress: null },
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
      result: { batchId: 'gb-54', totalItems: 1, items: [ITEMS[4]], progress: null },
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

// ---------------------------------------------------------------------------
// dsr-6d-b — the rows say what the BACKEND says (items[] first)
// ---------------------------------------------------------------------------

const state = (
  mediaId: string,
  title: string,
  status: GenerationBatchItemState['status'],
  reason: GenerationBatchItemState['reason'] = '',
  seriesTitle = ''
): GenerationBatchItemState => ({
  mediaId,
  title,
  mediaType: seriesTitle ? 'episode' : 'movie',
  seriesTitle,
  status,
  reason,
});

describe('GenerationBatchPanelV2 — rows from progress.items (dsr-6d-b AC #2/#3)', () => {
  it('[P0] 🔴 #1 an item the pipeline REFUSED renders 失敗 with its reason — it used to render 完成', () => {
    renderPanel({
      status: 'running',
      progress: progressOf({
        currentMediaId: M5,
        successCount: 2,
        failCount: 1,
        items: [
          state(M1, '沙丘：第二部', 'done'),
          state(M2, '奧本海默', 'done'),
          state(M3, '怪奇物語 S04E07', 'failed', 'busy_elsewhere'),
          state(M4, '星際效應', 'failed', 'skipped'),
          state(M5, '全面啟動', 'running'),
        ],
      }),
      items: [], // no 202 items[] at all — the queue comes from the snapshot
    });

    const busy = screen.getByTestId(`gen-batch-row-${M3}`);
    expect(busy).toHaveAttribute('data-state', 'failed');
    expect(busy).toHaveTextContent('這部正在別處處理');
    expect(busy).not.toHaveTextContent('完成');

    const skipped = screen.getByTestId(`gen-batch-row-${M4}`);
    expect(skipped).toHaveTextContent('沒有可用的字幕來源');
    expect(skipped).not.toHaveTextContent('已略過');

    expect(screen.getByTestId(`gen-batch-row-${M5}`)).toHaveAttribute('data-state', 'running');
  });

  it('[P0] 🔴 #9 a failed row draws NO stepper (it is not being worked on)', () => {
    renderPanel({
      status: 'running',
      progress: progressOf({
        currentMediaId: M2,
        items: [state(M1, '沙丘：第二部', 'failed', 'error'), state(M2, '奧本海默', 'running')],
      }),
      items: [],
      activeItemProgress: {
        phase: 'transcribing',
        failedPhase: null,
        percentage: null,
        message: '',
        jobId: null,
        error: null,
        srtPath: null,
        zhSrtPath: null,
        partial: false,
      } as never,
    });

    expect(screen.getByTestId(`gen-batch-row-${M1}`)).toHaveTextContent('生成失敗');
    // exactly one stepper — the running row's
    expect(screen.getAllByTestId('generation-progress-v2')).toHaveLength(1);
    expect(screen.getByTestId(`gen-batch-row-${M2}`)).toHaveTextContent('轉錄中');
  });

  it('[P0] a running row with no per-item event yet says 處理中, never a stage it cannot know', () => {
    renderPanel({
      status: 'running',
      progress: progressOf({ currentMediaId: M1, items: [state(M1, '沙丘：第二部', 'running')] }),
      items: [],
    });
    expect(screen.getByTestId(`gen-batch-row-${M1}`)).toHaveTextContent('處理中');
  });

  it('[P0] an episode row shows 劇名 + 集數標題; a movie shows neither a prefix nor "undefined"', () => {
    renderPanel({
      status: 'complete',
      progress: progressOf({
        status: 'complete',
        successCount: 2,
        items: [
          state(M3, 'S04E07 第七章', 'done', '', '怪奇物語'),
          state(M1, '沙丘：第二部', 'done'),
        ],
      }),
      items: [],
    });

    expect(screen.getByTestId(`gen-batch-row-${M3}`)).toHaveTextContent('怪奇物語 S04E07 第七章');
    const movie = screen.getByTestId(`gen-batch-row-${M1}`);
    expect(movie).toHaveTextContent('沙丘：第二部');
    expect(movie).not.toHaveTextContent('undefined');
  });

  it('[P0] budget_ceiling + cancelled rows come straight from the snapshot', () => {
    renderPanel({
      status: 'budget_ceiling',
      progress: progressOf({
        status: 'budget_ceiling',
        successCount: 1,
        pausedCount: 2,
        spentUsd: 5,
        items: [
          state(M1, '沙丘：第二部', 'done'),
          state(M2, '奧本海默', 'paused'),
          state(M3, '怪奇物語 S04E07', 'paused'),
        ],
      }),
      items: [],
    });

    expect(screen.getByTestId(`gen-batch-row-${M2}`)).toHaveAttribute('data-state', 'paused');
    expect(screen.getByTestId(`gen-batch-row-${M3}`)).toHaveTextContent('已暫停 — 下次繼續');
  });

  it('[P0] progress.items WINS over a stale 202 items[] prop', () => {
    renderPanel({
      status: 'complete',
      progress: progressOf({
        status: 'complete',
        successCount: 1,
        failCount: 1,
        items: [state(M1, '沙丘：第二部', 'done'), state(M2, '奧本海默', 'failed', 'error')],
      }),
      items: ITEMS, // five stale rows from the start-202
    });

    expect(screen.getByTestId('gen-batch-item-list').children).toHaveLength(2);
    expect(screen.getByTestId(`gen-batch-row-${M2}`)).toHaveTextContent('生成失敗');
  });
});

describe('GenerationBatchPanelV2 — the batch verdict + honest colour (dsr-6d-b AC #3/#4)', () => {
  it('[P0] complete with NO failures: 全部完成（N 部）in success text', () => {
    renderPanel({
      status: 'complete',
      progress: progressOf({ status: 'complete', successCount: 5, failCount: 0 }),
      items: ITEMS,
    });
    const summary = screen.getByTestId('gen-batch-summary');
    expect(summary).toHaveTextContent('全部完成（5 部）');
    expect(summary.className).toContain('success-text');
  });

  it('[P0] complete WITH failures is neutral — green would claim an outcome that is not good', () => {
    renderPanel({
      status: 'complete',
      progress: progressOf({ status: 'complete', successCount: 2, failCount: 3 }),
      items: ITEMS,
    });
    const summary = screen.getByTestId('gen-batch-summary');
    expect(summary).toHaveTextContent('完成 2 部、失敗 3 部');
    expect(summary.className).not.toContain('success');
    expect(summary.className).not.toContain('error');
  });

  it('[P0] cancelled says how much survived', () => {
    renderPanel({
      status: 'cancelled',
      progress: progressOf({ status: 'cancelled', successCount: 2 }),
      items: ITEMS,
    });
    expect(screen.getByTestId('gen-batch-summary')).toHaveTextContent('已取消：完成 2 部');
  });

  it('[P0] M4 the verdict IS announced, and exactly once — through the always-mounted live region', () => {
    renderPanel({
      status: 'complete',
      progress: progressOf({ status: 'complete', successCount: 5 }),
      items: ITEMS,
    });
    // The announcer is the sr-only region that exists for the panel's whole
    // life (a live region mounted together with its text is not spoken).
    const live = screen.getByTestId('gen-batch-status-live');
    expect(live).toHaveAttribute('aria-live', 'polite');
    expect(live).toHaveTextContent('全部完成（5 部）');
    // …and the VISIBLE line must not be a second live region, or AT says it twice.
    expect(screen.getByTestId('gen-batch-summary')).not.toHaveAttribute('aria-live');
  });

  it('[P0] M4 a cancelled batch is announced too', () => {
    renderPanel({
      status: 'cancelled',
      progress: progressOf({ status: 'cancelled', successCount: 2 }),
      items: ITEMS,
    });
    expect(screen.getByTestId('gen-batch-status-live')).toHaveTextContent('已取消：完成 2 部');
  });

  it('[P1] budget_ceiling keeps its sr-only announcement and shows no verdict line', () => {
    renderPanel({
      status: 'budget_ceiling',
      progress: progressOf({ status: 'budget_ceiling', successCount: 2, pausedCount: 3 }),
      items: ITEMS,
    });
    expect(screen.getByTestId('gen-batch-status-live')).toHaveTextContent('已達本次預算上限');
    expect(screen.queryByTestId('gen-batch-summary')).toBeNull();
  });

  it.each([
    ['running', 0, 'bg-[var(--accent-primary)]'],
    ['complete', 0, 'bg-[var(--success)]'],
    ['complete', 2, 'bg-[var(--text-muted)]'],
    ['cancelled', 0, 'bg-[var(--text-muted)]'],
    ['budget_ceiling', 0, 'bg-[var(--text-muted)]'],
    ['error', 0, 'bg-[var(--error)]'],
  ] as const)(
    '[P0] 🔴 #7 progress bar at %s (failCount %i) is %s — never bg-tertiary (an invisible bar)',
    (status, failCount, expected) => {
      renderPanel({
        status,
        progress: progressOf({ status, failCount, successCount: 2 }),
        items: ITEMS,
      });
      const bar = screen.getByTestId('gen-batch-progress-bar');
      expect(bar.className).toContain(expected);
      expect(bar.className).not.toContain('bg-[var(--bg-tertiary)]');
      if (status !== 'running') expect(bar.className).not.toContain('accent-primary');
    }
  );

  it('[P0] 🔴 #8 確定取消 and 重試失敗項目 are NEUTRAL secondaries, not 硃砂', () => {
    renderPanel({ status: 'running', progress: progressOf({}), items: ITEMS });
    fireEvent.click(screen.getByTestId('gen-batch-cancel-all'));
    const confirmBtn = screen.getByTestId('gen-batch-cancel-confirm-btn');
    expect(confirmBtn.className).toContain('bg-[var(--bg-tertiary)]');
    expect(confirmBtn.className).not.toContain('error');

    renderPanel({
      status: 'complete',
      progress: progressOf({ status: 'complete', failCount: 1 }),
      items: ITEMS,
      failedIds: new Set([M2]),
      onRetryFailed: vi.fn(),
    });
    const retry = screen.getByTestId('gen-batch-retry-failed-btn');
    expect(retry.className).toContain('bg-[var(--bg-tertiary)]');
    expect(retry.className).not.toContain('error');
  });
});

describe('GenerationBatchPanelV2 — cancel that can fail + focus (dsr-6d-b AC #6)', () => {
  it('[P0] 🔴 #5 a failed cancel SAYS SO and keeps the confirm row (the batch is still running)', async () => {
    const onConfirmCancelAll = vi.fn().mockRejectedValue(new Error('network'));
    renderPanel({
      status: 'running',
      progress: progressOf({}),
      items: ITEMS,
      onConfirmCancelAll,
    });

    fireEvent.click(screen.getByTestId('gen-batch-cancel-all'));
    fireEvent.click(screen.getByTestId('gen-batch-cancel-confirm-btn'));

    const alert = await screen.findByTestId('gen-batch-cancel-error');
    expect(alert).toHaveAttribute('role', 'alert');
    expect(alert).toHaveTextContent('取消失敗，批次仍在進行。請再試一次。');
    expect(screen.getByTestId('gen-batch-cancel-confirm')).toBeInTheDocument();
  });

  it('[P0] while the cancel is in flight the button says 取消中… and is aria-disabled (never disabled)', async () => {
    let release: () => void = () => {};
    const onConfirmCancelAll = vi.fn(
      () =>
        new Promise<void>((res) => {
          release = res;
        })
    );
    renderPanel({
      status: 'running',
      progress: progressOf({}),
      items: ITEMS,
      onConfirmCancelAll,
    });

    fireEvent.click(screen.getByTestId('gen-batch-cancel-all'));
    fireEvent.click(screen.getByTestId('gen-batch-cancel-confirm-btn'));

    const btn = await screen.findByText('取消中…');
    expect(btn).toHaveAttribute('aria-disabled', 'true');
    expect(btn).not.toHaveAttribute('disabled');

    // A second click while busy must not fire a second request.
    fireEvent.click(btn);
    expect(onConfirmCancelAll).toHaveBeenCalledTimes(1);

    release();
    // Still 取消中… after the request resolves — the batch has not reported a
    // terminal status yet, and claiming otherwise is the lie M7 is about.
    await waitFor(() => expect(onConfirmCancelAll).toHaveBeenCalledTimes(1));
    expect(screen.getByTestId('gen-batch-cancel-confirm')).toBeInTheDocument();
  });

  it('[P0] M3 the failure alert does NOT survive 繼續生成 — a new confirm row starts clean', async () => {
    const onConfirmCancelAll = vi.fn().mockRejectedValue(new Error('network'));
    renderPanel({
      status: 'running',
      progress: progressOf({}),
      items: ITEMS,
      onConfirmCancelAll,
    });

    fireEvent.click(screen.getByTestId('gen-batch-cancel-all'));
    fireEvent.click(screen.getByTestId('gen-batch-cancel-confirm-btn'));
    await screen.findByTestId('gen-batch-cancel-error');

    fireEvent.click(screen.getByText('繼續生成'));
    fireEvent.click(screen.getByTestId('gen-batch-cancel-all'));

    expect(screen.getByTestId('gen-batch-cancel-confirm')).toBeInTheDocument();
    expect(screen.queryByTestId('gen-batch-cancel-error')).toBeNull();
  });

  it('[P0] 🔴 #12 focus follows the confirm row instead of falling to <body>', async () => {
    renderPanel({ status: 'running', progress: progressOf({}), items: ITEMS });

    fireEvent.click(screen.getByTestId('gen-batch-cancel-all'));
    await waitFor(() => expect(screen.getByText('繼續生成')).toHaveFocus());

    fireEvent.click(screen.getByText('繼續生成'));
    await waitFor(() => expect(screen.getByTestId('gen-batch-cancel-all')).toHaveFocus());
    expect(document.body).not.toHaveFocus();
  });
});

describe('GenerationBatchPanelV2 — 再產生字幕 (dsr-6d-b AC #5)', () => {
  it('[P0] a terminal with nothing to retry still offers a way back to the consent flow', () => {
    const onRestart = vi.fn();
    renderPanel({
      status: 'complete',
      progress: progressOf({ status: 'complete', successCount: 5 }),
      items: ITEMS,
      onRestart,
    });
    fireEvent.click(screen.getByTestId('gen-batch-restart-btn'));
    expect(onRestart).toHaveBeenCalledOnce();
  });

  it('[P1] 重試失敗項目 takes precedence — the two never appear together', () => {
    renderPanel({
      status: 'complete',
      progress: progressOf({ status: 'complete', failCount: 1 }),
      items: ITEMS,
      failedIds: new Set([M2]),
      onRetryFailed: vi.fn(),
      onRestart: vi.fn(),
    });
    expect(screen.getByTestId('gen-batch-retry-failed-btn')).toBeInTheDocument();
    expect(screen.queryByTestId('gen-batch-restart-btn')).toBeNull();
  });

  it('[P1] never at budget_ceiling — 下次繼續 owns recovery there', () => {
    renderPanel({
      status: 'budget_ceiling',
      progress: progressOf({ status: 'budget_ceiling', pausedCount: 2 }),
      items: ITEMS,
      onRestart: vi.fn(),
    });
    expect(screen.queryByTestId('gen-batch-restart-btn')).toBeNull();
  });
});

// ---------------------------------------------------------------------------
// dsr-6d-b container — the on-open probe, `last`, 409, reconnect, caches
// ---------------------------------------------------------------------------

describe('GenerationBatchDialogV2 — probe / last / reconnect (dsr-6d-b AC #5)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    h.batchState.status = 'idle';
    h.batchState.batchId = '';
    h.batchState.currentMediaId = null;
    h.batchState.currentItem = '';
    h.batchState.totalItems = 0;
    h.batchState.successCount = 0;
    h.batchState.failCount = 0;
    h.batchState.pausedCount = 0;
    h.batchState.items = null;
    h.batchEpoch = 0;
    h.itemState.phase = 'idle';
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: false, progress: null });
  });

  function renderOpenable(initialOpen = true) {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const invalidate = vi.spyOn(queryClient, 'invalidateQueries');
    const remove = vi.spyOn(queryClient, 'removeQueries');
    let open = initialOpen;
    const makeUi = () => (
      <QueryClientProvider client={queryClient}>
        <GenerationBatchDialogV2 open={open} onOpenChange={vi.fn()} />
      </QueryClientProvider>
    );
    const view = render(makeUi());
    return {
      queryClient,
      invalidate,
      remove,
      rerender: () => view.rerender(makeUi()),
      setOpen: (next: boolean) => {
        open = next;
        view.rerender(makeUi());
      },
    };
  }

  const LAST = {
    batchId: 'gb-last',
    totalItems: 2,
    currentIndex: 2,
    currentMediaId: M2,
    currentItem: '奧本海默',
    successCount: 1,
    failCount: 1,
    pausedCount: 0,
    status: 'complete' as const,
    spentUsd: 1.2,
    budgetUsd: 5,
    items: [state(M1, '沙丘：第二部', 'done'), state(M2, '奧本海默', 'failed', 'error')],
  };

  it('[P0] path 1 — a RUNNING batch is tracked (stream attached)', async () => {
    const snapshot = progressOf({ status: 'running' });
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: true, progress: snapshot });

    renderOpenable();

    await waitFor(() => expect(h.batchStartTracking).toHaveBeenCalledWith(snapshot));
    expect(h.batchAttachSnapshot).not.toHaveBeenCalled();
  });

  it('[P0] path 2 — a finished batch is ATTACHED, not tracked (no stream, terminal status kept)', async () => {
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: false, last: LAST });

    renderOpenable();

    await waitFor(() => expect(h.batchAttachSnapshot).toHaveBeenCalledWith(LAST));
    expect(h.batchStartTracking).not.toHaveBeenCalled();
  });

  it('[P0] path 3 — nothing running and nothing remembered → the consent flow', async () => {
    mocked.getGenerationBatchStatus.mockResolvedValue({
      running: false,
      progress: null,
      last: null,
    });

    renderOpenable();

    expect(await screen.findByTestId('consent-view-stub')).toBeInTheDocument();
    expect(h.batchAttachSnapshot).not.toHaveBeenCalled();
  });

  it('[P0] 🚨 the SAME last batch is shown ONCE — reopening goes back to the consent flow', async () => {
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: false, last: LAST });
    const { setOpen } = renderOpenable();

    await waitFor(() => expect(h.batchAttachSnapshot).toHaveBeenCalledTimes(1));

    // Close (the hook resets to idle) and reopen.
    setOpen(false);
    h.batchState.status = 'idle';
    setOpen(true);

    expect(await screen.findByTestId('consent-view-stub')).toBeInTheDocument();
    expect(h.batchAttachSnapshot).toHaveBeenCalledTimes(1);
  });

  it('[P0] 再產生字幕 returns to the consent flow and starts NOTHING', async () => {
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: false, last: LAST });
    h.batchState.status = 'complete';
    h.batchState.batchId = 'gb-last';
    h.batchState.successCount = 2;
    h.batchState.items = [state(M1, '沙丘：第二部', 'done'), state(M2, '奧本海默', 'done')];
    const { rerender } = renderOpenable();

    fireEvent.click(await screen.findByTestId('gen-batch-restart-btn'));
    expect(h.batchReset).toHaveBeenCalled();
    expect(mocked.startGenerationBatch).not.toHaveBeenCalled();

    h.batchState.status = 'idle';
    rerender();
    expect(await screen.findByTestId('consent-view-stub')).toBeInTheDocument();
  });

  it('[P0] 🔴 #4 a 409 with an EMPTY body re-reads status instead of pinning the panel at 0 / 0', async () => {
    mocked.startGenerationBatch.mockResolvedValue({
      conflict: true,
      progress: null as never,
    });
    mocked.getGenerationBatchStatus
      .mockResolvedValueOnce({ running: false, progress: null })
      .mockResolvedValueOnce({ running: false, last: LAST });

    renderOpenable();
    fireEvent.click(await screen.findByTestId('consent-stub-start'));

    await waitFor(() => expect(mocked.getGenerationBatchStatus).toHaveBeenCalledTimes(2));
    expect(h.batchStartTracking).not.toHaveBeenCalled();
    await waitFor(() => expect(h.batchAttachSnapshot).toHaveBeenCalledWith(LAST));
  });

  it('[P0] a reconnect (connectionEpoch tick) re-reads status — the cure for a lost terminal event', async () => {
    const running = progressOf({ status: 'running' });
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: true, progress: running });
    h.batchState.status = 'running';
    h.batchState.currentItem = '怪奇物語 S04E07';
    const { rerender } = renderOpenable();
    await waitFor(() => expect(mocked.getGenerationBatchStatus).toHaveBeenCalledTimes(1));

    // The stream dropped and came back; meanwhile the batch finished.
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: false, last: LAST });
    h.batchEpoch = 1;
    rerender();

    await waitFor(() => expect(h.batchAttachSnapshot).toHaveBeenCalledWith(LAST));
  });

  it('[P0] the 202 snapshot seeds the panel (queue + real ceiling), no SSE event needed', async () => {
    const started = progressOf({
      batchId: 'gb-202',
      totalItems: 2,
      status: 'running',
      items: [state(M1, '沙丘：第二部', 'running'), state(M2, '奧本海默', 'queued')],
    });
    mocked.startGenerationBatch.mockResolvedValue({
      conflict: false,
      result: { batchId: 'gb-202', totalItems: 2, items: ITEMS.slice(0, 2), progress: started },
    });

    renderOpenable();
    fireEvent.click(await screen.findByTestId('consent-stub-start'));

    await waitFor(() => expect(h.batchStartTracking).toHaveBeenCalledWith(started));
  });

  it('[P0] an empty scope (200, batch_id null) stays in the consent flow and says why', async () => {
    mocked.startGenerationBatch.mockResolvedValue({
      conflict: false,
      result: { batchId: null, totalItems: 0, items: [], progress: null },
    });

    renderOpenable();
    fireEvent.click(await screen.findByTestId('consent-stub-start'));

    await waitFor(() =>
      expect(screen.getByTestId('consent-stub-error')).toHaveTextContent('沒有可以生成的項目')
    );
    expect(h.batchStartTracking).not.toHaveBeenCalled();
  });
});

describe('GenerationBatchDialogV2 — terminal caches (dsr-6d-b AC #6)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    h.batchState.status = 'idle';
    h.batchState.batchId = '';
    h.batchState.currentMediaId = null;
    h.batchState.currentItem = '';
    h.batchState.items = null;
    h.batchEpoch = 0;
    h.itemState.phase = 'idle';
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: false, progress: null });
  });

  function renderTerminal() {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const invalidate = vi.spyOn(queryClient, 'invalidateQueries');
    const remove = vi.spyOn(queryClient, 'removeQueries');
    const makeUi = () => (
      <QueryClientProvider client={queryClient}>
        <GenerationBatchDialogV2 open onOpenChange={vi.fn()} />
      </QueryClientProvider>
    );
    const view = render(makeUi());
    return { invalidate, remove, rerender: () => view.rerender(makeUi()) };
  }

  const keysOf = (invalidate: ReturnType<typeof vi.spyOn>) =>
    invalidate.mock.calls.map((c) => JSON.stringify((c[0] as { queryKey: unknown }).queryKey));

  it('[P0] 🔴 #11 a terminal refreshes library, preview, DETAILS, ACTIVITY and the per-title quote', async () => {
    h.batchState.status = 'complete';
    h.batchState.batchId = 'gb-t1';
    const { invalidate } = renderTerminal();

    await waitFor(() => expect(invalidate).toHaveBeenCalled());
    const keys = keysOf(invalidate);
    expect(keys).toContain(JSON.stringify(['library']));
    expect(keys).toContain(JSON.stringify(['subtitles', 'generation-batch', 'preview']));
    expect(keys).toContain(JSON.stringify(['details']));
    expect(keys).toContain(JSON.stringify(['activity']));
    expect(keys).toContain(JSON.stringify(['subtitles', 'transcription-estimate']));
  });

  it('[P0] 🔴 #6 the cached queue is removed ON TERMINAL (the file header promise)', async () => {
    h.batchState.status = 'complete';
    h.batchState.batchId = 'gb-t2';
    const { remove } = renderTerminal();

    await waitFor(() =>
      expect(
        remove.mock.calls.some(
          (c) =>
            JSON.stringify((c[0] as { queryKey: unknown }).queryKey) ===
            JSON.stringify(['subtitles', 'generation-batch', 'items'])
        )
      ).toBe(true)
    );
  });

  it('[P0] 🚨 the same terminal batch invalidates ONCE — reopening it must not re-sweep the library', async () => {
    h.batchState.status = 'complete';
    h.batchState.batchId = 'gb-t3';
    const { invalidate, rerender } = renderTerminal();

    await waitFor(() => expect(invalidate).toHaveBeenCalled());
    const first = invalidate.mock.calls.length;
    expect(first).toBeGreaterThan(0);

    // Close (hook back to idle), then reopen — the probe re-attaches the SAME
    // `last` snapshot, so the terminal effect fires again with the same id.
    h.batchState.status = 'idle';
    rerender();
    h.batchState.status = 'complete';
    rerender();
    await waitFor(() => expect(screen.getByTestId('gen-batch-close-btn')).toBeInTheDocument());

    expect(invalidate.mock.calls.length).toBe(first);
  });

  it('[P1] a DIFFERENT terminal batch does invalidate again', async () => {
    h.batchState.status = 'complete';
    h.batchState.batchId = 'gb-t4';
    const { invalidate, rerender } = renderTerminal();

    await waitFor(() => expect(invalidate).toHaveBeenCalled());
    const first = invalidate.mock.calls.length;

    h.batchState.status = 'running';
    rerender();
    h.batchState.status = 'complete';
    h.batchState.batchId = 'gb-t5';
    rerender();

    await waitFor(() => expect(invalidate.mock.calls.length).toBeGreaterThan(first));
  });
});

describe('GenerationBatchDialogV2 — items-first kills the two lies (dsr-6d-b 🔴 #1/#2)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    h.batchState.status = 'idle';
    h.batchState.batchId = '';
    h.batchState.currentMediaId = null;
    h.batchState.currentItem = '';
    h.batchState.successCount = 0;
    h.batchState.failCount = 0;
    h.batchState.items = null;
    h.batchEpoch = 0;
    h.itemState.phase = 'idle';
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: false, progress: null });
  });

  function renderPlain() {
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

  it('[P0] 🔴 #1 an item refused with busy_elsewhere is retryable — the button EXISTS again', async () => {
    h.batchState.status = 'complete';
    h.batchState.batchId = 'gb-busy';
    h.batchState.successCount = 1;
    h.batchState.failCount = 1;
    h.batchState.items = [
      state(M1, '沙丘：第二部', 'done'),
      state(M2, '奧本海默', 'failed', 'busy_elsewhere'),
    ];

    const { rerender } = renderPlain();

    const retry = await screen.findByTestId('gen-batch-retry-failed-btn');
    fireEvent.click(retry);
    h.batchState.status = 'idle';
    rerender();

    const stub = await screen.findByTestId('consent-view-stub');
    expect(stub.getAttribute('data-preselected')).toBe(M2);
  });

  it('[P0] 🔴 #2 a per-item failure never bleeds onto the NEXT item once the backend owns the queue', async () => {
    h.batchState.status = 'running';
    h.batchState.batchId = 'gb-race2';
    h.batchState.currentMediaId = M2;
    h.batchState.items = [
      state(M1, '沙丘：第二部', 'failed', 'error'),
      state(M2, '奧本海默', 'running'),
    ];
    // The per-item stream still reports `failed` — it belongs to M1, and the
    // stale phase used to get recorded against M2 on the next render.
    h.itemState.phase = 'failed';

    renderPlain();

    await waitFor(() =>
      expect(screen.getByTestId(`gen-batch-row-${M2}`)).toHaveAttribute('data-state', 'running')
    );
    expect(screen.getByTestId(`gen-batch-row-${M2}`)).not.toHaveTextContent('失敗');
    expect(screen.getByTestId(`gen-batch-row-${M1}`)).toHaveTextContent('生成失敗');
  });

  it('[P0] 下次繼續 preselects the unfinished rows straight from the backend queue', async () => {
    h.batchState.status = 'budget_ceiling';
    h.batchState.batchId = 'gb-ceil';
    h.batchState.pausedCount = 2;
    h.batchState.items = [
      state(M1, '沙丘：第二部', 'done'),
      state(M2, '奧本海默', 'failed', 'error'),
      state(M3, '怪奇物語 S04E07', 'paused'),
      state(M4, '星際效應', 'paused'),
    ];

    const { rerender } = renderPlain();

    fireEvent.click(await screen.findByTestId('gen-batch-resume-btn'));
    h.batchState.status = 'idle';
    rerender();

    const stub = await screen.findByTestId('consent-view-stub');
    expect(stub.getAttribute('data-preselected')).toBe([M2, M3, M4].join(','));
  });
});

// ---------------------------------------------------------------------------
// Honest behaviours that had NO test before (dsr-6d-b AC #7/#8)
// ---------------------------------------------------------------------------

describe('GenerationBatchPanelV2 — claims that must stay true', () => {
  it('[P0] the 即時更新（SSE）chip claims a LIVE stream — present while running', () => {
    renderPanel({ status: 'running', progress: progressOf({}), items: ITEMS });
    expect(screen.getByTestId('gen-batch-sse-chip')).toBeInTheDocument();
  });

  it.each(['complete', 'cancelled', 'error', 'budget_ceiling'] as const)(
    '[P0] the SSE chip is GONE at %s — the stream is closed, the badge must not say otherwise',
    (status) => {
      renderPanel({ status, progress: progressOf({ status }), items: ITEMS });
      expect(screen.queryByTestId('gen-batch-sse-chip')).toBeNull();
    }
  );

  // ⚠️ 「執行中點外面不關」 is NOT covered here on purpose: Radix's
  // DismissableLayer does not dispatch its outside-pointer event under jsdom,
  // so a negative assertion would pass whatever the code did (the exact
  // vacuous-test trap dsr-6d-b's CR called out). It is covered in a real
  // browser by tests/e2e/batch-subtitle.spec.ts instead.
  it('[P0] Escape stays gated while running (the same guard, in a form jsdom CAN observe)', () => {
    const props = renderPanel({ status: 'running', progress: progressOf({}), items: ITEMS });
    fireEvent.keyDown(screen.getByTestId('generation-batch-dialog-v2'), { key: 'Escape' });
    expect(props.onClose).not.toHaveBeenCalled();
  });
});

// ---------------------------------------------------------------------------
// dsr-6d-b CR — the dead ends the adversarial review found
// ---------------------------------------------------------------------------

describe('GenerationBatchDialogV2 — CR regressions (no dead ends)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    h.batchState.status = 'idle';
    h.batchState.batchId = '';
    h.batchState.currentMediaId = null;
    h.batchState.currentItem = '';
    h.batchState.totalItems = 0;
    h.batchState.successCount = 0;
    h.batchState.failCount = 0;
    h.batchState.pausedCount = 0;
    h.batchState.items = null;
    h.batchEpoch = 0;
    h.itemState.phase = 'idle';
    mocked.getGenerationBatchStatus.mockResolvedValue({ running: false, progress: null });
  });

  function renderCr() {
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

  it('[P0] H1 a batch that ALREADY FINISHED by the time the 202 is read is attached, never painted as running', async () => {
    // One title the pipeline refuses outright: Start() returns, the goroutine
    // fails + finishes, and SnapshotFor() hands back a TERMINAL snapshot. The
    // terminal SSE event was broadcast before we ever opened the stream.
    const finished = progressOf({
      batchId: 'gb-instant',
      totalItems: 1,
      status: 'complete',
      successCount: 0,
      failCount: 1,
      items: [state(M1, '沙丘：第二部', 'failed', 'busy_elsewhere')],
    });
    mocked.startGenerationBatch.mockResolvedValue({
      conflict: false,
      result: { batchId: 'gb-instant', totalItems: 1, items: [ITEMS[0]], progress: finished },
    });

    renderCr();
    fireEvent.click(await screen.findByTestId('consent-stub-start'));

    await waitFor(() => expect(h.batchAttachSnapshot).toHaveBeenCalledWith(finished));
    // startTracking would have forced status:'running' AND opened a stream for
    // a batch that is already over — the 進行中 dead end.
    expect(h.batchStartTracking).not.toHaveBeenCalled();
  });

  it('[P1] H1 a genuinely running 202 snapshot still goes through startTracking (stream opens)', async () => {
    const running = progressOf({
      batchId: 'gb-live',
      status: 'running',
      items: [state(M1, '沙丘：第二部', 'running')],
    });
    mocked.startGenerationBatch.mockResolvedValue({
      conflict: false,
      result: { batchId: 'gb-live', totalItems: 1, items: [ITEMS[0]], progress: running },
    });

    renderCr();
    fireEvent.click(await screen.findByTestId('consent-stub-start'));

    await waitFor(() => expect(h.batchStartTracking).toHaveBeenCalledWith(running));
    expect(h.batchAttachSnapshot).not.toHaveBeenCalled();
  });

  it('[P0] H2 a batch you WATCHED finish is not replayed on the next open — a new selection survives', async () => {
    // Watch batch A complete with a failure while the dialog is open.
    h.batchState.status = 'complete';
    h.batchState.batchId = 'gb-A';
    h.batchState.successCount = 1;
    h.batchState.failCount = 1;
    h.batchState.items = [
      state(M1, '沙丘：第二部', 'done'),
      state(M2, '奧本海默', 'failed', 'error'),
    ];
    // The server still remembers it (`last` is kept until the next batch).
    mocked.getGenerationBatchStatus.mockResolvedValue({
      running: false,
      last: { ...progressOf({ batchId: 'gb-A', status: 'complete' }), items: h.batchState.items },
    });

    const { rerender } = renderCr();
    await screen.findByTestId('gen-batch-close-btn');

    // Close → the hook resets → reopen (same mount, as ActivityHub does).
    h.batchState.status = 'idle';
    h.batchState.batchId = '';
    h.batchState.items = null;
    rerender();

    // The consent flow appears — NOT yesterday's completion screen, and
    // therefore no 重試失敗項目 to hijack preselectedIds.
    const stub = await screen.findByTestId('consent-view-stub');
    expect(stub.getAttribute('data-preselected')).toBe('');
    expect(h.batchAttachSnapshot).not.toHaveBeenCalled();
  });

  it('[P0] M5 re-attaching a terminal still forces a fresh candidate analysis (CR H2 must not regress)', async () => {
    h.batchState.status = 'complete';
    h.batchState.batchId = 'gb-M5';
    const { rerender } = renderCr();
    await screen.findByTestId('gen-batch-close-btn');

    // Same batch reaches the effect a second time (reopen → `last` re-attached).
    h.batchState.status = 'idle';
    rerender();
    h.batchState.status = 'complete';
    rerender();
    await screen.findByTestId('gen-batch-close-btn');

    // …and the consent flow that follows must still re-analyze: the completed
    // items are still listed in the stale snapshot with the wrong quotes.
    h.batchState.status = 'idle';
    rerender();
    const stub = await screen.findByTestId('consent-view-stub');
    expect(stub.getAttribute('data-force-analyze')).toBe('true');
  });

  it('[P0] M8 a 409 whose re-read finds nothing says so instead of silently doing nothing', async () => {
    mocked.startGenerationBatch.mockResolvedValue({ conflict: true, progress: null });
    mocked.getGenerationBatchStatus.mockResolvedValue({
      running: false,
      progress: null,
      last: null,
    });

    renderCr();
    fireEvent.click(await screen.findByTestId('consent-stub-start'));

    await waitFor(() => expect(mocked.getGenerationBatchStatus).toHaveBeenCalledTimes(2));
    expect(await screen.findByTestId('consent-stub-error')).toHaveTextContent(
      '剛才那個批次已經結束了'
    );
  });

  it('[P1] M8 …and the message is cleared once the re-read DOES find the batch', async () => {
    const running = progressOf({ status: 'running' });
    mocked.startGenerationBatch.mockResolvedValue({ conflict: true, progress: null });
    mocked.getGenerationBatchStatus
      .mockResolvedValueOnce({ running: false, progress: null })
      .mockResolvedValue({ running: true, progress: running });

    renderCr();
    fireEvent.click(await screen.findByTestId('consent-stub-start'));

    await waitFor(() => expect(h.batchStartTracking).toHaveBeenCalledWith(running));
    expect(screen.queryByTestId('consent-stub-error')).toBeNull();
  });

  it('[P0] L9 an idempotent cancel on an already-finished batch re-reads status instead of hanging on 取消中…', async () => {
    h.batchState.status = 'running';
    h.batchState.batchId = 'gb-late';
    h.batchState.currentMediaId = M3;
    h.batchState.currentItem = '怪奇物語 S04E07';
    // The server answers 200 {cancelled:false, running:false}: nothing to
    // cancel, because the batch ended and we missed the terminal event.
    mocked.cancelGenerationBatch.mockResolvedValue({ cancelled: false, running: false });
    mocked.getGenerationBatchStatus
      .mockResolvedValueOnce({ running: false, progress: null })
      .mockResolvedValue({
        running: false,
        last: progressOf({ batchId: 'gb-late', status: 'complete', successCount: 4 }),
      });

    renderCr();
    fireEvent.click(screen.getByTestId('gen-batch-cancel-all'));
    fireEvent.click(screen.getByTestId('gen-batch-cancel-confirm-btn'));

    await waitFor(() => expect(mocked.getGenerationBatchStatus).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(h.batchAttachSnapshot).toHaveBeenCalled());
  });
});

describe('GenerationBatchPanelV2 — phone layout (dsr-6f-3, F8-M-v2 H717g)', () => {
  const t = (el: Element) => el.className.split(/\s+/).filter(Boolean);

  it('pins 本次用量 to the footer on a phone: two copies, one per breakpoint', () => {
    renderPanel({ items: ITEMS });

    // The in-body copy keeps its testids (desktop, and everything already
    // written against them); it just stops rendering below sm:.
    const bodyCopy = screen.getByTestId('gen-batch-cost-line').parentElement!;
    expect(t(bodyCopy)).toContain('max-sm:hidden');
    expect(screen.getByTestId('gen-batch-body')).toContainElement(bodyCopy);

    // The phone copy lives in the FIXED footer, so the money spent cannot be
    // scrolled out of sight by a long queue.
    const footer = screen.getByTestId('gen-batch-footer');
    const phoneLine = screen.getByTestId('gen-batch-cost-line-mobile');
    expect(footer).toContainElement(phoneLine);
    expect(t(phoneLine.parentElement!)).toContain('sm:hidden');
    // F8-M draws the footer cost row in Label 12, the desktop body one is Body 14.
    expect(t(phoneLine)).toContain('text-xs');
    expect(phoneLine).toHaveTextContent('本次用量');
    expect(screen.getByTestId('gen-batch-sse-chip-mobile')).toBeInTheDocument();
  });

  it('the phone SSE chip follows the same isRunning gate as the desktop one', () => {
    renderPanel({
      status: 'complete',
      progress: progressOf({ status: 'complete', successCount: 5 }),
      items: ITEMS,
    });
    expect(screen.getByTestId('gen-batch-cost-line-mobile')).toBeInTheDocument();
    expect(screen.queryByTestId('gen-batch-sse-chip')).toBeNull();
    expect(screen.queryByTestId('gen-batch-sse-chip-mobile')).toBeNull();
  });

  it('the footer stacks on a phone and carries the sheet gutter + safe area', () => {
    renderPanel({ items: ITEMS });
    expect(t(screen.getByTestId('gen-batch-footer'))).toEqual(
      expect.arrayContaining([
        'max-sm:flex-col',
        'max-sm:items-stretch',
        'max-sm:px-4',
        'max-sm:pb-[max(0.75rem,env(safe-area-inset-bottom))]',
      ])
    );
    expect(t(screen.getByTestId('gen-batch-body'))).toEqual(
      expect.arrayContaining(['max-sm:px-4', 'max-sm:pt-1.5', 'max-sm:gap-3.5'])
    );
    // Column + items-stretch already makes the lone button full-width; all it
    // needs is its label centred.
    expect(t(screen.getByTestId('gen-batch-cancel-all'))).toContain('max-sm:justify-center');
  });

  it('cancel-confirm: both sentences take a line each, the two buttons split the row', () => {
    renderPanel({
      items: ITEMS,
      onConfirmCancelAll: vi.fn().mockRejectedValue(new Error('500')),
    });
    fireEvent.click(screen.getByTestId('gen-batch-cancel-all'));

    const row = screen.getByTestId('gen-batch-cancel-confirm');
    expect(t(row.querySelector('span')!)).toContain('max-sm:w-full');
    expect(t(screen.getByRole('button', { name: '繼續生成' }))).toEqual(
      expect.arrayContaining(['max-sm:flex-1', 'max-sm:justify-center'])
    );
    expect(t(screen.getByTestId('gen-batch-cancel-confirm-btn'))).toEqual(
      expect.arrayContaining(['max-sm:flex-1', 'max-sm:justify-center'])
    );
  });

  it('cancel-confirm: the FAILURE sentence gets its own line too (it is short enough to share one)', async () => {
    renderPanel({
      items: ITEMS,
      onConfirmCancelAll: vi.fn().mockRejectedValue(new Error('500')),
    });
    fireEvent.click(screen.getByTestId('gen-batch-cancel-all'));
    fireEvent.click(screen.getByTestId('gen-batch-cancel-confirm-btn'));
    const alert = await screen.findByTestId('gen-batch-cancel-error');
    expect(t(alert)).toContain('max-sm:w-full');
  });

  it('terminal / budget-ceiling: the two buttons share an sm:contents wrapper and split the row', () => {
    renderPanel({
      status: 'budget_ceiling',
      progress: progressOf({ status: 'budget_ceiling', pausedCount: 3, spentUsd: 5 }),
      items: ITEMS,
    });
    const group = screen.getByTestId('gen-batch-footer-actions');
    // sm:contents ⇒ on a desktop the wrapper has no box and both buttons are
    // direct flex children of the footer exactly as before.
    expect(t(group)).toEqual(expect.arrayContaining(['sm:contents', 'max-sm:w-full']));
    for (const id of ['gen-batch-close-btn', 'gen-batch-resume-btn']) {
      expect(group).toContainElement(screen.getByTestId(id));
      expect(t(screen.getByTestId(id))).toEqual(
        expect.arrayContaining(['max-sm:flex-1', 'max-sm:justify-center'])
      );
    }
  });

  it.each([
    ['gen-batch-retry-failed-btn', { onRetryFailed: vi.fn() }],
    ['gen-batch-restart-btn', { onRestart: vi.fn() }],
  ] as const)('terminal: 關閉 + %s split the row inside the same wrapper', (second, extra) => {
    renderPanel({
      status: 'complete',
      progress: progressOf({ status: 'complete', successCount: 4, failCount: 1 }),
      items: ITEMS,
      ...extra,
    });
    const group = screen.getByTestId('gen-batch-footer-actions');
    for (const id of ['gen-batch-close-btn', second]) {
      expect(group).toContainElement(screen.getByTestId(id));
      expect(t(screen.getByTestId(id))).toEqual(
        expect.arrayContaining(['max-sm:flex-1', 'max-sm:justify-center'])
      );
    }
  });

  it('the running card uses the 12px phone padding of F8-M', () => {
    renderPanel({ items: ITEMS });
    expect(t(screen.getByTestId(`gen-batch-row-${M1}`))).toContain('max-sm:p-3');
  });
});

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';

const h = vi.hoisted(() => ({
  analysisState: {
    status: 'idle' as string,
    analyzed: 0,
    total: 0,
    error: null as string | null,
  },
  startTracking: vi.fn(),
  reset: vi.fn(),
  models: undefined as unknown,
  modelsError: false,
}));

vi.mock('../../../hooks/useGenerationCandidatesProgress', () => ({
  useGenerationCandidatesProgress: () => ({
    progress: h.analysisState,
    status: h.analysisState.status,
    startTracking: h.startTracking,
    reset: h.reset,
  }),
}));

vi.mock('../../../services/subtitleService', () => ({
  subtitleService: {
    getGenerationCandidates: vi.fn(),
    startCandidateAnalysis: vi.fn(),
    cancelCandidateAnalysis: vi.fn(),
    getModels: vi.fn(),
  },
}));

// The catalog is server state behind TanStack Query; stubbing the hook keeps
// this file free of a QueryClientProvider (the hook itself is covered in
// useTranslationModels.spec.ts).
vi.mock('../../../hooks/useTranslationModels', () => ({
  useTranslationModels: () => ({ data: h.models, isError: h.modelsError }),
}));

import { GenerationConsentView } from './GenerationConsentView';
import {
  subtitleService,
  type CandidateAnalysisSnapshot,
  type TranslationModelList,
} from '../../../services/subtitleService';

const mocked = vi.mocked(subtitleService);

/** sub-6-8b: the catalog the F16/F19 picker offers (backend display order). */
const MODELS: TranslationModelList = {
  defaultModelId: 'claude-sonnet-5',
  models: [
    {
      id: 'claude-haiku-4-5',
      provider: 'claude',
      displayName: 'Claude Haiku 4.5',
      tier: 'fast',
      isDefault: false,
      qualityGrade: 'B',
      qualityNote: 'Vido 實測 2026-09',
    },
    {
      id: 'claude-sonnet-5',
      provider: 'claude',
      displayName: 'Claude Sonnet 5',
      tier: 'balanced',
      isDefault: true,
      qualityGrade: 'A',
      qualityNote: 'Vido 實測 2026-09',
    },
  ],
};

const A = '0a54a9e2-3a67-4f3e-9f8e-a1c2d3e4f501';
const B = '1b65baf3-4b78-4a4f-8a9f-b2d3e4f5a602';
const EP = '8fa9fed7-8fbc-4e8d-8edc-f6b7c8d9e006';

const READY: CandidateAnalysisSnapshot = {
  status: 'ready',
  analyzed: 3,
  total: 3,
  result: {
    candidates: [
      {
        mediaId: A,
        mediaType: 'movie',
        title: '沙丘：第二部',
        route: 'extract',
        runtimeMinutes: 166,
        runtimeKnown: true,
        estimatedUsd: 0.05,
      },
      {
        mediaId: EP,
        mediaType: 'episode',
        title: '怪奇物語 S04E07',
        route: 'asr',
        runtimeMinutes: 52,
        runtimeKnown: true,
        estimatedUsd: 0.31,
      },
      {
        mediaId: B,
        mediaType: 'movie',
        title: '略過的項目',
        route: 'skip',
        runtimeMinutes: 0,
        runtimeKnown: false,
        estimatedUsd: 0,
      },
    ],
    summary: {
      extractCount: 1,
      asrCount: 1,
      skippedCount: 1,
      estimatedTotalUsd: 0.36,
      selfHostedAsr: false,
    },
  },
};

/**
 * The same sweep as READY, priced under both models (sub-6-8a AC #3). Haiku is
 * ~2.5x cheaper here, which is the gap this whole story exists to surface.
 */
const READY_PRICED: CandidateAnalysisSnapshot = {
  ...READY,
  result: {
    ...READY.result!,
    estimatesByModel: {
      'claude-sonnet-5': { totalUsd: 0.36, perCandidate: { [A]: 0.05, [EP]: 0.31 } },
      'claude-haiku-4-5': { totalUsd: 0.14, perCandidate: { [A]: 0.02, [EP]: 0.12 } },
    },
    // Whole-sweep minutes over 166 + 52 = 218 runtime minutes.
    estimatedMinutesByModel: { 'claude-sonnet-5': 37, 'claude-haiku-4-5': 24 },
  },
};

function renderView(props: Partial<React.ComponentProps<typeof GenerationConsentView>> = {}) {
  const merged: React.ComponentProps<typeof GenerationConsentView> = {
    open: true,
    onStartBatch: vi.fn(),
    onClose: vi.fn(),
    ...props,
  };
  render(<GenerationConsentView {...merged} />);
  return merged;
}

describe('GenerationConsentView (sub-4-3 container)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    h.models = MODELS;
    h.modelsError = false;
    h.analysisState.status = 'idle';
    h.analysisState.analyzed = 0;
    h.analysisState.total = 0;
    h.analysisState.error = null;
  });

  it('[P0 AC #2] ready snapshot renders the list with the default extract-only selection; skip rows never render', async () => {
    mocked.getGenerationCandidates.mockResolvedValue(READY);

    renderView();

    await waitFor(() => expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument());
    // skip row is excluded entirely
    expect(screen.queryByText('略過的項目')).not.toBeInTheDocument();
    // default selection = extract only → summary $0.05
    expect(screen.getByTestId('consent-summary-usd')).toHaveTextContent('$0.05');
    expect(mocked.startCandidateAnalysis).not.toHaveBeenCalled();
  });

  it('[P0 AC #1] idle snapshot kicks a fresh analysis and shows F14', async () => {
    mocked.getGenerationCandidates.mockResolvedValue({ status: 'idle', analyzed: 0, total: 0 });
    mocked.startCandidateAnalysis.mockResolvedValue({ started: true });

    renderView();

    await waitFor(() => expect(mocked.startCandidateAnalysis).toHaveBeenCalledOnce());
    expect(h.startTracking).toHaveBeenCalled();
    expect(screen.getByTestId('consent-analysis-panel')).toBeInTheDocument();
  });

  it('[P0 AC #1] a 409-already-running analysis is a JOIN, not an error', async () => {
    mocked.getGenerationCandidates.mockResolvedValue({ status: 'idle', analyzed: 0, total: 0 });
    mocked.startCandidateAnalysis.mockResolvedValue({ alreadyRunning: true });

    renderView();

    await waitFor(() => expect(h.startTracking).toHaveBeenCalled());
    expect(screen.getByTestId('consent-analysis-panel')).toBeInTheDocument();
    expect(screen.queryByTestId('consent-load-error')).not.toBeInTheDocument();
  });

  it('analyzing snapshot joins with seeded counts (no new analyze POST)', async () => {
    mocked.getGenerationCandidates.mockResolvedValue({
      status: 'analyzing',
      analyzed: 234,
      total: 1247,
    });

    renderView();

    await waitFor(() =>
      expect(h.startTracking).toHaveBeenCalledWith({ analyzed: 234, total: 1247 })
    );
    expect(mocked.startCandidateAnalysis).not.toHaveBeenCalled();
  });

  it('[P0 AC #5] zero listable candidates renders the F20 empty state with 關閉 only', async () => {
    mocked.getGenerationCandidates.mockResolvedValue({
      ...READY,
      result: {
        candidates: [READY.result!.candidates[2]], // skip only
        summary: { ...READY.result!.summary, extractCount: 0, asrCount: 0 },
      },
    });

    const props = renderView();

    await waitFor(() => expect(screen.getByTestId('consent-empty-state')).toBeInTheDocument());
    // Appears both in the visible copy and the sr-only live region.
    expect(screen.getByTestId('consent-empty-state')).toHaveTextContent('所有影片都有繁中字幕了');
    expect(screen.queryByTestId('consent-start-btn')).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId('consent-empty-close'));
    expect(props.onClose).toHaveBeenCalled();
  });

  it('[P0 AC #7] preselected ids intersect the candidate list (mixed movie+episode)', async () => {
    mocked.getGenerationCandidates.mockResolvedValue(READY);

    renderView({ preselectedIds: [EP, '9ff0c000-dead-4bee-8f00-000000000999'] });

    await waitFor(() => expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument());
    // Only the ASR episode is preselected → summary shows its estimate.
    expect(screen.getByTestId('consent-summary-usd')).toHaveTextContent('$0.31');
  });

  it('[P0 AC #4] confirm flow hands over (list-order ids, WYSIWYG budget)', async () => {
    mocked.getGenerationCandidates.mockResolvedValue(READY);
    const props = renderView();

    await waitFor(() => expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument());
    // Select the episode too (default only has the extract movie).
    fireEvent.click(screen.getByLabelText('選取 怪奇物語 S04E07'));
    // Raise the ceiling to a custom value — the ON-SCREEN number must be sent.
    fireEvent.change(screen.getByTestId('consent-budget-input'), { target: { value: '2.50' } });
    fireEvent.click(screen.getByTestId('consent-start-btn'));

    // F16 confirm dialog totals ride the same selector.
    expect(screen.getByTestId('consent-confirm-total-usd')).toHaveTextContent('$0.36');
    fireEvent.click(screen.getByTestId('consent-confirm-start'));

    // sub-6-8b: the model whose price was on screen travels with the batch.
    expect(props.onStartBatch).toHaveBeenCalledWith([A, EP], 2.5, 'claude-sonnet-5');
  });

  // ─── sub-6-8b: per-run model selection ────────────────────────────────────

  it('[P0 AC #1] the picker pre-selects is_default and prices every model for THIS batch', async () => {
    mocked.getGenerationCandidates.mockResolvedValue(READY_PRICED);
    renderView();

    await waitFor(() => expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('consent-start-btn'));

    const sonnet = screen.getByTestId('consent-model-option-claude-sonnet-5');
    const haiku = screen.getByTestId('consent-model-option-claude-haiku-4-5');
    expect(sonnet).toHaveAttribute('data-selected', 'true');
    expect(haiku).toHaveAttribute('data-selected', 'false');

    // Default selection = the extract movie only ($0.05 Sonnet / $0.02 Haiku),
    // NOT the whole sweep — the row prices what is actually about to run.
    expect(screen.getByTestId('consent-model-usd-claude-sonnet-5')).toHaveTextContent('$0.05');
    expect(screen.getByTestId('consent-model-usd-claude-haiku-4-5')).toHaveTextContent('$0.02');
    expect(screen.getByTestId('consent-model-grade-claude-sonnet-5')).toHaveTextContent('品質 A');
    expect(screen.getByTestId('consent-model-grade-claude-haiku-4-5')).toHaveTextContent('品質 B');
    // 37 sweep-minutes x (166/218 selected runtime) = 28.
    expect(screen.getByTestId('consent-model-minutes-claude-sonnet-5')).toHaveTextContent(
      '約 28 分鐘'
    );
    expect(screen.getByTestId('consent-model-minutes-claude-haiku-4-5')).toHaveTextContent(
      '約 18 分鐘'
    );
  });

  it('[P0 AC #3] switching model re-prices the summary bar, the footer AND the confirm total together', async () => {
    mocked.getGenerationCandidates.mockResolvedValue(READY_PRICED);
    renderView();

    await waitFor(() => expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument());
    fireEvent.click(screen.getByLabelText('選取 怪奇物語 S04E07'));
    fireEvent.click(screen.getByTestId('consent-start-btn'));

    expect(screen.getByTestId('consent-summary-usd')).toHaveTextContent('$0.36');
    expect(screen.getByTestId('consent-footer-usd')).toHaveTextContent('$0.36');
    expect(screen.getByTestId('consent-confirm-total-usd')).toHaveTextContent('$0.36');

    fireEvent.click(screen.getByLabelText(/Claude Haiku 4.5/));

    // All three move, or the screen is lying to somebody.
    expect(screen.getByTestId('consent-summary-usd')).toHaveTextContent('$0.14');
    expect(screen.getByTestId('consent-footer-usd')).toHaveTextContent('$0.14');
    expect(screen.getByTestId('consent-confirm-total-usd')).toHaveTextContent('$0.14');
    // Per-row amounts follow too (抽取 movie: $0.05 -> $0.02).
    expect(screen.getByTestId(`consent-row-usd-${A}`)).toHaveTextContent('$0.02');
  });

  it('[P0 AC #4] the cheaper choice states the gap in money, not just in grade', async () => {
    mocked.getGenerationCandidates.mockResolvedValue(READY_PRICED);
    renderView();

    await waitFor(() => expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument());
    fireEvent.click(screen.getByLabelText('選取 怪奇物語 S04E07'));
    fireEvent.click(screen.getByTestId('consent-start-btn'));

    expect(screen.getByTestId('consent-model-note-claude-sonnet-5')).toHaveTextContent(
      'eval-1 實測品質最穩'
    );
    fireEvent.click(screen.getByLabelText(/Claude Haiku 4.5/));
    expect(screen.getByTestId('consent-model-note-claude-haiku-4-5')).toHaveTextContent(
      '比 Claude Sonnet 5 省 $0.22（61%）'
    );
  });

  it('[P0 AC #3] the F18 budget verdict follows the chosen model, not the default', async () => {
    mocked.getGenerationCandidates.mockResolvedValue(READY_PRICED);
    renderView();

    await waitFor(() => expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument());
    fireEvent.click(screen.getByLabelText('選取 怪奇物語 S04E07'));
    fireEvent.change(screen.getByTestId('consent-budget-input'), { target: { value: '0.20' } });
    fireEvent.click(screen.getByTestId('consent-start-btn'));

    // $0.36 under Sonnet blows the $0.20 ceiling…
    expect(screen.getByTestId('consent-over-budget-banner')).toBeInTheDocument();
    expect(screen.getByTestId('consent-confirm-start')).toHaveTextContent('仍要開始');

    // …and $0.14 under Haiku does not. Same ceiling, different verdict.
    fireEvent.click(screen.getByLabelText(/Claude Haiku 4.5/));
    expect(screen.queryByTestId('consent-over-budget-banner')).not.toBeInTheDocument();
    expect(screen.getByTestId('consent-confirm-start')).toHaveTextContent('確認並開始');
  });

  it('[P0 AC #2] the chosen model id rides the start payload', async () => {
    mocked.getGenerationCandidates.mockResolvedValue(READY_PRICED);
    const props = renderView();

    await waitFor(() => expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('consent-start-btn'));
    fireEvent.click(screen.getByLabelText(/Claude Haiku 4.5/));
    fireEvent.click(screen.getByTestId('consent-confirm-start'));

    expect(props.onStartBatch).toHaveBeenCalledWith([A], 5, 'claude-haiku-4-5');
  });

  it('[P0 AC #2] a pre-sub-6-8a server (no estimates_by_model) offers the default model ALONE', async () => {
    // Quoting Haiku at the default model rate would be a lie in a money field;
    // the honest degrade is one row, priced from what the server did send.
    mocked.getGenerationCandidates.mockResolvedValue(READY);
    renderView();

    await waitFor(() => expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('consent-start-btn'));

    expect(screen.getByTestId('consent-model-option-claude-sonnet-5')).toBeInTheDocument();
    expect(screen.queryByTestId('consent-model-option-claude-haiku-4-5')).not.toBeInTheDocument();
    // No estimated_minutes_by_model either — the time column stays absent
    // rather than inventing a duration.
    expect(screen.queryByTestId('consent-model-minutes-claude-sonnet-5')).not.toBeInTheDocument();
  });

  it('no catalog (no AI key configured) renders no picker and still starts', async () => {
    h.models = { models: [], defaultModelId: '' };
    mocked.getGenerationCandidates.mockResolvedValue(READY_PRICED);
    const props = renderView();

    await waitFor(() => expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('consent-start-btn'));

    expect(screen.queryByTestId('consent-model-picker')).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId('consent-confirm-start'));
    // Empty model id = "use the deployment default", which is what the
    // unpriced fallback quoted.
    expect(props.onStartBatch).toHaveBeenCalledWith([A], 5, '');
  });

  it('[P0 CR H1] a catalog that loses the picked model re-prices AND re-sends together — never one without the other', async () => {
    // The failure this guards: the catalog refetches without Haiku (a key was
    // edited) while the dialog is open. The old code kept pricing from
    // `modelId` but validated against `choices` at confirm time, so the screen
    // showed Haiku's cheap total and the batch went out with an empty model id
    // — billed at the server default. Quote and charge from different models is
    // the one outcome this whole story exists to prevent.
    mocked.getGenerationCandidates.mockResolvedValue(READY_PRICED);
    const props = renderView();

    await waitFor(() => expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument());
    fireEvent.click(screen.getByLabelText('選取 怪奇物語 S04E07'));
    fireEvent.click(screen.getByTestId('consent-start-btn'));
    fireEvent.click(screen.getByLabelText(/Claude Haiku 4.5/));
    expect(screen.getByTestId('consent-confirm-total-usd')).toHaveTextContent('$0.14');

    // Catalog shrinks to Sonnet only, mid-dialog. Any interaction re-renders and
    // picks up the new catalog — the budget field is the cheapest one to poke
    // (a same-value change fires no event, so it must differ from the prefill).
    h.models = { defaultModelId: 'claude-sonnet-5', models: [MODELS.models[1]] };
    fireEvent.change(screen.getByTestId('consent-budget-input'), { target: { value: '6.00' } });

    // The price snaps back to Sonnet's…
    expect(screen.getByTestId('consent-confirm-total-usd')).toHaveTextContent('$0.36');
    expect(screen.getByTestId('consent-summary-usd')).toHaveTextContent('$0.36');
    // …the radiogroup still has a row checked (CR M3 — never zero) …
    expect(screen.getByTestId('consent-model-option-claude-sonnet-5')).toHaveAttribute(
      'data-selected',
      'true'
    );
    // …and the id sent is the one just priced, not '' and not the vanished Haiku.
    fireEvent.click(screen.getByTestId('consent-confirm-start'));
    expect(props.onStartBatch).toHaveBeenCalledWith([A, EP], 6, 'claude-sonnet-5');
  });

  it('[CR M4] a FAILED catalog says so — it is not the same fact as「沒設金鑰」', async () => {
    h.models = undefined;
    h.modelsError = true;
    mocked.getGenerationCandidates.mockResolvedValue(READY_PRICED);
    renderView();

    await waitFor(() => expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('consent-start-btn'));

    expect(screen.queryByTestId('consent-model-picker')).not.toBeInTheDocument();
    expect(screen.getByTestId('consent-model-catalog-error')).toHaveTextContent('無法載入模型清單');
  });

  it('an empty catalog (no key) stays silent — nothing failed', async () => {
    h.models = { models: [], defaultModelId: '' };
    mocked.getGenerationCandidates.mockResolvedValue(READY_PRICED);
    renderView();

    await waitFor(() => expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('consent-start-btn'));

    expect(screen.queryByTestId('consent-model-picker')).not.toBeInTheDocument();
    expect(screen.queryByTestId('consent-model-catalog-error')).not.toBeInTheDocument();
  });

  it('SSE ready transition refetches the snapshot for the result', async () => {
    mocked.getGenerationCandidates
      .mockResolvedValueOnce({ status: 'analyzing', analyzed: 1, total: 3 })
      .mockResolvedValueOnce(READY);

    const view = render(<GenerationConsentView open onStartBatch={vi.fn()} onClose={vi.fn()} />);
    await waitFor(() => expect(h.startTracking).toHaveBeenCalled());

    h.analysisState.status = 'ready';
    view.rerender(<GenerationConsentView open onStartBatch={vi.fn()} onClose={vi.fn()} />);

    await waitFor(() => expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument());
    expect(mocked.getGenerationCandidates).toHaveBeenCalledTimes(2);
  });

  it('F14 取消 cancels the analysis and closes', async () => {
    mocked.getGenerationCandidates.mockResolvedValue({
      status: 'analyzing',
      analyzed: 10,
      total: 100,
    });
    mocked.cancelCandidateAnalysis.mockResolvedValue({ cancelled: true });

    const props = renderView();

    await waitFor(() => expect(screen.getByTestId('consent-analysis-panel')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('consent-analysis-cancel'));

    await waitFor(() => expect(mocked.cancelCandidateAnalysis).toHaveBeenCalledOnce());
    expect(props.onClose).toHaveBeenCalled();
  });

  it('[P0 CR H2] forceAnalyze ignores a ready snapshot and kicks a fresh analysis', async () => {
    mocked.getGenerationCandidates.mockResolvedValue(READY);
    mocked.startCandidateAnalysis.mockResolvedValue({ started: true });

    renderView({ forceAnalyze: true });

    await waitFor(() => expect(mocked.startCandidateAnalysis).toHaveBeenCalledOnce());
    expect(screen.getByTestId('consent-analysis-panel')).toBeInTheDocument();
    expect(screen.queryByTestId('consent-candidate-list')).not.toBeInTheDocument();
  });

  it('[P0 CR M5] the analyzing phase polls the snapshot as a missed-ready safety net', async () => {
    vi.useFakeTimers();
    try {
      mocked.getGenerationCandidates
        .mockResolvedValueOnce({ status: 'analyzing', analyzed: 1, total: 3 })
        .mockResolvedValue(READY);

      renderView();

      await vi.waitFor(() => expect(h.startTracking).toHaveBeenCalled());
      // The SSE ready event never arrives — the 5s poll must rescue F14.
      await vi.advanceTimersByTimeAsync(5_000);
      await vi.waitFor(() =>
        expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument()
      );
      expect(mocked.getGenerationCandidates.mock.calls.length).toBeGreaterThanOrEqual(2);
    } finally {
      vi.useRealTimers();
    }
  });

  it('[CR L8] the confirm dialog stays open while the start is in flight and closes on failure', async () => {
    mocked.getGenerationCandidates.mockResolvedValue(READY);
    const onStartBatch = vi.fn();
    const view = render(
      <GenerationConsentView open starting={false} onStartBatch={onStartBatch} onClose={vi.fn()} />
    );
    await waitFor(() => expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument());

    fireEvent.click(screen.getByTestId('consent-start-btn'));
    fireEvent.click(screen.getByTestId('consent-confirm-start'));
    expect(onStartBatch).toHaveBeenCalledOnce();

    // Still open (starting in flight) — spinner state is now reachable.
    view.rerender(
      <GenerationConsentView open starting onStartBatch={onStartBatch} onClose={vi.fn()} />
    );
    expect(screen.getByTestId('consent-confirm-dialog')).toBeInTheDocument();
    expect(screen.getByTestId('consent-confirm-start')).toBeDisabled();

    // Failure → the error effect closes the confirm and surfaces the message
    // in the list panel.
    view.rerender(
      <GenerationConsentView
        open
        starting={false}
        startError="批次生成啟動失敗"
        onStartBatch={onStartBatch}
        onClose={vi.fn()}
      />
    );
    await waitFor(() =>
      expect(screen.queryByTestId('consent-confirm-dialog')).not.toBeInTheDocument()
    );
    expect(screen.getByTestId('consent-start-error')).toHaveTextContent('批次生成啟動失敗');
  });
});

describe('GenerationConsentView budget prefill (sub-5-1 AC #6)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    h.models = MODELS;
    h.modelsError = false;
    h.analysisState.status = 'idle';
    h.analysisState.analyzed = 0;
    h.analysisState.total = 0;
    h.analysisState.error = null;
  });

  it('[P0] prefills the budget input from the snapshot default_budget_usd', async () => {
    mocked.getGenerationCandidates.mockResolvedValue({ ...READY, defaultBudgetUsd: 12 });

    render(<GenerationConsentView open onStartBatch={vi.fn()} onClose={vi.fn()} />);

    await waitFor(() => expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument());
    expect(screen.getByTestId('consent-budget-input')).toHaveValue(12);
  });

  it('[P0] keeps the $5.00 fallback when the snapshot carries no default (pre-sub-5-1 server)', async () => {
    mocked.getGenerationCandidates.mockResolvedValue(READY);

    render(<GenerationConsentView open onStartBatch={vi.fn()} onClose={vi.fn()} />);

    await waitFor(() => expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument());
    expect(screen.getByTestId('consent-budget-input')).toHaveValue(5);
  });

  it('keeps the fallback for a non-positive default (0 = unlimited must not prefill 0.00)', async () => {
    mocked.getGenerationCandidates.mockResolvedValue({ ...READY, defaultBudgetUsd: 0 });

    render(<GenerationConsentView open onStartBatch={vi.fn()} onClose={vi.fn()} />);

    await waitFor(() => expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument());
    expect(screen.getByTestId('consent-budget-input')).toHaveValue(5);
  });

  it('the kick-analyze exit gets the prefill too — it survives to the list shown after SSE ready', async () => {
    mocked.getGenerationCandidates
      .mockResolvedValueOnce({ status: 'idle', analyzed: 0, total: 0, defaultBudgetUsd: 8 })
      .mockResolvedValueOnce(READY); // SSE-ready refetch carries no field; prefill must persist
    mocked.startCandidateAnalysis.mockResolvedValue({ started: true });

    const view = render(<GenerationConsentView open onStartBatch={vi.fn()} onClose={vi.fn()} />);
    await waitFor(() => expect(h.startTracking).toHaveBeenCalled());

    h.analysisState.status = 'ready';
    view.rerender(<GenerationConsentView open onStartBatch={vi.fn()} onClose={vi.fn()} />);

    await waitFor(() => expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument());
    expect(screen.getByTestId('consent-budget-input')).toHaveValue(8);
  });

  it('WYSIWYG is untouched: the prefilled value is exactly what a confirm sends', async () => {
    mocked.getGenerationCandidates.mockResolvedValue({ ...READY, defaultBudgetUsd: 3.75 });
    const onStartBatch = vi.fn();

    render(<GenerationConsentView open onStartBatch={onStartBatch} onClose={vi.fn()} />);

    await waitFor(() => expect(screen.getByTestId('consent-candidate-list')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('consent-start-btn'));
    fireEvent.click(screen.getByTestId('consent-confirm-start'));

    expect(onStartBatch).toHaveBeenCalledWith([A], 3.75, 'claude-sonnet-5');
  });
});

// ---------------------------------------------------------------------------
// sub-5-3 — 三序同源 at the view seam + the error-phase retry pre-existing fix
// ---------------------------------------------------------------------------

describe('GenerationConsentView — grouped order (sub-5-3 AC #2)', () => {
  const SRS = 'b1c2d3e4-f5a6-4b7c-8d9e-0f1a2b3c4d5e';
  const EP1 = '9a0bfe08-1acd-4f9e-9fed-a7c8d9e0f201';
  const EP2 = '9a0bfe08-1acd-4f9e-9fed-a7c8d9e0f202';

  const GROUPED_READY: CandidateAnalysisSnapshot = {
    status: 'ready',
    analyzed: 3,
    total: 3,
    result: {
      candidates: [
        // Backend (title,id) order interleaves the episode BEFORE the movie —
        // grouped order must put the movie first, then the series run.
        {
          mediaId: EP2,
          mediaType: 'episode',
          title: 'Ep Beta S01E02',
          route: 'asr',
          runtimeMinutes: 50,
          runtimeKnown: true,
          estimatedUsd: 0.3,
          seriesId: SRS,
          seriesTitle: '怪奇物語',
          seasonNumber: 1,
          episodeNumber: 2,
        },
        {
          mediaId: A,
          mediaType: 'movie',
          title: '沙丘：第二部',
          route: 'extract',
          runtimeMinutes: 166,
          runtimeKnown: true,
          estimatedUsd: 0.05,
        },
        {
          mediaId: EP1,
          mediaType: 'episode',
          title: 'Ep Alpha S01E01',
          route: 'asr',
          runtimeMinutes: 50,
          runtimeKnown: true,
          estimatedUsd: 0.3,
          seriesId: SRS,
          seriesTitle: '怪奇物語',
          seasonNumber: 1,
          episodeNumber: 1,
        },
      ],
      summary: {
        extractCount: 1,
        asrCount: 2,
        skippedCount: 0,
        estimatedTotalUsd: 0.65,
        selfHostedAsr: false,
      },
    },
  };

  beforeEach(() => {
    vi.clearAllMocks();
    h.models = MODELS;
    h.modelsError = false;
    h.analysisState.status = 'idle';
  });

  it('submits ids in GROUPED order — display, submission and the feasible walk share one order', async () => {
    mocked.getGenerationCandidates.mockResolvedValue(GROUPED_READY);
    const onStartBatch = vi.fn();
    renderView({ onStartBatch });

    // Select everything, then start → confirm.
    const selectAll = await screen.findByTestId('consent-select-all');
    fireEvent.click(selectAll);
    fireEvent.click(screen.getByTestId('consent-start-btn'));
    fireEvent.click(await screen.findByTestId('consent-confirm-start'));

    expect(onStartBatch).toHaveBeenCalledTimes(1);
    const [ids] = onStartBatch.mock.calls[0];
    // movie first (flat section), then the series episodes in E01→E02 order —
    // NOT the backend's title order (which began with EP2).
    expect(ids).toEqual([A, EP1, EP2]);
  });

  it('renders the series header row from the grouped snapshot', async () => {
    mocked.getGenerationCandidates.mockResolvedValue(GROUPED_READY);
    renderView();
    expect(await screen.findByTestId(`consent-group-${SRS}`)).toHaveTextContent('怪奇物語');
  });
});

describe('GenerationConsentView — error-phase 重試 (pre-existing fix, sub-5-3)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    h.models = MODELS;
    h.modelsError = false;
    h.analysisState.status = 'idle';
  });

  it('重試 actually re-bootstraps instead of throwing on the missing guard param', async () => {
    // First load fails → error phase.
    mocked.getGenerationCandidates.mockRejectedValueOnce(new Error('網路爆炸'));
    renderView();
    expect(await screen.findByTestId('consent-load-error')).toHaveTextContent('網路爆炸');

    // Retry succeeds: bootstrap must run to the list phase, not TypeError.
    mocked.getGenerationCandidates.mockResolvedValue(READY);
    fireEvent.click(screen.getByRole('button', { name: '重試' }));

    expect(await screen.findByTestId('consent-candidate-list')).toBeInTheDocument();
  });
});

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
  },
}));

import { GenerationConsentView } from './GenerationConsentView';
import { subtitleService, type CandidateAnalysisSnapshot } from '../../../services/subtitleService';

const mocked = vi.mocked(subtitleService);

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

    expect(props.onStartBatch).toHaveBeenCalledWith([A, EP], 2.5);
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

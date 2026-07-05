import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
} from '@tanstack/react-router';

const h = vi.hoisted(() => ({
  genState: {
    phase: 'idle' as string,
    failedPhase: null as string | null,
    percentage: null as number | null,
    message: '',
    jobId: null,
    error: null as string | null,
    srtPath: null,
    zhSrtPath: null,
  },
  startTracking: vi.fn(),
  reset: vi.fn(),
  genOptions: undefined as { onComplete?: (p: unknown) => void } | undefined,
  glossaryTerms: [{ id: 't1' }, { id: 't2' }, { id: 't3' }] as unknown[],
  fetchHook: {
    search: vi.fn(),
    isSearching: false,
    searchError: null as Error | null,
    results: [] as unknown[],
    resultCount: 0,
    sortBy: 'score',
    sortOrder: 'desc',
    toggleSort: vi.fn(),
    download: vi.fn(),
    downloadingIds: new Set<string>(),
    downloadedIds: new Set<string>(),
    downloadErrorMap: {} as Record<string, string>,
    preview: vi.fn(),
    previewDataMap: {},
    previewingId: null,
    isPreviewing: false,
    downloadStage: null,
  },
}));

vi.mock('../../hooks/useGenerationProgress', () => ({
  useGenerationProgress: (options?: { onComplete?: (p: unknown) => void }) => {
    h.genOptions = options;
    return { progress: h.genState, startTracking: h.startTracking, reset: h.reset };
  },
}));

vi.mock('../../hooks/useGlossary', () => ({
  useGlossaryTerms: () => ({ data: h.glossaryTerms }),
}));

vi.mock('../../hooks/useSubtitleSearch', () => ({
  useSubtitleSearch: () => h.fetchHook,
}));

vi.mock('../../services/transcriptionService', () => ({
  transcriptionService: { startTranscription: vi.fn() },
}));

vi.mock('./GlossaryPanelV2', () => ({
  GlossaryPanelV2: ({ open }: { open: boolean }) =>
    open ? <div data-testid="glossary-panel-stub" /> : null,
}));

import { ManageSubtitleDialogV2 } from './ManageSubtitleDialogV2';
import { transcriptionService } from '../../services/transcriptionService';

const mockedTrigger = vi.mocked(transcriptionService.startTranscription);

type DialogProps = Partial<React.ComponentProps<typeof ManageSubtitleDialogV2>>;

function renderDialog(props: DialogProps = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const merged: React.ComponentProps<typeof ManageSubtitleDialogV2> = {
    mediaId: '42',
    mediaType: 'movie',
    mediaTitle: '怪奇物語',
    mediaFilePath: '/media/movies/st.mkv',
    subtitleTracks: JSON.stringify([{ language: 'en' }]),
    open: true,
    onOpenChange: vi.fn(),
    ...props,
  };
  const rootRoute = createRootRoute({
    component: () => (
      <QueryClientProvider client={queryClient}>
        <ManageSubtitleDialogV2 {...merged} />
      </QueryClientProvider>
    ),
  });
  const settings = createRoute({
    getParentRoute: () => rootRoute,
    path: '/settings',
    component: () => null,
  });
  const router = createRouter({
    routeTree: rootRoute.addChildren([settings]),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  });
  return render(<RouterProvider router={router} />);
}

beforeEach(() => {
  vi.clearAllMocks();
  h.genState = {
    phase: 'idle',
    failedPhase: null,
    percentage: null,
    message: '',
    jobId: null,
    error: null,
    srtPath: null,
    zhSrtPath: null,
  };
  h.glossaryTerms = [{ id: 't1' }, { id: 't2' }, { id: 't3' }];
  h.fetchHook.results = [];
  h.fetchHook.searchError = null;
  h.fetchHook.isSearching = false;
});

describe('ManageSubtitleDialogV2 (F1 管理字幕)', () => {
  it('renders tracks with language pill + source, 生成字幕 primary, glossary entry with Mono count', async () => {
    renderDialog({
      subtitleTracks: JSON.stringify([{ language: 'zh-Hant' }, { language: 'en' }]),
    });

    expect(await screen.findByTestId('manage-subtitle-dialog-v2')).toBeInTheDocument();
    expect(screen.getByText('管理字幕 — 怪奇物語')).toBeInTheDocument();
    expect(screen.getByTestId('subtitle-tracks-section')).toBeInTheDocument();
    expect(screen.getByText('繁中')).toBeInTheDocument();
    expect(screen.getByText('英文')).toBeInTheDocument();
    expect(screen.getByTestId('action-generate-subtitle')).toHaveTextContent('生成字幕');
    expect(screen.getByTestId('open-glossary')).toHaveTextContent('名詞對照表');
    expect(screen.getByTestId('open-glossary')).toHaveTextContent('3');
    // Dormant fetch: a footer text-link only — NO source chips, NO Zimuku.
    expect(screen.getByTestId('toggle-fetch')).toHaveTextContent('搜尋線上字幕（成功率低）');
    expect(screen.queryByText(/Zimuku/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Assrt/)).not.toBeInTheDocument();
  });

  it('shows the authoritative engine row when subtitleStatus=found (ux3-0-2 semantics)', async () => {
    renderDialog({
      subtitleTracks: undefined,
      subtitleStatus: 'found',
      subtitleLanguage: 'zh-Hant',
    });

    const row = await screen.findByTestId('subtitle-track-engine');
    expect(row).toHaveTextContent('繁中');
    expect(row).toHaveTextContent('字幕引擎');
  });

  it('renders the F2 缺字幕 empty state when there are no tracks (distinct from failure)', async () => {
    renderDialog({ subtitleTracks: undefined });

    expect(await screen.findByTestId('subtitle-empty-state')).toBeInTheDocument();
    expect(screen.getByText('尚無字幕')).toBeInTheDocument();
    expect(screen.getByText('此影片目前沒有任何字幕軌')).toBeInTheDocument();
    expect(screen.queryByTestId('generation-trigger-error')).not.toBeInTheDocument();
  });

  it('series: 生成字幕 renders DISABLED with hint 影集字幕生成即將推出 (capability honor)', async () => {
    renderDialog({ mediaType: 'series' });

    const cta = await screen.findByTestId('action-generate-subtitle');
    expect(cta).toBeDisabled();
    expect(screen.getByText('影集字幕生成即將推出')).toBeInTheDocument();

    fireEvent.click(cta);
    expect(mockedTrigger).not.toHaveBeenCalled();
  });

  it('movie: 生成字幕 POSTs the transcribe trigger with the int64 id and enters the progress view', async () => {
    mockedTrigger.mockResolvedValue({
      status: 'started',
      result: { jobId: 'job-9', message: 'ok' },
    });
    renderDialog();

    fireEvent.click(await screen.findByTestId('action-generate-subtitle'));

    await waitFor(() => expect(mockedTrigger).toHaveBeenCalledWith(42));
    await waitFor(() => expect(h.startTracking).toHaveBeenCalledWith(42));
    expect(screen.getByText('生成字幕 — 怪奇物語')).toBeInTheDocument();
    expect(screen.getByTestId('generation-progress-v2')).toBeInTheDocument();
    expect(screen.getByText('即時更新（SSE）')).toBeInTheDocument();
  });

  it('503 TRANSCRIPTION_DISABLED → F5 尚未設定 warning panel + 前往設定 (never hard-fails)', async () => {
    mockedTrigger.mockResolvedValue({ status: 'disabled' });
    renderDialog();

    fireEvent.click(await screen.findByTestId('action-generate-subtitle'));

    expect(await screen.findByTestId('generation-not-configured')).toBeInTheDocument();
    expect(screen.getByText('字幕生成尚未設定')).toBeInTheDocument();
    expect(screen.getByTestId('go-to-settings')).toHaveTextContent('前往設定');
    // The rest of the dialog survives (fail-soft): tracks + glossary still visible.
    expect(screen.getByTestId('subtitle-tracks-section')).toBeInTheDocument();
    expect(screen.getByTestId('open-glossary')).toBeInTheDocument();
  });

  it('409 TRANSCRIPTION_IN_PROGRESS → attaches to the running job SSE stream instead of erroring', async () => {
    mockedTrigger.mockResolvedValue({ status: 'inProgress' });
    renderDialog();

    fireEvent.click(await screen.findByTestId('action-generate-subtitle'));

    await waitFor(() => expect(h.startTracking).toHaveBeenCalledWith(42));
    expect(screen.getByTestId('generation-progress-v2')).toBeInTheDocument();
    expect(screen.queryByTestId('generation-trigger-error')).not.toBeInTheDocument();
  });

  it('trigger 404/400/500 → fail-soft error with 重試', async () => {
    mockedTrigger.mockRejectedValueOnce(new Error('找不到電影'));
    mockedTrigger.mockResolvedValueOnce({
      status: 'started',
      result: { jobId: 'job-9', message: 'ok' },
    });
    renderDialog();

    fireEvent.click(await screen.findByTestId('action-generate-subtitle'));

    const panel = await screen.findByTestId('generation-trigger-error');
    expect(panel).toHaveTextContent('無法開始生成：找不到電影');

    fireEvent.click(screen.getByTestId('generation-trigger-retry'));
    await waitFor(() => expect(mockedTrigger).toHaveBeenCalledTimes(2));
  });

  it('§9b CN policy: 簡中 track on CN content shows the policy line (policy-correct, not a defect)', async () => {
    renderDialog({
      subtitleTracks: JSON.stringify([{ language: 'zh-CN' }]),
      productionCountry: 'CN',
    });

    expect(await screen.findByTestId('cn-policy-note-track-0')).toHaveTextContent(
      '陸劇保留簡體字幕（對白一致）'
    );
  });

  it('no CN policy line for 簡中 tracks on non-CN content', async () => {
    renderDialog({ subtitleTracks: JSON.stringify([{ language: 'zh-CN' }]) });

    await screen.findByTestId('subtitle-tracks-section');
    expect(screen.queryByTestId('cn-policy-note-track-0')).not.toBeInTheDocument();
  });

  it('renders the F10 loading skeleton while the parent detail is loading', async () => {
    renderDialog({ isLoading: true });

    expect(await screen.findByTestId('manage-subtitle-skeleton')).toBeInTheDocument();
    expect(screen.queryByTestId('action-generate-subtitle')).not.toBeInTheDocument();
  });

  it('opens the glossary panel from the 名詞對照表 row', async () => {
    renderDialog();

    fireEvent.click(await screen.findByTestId('open-glossary'));
    expect(screen.getByTestId('glossary-panel-stub')).toBeInTheDocument();
  });

  it('dormant fetch expands on demand and searches WITHOUT source chips or score rows', async () => {
    h.fetchHook.results = [
      {
        id: 's1',
        source: 'assrt',
        filename: 'Stranger.Things.S01E01.zh.srt',
        language: 'zh-TW',
        downloadUrl: '',
        downloads: 100,
        group: 'grp',
        resolution: '1080p',
        format: 'srt',
        score: 0.9,
        scoreBreakdown: { language: 1, resolution: 1, sourceTrust: 1, group: 1, downloads: 1 },
      },
    ];
    renderDialog();

    fireEvent.click(await screen.findByTestId('toggle-fetch'));
    expect(screen.getByTestId('fetch-section')).toBeInTheDocument();

    fireEvent.click(screen.getByTestId('fetch-search'));
    expect(h.fetchHook.search).toHaveBeenCalledWith({
      mediaId: '42',
      mediaType: 'movie',
      query: '怪奇物語',
    });

    const row = screen.getByTestId('fetch-result-s1');
    expect(row).toHaveTextContent('Stranger.Things.S01E01.zh.srt');
    // NO score badge, NO score breakdown, NO source chip text in the row.
    expect(row).not.toHaveTextContent('0.9');
    expect(row).not.toHaveTextContent('assrt');
  });

  it('fires onGenerationComplete when the SSE complete callback lands (AC 6 hook-up)', async () => {
    const onGenerationComplete = vi.fn();
    renderDialog({ onGenerationComplete });

    await screen.findByTestId('manage-subtitle-dialog-v2');
    // The dialog registered an onComplete with the generation hook — simulate the SSE terminal.
    h.genOptions?.onComplete?.({ srtPath: '/a.srt', zhSrtPath: '/a.zh.srt' });

    expect(onGenerationComplete).toHaveBeenCalledTimes(1);
  });

  it('closing the dialog resets generation tracking (job continues server-side)', async () => {
    const onOpenChange = vi.fn();
    renderDialog({ onOpenChange });

    fireEvent.click(await screen.findByTestId('dialog-close'));

    expect(h.reset).toHaveBeenCalled();
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});

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
    srtPath: null as string | null,
    zhSrtPath: null as string | null,
    partial: false as boolean,
    englishKeptBlocks: null as number | null,
  },
  startTracking: vi.fn(),
  reset: vi.fn(),
  genOptions: undefined as { onComplete?: (p: unknown) => void } | undefined,
  glossaryTerms: [{ id: 't1' }, { id: 't2' }, { id: 't3' }] as unknown[],
  // 9R-10c red line 1: the glossary is per-SHOW, so episode mode must query it
  // with the SERIES id while the transcribe target stays the episode id.
  // Captured so the two ids are assertable as DIFFERENT at the dialog seam.
  glossaryQueriedId: undefined as string | undefined,
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
  useGlossaryTerms: (mediaId: string) => {
    h.glossaryQueriedId = mediaId;
    return { data: h.glossaryTerms };
  },
}));

vi.mock('../../hooks/useSubtitleSearch', () => ({
  useSubtitleSearch: () => h.fetchHook,
}));

vi.mock('../../services/transcriptionService', () => ({
  transcriptionService: {
    startTranscription: vi.fn(),
    startEpisodeTranscription: vi.fn(),
    // dsr-6a: the price on the paid buttons (GET …/transcribe/estimate).
    getTranscriptionEstimate: vi.fn(),
  },
}));

vi.mock('./GlossaryPanelV2', () => ({
  GlossaryPanelV2: ({ open }: { open: boolean }) =>
    open ? <div data-testid="glossary-panel-stub" /> : null,
}));

import { ManageSubtitleDialogV2 } from './ManageSubtitleDialogV2';
import {
  transcriptionService,
  type TranscriptionEstimate,
} from '../../services/transcriptionService';
import { ApiError } from '../../lib/apiError';

const mockedTrigger = vi.mocked(transcriptionService.startTranscription);
const mockedEpisodeTrigger = vi.mocked(transcriptionService.startEpisodeTranscription);
const mockedEstimate = vi.mocked(transcriptionService.getTranscriptionEstimate);

function readyEstimate(overrides: Partial<TranscriptionEstimate> = {}): TranscriptionEstimate {
  return {
    mediaId: '4f8c2d1a-5b6e-4c7d-8e9f-0a1b2c3d4e5f',
    mediaType: 'movie',
    plan: 'full',
    asrAvailable: true,
    selfHostedAsr: false,
    translationConfigured: true,
    modelId: 'claude-sonnet-5',
    runtimeMinutes: 30,
    runtimeKnown: true,
    runtimeSource: 'ffprobe',
    estimatedUsd: 0.42,
    ...overrides,
  };
}

/** The CTA is a skeleton until the estimate lands — wait for its amount first. */
async function findPricedGenerate() {
  await screen.findByTestId('action-generate-subtitle-amount');
  return screen.getByTestId('action-generate-subtitle');
}

type DialogProps = Partial<React.ComponentProps<typeof ManageSubtitleDialogV2>>;

// Media-id fixture convention (9R-18 AC 7): media ids are UUID STRINGS —
// mirror the prod creation path (uuid.New().String()); do NOT invent numeric ids.
const MOVIE_UUID = '4f8c2d1a-5b6e-4c7d-8e9f-0a1b2c3d4e5f';

/** Owns `open` like a real parent does, with a way to open the dialog again. */
function ControlledDialog(props: React.ComponentProps<typeof ManageSubtitleDialogV2>) {
  const [open, setOpen] = React.useState(true);
  return (
    <>
      <button type="button" data-testid="reopen" onClick={() => setOpen(true)}>
        reopen
      </button>
      <ManageSubtitleDialogV2 {...props} open={open} onOpenChange={setOpen} />
    </>
  );
}

/** Lets a test change `subtitleStatus` under an open dialog, as a parent refetch would. */
function StatusSwitchDialog(props: React.ComponentProps<typeof ManageSubtitleDialogV2>) {
  const [status, setStatus] = React.useState(props.subtitleStatus);
  return (
    <>
      <ManageSubtitleDialogV2 {...props} subtitleStatus={status} />
      {/* Portalled dialog content sits beside this; the button stays reachable. */}
      <button type="button" data-testid="mark-found" onClick={() => setStatus('found')}>
        mark found
      </button>
    </>
  );
}

function renderDialog(
  props: DialogProps = {},
  options: { controlled?: boolean; statusSwitch?: boolean } = {}
) {
  const queryClient = new QueryClient({
    defaultOptions: {
      // The controlled harness runs with production's 5-minute staleTime, so a
      // reopen proves the dialog DROPS the price rather than it merely going stale.
      queries: { retry: false, ...(options.controlled ? { staleTime: 5 * 60 * 1000 } : {}) },
      mutations: { retry: false },
    },
  });
  const merged: React.ComponentProps<typeof ManageSubtitleDialogV2> = {
    mediaId: MOVIE_UUID,
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
        {options.controlled ? (
          <ControlledDialog {...merged} />
        ) : options.statusSwitch ? (
          <StatusSwitchDialog {...merged} />
        ) : (
          <ManageSubtitleDialogV2 {...merged} />
        )}
      </QueryClientProvider>
    ),
  });
  const settings = createRoute({
    getParentRoute: () => rootRoute,
    path: '/settings',
    component: () => null,
  });
  // sub-2-1b AC #4 — the F5 前往設定 target. Registered so the navigation the
  // dialog performs resolves to a real route rather than silently 404-ing.
  const settingsKeys = createRoute({
    getParentRoute: () => rootRoute,
    path: '/settings/keys',
    component: () => null,
  });
  const router = createRouter({
    routeTree: rootRoute.addChildren([settings, settingsKeys]),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  });
  return { ...render(<RouterProvider router={router} />), router };
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
    partial: false,
    englishKeptBlocks: null,
  };
  h.glossaryTerms = [{ id: 't1' }, { id: 't2' }, { id: 't3' }];
  h.fetchHook.results = [];
  h.fetchHook.searchError = null;
  h.fetchHook.isSearching = false;
  // Default: a measured-runtime movie, translation configured → ① $0.42.
  mockedEstimate.mockResolvedValue(readyEstimate());
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
    expect(await findPricedGenerate()).toHaveTextContent('生成字幕');
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

  // sub-2-2d AC #2 — the degraded CTA helper; since dsr-6a the signal is the
  // estimate's translationConfigured (the run's own check), not /settings/keys.
  it('helper・default (key configured): 語音辨識＋AI 翻譯，約需數分鐘 — the ruled verb', async () => {
    renderDialog();
    await findPricedGenerate();
    const helper = screen.getByTestId('generation-helper');
    expect(helper).toHaveTextContent('語音辨識＋AI 翻譯，約需數分鐘');
    expect(screen.queryByTestId('helper-goto-settings')).toBeNull();
  });

  it('helper・degraded (key unconfigured): states the en-only truth + 前往設定, CTA stays ENABLED', async () => {
    mockedEstimate.mockResolvedValue(
      readyEstimate({ translationConfigured: false, modelId: '', estimatedUsd: 0.18 })
    );
    renderDialog();

    const cta = await findPricedGenerate();
    const helper = screen.getByTestId('generation-helper');
    expect(helper).toHaveTextContent('僅能產生英文字幕——尚未設定翻譯金鑰');
    expect(screen.getByTestId('helper-goto-settings')).toBeInTheDocument();
    // Degraded ≠ blocked (the party-mode asymmetry): the CTA must stay live —
    // and it charges only the speech-recognition half.
    expect(cta).toBeEnabled();
    expect(screen.getByTestId('action-generate-subtitle-amount').textContent).toBe('$0.18');
  });

  it('helper・loading: the default line and a skeleton — never a clickable button without a price', async () => {
    mockedEstimate.mockReturnValue(new Promise(() => {}));
    renderDialog();

    const cta = await screen.findByTestId('action-generate-subtitle');
    expect(screen.getByTestId('generation-helper')).toHaveTextContent(
      '語音辨識＋AI 翻譯，約需數分鐘'
    );
    expect(screen.queryByTestId('helper-goto-settings')).toBeNull();
    expect(cta).toHaveAttribute('aria-disabled', 'true');
    expect(screen.getByTestId('action-generate-subtitle-amount-skeleton')).toBeInTheDocument();
    fireEvent.click(cta);
    expect(mockedTrigger).not.toHaveBeenCalled();
  });

  // The open-gating contract at the dialog seam — a closed dialog must not
  // fetch the estimate (it may probe the file on the server).
  it('a closed dialog does not ask for the estimate; an open one does, for THIS movie', async () => {
    const first = renderDialog({ open: false });
    await waitFor(() => expect(first.router.state.status).toBe('idle'));
    expect(mockedEstimate).not.toHaveBeenCalled();
    first.unmount();

    renderDialog({ open: true });
    await findPricedGenerate();
    expect(mockedEstimate).toHaveBeenCalledWith('movie', MOVIE_UUID, expect.anything());
  });

  it('helper link navigates to /settings/keys', async () => {
    mockedEstimate.mockResolvedValue(readyEstimate({ translationConfigured: false }));
    const { router } = renderDialog();

    fireEvent.click(await screen.findByTestId('helper-goto-settings'));
    await waitFor(() => expect(router.state.location.pathname).toBe('/settings/keys'));
  });

  it('series: 生成字幕 renders DISABLED and points at the episode list (9R-10c AC #5)', async () => {
    renderDialog({ mediaType: 'series' });

    const cta = await screen.findByTestId('action-generate-subtitle');
    expect(cta).toBeDisabled();
    // dsr-6a: a series has no generate route, so no price is asked for or shown.
    expect(mockedEstimate).not.toHaveBeenCalled();
    expect(cta.textContent).not.toMatch(/\$/);
    // 9R-10c AC #5: the old 影集字幕生成即將推出 became a lie once sub-4-2 /
    // sub-5-3 shipped episode batching — a series CAN be generated, just not
    // at series level. J3-D rules the copy points at the real entry instead.
    expect(screen.getByText('請於下方分集清單逐集生成')).toBeInTheDocument();
    expect(screen.queryByText('影集字幕生成即將推出')).not.toBeInTheDocument();

    fireEvent.click(cta);
    expect(mockedTrigger).not.toHaveBeenCalled();
  });

  it('movie: 生成字幕 POSTs the transcribe trigger with the int64 id and enters the progress view', async () => {
    mockedTrigger.mockResolvedValue({
      status: 'started',
      result: { jobId: 'job-9', message: 'ok' },
    });
    renderDialog();

    fireEvent.click(await findPricedGenerate());

    await waitFor(() => expect(mockedTrigger).toHaveBeenCalledWith(MOVIE_UUID));
    await waitFor(() => expect(h.startTracking).toHaveBeenCalledWith(MOVIE_UUID));
    expect(screen.getByText('生成字幕 — 怪奇物語')).toBeInTheDocument();
    expect(screen.getByTestId('generation-progress-v2')).toBeInTheDocument();
    expect(screen.getByText('即時更新（SSE）')).toBeInTheDocument();
  });

  it('503 TRANSCRIPTION_DISABLED → F5 尚未設定 warning panel + 前往設定 (never hard-fails)', async () => {
    mockedTrigger.mockResolvedValue({ status: 'disabled' });
    renderDialog();

    fireEvent.click(await findPricedGenerate());

    expect(await screen.findByTestId('generation-not-configured')).toBeInTheDocument();
    // sub-2-2d AC #1 — γ's ratified copy: the panel names ASR, not a vague
    // "generation" and never the translation key (the 503's real trigger).
    expect(screen.getByText('語音辨識尚未設定')).toBeInTheDocument();
    // sub-5-2 AC #5: the restart clause was retired when the ASR key started
    // resolving per call — saving is now sufficient.
    expect(
      screen.getByText('生成字幕需要雲端語音辨識（ASR）金鑰。請至金鑰設定儲存後即可使用。')
    ).toBeInTheDocument();
    expect(screen.getByTestId('generation-not-configured')).not.toHaveTextContent('重啟');
    expect(screen.getByTestId('go-to-settings')).toHaveTextContent('前往設定');
    // The rest of the dialog survives (fail-soft): tracks + glossary still visible.
    expect(screen.getByTestId('subtitle-tracks-section')).toBeInTheDocument();
    expect(screen.getByTestId('open-glossary')).toBeInTheDocument();
  });

  // sub-2-1b AC #4 — the dead END. /settings renders but carries no key surface,
  // so the button has to land on the page that actually holds the key.
  it('前往設定 navigates to /settings/keys, keeping the go-to-settings testid', async () => {
    mockedTrigger.mockResolvedValue({ status: 'disabled' });
    const { router } = renderDialog();

    fireEvent.click(await findPricedGenerate());
    const goToSettings = await screen.findByTestId('go-to-settings');
    fireEvent.click(goToSettings);

    await waitFor(() => expect(router.state.location.pathname).toBe('/settings/keys'));
  });

  it('409 TRANSCRIPTION_IN_PROGRESS → attaches to the running job SSE stream instead of erroring', async () => {
    mockedTrigger.mockResolvedValue({ status: 'inProgress' });
    renderDialog();

    fireEvent.click(await findPricedGenerate());

    await waitFor(() => expect(h.startTracking).toHaveBeenCalledWith(MOVIE_UUID));
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

    fireEvent.click(await findPricedGenerate());

    const panel = await screen.findByTestId('generation-trigger-error');
    expect(panel).toHaveTextContent('無法開始生成：找不到電影');

    // dsr-6a: 重試 spends money again — it carries the (re-fetched) price.
    await screen.findByTestId('generation-trigger-retry-amount');
    expect(screen.getByTestId('generation-trigger-retry-amount').textContent).toBe('$0.42');
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
      mediaId: MOVIE_UUID,
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

  // sub-2-2b AC #3: the completion note must not claim 完成 when translation
  // did not run — an en-only result (zhSrtPath null) says exactly what it is.
  it('complete WITH a zh path → 字幕已生成完成', async () => {
    mockedTrigger.mockResolvedValue({ status: 'started', result: { jobId: 'j', message: 'ok' } });
    h.genState.phase = 'complete';
    h.genState.zhSrtPath = '/media/m.zh-Hant.srt';
    renderDialog();

    fireEvent.click(await findPricedGenerate());

    const note = await screen.findByTestId('generation-complete-note');
    expect(note).toHaveTextContent('字幕已生成完成');
  });

  it('complete WITHOUT a zh path (en-only, key unconfigured) → 已生成英文字幕；尚未翻譯', async () => {
    mockedTrigger.mockResolvedValue({ status: 'started', result: { jobId: 'j', message: 'ok' } });
    h.genState.phase = 'complete';
    h.genState.zhSrtPath = null;
    renderDialog();

    fireEvent.click(await findPricedGenerate());

    const note = await screen.findByTestId('generation-complete-note');
    expect(note).toHaveTextContent('已生成英文字幕；尚未翻譯');
    expect(note).not.toHaveTextContent('字幕已生成完成');
  });

  // bugfix-j CR H2: a PARTIAL completion has a zh path (the mixed file was
  // deliberately placed) but must NOT claim 完成 — the row is `untranslated`.
  it('complete PARTIAL (zh path present but English cues kept) → 部分翻譯失敗, not 完成', async () => {
    mockedTrigger.mockResolvedValue({ status: 'started', result: { jobId: 'j', message: 'ok' } });
    h.genState.phase = 'complete';
    h.genState.zhSrtPath = '/media/m.zh-Hant.srt';
    h.genState.partial = true;
    h.genState.englishKeptBlocks = 5;
    renderDialog();

    fireEvent.click(await findPricedGenerate());

    const note = await screen.findByTestId('generation-complete-note');
    expect(note).toHaveTextContent('部分翻譯失敗');
    expect(note).toHaveTextContent('5 句保留英文');
    expect(note).not.toHaveTextContent('字幕已生成完成');
  });

  it('closing the dialog resets generation tracking (job continues server-side)', async () => {
    const onOpenChange = vi.fn();
    renderDialog({ onOpenChange });

    fireEvent.click(await screen.findByTestId('dialog-close'));

    expect(h.reset).toHaveBeenCalled();
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it('closing the dialog also closes the glossary panel (no resurrect on reopen)', async () => {
    renderDialog(); // onOpenChange stub keeps `open` true, so internal state stays observable

    fireEvent.click(await screen.findByTestId('open-glossary'));
    expect(screen.getByTestId('glossary-panel-stub')).toBeInTheDocument();

    fireEvent.click(screen.getByTestId('dialog-close'));
    expect(screen.queryByTestId('glossary-panel-stub')).not.toBeInTheDocument();
  });
});

// ── 9R-10c — episode mode ─────────────────────────────────────────────────

const SERIES_UUID = '11111111-2222-4333-8444-555555555555';
const EPISODE_UUID = '9c3b7e21-4d5a-4b8c-9f01-2e3d4c5b6a7f';

function renderEpisodeDialog(props: DialogProps = {}) {
  return renderDialog({
    mediaId: EPISODE_UUID,
    mediaType: 'episode',
    glossaryMediaId: SERIES_UUID,
    mediaTitle: '第七集',
    mediaCode: 'S04E07',
    subtitleTracks: undefined,
    ...props,
  });
}

describe('ManageSubtitleDialogV2 — episode mode (9R-10c)', () => {
  beforeEach(() => {
    h.glossaryQueriedId = undefined;
    mockedEpisodeTrigger.mockReset();
    mockedTrigger.mockReset();
  });

  // RED LINE 1. /media/:id/glossary takes the route id VERBATIM — it will not
  // resolve an episode id to its series. Sending the episode id would strand
  // terms in rows no other episode can ever see (the frontend twin of the bug
  // CR sub-5-5 H1 fixed in the pipeline). The two ids MUST differ here.
  it('queries the glossary with the SERIES id while triggering with the EPISODE id', async () => {
    mockedEpisodeTrigger.mockResolvedValue({
      status: 'started',
      result: { jobId: 'j1', message: '' },
    });
    renderEpisodeDialog();

    fireEvent.click(await findPricedGenerate());
    await waitFor(() => expect(mockedEpisodeTrigger).toHaveBeenCalledWith(EPISODE_UUID));

    expect(h.glossaryQueriedId).toBe(SERIES_UUID);
    // The whole point: these are two different ids, not one reused.
    expect(h.glossaryQueriedId).not.toBe(EPISODE_UUID);
    expect(mockedEpisodeTrigger).not.toHaveBeenCalledWith(SERIES_UUID);
    // …and the MOVIE route is never touched for an episode.
    expect(mockedTrigger).not.toHaveBeenCalled();
  });

  it('falls back to mediaId for the glossary when glossaryMediaId is omitted (movie/series unchanged)', async () => {
    renderDialog({ mediaType: 'movie' });
    await screen.findByTestId('manage-subtitle-dialog-v2');
    expect(h.glossaryQueriedId).toBe(MOVIE_UUID);
  });

  // RED LINE 2. The search endpoints bind `oneof=movie series`; an episode
  // would 400. Capability honor: don't draw a control that cannot work.
  it('hides the dormant online-search secondary', async () => {
    renderEpisodeDialog();
    await screen.findByTestId('manage-subtitle-dialog-v2');
    expect(screen.queryByTestId('toggle-fetch')).not.toBeInTheDocument();
  });

  it('keeps the online-search secondary for movies', async () => {
    renderDialog({ mediaType: 'movie' });
    expect(await screen.findByTestId('toggle-fetch')).toBeInTheDocument();
  });

  it('renders the SxxExx code chip so the dialog says WHICH episode it acts on', async () => {
    renderEpisodeDialog();
    expect(await screen.findByTestId('dialog-title-code')).toHaveTextContent('S04E07');
  });

  it('enables the CTA (an episode has a generate route, unlike a series)', async () => {
    mockedEstimate.mockResolvedValue(
      readyEstimate({ mediaId: EPISODE_UUID, mediaType: 'episode' })
    );
    renderEpisodeDialog();
    expect(await findPricedGenerate()).not.toBeDisabled();
    expect(mockedEstimate).toHaveBeenCalledWith('episode', EPISODE_UUID, expect.anything());
  });

  // Design string (F1 note-untranslated): a resume skips the expensive ASR leg,
  // so the user is told this run is cheap rather than being left to guess.
  it('shows the cheap-resume helper when the episode is untranslated (from props while the price loads)', async () => {
    mockedEstimate.mockReturnValue(new Promise(() => {}));
    renderEpisodeDialog({ subtitleStatus: 'untranslated' });
    expect(await screen.findByTestId('generation-helper')).toHaveTextContent(
      '僅需翻譯，不再重跑語音辨識'
    );
  });

  it('shows the default helper for other episode statuses', async () => {
    renderEpisodeDialog({ subtitleStatus: 'not_found' });
    await findPricedGenerate();
    expect(screen.getByTestId('generation-helper')).toHaveTextContent('語音辨識＋AI 翻譯');
  });
});

// ── dsr-6a — the price on the paid buttons (J9-D) ─────────────────────────

describe('ManageSubtitleDialogV2 — cost on paid buttons (dsr-6a)', () => {
  it('① shows the amount verbatim on 生成字幕, described by the helper line', async () => {
    renderDialog();
    const cta = await findPricedGenerate();
    expect(screen.getByTestId('action-generate-subtitle-amount').textContent).toBe('$0.42');
    expect(cta).toHaveAccessibleDescription('語音辨識＋AI 翻譯，約需數分鐘');
  });

  it('② assumed runtime: ≈ on the amount and the 片長未知 line', async () => {
    mockedEstimate.mockResolvedValue(
      readyEstimate({ runtimeSource: 'fallback', runtimeKnown: false, runtimeMinutes: 45 })
    );
    renderDialog();
    await findPricedGenerate();
    expect(screen.getByTestId('action-generate-subtitle-amount').textContent).toBe('≈ $0.42');
    expect(screen.getByTestId('generation-helper')).toHaveTextContent(
      '片長未知（估 45 分）——實際費用依內容長度而定'
    );
  });

  it('③ self-hosted ASR with no translation key: $0.00 explained, never 「免費」', async () => {
    mockedEstimate.mockResolvedValue(
      readyEstimate({
        selfHostedAsr: true,
        translationConfigured: false,
        modelId: '',
        estimatedUsd: 0,
      })
    );
    renderDialog();
    await findPricedGenerate();
    expect(screen.getByTestId('action-generate-subtitle-amount').textContent).toBe('$0.00');
    expect(screen.getByTestId('generation-helper')).toHaveTextContent(
      '語音辨識：自架（不另計費）。僅能產生英文字幕——尚未設定翻譯金鑰'
    );
    expect(screen.queryByText(/免費/)).toBeNull();
  });

  it('⑤ estimate failed: disabled, no amount, the reason in the secondary tone', async () => {
    mockedEstimate.mockRejectedValue(new ApiError('boom', 500, 'INTERNAL_ERROR'));
    renderDialog();
    const helper = await screen.findByText('暫時算不出費用，因此先不開放。重新整理或稍後再試。');
    const cta = screen.getByTestId('action-generate-subtitle');
    expect(cta).toBeDisabled();
    expect(cta.textContent).not.toMatch(/\$/);
    expect(helper).toHaveAttribute('data-tone', 'secondary');
    fireEvent.click(cta);
    expect(mockedTrigger).not.toHaveBeenCalled();
  });

  it('the video file cannot be read: disabled with its own reason', async () => {
    mockedEstimate.mockRejectedValue(
      new ApiError('Movie file not accessible', 400, 'VALIDATION_REQUIRED_FIELD')
    );
    renderDialog();
    expect(
      await screen.findByText('讀不到影片檔案，因此先不開放。請確認檔案還在，或重新掃描媒體庫。')
    ).toBeInTheDocument();
    expect(screen.getByTestId('action-generate-subtitle')).toBeDisabled();
  });

  it('⑥ no ASR key: disabled BEFORE the click, with 前往設定', async () => {
    mockedEstimate.mockResolvedValue(readyEstimate({ asrAvailable: false }));
    const { router } = renderDialog();
    expect(
      await screen.findByText('生成字幕需要雲端語音辨識（ASR）金鑰。請至金鑰設定儲存後即可使用。', {
        exact: false,
      })
    ).toBeInTheDocument();
    const cta = screen.getByTestId('action-generate-subtitle');
    expect(cta).toBeDisabled();
    fireEvent.click(cta);
    expect(mockedTrigger).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId('helper-goto-settings'));
    await waitFor(() => expect(router.state.location.pathname).toBe('/settings/keys'));
  });

  it('translate-only without a translation key: disabled — the click would produce nothing', async () => {
    mockedEstimate.mockResolvedValue(
      readyEstimate({ plan: 'translate_only', translationConfigured: false, estimatedUsd: 0 })
    );
    renderDialog({ subtitleStatus: 'untranslated' });
    expect(await screen.findByText('尚未設定翻譯金鑰', { exact: false })).toBeInTheDocument();
    expect(screen.getByTestId('action-generate-subtitle')).toBeDisabled();
  });

  it('a MOVIE that can resume translate-only gets the cheap line too', async () => {
    mockedEstimate.mockResolvedValue(readyEstimate({ plan: 'translate_only', estimatedUsd: 0.24 }));
    renderDialog({ subtitleStatus: 'untranslated' });
    await findPricedGenerate();
    expect(screen.getByTestId('action-generate-subtitle-amount').textContent).toBe('$0.24');
    expect(screen.getByTestId('generation-helper')).toHaveTextContent(
      '僅需翻譯，不再重跑語音辨識——這次很快也很便宜'
    );
  });

  it('the failed-run 重試 carries the price, re-fetched when the run fails', async () => {
    mockedTrigger.mockResolvedValue({ status: 'started', result: { jobId: 'j', message: 'ok' } });
    renderDialog();
    const cta = await findPricedGenerate();
    expect(mockedEstimate).toHaveBeenCalledTimes(1);

    // The SSE stream reports the run failed by the time the view re-renders —
    // and the run left its English SRT behind, so the retry is translate-only.
    mockedEstimate.mockResolvedValue(readyEstimate({ plan: 'translate_only', estimatedUsd: 0.24 }));
    h.genState.phase = 'failed';
    h.genState.failedPhase = 'translating';
    fireEvent.click(cta);
    expect(await screen.findByTestId('gen-failed-panel')).toBeInTheDocument();
    // Entering `failed` re-prices, and 重試 shows the NEW price.
    await waitFor(() => expect(screen.getByTestId('gen-retry-amount').textContent).toBe('$0.24'));
  });

  it('re-prices when the subtitle status changes under an open dialog (e.g. a downloaded subtitle ends a resume)', async () => {
    mockedEstimate.mockResolvedValue(readyEstimate({ plan: 'translate_only', estimatedUsd: 0.24 }));
    renderDialog({ subtitleStatus: 'untranslated' }, { statusSwitch: true });
    await findPricedGenerate();
    expect(screen.getByTestId('action-generate-subtitle-amount').textContent).toBe('$0.24');

    // The parent refetched after a download: the row is `found` now, so the next
    // click is a FULL run — the cheap translate-only quote must not survive.
    mockedEstimate.mockResolvedValue(readyEstimate({ estimatedUsd: 0.42 }));
    fireEvent.click(screen.getByTestId('mark-found'));
    await waitFor(() =>
      expect(screen.getByTestId('action-generate-subtitle-amount').textContent).toBe('$0.42')
    );
  });

  it('a successful online download re-prices at once', async () => {
    h.fetchHook.results = [
      {
        id: 's1',
        source: 'assrt',
        filename: 'a.zh.srt',
        language: 'zh-TW',
        downloadUrl: '',
        downloads: 1,
        group: '',
        resolution: '1080p',
        format: 'srt',
        score: 0.5,
        scoreBreakdown: { language: 1, resolution: 1, sourceTrust: 1, group: 1, downloads: 1 },
      },
    ];
    h.fetchHook.download.mockImplementation((_params: unknown, opts?: { onSuccess?: () => void }) =>
      opts?.onSuccess?.()
    );
    renderDialog();
    await findPricedGenerate();
    expect(mockedEstimate).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByTestId('toggle-fetch'));
    fireEvent.click(screen.getByTestId('fetch-download-s1'));
    await waitFor(() => expect(mockedEstimate).toHaveBeenCalledTimes(2));
  });

  it('a blocked retry says why under the button', async () => {
    mockedTrigger.mockRejectedValueOnce(new Error('找不到電影'));
    renderDialog();
    const cta = await findPricedGenerate();

    // The trigger error re-prices; by then the ASR key has been removed.
    mockedEstimate.mockResolvedValue(readyEstimate({ asrAvailable: false }));
    fireEvent.click(cta);
    await screen.findByTestId('generation-trigger-error');
    const note = await screen.findByTestId('generation-trigger-retry-note');
    expect(note).toHaveTextContent('生成字幕需要雲端語音辨識（ASR）金鑰');
    expect(screen.getByTestId('generation-trigger-retry')).toBeDisabled();
    expect(screen.getByTestId('generation-trigger-retry')).toHaveAccessibleDescription(
      /生成字幕需要雲端語音辨識/
    );
  });

  it('closing the dialog drops the cached price, so the next open re-estimates', async () => {
    renderDialog({}, { controlled: true });
    await findPricedGenerate();
    expect(mockedEstimate).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByTestId('dialog-close'));
    await waitFor(() => expect(screen.queryByTestId('manage-subtitle-dialog-v2')).toBeNull());

    // A reopen within the global 5-minute staleTime would otherwise reuse the
    // old number — the price must be the one current at the moment of the click.
    mockedEstimate.mockResolvedValue(readyEstimate({ estimatedUsd: 0.5 }));
    fireEvent.click(screen.getByTestId('reopen'));
    await waitFor(() => expect(mockedEstimate).toHaveBeenCalledTimes(2));
    expect((await screen.findByTestId('action-generate-subtitle-amount')).textContent).toBe(
      '$0.50'
    );
  });

  it('while the trigger is in flight the button keeps its price and cannot be pressed twice', async () => {
    mockedTrigger.mockReturnValue(new Promise(() => {}));
    renderDialog();
    const cta = await findPricedGenerate();
    fireEvent.click(cta);
    await waitFor(() => expect(cta).toHaveAttribute('aria-busy', 'true'));
    expect(cta).toBeDisabled();
    expect(screen.getByTestId('action-generate-subtitle-amount').textContent).toBe('$0.42');
    expect(cta.querySelector('.animate-spin')).toBeNull();
  });
});

// Design ref: ux-design.pen Screen F1-D-v2 (r1EY9)
/**
 * 管理字幕 dialog v2 (ux3-subtitle-v2 AC 1/2/5 — generation-centric per ADR
 * adr-subtitle-route-c-generation D1). Screens: F1-D-v2 r1EY9 / F1-M-v2 JkdfH
 * (mobile sheet = the same Radix Dialog with bottom-sheet positioning at <sm —
 * AC 7 mandates Radix for modals; Radix gives focus-trap/Escape/scrim), F2-D-v2
 * S9Rbrq 缺字幕, F3-D-v2 JbXai 生成進度, F4-D-v2 U8rRtv 生成失敗 (per-item state),
 * F5-D-v2 f6ZxY 尚未設定 fail-soft, F10-D-v2 olDlj 載入骨架.
 *
 * - 生成字幕 is the ONLY primary action; movies call
 *   POST /movies/{id}/transcribe?translate=true (UUID-string id passed through
 *   as-is, 9R-18 — the old `Number(uuid)` produced NaN);
 *   EPISODES call POST /episodes/{id}/transcribe (9R-10c, consuming 9R-10a
 *   [@contract-v1]); series keep the CTA DISABLED and point at the episode list
 *   (there is no series-level generate — J3-D ruling: generation is per episode).
 * - EPISODE mode takes a SEPARATE glossaryMediaId (the SERIES id). The glossary
 *   is per-show and /media/:id/glossary uses the route id verbatim, so passing
 *   the episode id would strand terms in rows no other episode can see.
 * - 503 TRANSCRIPTION_DISABLED → 語音辨識尚未設定 warning panel + 前往設定
 *   (γ-ratified ASR copy, sub-2-2d — the 503 gate is FFmpeg+ASR, never the
 *   translation key; dialog never hard-fails); 409 → attach to the running
 *   job's SSE stream; 404/400/500 → fail-soft error + 重試.
 * - Translation key missing is a DEGRADATION, not a block (party-mode ruling
 *   2026-08-05): the helper line states the en-only truth + 前往設定 while the
 *   CTA stays enabled; the row lands `untranslated` and a re-run resumes
 *   translate-only (sub-2-2a).
 * - Fetch is demoted to a dormant secondary 搜尋線上字幕（成功率低） — NO source
 *   chips, NO score-breakdown rows, NO Zimuku (9R-14 removed it).
 * - CN policy (§9b, note v16pVI): a 簡中 track on CN content shows the policy
 *   line 陸劇保留簡體字幕（對白一致） — policy-correct, NOT a defect. The design's
 *   轉為繁中/仍要轉換 actions are NOT rendered: no backend endpoint converts an
 *   existing local track today (capability honor — see Discovery Triage).
 * - No cancel control for a running job: the backend exposes no cancel route;
 *   closing the dialog only stops watching (job continues server-side).
 */
import { useCallback, useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import {
  BookOpen,
  CaptionsOff,
  ChevronRight,
  CircleAlert,
  Download,
  Info,
  Loader2,
  Radio,
  Settings,
} from 'lucide-react';
import { Dialog, DialogContent, DialogTitle } from '../ui/Dialog';
import { cn } from '../../lib/utils';
// Canonical zh-script sets (deriveSubtitleStatus semantics, AC 1a) — single source.
import { HANT, HANS } from '../../utils/libraryStatus';
import { transcriptionService } from '../../services/transcriptionService';
import type { SubtitleSearchResult } from '../../services/subtitleService';
import { useGenerationProgress } from '../../hooks/useGenerationProgress';
import { useGlossaryTerms } from '../../hooks/useGlossary';
import { useKeySettings } from '../../hooks/useKeySettings';
import { useSubtitleSearch } from '../../hooks/useSubtitleSearch';
import { GenerationProgressV2 } from './GenerationProgressV2';
import { GlossaryPanelV2 } from './GlossaryPanelV2';

interface TrackRow {
  key: string;
  label: string;
  pillClass: string;
  source: string;
  isHans: boolean;
}

function languageDescriptor(lang: string): { label: string; pillClass: string; isHans: boolean } {
  const l = lang.toLowerCase();
  if (HANT.has(l))
    return {
      label: '繁中',
      pillClass: 'bg-[var(--success-tint)] text-[var(--success-text)]',
      isHans: false,
    };
  if (HANS.has(l))
    return {
      label: '簡中',
      pillClass: 'bg-[var(--info-tint)] text-[var(--info-text)]',
      isHans: true,
    };
  if (l === 'en' || l.startsWith('en-'))
    return {
      label: '英文',
      pillClass: 'bg-[var(--bg-primary)] text-[var(--text-secondary)]',
      isHans: false,
    };
  return {
    label: lang || '未知',
    pillClass: 'bg-[var(--bg-primary)] text-[var(--text-secondary)]',
    isHans: false,
  };
}

/** Rows from the embedded-tracks JSON + the authoritative engine result (ux3-0-2 semantics). */
function buildTrackRows(
  subtitleTracks?: string,
  subtitleStatus?: string,
  subtitleLanguage?: string
): TrackRow[] {
  const rows: TrackRow[] = [];

  if (subtitleStatus === 'found' && subtitleLanguage) {
    const d = languageDescriptor(subtitleLanguage);
    rows.push({ key: 'engine', source: '字幕引擎', ...d });
  }

  if (subtitleTracks) {
    try {
      const parsed = JSON.parse(subtitleTracks);
      if (Array.isArray(parsed)) {
        parsed.forEach((t: { language?: string; lang?: string }, i: number) => {
          const lang = t.language ?? t.lang ?? '';
          const d = languageDescriptor(lang);
          rows.push({ key: `track-${i}`, source: '本地檔案', ...d });
        });
      }
    } catch {
      // non-JSON legacy value → cannot classify, show nothing for it
    }
  }

  return rows;
}

type GenView = 'idle' | 'progress' | 'notConfigured' | 'triggerError';

export interface ManageSubtitleDialogV2Props {
  /** STRING local media id (glossary + fetch contract; transcribe converts to int64). */
  mediaId: string;
  mediaType: 'movie' | 'series' | 'episode';
  /** Glossary owner id. The glossary is per-SHOW, so an episode passes its
   *  SERIES id here while `mediaId` stays the episode row id (9R-10c red line 1).
   *  Omitted → falls back to mediaId (movie/series unchanged, zero regression). */
  glossaryMediaId?: string;
  mediaTitle: string;
  /** Optional code chip beside the title, e.g. `S04E07` (design node tO72N).
   *  Episode mode passes it so the dialog says WHICH episode it is acting on. */
  mediaCode?: string;
  mediaFilePath: string;
  mediaResolution?: string;
  /** LibraryMovie.subtitleTracks JSON string (embedded tracks). */
  subtitleTracks?: string;
  subtitleStatus?: string;
  subtitleLanguage?: string;
  /** ISO 3166-1 codes; contains "CN" → §9b policy display (no local-detail source today). */
  productionCountry?: string;
  /** True while the parent detail query loads — renders the F10 skeleton. */
  isLoading?: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Fired on transcription_complete (AC 6 — parent invalidates detail + library caches). */
  onGenerationComplete?: () => void;
  /** Fired when a dormant-fetch download succeeds (parity with the v1 dialog). */
  onDownloadSuccess?: () => void;
}

export function ManageSubtitleDialogV2({
  mediaId,
  mediaType,
  glossaryMediaId,
  mediaTitle,
  mediaCode,
  mediaFilePath,
  mediaResolution,
  subtitleTracks,
  subtitleStatus,
  subtitleLanguage,
  productionCountry,
  isLoading = false,
  open,
  onOpenChange,
  onGenerationComplete,
  onDownloadSuccess,
}: ManageSubtitleDialogV2Props) {
  const navigate = useNavigate();
  const isMovie = mediaType === 'movie';
  const isEpisode = mediaType === 'episode';
  // Positive intent, not a pile of negations (sub-2-2b CR L1 fixed the same
  // smell): a SERIES is the only mediaType with no generate route of its own.
  const canGenerate = isMovie || isEpisode;
  const isCNContent = productionCountry?.includes('CN') ?? false;

  const [genView, setGenView] = useState<GenView>('idle');
  const [triggerError, setTriggerError] = useState<string | null>(null);
  const [glossaryOpen, setGlossaryOpen] = useState(false);
  const [fetchOpen, setFetchOpen] = useState(false);

  const generation = useGenerationProgress({
    onComplete: () => onGenerationComplete?.(),
  });

  const glossary = useGlossaryTerms(glossaryMediaId ?? mediaId, open);
  const glossaryCount = glossary.data?.length ?? 0;

  // sub-2-2d AC #2 — the degraded-CTA pre-flight (β Task 4's deferral, γ's
  // ratified copy). Open-gated so a closed dialog costs no request. The signal
  // is GET /settings/keys' claude.configured (2-1a CR L1: never branch on an
  // error code). Loading/error keep the DEFAULT line — fail-soft, never flash
  // the degraded warning on an unresolved query.
  const keySettings = useKeySettings({ enabled: open });
  const translationDegraded =
    keySettings.data?.keys.find((k) => k.name === 'claude')?.configured === false;

  // Dormant fetch section (reuses the Epic 8 hook; results WITHOUT chips/scores).
  // Named onlineSearch — `fetch` would shadow window.fetch inside this component.
  const onlineSearch = useSubtitleSearch();

  const trigger = useMutation({
    mutationFn: () =>
      isEpisode
        ? transcriptionService.startEpisodeTranscription(mediaId)
        : transcriptionService.startTranscription(mediaId),
    onSuccess: (outcome) => {
      if (outcome.status === 'disabled') {
        setGenView('notConfigured');
        return;
      }
      // started AND inProgress (409) both attach to the job's SSE stream.
      setGenView('progress');
      generation.startTracking(mediaId);
    },
    onError: (error) => {
      setTriggerError(error instanceof Error ? error.message : '生成字幕失敗');
      setGenView('triggerError');
    },
  });

  const startGeneration = useCallback(() => {
    setTriggerError(null);
    trigger.mutate();
  }, [trigger]);

  const handleOpenChange = useCallback(
    (next: boolean) => {
      if (!next) {
        // Closing only stops WATCHING — a running job continues server-side.
        generation.reset();
        setGenView('idle');
        setTriggerError(null);
        setFetchOpen(false);
        setGlossaryOpen(false); // don't resurrect the glossary panel on reopen
      }
      onOpenChange(next);
    },
    [generation, onOpenChange]
  );

  const tracks = buildTrackRows(subtitleTracks, subtitleStatus, subtitleLanguage);
  const inProgressView = genView === 'progress';
  const dialogTitle = inProgressView ? `生成字幕 — ${mediaTitle}` : `管理字幕 — ${mediaTitle}`;

  const handleFetchSearch = useCallback(() => {
    // CR H1 — red line 2 enforced by the TYPE SYSTEM, not just by hiding the
    // control: the search endpoints bind `oneof=movie series`, so an episode
    // would 400. This early return also narrows `mediaType` for the call below.
    if (mediaType === 'episode') return;
    onlineSearch.search({ mediaId, mediaType, query: mediaTitle });
  }, [onlineSearch, mediaId, mediaType, mediaTitle]);

  const handleFetchDownload = useCallback(
    (result: SubtitleSearchResult) => {
      // CR H1 — same narrowing as handleFetchSearch: unreachable for an
      // episode (the control is hidden), and now un-typeable too.
      if (mediaType === 'episode') return;
      onlineSearch.download(
        {
          mediaId,
          mediaType,
          mediaFilePath,
          subtitleId: result.id,
          provider: result.source,
          resolution: mediaResolution,
          convertToTraditional: !isCNContent, // §9b default: CN content keeps simplified
          score: result.score,
        },
        { onSuccess: () => onDownloadSuccess?.() }
      );
    },
    [
      onlineSearch,
      mediaId,
      mediaType,
      mediaFilePath,
      mediaResolution,
      isCNContent,
      onDownloadSuccess,
    ]
  );

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent
        data-testid="manage-subtitle-dialog-v2"
        aria-describedby={undefined}
        className={cn(
          'flex max-h-[85vh] flex-col gap-0 overflow-hidden p-0',
          // Mobile: bottom sheet (F1-M-v2 JkdfH). Desktop: centered dialog (F1-D-v2 r1EY9).
          'bottom-0 left-0 right-0 top-auto w-full max-w-none translate-x-0 translate-y-0 rounded-b-none rounded-t-[var(--radius-xl)]',
          'sm:bottom-auto sm:left-1/2 sm:right-auto sm:top-1/2 sm:w-[calc(100vw-4rem)] sm:max-w-3xl sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-[var(--radius-lg)]'
        )}
      >
        {/* Header */}
        <div className="flex h-14 shrink-0 items-center justify-between border-b border-[var(--border-subtle)] pl-6 pr-12">
          <div className="flex min-w-0 items-center gap-2">
            <DialogTitle className="truncate text-base font-semibold">{dialogTitle}</DialogTitle>
            {mediaCode && (
              <span
                data-testid="dialog-title-code"
                className="shrink-0 rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)] px-2 py-0.5 font-mono text-xs text-[var(--text-secondary)]"
              >
                {mediaCode}
              </span>
            )}
          </div>
        </div>

        <div className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto p-6">
          {isLoading ? (
            /* F10-D-v2 (olDlj) 載入骨架 — animation respects prefers-reduced-motion. */
            <div
              data-testid="manage-subtitle-skeleton"
              aria-hidden="true"
              className="flex flex-col gap-3"
            >
              {[0, 1, 2].map((i) => (
                <div
                  key={i}
                  className="h-11 animate-pulse rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] motion-reduce:animate-none"
                />
              ))}
              <div className="mt-3 h-11 w-32 animate-pulse rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] motion-reduce:animate-none" />
            </div>
          ) : inProgressView ? (
            /* F3-D-v2 (JbXai) 生成進度 / F4-D-v2 (U8rRtv) 生成失敗 — per-item states. */
            <>
              <GenerationProgressV2
                phase={generation.progress.phase}
                failedPhase={generation.progress.failedPhase}
                percentage={generation.progress.percentage}
                message={generation.progress.message}
                error={generation.progress.error}
                onRetry={startGeneration}
              />
              <div className="flex justify-center">
                <span className="flex items-center gap-1.5 rounded-[var(--radius-sm)] bg-[var(--info-tint)] px-2 py-1 text-[11px] text-[var(--info-text)]">
                  <Radio className="h-3 w-3" aria-hidden="true" />
                  即時更新（SSE）
                </span>
              </div>
              {generation.progress.phase === 'complete' && (
                <p
                  data-testid="generation-complete-note"
                  className="text-center text-[13px] text-[var(--text-secondary)]"
                >
                  {/* sub-2-2b AC #3: an en-only completion (no zh path — key
                      unconfigured or translate failed non-fatally) must not
                      claim 完成. The row is `untranslated`; setting the key and
                      re-running resumes translate-only (sub-2-2a).
                      bugfix-j CR H2: a PARTIAL completion places a mixed zh
                      file, so zhSrtPath alone must not claim 完成 either.
                      Disclosure is UNCONDITIONAL (any kept-English cue sets
                      partial); the row's verdict is thresholded (H3 ruling,
                      option B) — a material residue records `untranslated`,
                      an immaterial one stays `found`. Either way re-running
                      resumes translate-only from the EN source. */}
                  {generation.progress.partial
                    ? `字幕已生成（部分翻譯失敗，${generation.progress.englishKeptBlocks ?? '若干'} 句保留英文；重新生成可補譯）`
                    : generation.progress.zhSrtPath
                      ? '字幕已生成完成'
                      : '已生成英文字幕；尚未翻譯'}
                </p>
              )}
            </>
          ) : (
            <>
              {/* 現有字幕 (F1) or 缺字幕 empty state (F2-D-v2 S9Rbrq) */}
              {tracks.length > 0 ? (
                <section data-testid="subtitle-tracks-section" className="flex flex-col gap-2.5">
                  <h3 className="text-[13px] font-semibold text-[var(--text-secondary)]">
                    現有字幕
                  </h3>
                  {tracks.map((track) => (
                    <div key={track.key} className="flex flex-col gap-1">
                      <div
                        data-testid={`subtitle-track-${track.key}`}
                        className="flex items-center gap-3 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-3.5 py-3"
                      >
                        <span
                          className={cn(
                            'shrink-0 rounded-full px-2.5 py-0.5 text-[11px] font-medium',
                            track.pillClass
                          )}
                        >
                          {track.label}
                        </span>
                        <span className="text-xs text-[var(--text-secondary)]">{track.source}</span>
                      </div>
                      {/* §9b CN policy: 簡中 on CN content is policy-correct, NOT a defect. */}
                      {track.isHans && isCNContent && (
                        <div
                          data-testid={`cn-policy-note-${track.key}`}
                          className="flex items-center gap-2 px-3.5 text-xs text-[var(--text-muted)]"
                        >
                          <Info className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                          陸劇保留簡體字幕（對白一致）
                        </div>
                      )}
                    </div>
                  ))}
                </section>
              ) : (
                <section
                  data-testid="subtitle-empty-state"
                  className="flex flex-col items-center gap-2 pb-2 pt-7"
                >
                  <CaptionsOff className="h-9 w-9 text-[var(--text-muted)]" aria-hidden="true" />
                  <p className="text-[15px] font-semibold text-[var(--text-primary)]">尚無字幕</p>
                  <p className="text-xs text-[var(--text-muted)]">此影片目前沒有任何字幕軌</p>
                </section>
              )}

              {/* Generate section — the ONLY primary action (or its 尚未設定/error stand-ins). */}
              {genView === 'notConfigured' ? (
                /* F5-D-v2 (f6ZxY) fail-soft: dialog never hard-fails. */
                <div
                  data-testid="generation-not-configured"
                  className="flex items-center gap-3.5 rounded-[var(--radius-md)] bg-[var(--warning-tint)] p-4"
                >
                  <Settings
                    className="h-5 w-5 shrink-0 text-[var(--warning-text)]"
                    aria-hidden="true"
                  />
                  <div className="flex min-w-0 flex-1 flex-col gap-1">
                    {/* sub-2-2d AC #1 — γ's ratified ASR copy: the 503 gate is
                        FFmpeg+ASR (transcription_service.go), so the panel names
                        語音辨識. sub-5-2 AC #5 retired the restart clause — the
                        ASR client is no longer boot-built (ASRProviderHolder
                        resolves the key per call), so saving is now sufficient
                        and 「重啟伺服器」 would send the user to reboot their NAS
                        for nothing. The replacement phrasing reuses the already
                        ratified Claude-row wording in ApiKeysForm. */}
                    <p className="text-sm font-semibold text-[var(--text-primary)]">
                      語音辨識尚未設定
                    </p>
                    <p className="text-xs text-[var(--text-secondary)]">
                      生成字幕需要雲端語音辨識（ASR）金鑰。請至金鑰設定儲存後即可使用。
                    </p>
                  </div>
                  <button
                    type="button"
                    // sub-2-1b AC #4: /settings renders but has no key surface —
                    // a dead END, not a dead loop. /settings/keys is where the
                    // AI key this panel is asking for actually lives. The
                    // data-testid is kept: TestSprite/e2e selectors depend on it.
                    onClick={() => navigate({ to: '/settings/keys' })}
                    data-testid="go-to-settings"
                    className="flex min-h-[44px] shrink-0 items-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-4 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)]"
                  >
                    前往設定
                  </button>
                </div>
              ) : genView === 'triggerError' ? (
                <div
                  data-testid="generation-trigger-error"
                  className="flex items-center gap-2 rounded-[var(--radius-md)] bg-[var(--error-tint)] p-3"
                >
                  <CircleAlert
                    className="h-4 w-4 shrink-0 text-[var(--error-text)]"
                    aria-hidden="true"
                  />
                  <p className="flex-1 text-[13px] text-[var(--error-text)]">
                    無法開始生成{triggerError ? `：${triggerError}` : ''}
                  </p>
                  <button
                    type="button"
                    onClick={startGeneration}
                    data-testid="generation-trigger-retry"
                    className="flex min-h-[44px] shrink-0 items-center px-3 text-[13px] font-semibold text-[var(--accent-text)]"
                  >
                    重試
                  </button>
                </div>
              ) : (
                <section
                  data-testid="generation-section"
                  className={cn(
                    'flex items-center gap-4',
                    tracks.length === 0 && 'flex-col justify-center gap-2.5'
                  )}
                >
                  <button
                    type="button"
                    onClick={startGeneration}
                    disabled={!canGenerate || trigger.isPending}
                    data-testid="action-generate-subtitle"
                    className="flex min-h-[44px] items-center gap-2 rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-6 text-sm font-medium text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-pressed)] disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    {trigger.isPending && (
                      <Loader2
                        className="h-4 w-4 animate-spin motion-reduce:animate-none"
                        aria-hidden="true"
                      />
                    )}
                    生成字幕
                  </button>
                  <p data-testid="generation-helper" className="text-xs text-[var(--text-muted)]">
                    {!canGenerate ? (
                      /* J3-D ruling: there is no series-level generate — a
                         series is a container. The old 影集字幕生成即將推出
                         became a lie once sub-4-2/sub-5-3 shipped episode
                         batching, so the copy points at the real entry. */
                      '請於下方分集清單逐集生成'
                    ) : isEpisode && subtitleStatus === 'untranslated' ? (
                      /* Design string (F1 note-untranslated): this run resumes
                         translate-only, so it skips the expensive ASR leg. */
                      '僅需翻譯，不再重跑語音辨識——這次很快也很便宜'
                    ) : translationDegraded ? (
                      /* Degraded ≠ blocked: the truth up front, the CTA stays
                         live — an English subtitle beats no subtitle, and the
                         row will land `untranslated` for a translate-only
                         resume once the key exists (sub-2-2a). */
                      <>
                        僅能產生英文字幕——尚未設定翻譯金鑰{' '}
                        <button
                          type="button"
                          onClick={() => navigate({ to: '/settings/keys' })}
                          data-testid="helper-goto-settings"
                          className="underline transition-colors hover:text-[var(--text-primary)]"
                        >
                          前往設定
                        </button>
                      </>
                    ) : (
                      /* Verb ruled 2026-08-06 (party mode): this button calls the
                         transcribe (ASR) endpoint — 語音辨識, matching F5's
                         vocabulary; 抽取 belongs to the D2 pipeline's screens. */
                      '語音辨識＋AI 翻譯，約需數分鐘'
                    )}
                  </p>
                </section>
              )}

              {/* Glossary entry (F3 → opens F6/F7 panel) */}
              <button
                type="button"
                onClick={() => setGlossaryOpen(true)}
                data-testid="open-glossary"
                className="flex min-h-[44px] w-full items-center gap-2 rounded-[var(--radius-md)] px-1 text-left transition-colors hover:bg-[var(--bg-tertiary)]"
              >
                <BookOpen
                  className="h-4 w-4 shrink-0 text-[var(--text-secondary)]"
                  aria-hidden="true"
                />
                <span className="text-[13px] text-[var(--text-secondary)]">名詞對照表</span>
                <span className="text-[13px] text-[var(--text-secondary)]">
                  （<span className="font-mono tabular-nums">{glossaryCount}</span> 條）
                </span>
                <span className="flex-1" />
                <ChevronRight
                  className="h-4 w-4 shrink-0 text-[var(--text-muted)]"
                  aria-hidden="true"
                />
              </button>

              {/* Dormant fetch section — NO source chips, NO score rows, NO Zimuku (9R-14). */}
              {fetchOpen && (
                <section data-testid="fetch-section" className="flex flex-col gap-2.5">
                  <div className="flex items-center gap-3">
                    <h3 className="text-[13px] font-semibold text-[var(--text-secondary)]">
                      線上字幕搜尋
                    </h3>
                    <button
                      type="button"
                      onClick={handleFetchSearch}
                      disabled={onlineSearch.isSearching}
                      data-testid="fetch-search"
                      className="flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-4 text-sm text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)] disabled:opacity-50"
                    >
                      {onlineSearch.isSearching ? '搜尋中…' : '搜尋'}
                    </button>
                  </div>
                  {onlineSearch.searchError && (
                    <p className="text-[13px] text-[var(--error-text)]">
                      搜尋失敗：{onlineSearch.searchError.message}
                    </p>
                  )}
                  {!onlineSearch.isSearching &&
                    onlineSearch.results.length === 0 &&
                    !onlineSearch.searchError && (
                      <p className="text-xs text-[var(--text-muted)]">
                        尚無結果 — 線上來源成功率低，建議改用生成字幕
                      </p>
                    )}
                  {onlineSearch.results.map((result) => (
                    <div
                      key={result.id}
                      data-testid={`fetch-result-${result.id}`}
                      className="flex items-center gap-3 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-3.5 py-2"
                    >
                      <span className="min-w-0 flex-1 truncate font-mono text-xs text-[var(--text-primary)]">
                        {result.filename}
                      </span>
                      <span className="shrink-0 text-[11px] text-[var(--text-secondary)]">
                        {result.language}
                      </span>
                      {onlineSearch.downloadErrorMap[result.id] && (
                        <span className="shrink-0 text-[11px] text-[var(--error-text)]">
                          下載失敗
                        </span>
                      )}
                      <button
                        type="button"
                        onClick={() => handleFetchDownload(result)}
                        disabled={
                          onlineSearch.downloadingIds.has(result.id) ||
                          onlineSearch.downloadedIds.has(result.id)
                        }
                        aria-label={`下載 ${result.filename}`}
                        data-testid={`fetch-download-${result.id}`}
                        className="flex min-h-[44px] w-10 shrink-0 items-center justify-center text-[var(--accent-text)] disabled:opacity-50"
                      >
                        {onlineSearch.downloadingIds.has(result.id) ? (
                          <Loader2
                            className="h-4 w-4 animate-spin motion-reduce:animate-none"
                            aria-hidden="true"
                          />
                        ) : (
                          <Download className="h-4 w-4" aria-hidden="true" />
                        )}
                      </button>
                    </div>
                  ))}
                </section>
              )}
            </>
          )}
        </div>

        {/* Footer */}
        <div className="flex shrink-0 items-center justify-between border-t border-[var(--border-subtle)] px-6 py-3">
          {/* Red line 2: the search endpoints bind `oneof=movie series`, so an
              episode would 400. Capability honor — don't draw a dead control. */}
          {!inProgressView && !isLoading && !isEpisode ? (
            <button
              type="button"
              onClick={() => setFetchOpen((v) => !v)}
              data-testid="toggle-fetch"
              className="flex min-h-[44px] items-center px-1 text-xs text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)]"
            >
              搜尋線上字幕（成功率低）
            </button>
          ) : (
            <span className="text-xs text-[var(--text-muted)]">
              {inProgressView &&
              generation.progress.phase !== 'complete' &&
              generation.progress.phase !== 'failed'
                ? '關閉後生成會在背景繼續'
                : ''}
            </span>
          )}
          <button
            type="button"
            onClick={() => handleOpenChange(false)}
            data-testid="dialog-close"
            className="flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)]"
          >
            關閉
          </button>
        </div>

        <GlossaryPanelV2
          mediaId={mediaId}
          mediaTitle={mediaTitle}
          open={glossaryOpen}
          onOpenChange={setGlossaryOpen}
        />
      </DialogContent>
    </Dialog>
  );
}

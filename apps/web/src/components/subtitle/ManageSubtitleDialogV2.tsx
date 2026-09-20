// Design ref: ux-design.pen Screen F1-D-v2 (r1EY9) + Screen F2-D-v2 (S9Rbrq) + Screen F1-M-v2 (JkdfH) + Screen F3-D-v2 (JbXai) + Screen F3-M-v2 (k8sJl4) + Screen F4-D-v2 (U8rRtv) + Screen F5-D-v2 (f6ZxY)
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
 *   is per-show. Since sub-7-1 the backend resolves an episode id to its
 *   series' scope, but the frontend caches by the id it was handed — so the
 *   entry's count AND the panel must both use the series id, or the count goes
 *   stale after an edit (dsr-6b).
 * - 503 TRANSCRIPTION_DISABLED → 語音辨識尚未設定 warning panel + 前往設定
 *   (γ-ratified ASR copy, sub-2-2d — the 503 gate is FFmpeg+ASR, never the
 *   translation key; dialog never hard-fails); 409 → attach to the running
 *   job's SSE stream; 404/400/500 → fail-soft error + 重試.
 * - Translation key missing is a DEGRADATION, not a block (party-mode ruling
 *   2026-08-05): the helper line states the en-only truth + 前往設定 while the
 *   CTA stays enabled; the row lands `untranslated` and a re-run resumes
 *   translate-only (sub-2-2a).
 * - Every paid button — 生成字幕 and both 重試 — carries its estimated amount
 *   (story dsr-6a, DESIGN.md「會花錢的動作要有記號」, J9-D). The price comes from
 *   GET …/transcribe/estimate, which prices what THIS trigger does (speech
 *   recognition + translation, or translation only on resume). No price → no
 *   clickable button: loading shows a skeleton, a failed estimate or a missing
 *   ASR key disables it with the reason. The states and lines live in ONE pure
 *   function (generateCostView) shared by all three buttons.
 * - Fetch is demoted to a dormant secondary 搜尋線上字幕（成功率低） — NO source
 *   chips, NO score-breakdown rows, NO Zimuku (9R-14 removed it).
 * - CN policy (§9b, note v16pVI): a 簡中 track on CN content shows the policy
 *   line 陸劇保留簡體字幕（對白一致） — policy-correct, NOT a defect. The design's
 *   轉為繁中/仍要轉換 actions are NOT rendered: POST /api/v1/subtitles/convert
 *   exists but only converts a sidecar {name}.{lang}.srt|ass for movie/series —
 *   not embedded tracks, not episodes — and no client wires it yet
 *   (disc-2026-09-dialog-track-convert-not-wired).
 * - No cancel control for a running job: the backend exposes no cancel route;
 *   closing the dialog only stops watching (job continues server-side).
 */
import { useCallback, useEffect, useId, useRef, useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
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
import { MOBILE_SHEET_CLOSE, MOBILE_SHEET_CONTENT, SheetGrabber } from '../ui/mobileSheet';
import { cn } from '../../lib/utils';
// Shared subtitle-language labels (the detail page uses the same function, dsr-6b).
import { subtitleLangLabel, type SubtitleLangFamily } from '../../utils/libraryStatus';
import { transcriptionService } from '../../services/transcriptionService';
import type { SubtitleSearchResult } from '../../services/subtitleService';
import { useGenerationProgress } from '../../hooks/useGenerationProgress';
import { useGlossaryTerms } from '../../hooks/useGlossary';
import {
  transcriptionEstimateKeys,
  useTranscriptionEstimate,
} from '../../hooks/useTranscriptionEstimate';
import { useSubtitleSearch } from '../../hooks/useSubtitleSearch';
import { ButtonCost } from '../ui/ButtonCost';
import { GenerationProgressV2 } from './GenerationProgressV2';
import { GlossaryPanelV2 } from './GlossaryPanelV2';
import { deriveGenerateCostView, type RetryNote } from './generateCostView';

interface TrackRow {
  key: string;
  label: string;
  pillClass: string;
  source: string;
  isHans: boolean;
}

const PILL_CLASS: Record<SubtitleLangFamily, string> = {
  hant: 'bg-[var(--success-tint)] text-[var(--success-text)]',
  hans: 'bg-[var(--info-tint)] text-[var(--info-text)]',
  en: 'bg-[var(--bg-primary)] text-[var(--text-secondary)]',
  other: 'bg-[var(--bg-primary)] text-[var(--text-secondary)]',
};

/** Pill for one track. The label comes from the SHARED subtitleLangLabel, so this
 *  dialog and the detail page name a track identically (dsr-6b AC #3). */
function languageDescriptor(lang: string): { label: string; pillClass: string; isHans: boolean } {
  const { label, family } = subtitleLangLabel(lang);
  return { label, pillClass: PILL_CLASS[family], isHans: family === 'hans' };
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
    // `found` cannot tell a generated subtitle from a downloaded one, so the
    // source stays the neutral 字幕引擎 (design note on F1, dsr-6b).
    rows.push({ key: 'engine', source: '字幕引擎', ...d });
  } else if (subtitleStatus === 'untranslated') {
    // The English SRT a run already produced — the reason a re-run is
    // translate-only. Without this row the dialog said 尚無字幕 right above
    // 「僅需翻譯…」(dsr-6b 🔴 #3). An `untranslated` row can only come from
    // generation, so its source is 已生成.
    const d = languageDescriptor(subtitleLanguage || 'en');
    rows.push({ key: 'engine', source: '已生成', ...d });
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
  /** ISO 3166-1 codes; contains "CN" → §9b policy display. Movies pass it
   *  (LocalDetailV2); series/episodes have no production_countries. */
  productionCountry?: string;
  /** True while the parent detail query loads — renders the F10 skeleton. */
  isLoading?: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Fired on transcription_complete (AC 6 — parent invalidates detail + library caches). */
  onGenerationComplete?: () => void;
  /** Fired ONCE each time a run enters `failed` (dsr-6b AC #6) — the parent
   *  refetches so a kept English SRT shows under 現有字幕. May be a new function
   *  on every render; it is read through a ref, never an effect dependency. */
  onGenerationFailed?: () => void;
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
  onGenerationFailed,
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

  // dsr-6a AC #5 — the price on the paid buttons. The estimate replaces the
  // sub-2-2d GET /settings/keys pre-flight: it carries the run's own
  // translate check, so the degraded line and the amount can never come from
  // two different answers. Open-gated; a series has no generate route and
  // never asks.
  const queryClient = useQueryClient();
  const estimateMediaType = isMovie ? 'movie' : isEpisode ? 'episode' : null;
  const estimate = useTranscriptionEstimate(estimateMediaType, mediaId, {
    enabled: open && canGenerate,
  });
  const costView = deriveGenerateCostView({
    mediaType,
    subtitleStatus,
    estimate: { data: estimate.data, isError: estimate.isError, error: estimate.error },
  });
  const helperId = useId();
  const triggerRetryNoteId = useId();
  const retryNoteId = useId();

  const refreshEstimate = useCallback(() => {
    if (!estimateMediaType) return;
    void queryClient.invalidateQueries({
      queryKey: transcriptionEstimateKeys.item(estimateMediaType, mediaId),
    });
  }, [queryClient, estimateMediaType, mediaId]);

  // A run that failed may already have left the English SRT behind — the retry
  // is then translate-only and cheaper. Re-price whenever a run ends.
  const runPhase = generation.progress.phase;
  useEffect(() => {
    if (runPhase === 'failed' || runPhase === 'complete') refreshEstimate();
  }, [runPhase, refreshEstimate]);

  // dsr-6b AC #6 — tell the parent once per entry into `failed`. The callback
  // is read through a ref: SeasonAccordion passes a fresh inline arrow every
  // render, and as an effect dependency that would refetch → re-render → new
  // arrow → effect → refetch forever.
  const onGenerationFailedRef = useRef(onGenerationFailed);
  useEffect(() => {
    onGenerationFailedRef.current = onGenerationFailed;
  });
  const previousPhaseRef = useRef(runPhase);
  useEffect(() => {
    const previous = previousPhaseRef.current;
    previousPhaseRef.current = runPhase;
    if (runPhase === 'failed' && previous !== 'failed') onGenerationFailedRef.current?.();
  }, [runPhase]);

  // The row's subtitle state decides full vs translate-only. When it changes
  // under an open dialog (a parent refetch after a download, another job
  // finishing), the quote on screen may now be for the wrong run — and the
  // one direction that matters is a cheap resume quote surviving into a full
  // run. The first render is skipped: the query itself just fetched.
  const lastStatusRef = useRef(subtitleStatus);
  useEffect(() => {
    if (lastStatusRef.current === subtitleStatus) return;
    lastStatusRef.current = subtitleStatus;
    refreshEstimate();
  }, [subtitleStatus, refreshEstimate]);

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
      refreshEstimate();
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
        // dsr-6a: every open re-estimates — a price is a promise made at the
        // moment of the click, not five minutes earlier.
        if (estimateMediaType) {
          queryClient.removeQueries({
            queryKey: transcriptionEstimateKeys.item(estimateMediaType, mediaId),
          });
        }
      }
      onOpenChange(next);
    },
    [generation, onOpenChange, queryClient, estimateMediaType, mediaId]
  );

  const goToKeySettings = useCallback(() => navigate({ to: '/settings/keys' }), [navigate]);

  const renderRetryNote = (note: RetryNote | null, linkTestId: string) =>
    note ? (
      <>
        {note.text}
        {note.settingsLink && (
          <>
            {' '}
            <button
              type="button"
              onClick={goToKeySettings}
              data-testid={linkTestId}
              className="underline transition-colors hover:text-[var(--text-primary)]"
            >
              前往設定
            </button>
          </>
        )}
      </>
    ) : null;

  const tracks = buildTrackRows(subtitleTracks, subtitleStatus, subtitleLanguage);
  const inProgressView = genView === 'progress';
  const runIsLive = runPhase !== 'complete' && runPhase !== 'failed';
  const runFailed = inProgressView && runPhase === 'failed';

  // F4 footer hint: a blocking reason / ≈ first, then the translate-only resume.
  // The resume line is only said once the failure has been RE-priced: until then
  // `estimate.data` is the pre-failure answer, and a run that failed because its
  // English SRT vanished would claim a resume that is no longer true (dsr-6b CR L2).
  const failedHint: RetryNote | null =
    costView.retryNote ??
    (estimate.data?.plan === 'translate_only' && !estimate.isFetching
      ? { text: '已保留轉錄結果，重試只需翻譯', settingsLink: false }
      : null);

  /** 現有字幕. On the idle view an empty list is the F2 缺字幕 state; under a
   *  failed run an empty list renders nothing at all. */
  const renderTracksSection = (showEmptyState: boolean) => {
    if (tracks.length === 0) {
      return showEmptyState ? (
        <section
          data-testid="subtitle-empty-state"
          className="flex flex-col items-center gap-2 pb-2 pt-7"
        >
          <CaptionsOff className="h-9 w-9 text-[var(--text-muted)]" aria-hidden="true" />
          <p className="text-base font-semibold text-[var(--text-primary)]">尚無字幕</p>
          <p className="text-xs text-[var(--text-muted)]">此影片目前沒有任何字幕軌</p>
        </section>
      ) : null;
    }
    return (
      <section data-testid="subtitle-tracks-section" className="flex flex-col gap-2.5">
        <h3 className="text-sm font-semibold text-[var(--text-secondary)]">現有字幕</h3>
        {tracks.map((track) => (
          <div key={track.key} className="flex flex-col gap-1">
            <div
              data-testid={`subtitle-track-${track.key}`}
              className="flex items-center gap-3 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-3.5 py-3 max-sm:gap-2.5 max-sm:px-3 max-sm:py-2.5"
            >
              <span
                className={cn(
                  // 11px stays until disc-2026-09-11px-micro-label-not-on-type-scale is ruled.
                  'shrink-0 rounded-full px-2.5 py-1 text-[11px] font-medium',
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
    );
  };
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
        {
          onSuccess: () => {
            // A placed subtitle ends any translate-only resume — re-price now
            // rather than waiting for the parent's refetch to reach us.
            refreshEstimate();
            onDownloadSuccess?.();
          },
        }
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
      refreshEstimate,
    ]
  );

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent
        data-testid="manage-subtitle-dialog-v2"
        aria-describedby={undefined}
        closeClassName={cn(MOBILE_SHEET_CLOSE, 'max-sm:top-4')}
        className={cn(
          'flex max-h-[85vh] flex-col gap-0 overflow-hidden p-0',
          // Mobile: bottom sheet (F1-M-v2 JkdfH). Desktop: centered dialog (F1-D-v2 r1EY9).
          MOBILE_SHEET_CONTENT,
          // Desktop F1-D-v2 gD99f / F3-D-v2 wIihe: 880 wide, 1px hairline. The
          // hairline is sm-only — the phone sheet (F1-M FtarQ) is the shared shell above.
          'sm:bottom-auto sm:left-1/2 sm:right-auto sm:top-1/2 sm:w-[calc(100vw-4rem)] sm:max-w-[880px] sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-[var(--radius-lg)] sm:border sm:border-[var(--border-subtle)]'
        )}
      >
        {/* Phone: the sheet's drag handle (F1-M-v2 j53Qt). The other three
            subtitle sheets always had one; this dialog was the odd one out. */}
        <SheetGrabber data-testid="manage-sheet-grabber" />

        {/* Header. Phone (dsr-6f-1): 44 tall — the height of the ✕ — and 16px
            in. F1-M (CGvIz) draws NO rule under it; F3-M (B9gEpS) does, so the
            rule is dropped only outside the progress view. */}
        <div
          data-testid="manage-sheet-header"
          className={cn(
            'flex h-14 shrink-0 items-center justify-between border-b border-[var(--border-subtle)] pl-6 pr-12 max-sm:h-11 max-sm:pl-4',
            !inProgressView && 'max-sm:border-b-0'
          )}
        >
          <div className="flex min-w-0 items-center gap-1.5">
            <DialogTitle className="truncate text-base font-semibold">{dialogTitle}</DialogTitle>
            {mediaCode && (
              <span
                data-testid="dialog-title-code"
                // tO72N / Dey4O: plain Mono BodyLg 16/600, not a chip.
                // Phone: CvLdG is Body 14.
                className="shrink-0 font-mono text-base font-semibold text-[var(--text-primary)] max-sm:text-sm"
              >
                {mediaCode}
              </span>
            )}
          </div>
        </div>

        {/* Phone (PPLQr / tRFbG): [6,16,16,16] gap 14. Outside the progress view
            the phone sheet has NO footer, so this block is what meets the bottom
            edge and carries the safe-area inset. (Inert today — index.html has no
            viewport-fit=cover, so the inset is 0; see
            disc-2026-09-viewport-fit-cover-missing.) */}
        <div
          className={cn(
            'flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto px-6 py-5 max-sm:gap-3.5 max-sm:px-4 max-sm:pt-1.5',
            inProgressView ? 'max-sm:pb-4' : 'max-sm:pb-[max(1rem,env(safe-area-inset-bottom))]'
          )}
        >
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
              />
              {/* The stream is closed once a run ends — F4 draws no live chip. */}
              {runIsLive && (
                <div className="flex justify-center">
                  <span className="flex items-center gap-1.5 rounded-[var(--radius-sm)] bg-[var(--info-tint)] px-2 py-1 text-[11px] text-[var(--info-text)]">
                    <Radio className="h-3 w-3" aria-hidden="true" />
                    即時更新（SSE）
                  </span>
                </div>
              )}
              {generation.progress.phase === 'complete' && (
                <p
                  data-testid="generation-complete-note"
                  className="text-center text-sm text-[var(--text-secondary)]"
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
              {/* F4-D-v2 c6TLrS: what is already on disk after a failure — the
                  parent refetches on failure, so a kept English SRT appears. */}
              {runPhase === 'failed' && renderTracksSection(false)}
            </>
          ) : (
            <>
              {/* 現有字幕 (F1) or 缺字幕 empty state (F2-D-v2 S9Rbrq) */}
              {renderTracksSection(true)}

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
                  className="flex flex-col gap-2 rounded-[var(--radius-md)] bg-[var(--error-tint)] p-3"
                >
                  <div className="flex items-center gap-2">
                    <CircleAlert
                      className="h-4 w-4 shrink-0 text-[var(--error-text)]"
                      aria-hidden="true"
                    />
                    <p className="flex-1 text-sm text-[var(--error-text)]">
                      無法開始生成{triggerError ? `：${triggerError}` : ''}
                    </p>
                    {/* dsr-6a: a retry spends money again — same price, same rules. */}
                    <ButtonCost
                      label="重試"
                      cost={costView.cost}
                      busy={trigger.isPending}
                      onClick={startGeneration}
                      data-testid="generation-trigger-retry"
                      aria-describedby={costView.retryNote ? triggerRetryNoteId : undefined}
                      className="shrink-0"
                    />
                  </div>
                  {costView.retryNote && (
                    <p
                      id={triggerRetryNoteId}
                      data-testid="generation-trigger-retry-note"
                      className="text-right text-xs text-[var(--text-secondary)]"
                    >
                      {renderRetryNote(costView.retryNote, 'trigger-retry-goto-settings')}
                    </p>
                  )}
                </div>
              ) : (
                <section
                  data-testid="generation-section"
                  className={cn(
                    'flex items-center gap-4',
                    tracks.length === 0 && 'flex-col justify-center gap-2.5',
                    // Phone (F1-M zLRkb): a full-width button with its helper
                    // centred underneath. F2 (no tracks) has no phone drawing
                    // and follows the same rule.
                    'max-sm:flex-col max-sm:items-stretch max-sm:gap-1.5 max-sm:pt-1'
                  )}
                >
                  {/* dsr-6a: the ONLY primary action carries its price (J9-D). */}
                  <ButtonCost
                    label="生成字幕"
                    cost={costView.cost}
                    busy={trigger.isPending}
                    onClick={startGeneration}
                    data-testid="action-generate-subtitle"
                    aria-describedby={helperId}
                    className="max-sm:w-full max-sm:py-3"
                  />
                  <p
                    id={helperId}
                    data-testid="generation-helper"
                    data-tone={costView.helper.tone}
                    className={cn(
                      'text-xs max-sm:text-center',
                      costView.helper.tone === 'secondary'
                        ? 'text-[var(--text-secondary)]'
                        : 'text-[var(--text-muted)]'
                    )}
                  >
                    {/* Lines and their precedence: generateCostView (J9-D six
                        states + the SM supplement). Verb ruled 2026-08-06: this
                        button calls the transcribe (ASR) endpoint — 語音辨識. */}
                    {costView.helper.text}
                    {costView.helper.settingsLink && (
                      <>
                        {' '}
                        <button
                          type="button"
                          onClick={goToKeySettings}
                          data-testid="helper-goto-settings"
                          className="underline transition-colors hover:text-[var(--text-primary)]"
                        >
                          前往設定
                        </button>
                      </>
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
                <span className="text-sm text-[var(--text-secondary)]">名詞對照表</span>
                {/* h43R3: 「（」「8」「條）」 — no spaces inside the brackets. */}
                <span className="inline-flex items-baseline gap-0.5 text-sm text-[var(--text-secondary)]">
                  （<span className="font-mono tabular-nums">{glossaryCount}</span>條）
                </span>
                <span className="flex-1" />
                <ChevronRight
                  className="h-4 w-4 shrink-0 text-[var(--text-muted)]"
                  aria-hidden="true"
                />
              </button>

              {/* Phone: the idle sheet has no footer (F1-M V5LMv), so its
                  online-search toggle lives here — after the glossary row and
                  BEFORE the panel, so the panel opens under its trigger. Same
                  gate as the footer control (an episode would 400). */}
              {!isLoading && !isEpisode && (
                <button
                  type="button"
                  onClick={() => setFetchOpen((v) => !v)}
                  data-testid="toggle-fetch-mobile"
                  className="flex min-h-[44px] w-full items-center justify-center text-xs text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)] sm:hidden"
                >
                  搜尋線上字幕（成功率低）
                </button>
              )}

              {/* Dormant fetch section — NO source chips, NO score rows, NO Zimuku (9R-14). */}
              {fetchOpen && (
                <section data-testid="fetch-section" className="flex flex-col gap-2.5">
                  <div className="flex items-center gap-3">
                    <h3 className="text-sm font-semibold text-[var(--text-secondary)]">
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
                    <p className="text-sm text-[var(--error-text)]">
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

        {/* Footer — F1 idle: 搜尋線上字幕 + 關閉; F3 running (H2VIe): hint + 關閉;
            F4 failed (dg5rH): hint + 稍後再試 + 重試. */}
        {/* Phone (dsr-6f-1), by state:
            · outside the progress view — hidden. All it holds there is the
              online-search toggle (now in the body) and 關閉 (the 44px ✕); the
              paid 重試 of a TRIGGER error lives in the body, not here.
            · generating / complete (F3-M) — no footer is drawn: 關閉 is a
              full-width button with the hint under it, as a continuation of
              the body (no rule, 16px in).
            · failed — unchanged row. 重試 is a paid button with no phone
              drawing; it must not vanish because nobody drew it. */}
        <div
          className={cn(
            // max-sm:px-4: the phone body is 16px in; the footer lines up with it
            // in every state that keeps one (CR L4 — the failed row sat 8px inside).
            'flex shrink-0 flex-wrap items-center justify-between gap-3 border-t border-[var(--border-subtle)] px-6 py-3 max-sm:px-4',
            !inProgressView && 'max-sm:hidden',
            inProgressView &&
              !runFailed &&
              'max-sm:flex-col-reverse max-sm:items-stretch max-sm:border-t-0 max-sm:pt-0',
            inProgressView && 'max-sm:pb-[max(0.75rem,env(safe-area-inset-bottom))]'
          )}
        >
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
          ) : runFailed ? (
            failedHint ? (
              <p
                id={retryNoteId}
                data-testid="gen-retry-note"
                className="min-w-0 basis-full text-xs text-[var(--text-secondary)] sm:basis-auto sm:flex-1"
              >
                {renderRetryNote(failedHint, 'retry-goto-settings')}
              </p>
            ) : (
              <span />
            )
          ) : inProgressView && runIsLive ? (
            <span className="text-xs text-[var(--text-secondary)] max-sm:text-center">
              關閉後生成會在背景繼續
            </span>
          ) : (
            // Desktop needs SOMETHING on the left for justify-between; the phone
            // column must not get an empty 12px row out of it.
            <span className="max-sm:hidden" />
          )}
          {/* ml-auto: when the failed-run hint wraps to its own row on a phone,
              the buttons stay right-aligned on the row below. */}
          <div
            className={cn(
              'ml-auto flex shrink-0 items-center gap-3',
              // `ml-auto` cancels align-items:stretch; without this the button's
              // w-full resolves against a shrink-wrapped wrapper.
              !runFailed && 'max-sm:ml-0 max-sm:w-full'
            )}
          >
            <button
              type="button"
              onClick={() => handleOpenChange(false)}
              data-testid="dialog-close"
              className={cn(
                'flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)]',
                !runFailed && 'max-sm:w-full max-sm:justify-center'
              )}
            >
              {/* F4 RLbWb: after a failure the same close reads 稍後再試. */}
              {runFailed ? '稍後再試' : '關閉'}
            </button>
            {runFailed && (
              /* F4 Bfuec — a retry spends money again (dsr-6a): same price, same rules. */
              <ButtonCost
                label="重試"
                cost={costView.cost}
                busy={trigger.isPending}
                onClick={startGeneration}
                data-testid="gen-retry"
                aria-describedby={failedHint ? retryNoteId : undefined}
              />
            )}
          </div>
        </div>

        <GlossaryPanelV2
          mediaId={glossaryMediaId ?? mediaId}
          mediaTitle={mediaTitle}
          open={glossaryOpen}
          onOpenChange={setGlossaryOpen}
        />
      </DialogContent>
    </Dialog>
  );
}

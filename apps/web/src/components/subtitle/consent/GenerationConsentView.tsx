// Design ref: ux-design.pen Screen F14-D-v2 (nBT3M) · F15-D-v2 (pwMzT) · F15-M-v2 (fdu4y) · F18-D-v2 (zBik1) · F20-D-v2 (D7MOm)
/**
 * 產生字幕 consent flow container (sub-4-3 AC #1/#2/#3/#5) — replaces the old
 * idle branch of GenerationBatchDialogV2 (the scope-chips + aggregate-count
 * idle state is history: nothing starts without an explicit, priced, consented
 * selection).
 *
 * State machine (driven by the GET state envelope + the candidates SSE hook):
 *   open → GET snapshot
 *     ready              → F15 list (result from the snapshot)
 *     analyzing          → join: startTracking(seed) → F14
 *     idle/cancelled/err → POST analyze (409 = join, not an error) → F14
 *   SSE ready → GET again for the result (events carry counts only) → F15
 *   zero listable candidates → F20 empty state
 *   開始產生 → F16/F19 confirm → onStartBatch(selected ids, budgetUsd)
 *
 * budget_usd is WYSIWYG consent: the number on screen is ALWAYS sent — the
 * server-side env default is a fallback for the legacy dialog, never a hidden
 * data source for this flow. Prefill reads the snapshot's default_budget_usd
 * (sub-5-1 AC #6 — the operator's real AI_RUN_BUDGET_USD follows through);
 * DEFAULT_BUDGET_TEXT stays as the fallback for the error phase and
 * pre-sub-5-1 servers.
 *
 * Rule 23: zero wall-clock reads (`analyzed_at` is deliberately not rendered).
 */
import { useCallback, useEffect, useMemo, useState } from 'react';
import { Dialog, DialogContent, DialogTitle } from '../../ui/Dialog';
import { cn } from '../../../lib/utils';
import { subtitleService, type GenerationCandidate } from '../../../services/subtitleService';
import { useGenerationCandidatesProgress } from '../../../hooks/useGenerationCandidatesProgress';
import { AnalysisProgressPanel } from './AnalysisProgressPanel';
import { CandidateListPanel } from './CandidateListPanel';
import { ConsentEmptyState } from './ConsentEmptyState';
import { ConfirmGenerationDialog } from './ConfirmGenerationDialog';
import {
  computeTotals,
  defaultSelection,
  groupOrder,
  listableCandidates,
  parseBudgetInput,
  type ConsentRouteFilter,
} from './consentSelection';

/** Fallback prefill when the snapshot carries no default_budget_usd (error
 * phase, or a pre-sub-5-1 server). Matches the AI_RUN_BUDGET_USD factory
 * default. */
const DEFAULT_BUDGET_TEXT = '5.00';

type ConsentPhase = 'loading' | 'analyzing' | 'list' | 'empty' | 'error';

export interface GenerationConsentViewProps {
  open: boolean;
  /**
   * Media ids (movie AND episode UUIDs — D1) pre-checked when the flow opens
   * from a library selection. Intersected with the candidate list once the
   * analysis is ready; an empty intersection falls back to the default
   * (extract-only) selection. Callers MUST pass a render-stable array (state,
   * not an inline literal) — it participates in the bootstrap effect deps.
   */
  preselectedIds?: string[];
  /**
   * CR sub-4-3 H2: force a fresh (free) analysis even when a ready snapshot
   * exists. Set by the F17 post-scan deep link and after a batch terminal —
   * the two moments the library's candidate reality just changed and a stale
   * ready snapshot would silently show a pre-scan / pre-batch list.
   */
  forceAnalyze?: boolean;
  starting?: boolean;
  startError?: string | null;
  onStartBatch: (mediaIds: string[], budgetUsd: number) => void;
  onClose: () => void;
}

export function GenerationConsentView({
  open,
  preselectedIds,
  forceAnalyze = false,
  starting = false,
  startError = null,
  onStartBatch,
  onClose,
}: GenerationConsentViewProps) {
  const [phase, setPhase] = useState<ConsentPhase>('loading');
  const [candidates, setCandidates] = useState<GenerationCandidate[]>([]);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [filter, setFilter] = useState<ConsentRouteFilter>('all');
  const [budgetText, setBudgetText] = useState(DEFAULT_BUDGET_TEXT);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  /**
   * Why the list came back empty. `listableCandidates()` keeps only
   * extract/asr routes, so an empty list means「nothing is generatable」— which
   * is NOT the same claim as「everything already has subtitles」. Recording
   * which one actually happened is what stops the empty state lying; see the
   * header of ConsentEmptyState.tsx. Defaults to the non-claiming reason.
   */
  const [emptyAllCovered, setEmptyAllCovered] = useState(false);
  /**
   * How many titles the analysis looked at, for the empty state's readout.
   * Taken from the SNAPSHOT, not from `analysis.progress`: the `ready` path
   * never starts SSE tracking, so the progress hook is still 0/0 there — which
   * is exactly the path a cached analysis takes, and exactly the path that was
   * printing the false claim.
   */
  const [analyzedCount, setAnalyzedCount] = useState<number | undefined>(undefined);

  const analysis = useGenerationCandidatesProgress();
  const { startTracking: startAnalysisTracking, reset: resetAnalysis } = analysis;

  const seedList = useCallback(
    (all: GenerationCandidate[]) => {
      // 三序同源 (sub-5-3 AC #2): re-order the STATE itself into grouped
      // order, so display, submission (handleConfirm's filter walk) and the
      // F18 feasibleCount cumulative walk inherit ONE order with zero further
      // changes — a render-layer mapping would leave three orders to drift.
      const listable = groupOrder(listableCandidates(all));
      setCandidates(listable);
      if (listable.length === 0) {
        // The ONLY case that earns「所有影片都有繁中字幕了」: the analysis
        // returned records and every one was filtered out as already covered.
        // `all.length === 0` means nothing generatable was found at all — a
        // different fact, and the one a series-only library hits (ASR is
        // movies-only today).
        setEmptyAllCovered(all.length > 0);
        setPhase('empty');
        return;
      }
      const preselected = new Set(
        (preselectedIds ?? []).filter((id) => listable.some((c) => c.mediaId === id))
      );
      setSelectedIds(preselected.size > 0 ? preselected : defaultSelection(listable));
      setPhase('list');
    },
    [preselectedIds]
  );

  /** Fetch the snapshot; ready → list, analyzing → join, else → kick analyze.
   * `isCancelled` guards every post-await state write (CR M3 — the promise can
   * settle after unmount/close; the container's probe effect precedent). */
  const bootstrap = useCallback(
    async (isCancelled: () => boolean) => {
      setPhase('loading');
      setLoadError(null);
      try {
        const snap = await subtitleService.getGenerationCandidates();
        if (isCancelled()) return;
        // sub-5-1 AC #6: prefill from the operator's configured default —
        // BEFORE the phase branches so the ready / analyzing / kick-analyze
        // exits all get it. A non-positive or absent value keeps the
        // DEFAULT_BUDGET_TEXT fallback (pre-sub-5-1 server / unlimited=0).
        if (typeof snap.defaultBudgetUsd === 'number' && snap.defaultBudgetUsd > 0) {
          setBudgetText(snap.defaultBudgetUsd.toFixed(2));
        }
        if (!forceAnalyze && snap.status === 'ready' && snap.result) {
          setAnalyzedCount(snap.total || snap.analyzed);
          seedList(snap.result.candidates);
          return;
        }
        if (snap.status === 'analyzing') {
          startAnalysisTracking({ analyzed: snap.analyzed, total: snap.total });
          setPhase('analyzing');
          return;
        }
        // idle / cancelled / error — or a ready snapshot the caller declared
        // stale (forceAnalyze) → start a fresh (free) analysis.
        await subtitleService.startCandidateAnalysis(); // 409 = join, resolves either way
        if (isCancelled()) return;
        startAnalysisTracking();
        setPhase('analyzing');
      } catch (err) {
        if (isCancelled()) return;
        setLoadError(err instanceof Error ? err.message : '無法載入候選清單');
        setPhase('error');
      }
    },
    [forceAnalyze, seedList, startAnalysisTracking]
  );

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    void bootstrap(() => cancelled);
    return () => {
      cancelled = true;
      resetAnalysis();
    };
  }, [open, bootstrap, resetAnalysis]);

  // CR M5: F14 must not hang forever when the terminal `ready` SSE event is
  // missed (sub-second analysis finishing before the stream registers, or a
  // frame lost across the 10s reconnect backoff). A slow snapshot poll while
  // analyzing is the safety net; the SSE path stays the fast path.
  useEffect(() => {
    if (phase !== 'analyzing') return;
    let cancelled = false;
    const timer = setInterval(() => {
      void subtitleService
        .getGenerationCandidates()
        .then((snap) => {
          if (cancelled) return;
          if (snap.status === 'ready' && snap.result) {
            setAnalyzedCount(snap.total || snap.analyzed);
            seedList(snap.result.candidates);
          } else if (snap.status === 'cancelled') onClose();
          else if (snap.status === 'error') {
            setLoadError(snap.error || '分析失敗');
            setPhase('error');
          }
        })
        .catch(() => {
          // Transient poll failure — the next tick or the SSE path recovers.
        });
    }, 5000);
    return () => {
      cancelled = true;
      clearInterval(timer);
    };
  }, [phase, seedList, onClose]);

  // SSE terminal transitions: ready → fetch the result; cancelled/error → surface.
  const analysisStatus = analysis.status;
  useEffect(() => {
    if (phase !== 'analyzing') return;
    if (analysisStatus === 'ready') {
      void subtitleService
        .getGenerationCandidates()
        .then((snap) => {
          setAnalyzedCount(snap.total || snap.analyzed);
          if (snap.result) seedList(snap.result.candidates);
          else {
            // No result object at all — the analysis produced nothing to
            // classify, so this can never be the「everything is covered」case.
            setEmptyAllCovered(false);
            setPhase('empty');
          }
        })
        .catch((err) => {
          setLoadError(err instanceof Error ? err.message : '無法載入候選清單');
          setPhase('error');
        });
    } else if (analysisStatus === 'error') {
      setLoadError(analysis.progress.error || '分析失敗');
      setPhase('error');
    } else if (analysisStatus === 'cancelled') {
      onClose();
    }
  }, [analysisStatus, phase, seedList, analysis.progress.error, onClose]);

  const budgetUsd = parseBudgetInput(budgetText);
  const totals = useMemo(
    () => computeTotals(candidates, selectedIds, budgetUsd),
    [candidates, selectedIds, budgetUsd]
  );

  const handleToggle = useCallback((mediaId: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(mediaId)) next.delete(mediaId);
      else next.add(mediaId);
      return next;
    });
  }, []);

  /** 整劇/整季 toggle (sub-5-3): set the group's ids to one target state. */
  const handleToggleGroup = useCallback((mediaIds: string[], next: boolean) => {
    setSelectedIds((prev) => {
      const nextSet = new Set(prev);
      for (const id of mediaIds) {
        if (next) nextSet.add(id);
        else nextSet.delete(id);
      }
      return nextSet;
    });
  }, []);

  const handleToggleAll = useCallback(() => {
    setSelectedIds((prev) =>
      prev.size === candidates.length ? new Set() : new Set(candidates.map((c) => c.mediaId))
    );
  }, [candidates]);

  const handleSelectAllExtract = useCallback(() => {
    setSelectedIds(defaultSelection(candidates));
  }, [candidates]);

  const handleClearSelection = useCallback(() => setSelectedIds(new Set()), []);

  const handleCancelAnalysis = useCallback(async () => {
    setCancelling(true);
    try {
      await subtitleService.cancelCandidateAnalysis();
      // The terminal `cancelled` SSE event closes the view (effect above);
      // fall back to closing directly if the stream missed it.
      onClose();
    } catch {
      onClose();
    } finally {
      setCancelling(false);
    }
  }, [onClose]);

  const handleConfirm = useCallback(() => {
    if (budgetUsd === null) return;
    // Submission order = list order (= the F18 feasibleCount walk order).
    // The confirm dialog STAYS OPEN while the start is in flight (CR L8):
    // success unmounts the whole consent view (status flips to running);
    // failure closes it via the startError effect below, surfacing the error
    // in the list panel.
    const ids = candidates.filter((c) => selectedIds.has(c.mediaId)).map((c) => c.mediaId);
    onStartBatch(ids, budgetUsd);
  }, [budgetUsd, candidates, selectedIds, onStartBatch]);

  useEffect(() => {
    if (startError) setConfirmOpen(false);
  }, [startError]);

  return (
    <>
      <Dialog
        open={open}
        onOpenChange={(next) => {
          if (!next) onClose();
        }}
      >
        <DialogContent
          data-testid="generation-consent-view"
          aria-describedby={undefined}
          className={cn(
            'flex max-h-[85vh] flex-col gap-0 overflow-hidden p-0',
            // Mobile: bottom sheet (F15-M-v2 fdu4y). Desktop: centered dialog.
            'bottom-0 left-0 right-0 top-auto w-full max-w-none translate-x-0 translate-y-0 rounded-b-none rounded-t-[var(--radius-xl)]',
            'sm:bottom-auto sm:left-1/2 sm:right-auto sm:top-1/2 sm:w-[calc(100vw-4rem)] sm:max-w-3xl sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-[var(--radius-lg)]'
          )}
        >
          {/* Mobile bottom-sheet drag handle (F8 precedent, sm:hidden). */}
          <div className="flex shrink-0 justify-center pb-1 pt-2 sm:hidden">
            <span aria-hidden="true" className="h-1 w-9 rounded-full bg-[var(--bg-tertiary)]" />
          </div>

          <div className="flex h-14 shrink-0 items-center justify-between border-b border-[var(--border-subtle)] pl-6 pr-12">
            <DialogTitle className="truncate text-base font-semibold">產生字幕</DialogTitle>
          </div>

          {/* Phase transitions announced to AT. */}
          <p aria-live="polite" className="sr-only" data-testid="consent-phase-live">
            {phase === 'analyzing'
              ? '正在分析字幕軌'
              : phase === 'list'
                ? '候選清單已就緒'
                : phase === 'empty'
                  ? emptyAllCovered
                    ? '所有影片都有繁中字幕了'
                    : '沒有找到可以產生字幕的項目'
                  : ''}
          </p>

          {(phase === 'loading' || phase === 'analyzing') && (
            <AnalysisProgressPanel
              analyzed={analysis.progress.analyzed}
              total={analysis.progress.total}
              cancelling={cancelling}
              onCancel={() => void handleCancelAnalysis()}
            />
          )}

          {phase === 'list' && (
            <CandidateListPanel
              candidates={candidates}
              selectedIds={selectedIds}
              filter={filter}
              totals={totals}
              budgetText={budgetText}
              budgetUsd={budgetUsd}
              starting={starting}
              startError={startError}
              onToggle={handleToggle}
              onToggleGroup={handleToggleGroup}
              onToggleAll={handleToggleAll}
              onSelectAllExtract={handleSelectAllExtract}
              onClearSelection={handleClearSelection}
              onFilterChange={setFilter}
              onBudgetTextChange={setBudgetText}
              onStartClick={() => setConfirmOpen(true)}
            />
          )}

          {phase === 'empty' && (
            <>
              <ConsentEmptyState allCovered={emptyAllCovered} analyzed={analyzedCount} />
              <div className="flex shrink-0 items-center justify-end border-t border-[var(--border-subtle)] px-6 py-3.5">
                <button
                  type="button"
                  onClick={onClose}
                  data-testid="consent-empty-close"
                  className="flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)]"
                >
                  關閉
                </button>
              </div>
            </>
          )}

          {phase === 'error' && (
            <div className="flex flex-1 flex-col items-center justify-center gap-4 px-6 py-12">
              <p data-testid="consent-load-error" className="text-sm text-[var(--error-text)]">
                {loadError}
              </p>
              <button
                type="button"
                // Pre-existing fix (sub-5-3): bootstrap REQUIRES the isCancelled
                // guard param (CR sub-4-3 M3 added it) — calling it bare threw
                // TypeError inside the try, so 重試 just re-rendered the error.
                onClick={() => void bootstrap(() => false)}
                className="flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)]"
              >
                重試
              </button>
            </div>
          )}
        </DialogContent>
      </Dialog>

      {budgetUsd !== null && (
        <ConfirmGenerationDialog
          open={confirmOpen}
          totals={totals}
          budgetUsd={budgetUsd}
          confirming={starting}
          onConfirm={handleConfirm}
          onCancel={() => {
            // CR L8: no silent dismiss while a paid start is in flight.
            if (!starting) setConfirmOpen(false);
          }}
        />
      )}
    </>
  );
}

// Design ref: ux-design.pen Screen L3-D-v2 (He04g)
// Source: ux-design.pen (Pencil app)
/**
 * L3 · 部分請求 — pick exactly which seasons or episodes of a show to request
 * (Story 13-2b). A master 整部影集 row, one row per season (expand → its
 * episodes, loaded lazily), owned/requested episodes shown locked with a pill,
 * and a footer that reads 已選 N 季 · M 集. The selection rules and the
 * selection → wire mapping live in ./seasonSelection (pure, unit-tested).
 *
 * Consumes: confirmed against [@contract-v1] (13-2a AC #1 selection, AC #5
 * coverage); GET /tmdb/tv/:id/season/:n (13-2a route).
 */
import { useEffect, useMemo, useRef, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { ChevronDown, ChevronRight, X } from 'lucide-react';
import { Link } from '@tanstack/react-router';
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '../ui/Dialog';
import { Button } from '../ui/Button';
import { CheckboxBox, type CheckboxBoxState } from '../ui/CheckboxBox';
import { useTVShowDetails } from '../../hooks/useMediaDetails';
import { requestKeys } from '../../hooks/useRequestedMedia';
import { requestService } from '../../services/requestService';
import { tmdbService } from '../../services/tmdb';
import type { TMDbSeasonDetails } from '../../types/tmdb';
import {
  EMPTY_SELECTION,
  buildRequestPayload,
  episodeChecked,
  episodeLock,
  isEmptySelection,
  masterState,
  seasonFullyOwned,
  seasonLocked,
  seasonState,
  seasonsNeedingEpisodeList,
  selectionSummary,
  toggleEpisode,
  toggleMaster,
  toggleSeason,
  treeSeasons,
  type CheckState,
  type RequestCoverage,
  type RequestPayload,
  type TreeSeason,
  type TreeSelection,
} from '../../utils/requestSelection';

/** Episodes shown before 顯示其餘 N 集 (the drawn S1 lists E01–E06). */
const EPISODE_PREVIEW = 6;

export const seasonEpisodesKey = (tmdbId: number, season: number) =>
  ['tmdb', 'tv', tmdbId, 'season', season] as const;
export const coverageKey = (tmdbId: number) => [...requestKeys.all, 'coverage', tmdbId] as const;

export interface SeasonEpisodeTreeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  tmdbId: number;
  title: string;
  /** Called with the wire selection; the caller creates the request. */
  onConfirm: (payload: RequestPayload) => void;
  /** Seasons to open expanded (gallery baselines; nothing in the product sets it). */
  defaultExpanded?: number[];
  /** The caller's create is in flight — 確認請求 stays disabled, the tree stays open. */
  submitting?: boolean;
  /** The caller's create failed — shown in the footer, picks kept (13-2b CR). */
  submitError?: string | null;
  /**
   * 13-2c (owned series): no coverage means "unknown", not "nothing owned" —
   * show an error with 重試 instead of a tree that could send the whole title.
   */
  requireCoverage?: boolean;
  /** Forwarded to the dialog: where focus goes when it closes. */
  onCloseAutoFocus?: (event: Event) => void;
}

export function SeasonEpisodeTreeDialog({
  open,
  onOpenChange,
  tmdbId,
  title,
  onConfirm,
  defaultExpanded,
  submitting = false,
  submitError = null,
  requireCoverage = false,
  onCloseAutoFocus,
}: SeasonEpisodeTreeDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        data-testid="season-tree-dialog"
        // The drawn modal: 560 wide, $bg-secondary, $radius-lg, hairline border;
        // the header carries its own 44px ✕, so the primitive's is hidden.
        className="flex max-h-[85vh] w-[calc(100vw-2rem)] max-w-[560px] flex-col overflow-hidden rounded-[var(--radius-lg)] border border-[var(--border-subtle)] p-0"
        closeClassName="hidden"
        onCloseAutoFocus={onCloseAutoFocus}
      >
        {/* No {open && …} gate: Radix unmounts the content itself once its exit
            animation ends, which also resets the tree's state — gating here
            emptied the box while it was still animating out (13-2b CR). */}
        <TreeBody
          tmdbId={tmdbId}
          title={title}
          onConfirm={onConfirm}
          onClose={() => onOpenChange(false)}
          defaultExpanded={defaultExpanded}
          parentSubmitting={submitting}
          parentError={submitError}
          requireCoverage={requireCoverage}
        />
      </DialogContent>
    </Dialog>
  );
}

function TreeBody({
  tmdbId,
  title,
  onConfirm,
  onClose,
  defaultExpanded,
  parentSubmitting,
  parentError,
  requireCoverage,
}: {
  tmdbId: number;
  title: string;
  onConfirm: (payload: RequestPayload) => void;
  onClose: () => void;
  defaultExpanded?: number[];
  parentSubmitting: boolean;
  parentError: string | null;
  requireCoverage: boolean;
}) {
  const queryClient = useQueryClient();
  const details = useTVShowDetails(tmdbId);
  const coverageQuery = useQuery({
    queryKey: coverageKey(tmdbId),
    queryFn: () => requestService.getCoverage(tmdbId),
    // Same freshness as the request list; the key sits under requestKeys.all
    // (['requests', …]), so every create/cancel/retry invalidates it anyway.
    staleTime: 30 * 1000,
    retry: false,
  });
  // AC #2 fail-soft: no coverage → the tree still opens without badges; the
  // 13-2a backend guard still refuses an overlapping selection.
  useEffect(() => {
    if (coverageQuery.isError) {
      console.warn(
        '[13-2b] request coverage unavailable — tree opens without badges',
        coverageQuery.error
      );
    }
  }, [coverageQuery.isError, coverageQuery.error]);
  const coverage = coverageQuery.data;

  const seasons = useMemo(() => treeSeasons(details.data?.seasons), [details.data?.seasons]);
  const selectableSeasons = useMemo(
    () => seasons.filter((s) => !seasonLocked(coverage, s)).map((s) => s.seasonNumber),
    [seasons, coverage]
  );
  const totalEpisodes = seasons.reduce((n, s) => n + s.episodeCount, 0);

  const [selection, setSelection] = useState<TreeSelection>(EMPTY_SELECTION);
  const [expanded, setExpanded] = useState<ReadonlySet<number>>(
    () => new Set(defaultExpanded ?? [])
  );
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  // Closing the dialog mid-submit (取消 / Esc / ✕ while unexpanded seasons
  // load) must not create a request afterwards (13-2b CR).
  const aliveRef = useRef(true);
  useEffect(
    () => () => {
      aliveRef.current = false;
    },
    []
  );

  const summary = selectionSummary(selection);
  const loading = details.isLoading || coverageQuery.isLoading;
  const blocked = !!(coverage?.activeRequest || coverage?.wholeSeriesRequested);
  const coverageMissing = requireCoverage && coverageQuery.isError && !coverage;
  const busy = submitting || parentSubmitting;
  const error = submitError ?? parentError;
  const showTree =
    !loading &&
    !(details.isError && !details.data) &&
    !coverageMissing &&
    !blocked &&
    seasons.length > 0;

  const submit = async () => {
    setSubmitting(true);
    setSubmitError(null);
    try {
      const need = seasonsNeedingEpisodeList(selection, seasons, coverage);
      const lists = new Map<number, number[]>();
      await Promise.all(
        need.map(async (n) => {
          const season = await queryClient.fetchQuery({
            queryKey: seasonEpisodesKey(tmdbId, n),
            queryFn: () => tmdbService.getSeasonDetails(tmdbId, n),
            staleTime: 10 * 60 * 1000,
          });
          lists.set(
            n,
            season.episodes.map((e) => e.episodeNumber)
          );
        })
      );
      if (!aliveRef.current) return;
      const payload = buildRequestPayload(selection, seasons, coverage, lists, requireCoverage);
      // A partial selection that resolves to nothing (every picked episode is
      // already owned/requested) must never go out empty — on the wire that
      // means the WHOLE title (13-2b CR).
      if (
        !payload.whole &&
        payload.seasons.length === 0 &&
        Object.keys(payload.episodes).length === 0
      ) {
        setSubmitError('選到的集數都已入庫或已請求');
        return;
      }
      onConfirm(payload);
    } catch {
      if (aliveRef.current) setSubmitError('無法載入集數資料，請稍後再試');
    } finally {
      if (aliveRef.current) setSubmitting(false);
    }
  };

  return (
    <>
      {/* modal-header (bITma): padding [20,16,12,24] */}
      <div className="flex items-start justify-between gap-3 pb-3 pl-6 pr-4 pt-5">
        <div className="flex min-w-0 flex-col gap-1">
          <DialogTitle className="text-lg font-semibold text-[var(--text-primary)] md:text-xl">
            選擇要請求的內容
          </DialogTitle>
          <DialogDescription className="truncate text-xs text-[var(--text-secondary)]">
            {title} · 影集
          </DialogDescription>
        </div>
        <button
          type="button"
          onClick={onClose}
          aria-label="關閉"
          data-testid="season-tree-close"
          className="flex size-11 shrink-0 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]"
        >
          <X className="size-[18px]" aria-hidden="true" />
        </button>
      </div>
      <div className="h-px shrink-0 bg-[var(--border-subtle)]" />

      <div className="relative min-h-0 flex-1 overflow-y-auto px-3 py-2">
        {loading ? (
          <TreeSkeleton />
        ) : details.isError && !details.data ? (
          <TreeMessage testId="season-tree-error" text="無法載入季資料">
            <Button variant="secondary" className="h-11" onClick={() => details.refetch()}>
              重試
            </Button>
          </TreeMessage>
        ) : coverageMissing ? (
          // 13-2c: on an owned series "no coverage" would read as "nothing
          // owned" and could send the whole title — refuse to guess.
          <TreeMessage testId="season-tree-coverage-error" text="無法確認哪些集數已經有了">
            <Button variant="secondary" className="h-11" onClick={() => coverageQuery.refetch()}>
              重試
            </Button>
          </TreeMessage>
        ) : blocked ? (
          // ⚖️ A (2026-08-19): an open request for this show blocks the tree —
          // one active request per title (13-2a duplicate guard).
          <TreeMessage testId="season-tree-active" text="這部影集已有進行中的請求">
            <Link
              to="/discover"
              search={{ view: 'requests' }}
              onClick={onClose}
              className="flex h-11 items-center px-2.5 text-sm font-semibold text-[var(--accent-text)]"
            >
              查看清單
            </Link>
          </TreeMessage>
        ) : seasons.length === 0 ? (
          <TreeMessage testId="season-tree-empty" text="這部影集還沒有可選的季資料" />
        ) : (
          <div
            role="group"
            aria-label="季與集"
            className="flex flex-col gap-0.5"
            data-testid="season-tree"
          >
            <TreeCheckRow
              testId="season-tree-master"
              state={toBox(
                masterState(selection, selectableSeasons),
                selectableSeasons.length === 0,
                false
              )}
              onToggle={() => setSelection((s) => toggleMaster(s, selectableSeasons))}
              label={
                <span className="text-sm font-semibold text-[var(--text-primary)]">整部影集</span>
              }
              right={
                <span className="text-xs text-[var(--text-secondary)]">
                  <span className="font-mono">{seasons.length}</span> 季 ·{' '}
                  <span className="font-mono">{totalEpisodes}</span> 集
                </span>
              }
            />
            {seasons.map((season) => (
              <SeasonBlock
                key={season.seasonNumber}
                tmdbId={tmdbId}
                season={season}
                coverage={coverage}
                selection={selection}
                expanded={expanded.has(season.seasonNumber)}
                onToggleExpand={() =>
                  setExpanded((prev) => {
                    const next = new Set(prev);
                    if (next.has(season.seasonNumber)) next.delete(season.seasonNumber);
                    else next.add(season.seasonNumber);
                    return next;
                  })
                }
                onToggleSeason={() => setSelection((s) => toggleSeason(s, season.seasonNumber))}
                onToggleEpisode={(episode, selectable) =>
                  setSelection((s) => toggleEpisode(s, season.seasonNumber, episode, selectable))
                }
              />
            ))}
          </div>
        )}
      </div>

      {/* Footer only with a tree to confirm: the blocked / error / empty
          states have nothing to submit, and the header ✕ closes them. */}
      {showTree && (
        <>
          <div className="h-px shrink-0 bg-[var(--border-subtle)]" />
          {/* modal-footer (ptDR3): padding [16,24], summary left, buttons right */}
          <div className="flex flex-wrap items-center justify-between gap-3 px-6 py-4">
            <p
              className="text-sm text-[var(--text-secondary)]"
              data-testid="season-tree-summary"
              aria-live="polite"
            >
              已選{' '}
              <span className="font-mono font-medium text-[var(--text-primary)]">
                {summary.seasons}
              </span>{' '}
              季 ·{' '}
              <span className="font-mono font-medium text-[var(--text-primary)]">
                {summary.episodes}
              </span>{' '}
              集
            </p>
            <div className="flex items-center gap-3">
              {error && (
                <span role="alert" className="text-xs text-[var(--error-text)]">
                  {error}
                </span>
              )}
              <Button variant="secondary" className="h-11 px-5" onClick={onClose}>
                取消
              </Button>
              <Button
                className="h-11 px-5"
                data-testid="season-tree-confirm"
                disabled={isEmptySelection(selection) || busy}
                onClick={submit}
              >
                確認請求
              </Button>
            </div>
          </div>
        </>
      )}
    </>
  );
}

function toBox(state: CheckState, disabled: boolean, disabledChecked: boolean): CheckboxBoxState {
  if (disabled) return disabledChecked ? 'disabled-checked' : 'disabled-empty';
  return state === 'checked' ? 'checked' : state === 'mixed' ? 'mixed' : 'empty';
}

/**
 * One 44px tree row: a native checkbox (sr-only) + the drawn box + label,
 * wrapped in a <label> so the whole row toggles. `indent` = episode rows.
 */
function TreeCheckRow({
  state,
  onToggle,
  label,
  right,
  indent = false,
  grow = false,
  testId,
}: {
  state: CheckboxBoxState;
  onToggle: () => void;
  label: React.ReactNode;
  right?: React.ReactNode;
  indent?: boolean;
  /** Fill the row's remaining width (the season row shares it with the chevron). */
  grow?: boolean;
  testId?: string;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const disabled = state === 'disabled-checked' || state === 'disabled-empty';
  useEffect(() => {
    if (inputRef.current) inputRef.current.indeterminate = state === 'mixed';
  }, [state]);
  return (
    <label
      data-testid={testId}
      className={`flex h-11 min-w-0 shrink-0 items-center gap-3 rounded-[var(--radius-md)] pr-3 ${
        grow ? 'flex-1' : ''
      } ${
        indent ? 'pl-12' : 'pl-3'
      } ${disabled ? 'cursor-default' : 'cursor-pointer hover:bg-[var(--bg-tertiary)]'}`}
    >
      <input
        ref={inputRef}
        type="checkbox"
        className="peer sr-only"
        checked={state === 'checked' || state === 'disabled-checked'}
        disabled={disabled}
        onChange={onToggle}
      />
      <CheckboxBox state={state} />
      {label}
      {right && <span className="ml-auto flex shrink-0 items-center gap-1.5">{right}</span>}
    </label>
  );
}

function SeasonBlock({
  tmdbId,
  season,
  coverage,
  selection,
  expanded,
  onToggleExpand,
  onToggleSeason,
  onToggleEpisode,
}: {
  tmdbId: number;
  season: TreeSeason;
  coverage: RequestCoverage | undefined;
  selection: TreeSelection;
  expanded: boolean;
  onToggleExpand: () => void;
  onToggleSeason: () => void;
  onToggleEpisode: (episode: number, selectable: number[]) => void;
}) {
  const n = season.seasonNumber;
  const locked = seasonLocked(coverage, season);
  const [showAll, setShowAll] = useState(false);
  const episodesQuery = useQuery<TMDbSeasonDetails>({
    queryKey: seasonEpisodesKey(tmdbId, n),
    queryFn: () => tmdbService.getSeasonDetails(tmdbId, n),
    staleTime: 10 * 60 * 1000,
    enabled: expanded,
  });
  const episodes = episodesQuery.data?.episodes ?? [];
  const selectable = episodes
    .map((e) => e.episodeNumber)
    .filter((e) => episodeLock(coverage, n, e) === null);
  const visible = showAll ? episodes : episodes.slice(0, EPISODE_PREVIEW);
  const code = `S${n}`;

  return (
    <div data-testid={`season-row-${n}`}>
      <div className="flex items-center">
        <TreeCheckRow
          grow
          state={toBox(seasonState(selection, n), locked, seasonFullyOwned(coverage, season))}
          onToggle={onToggleSeason}
          label={
            <span className="flex min-w-0 items-center gap-3">
              <span className="font-mono text-sm text-[var(--text-secondary)]">{code}</span>
              <span className="truncate text-sm text-[var(--text-primary)]">{season.name}</span>
            </span>
          }
          right={
            <span className="text-xs text-[var(--text-secondary)]">
              <span className="font-mono">{season.episodeCount}</span> 集
            </span>
          }
        />
        <button
          type="button"
          onClick={onToggleExpand}
          aria-expanded={expanded}
          aria-label={`${expanded ? '收起' : '展開'}${season.name}的集數`}
          data-testid={`season-expand-${n}`}
          className="flex size-11 shrink-0 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]"
        >
          {expanded ? (
            <ChevronDown className="size-4" aria-hidden="true" />
          ) : (
            <ChevronRight className="size-4" aria-hidden="true" />
          )}
        </button>
      </div>

      {expanded && (
        <div className="flex flex-col gap-0.5" data-testid={`season-episodes-${n}`}>
          {episodesQuery.isLoading && (
            <p className="h-11 py-3 pl-12 text-xs text-[var(--text-secondary)]">載入集數中…</p>
          )}
          {episodesQuery.isError && (
            <p className="h-11 py-3 pl-12 text-xs text-[var(--error-text)]">無法載入集數</p>
          )}
          {visible.map((ep) => {
            const lock = episodeLock(coverage, n, ep.episodeNumber);
            return (
              <TreeCheckRow
                key={ep.episodeNumber}
                indent
                testId={`episode-row-${n}-${ep.episodeNumber}`}
                state={
                  lock
                    ? lock === 'owned'
                      ? 'disabled-checked'
                      : 'disabled-empty'
                    : episodeChecked(selection, n, ep.episodeNumber)
                      ? 'checked'
                      : 'empty'
                }
                onToggle={() => onToggleEpisode(ep.episodeNumber, selectable)}
                label={
                  <span className="flex min-w-0 items-center gap-3">
                    <span className="font-mono text-xs text-[var(--text-secondary)]">
                      E{String(ep.episodeNumber).padStart(2, '0')}
                    </span>
                    <span
                      className={`truncate text-sm ${
                        lock ? 'text-[var(--text-secondary)]' : 'text-[var(--text-primary)]'
                      }`}
                    >
                      {ep.name}
                    </span>
                  </span>
                }
                right={lock ? <LockPill lock={lock} /> : undefined}
              />
            );
          })}
          {!showAll && episodes.length > EPISODE_PREVIEW && (
            <button
              type="button"
              onClick={() => setShowAll(true)}
              className="flex h-11 items-center gap-1 rounded-[var(--radius-md)] pl-20 text-left text-xs text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]"
            >
              顯示其餘 <span className="font-mono">{episodes.length - EPISODE_PREVIEW}</span> 集
            </button>
          )}
        </div>
      )}
    </div>
  );
}

function LockPill({ lock }: { lock: 'owned' | 'requested' }) {
  return lock === 'owned' ? (
    <span className="rounded-full bg-[var(--success-tint)] px-2 py-0.5 text-xs font-semibold text-[var(--success-text)]">
      已入庫
    </span>
  ) : (
    <span className="rounded-full bg-[var(--info-tint)] px-2 py-0.5 text-xs font-semibold text-[var(--info-text)]">
      已請求
    </span>
  );
}

function TreeMessage({
  text,
  testId,
  children,
}: {
  text: string;
  testId: string;
  children?: React.ReactNode;
}) {
  return (
    <div data-testid={testId} className="flex flex-col items-center gap-3 px-6 py-10 text-center">
      <p className="text-sm text-[var(--text-secondary)]">{text}</p>
      {children}
    </div>
  );
}

function TreeSkeleton() {
  return (
    <div data-testid="season-tree-skeleton" aria-busy="true" className="flex flex-col gap-0.5">
      {Array.from({ length: 5 }).map((_, i) => (
        <div key={i} className="flex h-11 items-center gap-3 px-3">
          <span className="size-5 shrink-0 animate-pulse rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)] motion-reduce:animate-none" />
          <span className="h-3 w-40 animate-pulse rounded bg-[var(--bg-tertiary)] motion-reduce:animate-none" />
        </div>
      ))}
    </div>
  );
}

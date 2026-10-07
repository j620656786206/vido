// Design ref: ux-design.pen Screen B4p-D (N2fmG6) + Screen J3-D (Z54xAd) + Screen J11-D (w2Opax)
/**
 * EpisodeList (Story 12-2)
 *
 * Presentational list of a season's episodes. The parent (SeasonAccordion) owns
 * the TanStack Query and passes loading/error/retry down — mirroring the
 * DualRatingDisplay (Story 12-1) presentational pattern.
 *
 * Each episode row shows: SxxExx code, title, air date, runtime, and — only when
 * the episode has a local file — a subtitle status indicator (AC #4/#5/#6).
 * On mobile, rows stack (title/date line, metadata below) for readability (AC #8).
 *
 * 9R-10c: rows with a local file also carry a 管理字幕 action (design J3-D
 * `Z54xAd`, node `btn-subtitle`). J3-D's ruling is that the action is IDENTICAL
 * for all ten subtitle_status values — the status indicator already encodes
 * state, and a status-dependent action would encode the same fact twice and
 * make buttons flicker in and out down a 25-episode list. The only gate is
 * hasLocalFile. Feasibility is explained inside the dialog, which has room.
 *
 * Presentational by contract: the dialog state lives in SeasonAccordionItem;
 * this component only raises onManageSubtitle.
 */

import {
  CheckCircle2,
  XCircle,
  Loader2,
  Minus,
  CircleSlash,
  CircleDashed,
  AlertCircle,
} from 'lucide-react';
import { cn } from '../../lib/utils';
import { Tooltip } from '../ui/Tooltip';
import type { ChineseSubtitleSource, MergedEpisode } from '../../types/library';

/**
 * The single gate for the per-episode subtitle entry (CR M3).
 *
 * `hasLocalFile` alone is NOT enough: `episode_id` is `omitempty` on the wire,
 * so a frontend deployed against a backend older than 9R-10a gets rows that
 * have a file but no address. Gating the BUTTON on one predicate and the DIALOG
 * on another produced buttons that rendered, clicked, and silently did nothing.
 * Both sides now ask this one question.
 */
export function canManageEpisodeSubtitle(
  episode: MergedEpisode
): episode is MergedEpisode & { episodeId: string; filePath: string } {
  return Boolean(episode.hasLocalFile && episode.episodeId && episode.filePath);
}

interface EpisodeListProps {
  episodes: MergedEpisode[];
  seasonNumber: number;
  isLoading?: boolean;
  isError?: boolean;
  onRetry?: () => void;
  /** Raised when a row's 管理字幕 is activated (9R-10c). Omitted → no action
   *  is rendered at all, keeping every existing caller byte-identical. */
  onManageSubtitle?: (episode: MergedEpisode) => void;
}

/** Formats a season/episode pair as SxxExx (e.g. S01E05). Exported so callers
 *  hosting the subtitle dialog label it identically (CR L1 — it was duplicated). */
export function episodeCode(seasonNumber: number, episodeNumber: number): string {
  const s = String(seasonNumber).padStart(2, '0');
  const e = String(episodeNumber).padStart(2, '0');
  return `S${s}E${e}`;
}

// The 10 `subtitle_status` values (sub-1-2 [@contract-v2]); treatment ratified by
// sub-1-7a, spec screen `flow-j-specs/j2-d` (the untranslated row lands with γ).
//
// `label` is the LONG form on purpose: the icon carries no visible text, so the
// accessible name is the only place the full explanation can live. That is where
// "已略過 must read as deliberate, not broken" is actually solved — the poster
// badge's 3–4-char label (libraryStatus.ts) cannot carry it.
//
// ICON GRAMMAR (J2-D, Sally 2026-08-04) — read this before adding a state:
//   CIRCLED glyph = a settled outcome. The pipeline has an answer for this file.
//     CheckCircle2 (found) · XCircle (not_found / no_text_source) · CircleSlash (skipped)
//   BARE glyph = not an outcome yet.
//     Minus (not_searched) · Loader2 (the four in-flight states)
// Colour is URGENCY, not outcome: --error-text = the user can do something about it,
// --text-muted = nothing to do, --success = done, --accent-text = in progress
// (accent stays reserved for in-progress — Sally 2026-07-05).
//
// `--accent-text` rather than `--accent-primary`: an icon is foreground, and
// #3b82f6 measures 3.04:1 on `--bg-tertiary` — at the WCAG 1.4.11 non-text
// threshold, so any hover surface puts it under. #60a5fa holds 4.40:1 worst case.
const SUBTITLE_STATUS: Record<
  string,
  { Icon: typeof CheckCircle2; color: string; label: string; spin?: boolean }
> = {
  // Uniform with its siblings below. `--success` DOES clear the 3:1 icon floor
  // (6.44:1), unlike cinnabar — this moves for vocabulary, not for contrast.
  found: { Icon: CheckCircle2, color: 'text-[var(--success-text)]', label: '已找到字幕' },
  // `--error-text`, not `--error`, for the same reason `searching` moved to the
  // accent twin below: an icon only owes WCAG 1.4.11's 3:1, and #c0392b does not
  // clear even that — 2.57:1 on `--bg-tertiary`, 2.42:1 on an error-tinted row.
  not_found: { Icon: XCircle, color: 'text-[var(--error-text)]', label: '找不到字幕' },
  // Re-tinted --warning → --accent-text per sub-1-7a AC #5: `searching` IS an
  // in-progress state, and two colours for one meaning next to the three spinners
  // below would read as a distinction that does not exist.
  searching: { Icon: Loader2, color: 'text-[var(--accent-text)]', label: '字幕搜尋中', spin: true },
  not_searched: { Icon: Minus, color: 'text-[var(--text-muted)]', label: '尚未搜尋字幕' },
  probing: { Icon: Loader2, color: 'text-[var(--accent-text)]', label: '偵測字幕軌中', spin: true },
  extracting: {
    Icon: Loader2,
    color: 'text-[var(--accent-text)]',
    label: '抽取內嵌字幕中',
    spin: true,
  },
  translating: {
    Icon: Loader2,
    color: 'text-[var(--accent-text)]',
    label: '翻譯字幕中',
    spin: true,
  },
  // Terminal: the pipeline will not produce a subtitle for this file automatically.
  // Distinct from not_found (缺字幕) by recovery — only P2 ASR can change this one.
  no_text_source: {
    Icon: XCircle,
    color: 'text-[var(--text-muted)]',
    label: '無可用的文字字幕軌',
  },
  // Terminal and DELIBERATE: the track's language tag did not qualify (P0 — `und`
  // is never treated as English). Muted, not error-red: nothing went wrong.
  //
  // `CircleSlash`, not `Minus` (Sally's ruling 2026-08-04). A bare `Minus` put a
  // SETTLED verdict into the not-settled-yet family, which is why it rendered
  // identically to `not_searched`. The slash reads "excluded by rule", which is
  // exactly what happened here.
  skipped: {
    Icon: CircleSlash,
    color: 'text-[var(--text-muted)]',
    label: '已略過（字幕軌語言不符）',
  },
  // Terminal (10th value, sub-1-2 [@contract-v2] via sub-2-2a): a generated
  // ENGLISH subtitle exists but the expected translation step did not run.
  // Unreachable for episodes until series generation (9R-10a) lands — added
  // NOW (sub-2-2b CR M1) so that story does not inherit a silent fallback to
  // `not_searched`'s 「尚未搜尋字幕」, which mislabels a settled verdict as
  // not-started (the exact skipped-vs-not_searched class Sally ruled on
  // 2026-08-04). Grammar: CIRCLED (settled outcome) + muted (nothing broke —
  // the recovery is setting the translation key, said by the label, and the
  // poster badge ruling made 未翻譯 neutral-not-error). CircleDashed glyph is
  // PROVISIONAL pending γ (sub-2-2c) ratification — the label is the contract.
  untranslated: {
    Icon: CircleDashed,
    color: 'text-[var(--text-muted)]',
    label: '已生成英文字幕，尚未翻譯',
  },
};

// ─── J11-D: "does this episode have Chinese subtitles" (badge-b) ───────────
//
// When the API sends chinese_subtitle (badge-a), the mark answers THAT
// question — the one the user opens the season for — instead of Vido's own
// search record. Same J2 grammar: circled = settled, bare = not yet; colour
// = does the user need to act. In-flight pipeline states still win: while a
// subtitle is being made, "it is being made" is the news.
//
// `zh` (Chinese, script untold — after badge-c only bitmap tracks stay here)
// reads like zh_hant: there IS Chinese to watch (Alexyu 1A, 2026-10-07).
type Mark = {
  Icon: typeof CheckCircle2;
  color: string;
  /** Line 1: the verdict. */
  title: string;
  /** Line 2: where it comes from, or what to do next. */
  detail: string;
  spin?: boolean;
};

const IN_FLIGHT = new Set(['searching', 'probing', 'extracting', 'translating']);

const SOURCE_LABEL: Record<ChineseSubtitleSource['kind'], (s: ChineseSubtitleSource) => string> = {
  embedded: () => '片內字幕',
  sidecar: (s) => (s.label ? `旁邊的 ${s.label} 檔` : '旁邊的字幕檔'),
  vido: () => 'Vido 生成',
};

/** "片內字幕・旁邊的 zh-TW 檔・Vido 生成" — each source once, in API order. */
function describeSources(sources: ChineseSubtitleSource[] | undefined): string {
  const seen = new Set<string>();
  for (const s of sources ?? []) seen.add(SOURCE_LABEL[s.kind](s));
  return [...seen].join('・');
}

function chineseMark(episode: MergedEpisode): Mark | null {
  const verdict = episode.chineseSubtitle;
  if (!verdict) return null;
  if (IN_FLIGHT.has(episode.subtitleStatus ?? '')) {
    return {
      Icon: Loader2,
      color: 'text-[var(--accent-text)]',
      title: '字幕生成中',
      detail: '打開「管理字幕」看進度',
      spin: true,
    };
  }
  // Alexyu 2A: before the tracks inside the file are read, a verdict can still
  // move — say so rather than change the rule.
  const stillReading = episode.embeddedSubtitlesRead === false;
  const sources = describeSources(episode.chineseSubtitleSources);
  switch (verdict) {
    case 'zh_hant':
      return {
        Icon: CheckCircle2,
        color: 'text-[var(--success-text)]',
        title: '有繁中字幕',
        detail: sources,
      };
    case 'zh':
      return {
        Icon: CheckCircle2,
        color: 'text-[var(--success-text)]',
        title: '有中文字幕（繁簡未知）',
        detail: sources,
      };
    case 'zh_hans':
      return {
        Icon: AlertCircle,
        color: 'text-[var(--error-text)]',
        title: '只有簡中字幕',
        detail: '可以在「管理字幕」轉成繁中',
      };
    case 'none':
      return {
        Icon: XCircle,
        color: 'text-[var(--error-text)]',
        title: '缺中文字幕',
        detail: stillReading ? '還在讀片內字幕，讀完可能會變' : '只有英文等其他語言——可以生成',
      };
    case 'unknown':
      return {
        Icon: Minus,
        color: 'text-[var(--text-muted)]',
        title: '還沒檢查這一集的字幕',
        detail: '下次掃描媒體庫時會讀',
      };
  }
  // A verdict this client does not know yet (a newer backend): fall back to
  // Vido's own record rather than claim "not checked" (CR L4).
  return null;
}

/** Pre-badge-a backends send no chinese_subtitle: fall back to Vido's own
 *  record (J2-D), with its long-form label as the only line. */
function statusMark(episode: MergedEpisode): Mark {
  const status = episode.subtitleStatus ?? 'not_searched';
  const meta = SUBTITLE_STATUS[status] ?? SUBTITLE_STATUS.not_searched;
  return { Icon: meta.Icon, color: meta.color, title: meta.label, detail: '', spin: meta.spin };
}

/**
 * Subtitle indicator — hidden entirely when no local file exists (AC #6).
 *
 * An icon with no visible text, so its explanation lives in a Tooltip (J11-D
 * rules 3–4, 6): instant on hover, on Tab focus, on a tap; Esc or a press
 * elsewhere closes it. The trigger is a real button so keyboards reach it; it
 * does nothing else — 管理字幕 stays its own action. The accessible name is
 * the same sentence the tooltip shows. No `title` (rule 6).
 */
function SubtitleStatusIcon({ episode, code }: { episode: MergedEpisode; code: string }) {
  if (!episode.hasLocalFile) return null;

  const mark = chineseMark(episode) ?? statusMark(episode);
  const { Icon } = mark;
  const sentence = mark.detail ? `${mark.title}，${mark.detail}` : mark.title;

  return (
    <Tooltip
      side="right"
      delay={0}
      openOnPress
      content={
        <span className="flex flex-col gap-0.5 text-left" data-testid="episode-subtitle-tooltip">
          <span className="font-semibold">{mark.title}</span>
          {mark.detail && (
            <span className="font-normal text-[var(--text-secondary)]">{mark.detail}</span>
          )}
        </span>
      }
    >
      <button
        type="button"
        aria-label={sentence}
        data-testid="episode-subtitle-indicator"
        data-episode={code}
        className={cn(
          '-m-1 inline-flex shrink-0 items-center rounded-[var(--radius-sm)] p-1 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]',
          mark.color
        )}
      >
        <Icon className={cn('h-4 w-4', mark.spin && 'animate-spin')} aria-hidden="true" />
      </button>
    </Tooltip>
  );
}

function formatRuntime(runtime?: number): string | null {
  if (!runtime || runtime <= 0) return null;
  // 分, not 分鐘: B4p-D writes 「24 分」 and the movie meta row says 「107 分」 (dsr-2).
  return `${runtime} 分`;
}

/** Loading skeleton shown while a season's episodes are being fetched (AC #4, Task 7.5). */
function EpisodeListSkeleton() {
  return (
    <ul className="divide-y divide-[var(--border-subtle)]" data-testid="episode-list-skeleton">
      {[0, 1, 2, 3].map((i) => (
        <li key={i} className="flex items-center gap-3 px-4 py-3" aria-hidden="true">
          <div className="h-4 w-14 animate-pulse rounded bg-[var(--bg-secondary)]" />
          <div className="h-4 flex-1 animate-pulse rounded bg-[var(--bg-secondary)]" />
          <div className="h-4 w-10 animate-pulse rounded bg-[var(--bg-secondary)]" />
        </li>
      ))}
    </ul>
  );
}

/** Retry-able error state shown when the TMDb season fetch fails (AC #7, Task 7.6). */
function EpisodeListError({ onRetry }: { onRetry?: () => void }) {
  return (
    <div
      role="alert"
      className="flex flex-col items-center gap-3 px-4 py-6 text-center"
      data-testid="episode-list-error"
    >
      <p className="text-sm text-[var(--text-secondary)]">無法載入劇集列表，請稍後再試。</p>
      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="rounded-md border border-[var(--border-subtle)] px-3 py-1.5 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-secondary)]"
        >
          重試
        </button>
      )}
    </div>
  );
}

export function EpisodeList({
  episodes,
  seasonNumber,
  isLoading,
  isError,
  onRetry,
  onManageSubtitle,
}: EpisodeListProps) {
  if (isLoading) return <EpisodeListSkeleton />;
  if (isError) return <EpisodeListError onRetry={onRetry} />;

  if (episodes.length === 0) {
    return (
      <p className="px-4 py-6 text-center text-sm text-[var(--text-secondary)]">
        此季沒有劇集資料。
      </p>
    );
  }

  return (
    <ul className="divide-y divide-[var(--border-subtle)]" data-testid="episode-list">
      {episodes.map((ep) => {
        const runtime = formatRuntime(ep.runtime);
        return (
          <li
            key={ep.episodeNumber}
            className="flex flex-col gap-1 px-4 py-3 sm:flex-row sm:items-center sm:gap-4"
            data-testid="episode-row"
          >
            {/* Title line: code + title + subtitle status */}
            <div className="flex min-w-0 flex-1 items-center gap-2">
              <span className="shrink-0 font-mono text-xs text-[var(--text-muted)]">
                {episodeCode(seasonNumber, ep.episodeNumber)}
              </span>
              <span className="truncate text-sm font-medium text-[var(--text-primary)]">
                {ep.name || `第 ${ep.episodeNumber} 集`}
              </span>
              <SubtitleStatusIcon episode={ep} code={episodeCode(seasonNumber, ep.episodeNumber)} />
            </div>

            {/* 9R-10c: the per-episode subtitle entry. Rendered for EVERY
                subtitle_status (J3-D ruling) but ONLY when the episode has a
                local file — a TMDb episode the NAS does not hold is not
                addressable and must not offer an action.
                Hover treatment is this story's own: the design draws no hover
                for row actions anywhere, so it is flagged for Sally to ratify. */}
            {onManageSubtitle && canManageEpisodeSubtitle(ep) && (
              <button
                type="button"
                onClick={() => onManageSubtitle(ep)}
                data-testid="episode-manage-subtitle"
                aria-label={`管理 ${episodeCode(seasonNumber, ep.episodeNumber)} 的字幕`}
                className="flex min-h-[44px] shrink-0 items-center self-start rounded-[var(--radius-md)] px-3 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-secondary)] hover:text-[var(--text-primary)] sm:self-auto"
              >
                管理字幕
              </button>
            )}

            {/* Metadata: air date + runtime (below on mobile, inline on desktop) */}
            <div className="flex shrink-0 items-center gap-3 text-xs text-[var(--text-secondary)]">
              {ep.airDate && <span>{ep.airDate}</span>}
              {runtime && <span>{runtime}</span>}
            </div>
          </li>
        );
      })}
    </ul>
  );
}

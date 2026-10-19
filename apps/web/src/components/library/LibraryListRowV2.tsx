// Design ref: ux-design.pen Screen A4p-D (b1H71g) · A4p-M (EkB8g)
/**
 * One row of the library list — the 修字幕清單 (⚖️ Alexyu 2026-10-19 1A:
 * disc-2026-10-library-list-triage-b1). The subtitle state sits right after the
 * title, at the same x on every row, and is never hidden at any width.
 *
 * One DOM, three layouts, picked by the LIST's width (`@container` on
 * `LibraryListV2`, columns in `libraryListCols.ts`):
 * - base (< 576, A4p-M): thumb 40×60 · [title / meta / 字幕＋★] · ›. The third line
 *   is a flex row.
 * - `@xl` / `@2xl` (A4p-D): the third line turns `display: contents`, so its two
 *   cells become grid items and land in the 字幕 and 評分 columns; 規格 appears at
 *   `@2xl`.
 * The lifecycle pill (失敗 / 整理中) is drawn twice for that reason — after the
 * title in the table, first on the third line in the three-line row — and CSS shows
 * one. Both copies are inside the link, whose `aria-label` is the row's whole name.
 *
 * Failed match (⚖️ Sally 7): B10p's sentence + 失敗 instead of the raw title, the
 * file name middle-truncated underneath, a film-strip thumb. The row stays ONE link
 * into the detail page, where 手動選片 lives — no second target in the row.
 */
import { Link } from '@tanstack/react-router';
import { Check, ChevronRight, Film, Star } from 'lucide-react';
import { getImageUrl } from '../../lib/image';
import { deriveSubtitleStatus, deriveLifecycleStatus } from '../../utils/libraryStatus';
import { filenameToGradient } from '../../utils/filenameToGradient';
import { fallbackInitial } from '../../utils/fallbackInitial';
import { noMetadataHeading } from '../../utils/noMetadataHeading';
import { fileNameOf, formatSpecLine, splitFileNameTail } from '../../utils/libraryListRow';
import { Tooltip } from '../ui/Tooltip';
import { LIBRARY_LIST_COLS, LIBRARY_LIST_GAP } from './libraryListCols';
import type { LibraryMovie, LibrarySeries } from '../../types/library';

/** What the 中文 badge's tooltip (and the row's spoken name) says — ⚖️ D2 keeps the badge itself. */
export const ZH_UNMARKED_HINT = '字幕軌沒標示繁簡';

export interface LibraryListRowData {
  id: string;
  type: 'movie' | 'tv';
  title: string;
  posterPath?: string | null;
  /** 年份 · 片長／季數 · 類型 — already joined. */
  meta: string;
  voteAverage?: number;
  media: LibraryMovie | LibrarySeries;
}

interface LibraryListRowV2Props extends LibraryListRowData {
  /** ux3-cutover-2: selection mode — row toggles instead of navigating. */
  selectable?: boolean;
  selected?: boolean;
  onSelect?: (e: React.MouseEvent) => void;
}

interface AriaLabelParts {
  heading: string;
  failed: boolean;
  fileName: string;
  pending: boolean;
  meta: string;
  subtitleLabel: string | null;
  /** The 中文 verdict (script not marked) — gets the hint, keyed on the verdict, not the wording. */
  zhUnmarked: boolean;
  rating: string | null;
  spec: string | null;
}

/** The row's spoken name, in column order (AC #9). Missing parts are left out, not read as blanks. */
export function rowAriaLabel(p: AriaLabelParts): string {
  const parts = [p.heading];
  if (p.failed) {
    parts.push('比對失敗', `檔名 ${p.fileName}`);
  } else {
    if (p.pending) parts.push('整理中');
    if (p.meta) parts.push(p.meta);
  }
  const subtitle =
    p.subtitleLabel === null
      ? '不知道'
      : p.zhUnmarked
        ? `${p.subtitleLabel}（${ZH_UNMARKED_HINT}）`
        : p.subtitleLabel;
  parts.push(`字幕：${subtitle}`);
  if (p.rating) parts.push(`評分 ${p.rating}`);
  if (p.spec) parts.push(p.spec);
  return parts.join('，');
}

const PILL = 'shrink-0 items-center rounded-full px-1.5 py-0.5 text-xs font-semibold';

export function LibraryListRowV2({
  id,
  type,
  title,
  posterPath,
  meta,
  voteAverage,
  media,
  selectable,
  selected,
  onSelect,
}: LibraryListRowV2Props) {
  const failed = media.parseStatus === 'failed';
  const lifecycle = deriveLifecycleStatus(media);
  // 已入庫 is the steady state and never shows; 失敗 / 整理中 do.
  const pill = lifecycle && !lifecycle.steadyState ? lifecycle : null;
  const subtitle = deriveSubtitleStatus(media);
  const zhUnmarked = !!subtitle && media.chineseSubtitle === 'zh';
  const rating = typeof voteAverage === 'number' && voteAverage > 0 ? voteAverage.toFixed(1) : null;
  const spec = formatSpecLine(media);
  const heading = failed ? noMetadataHeading(type) : title;
  const fileName = fileNameOf(media.filePath, title);
  const img = failed ? null : getImageUrl(posterPath ?? null, 'w185');

  const ariaLabel = rowAriaLabel({
    heading,
    failed,
    fileName,
    pending: media.parseStatus === 'pending',
    meta,
    subtitleLabel: subtitle?.label ?? null,
    zhUnmarked,
    rating,
    spec,
  });

  const pillEl = (className: string) =>
    pill && (
      <span aria-hidden="true" className={`${PILL} ${className} ${pill.className}`}>
        {pill.label}
      </span>
    );

  const subtitleBadge = subtitle ? (
    <span
      className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ${subtitle.className}`}
    >
      {subtitle.label}
    </span>
  ) : (
    // Unknown (or still being worked on — the Activity hub owns progress) is said
    // out loud: a blank column reads as "nothing wrong" (⚖️ Sally 4).
    <span className="text-xs text-[var(--text-muted)]">不知道</span>
  );

  return (
    <li>
      <Link
        to="/media/$type/$id"
        params={{ type, id }}
        data-testid={`list-row-v2-${id}`}
        aria-label={ariaLabel}
        aria-pressed={selectable ? selected : undefined}
        onClick={
          selectable
            ? (e) => {
                e.preventDefault();
                onSelect?.(e);
              }
            : undefined
        }
        // The list clips to its rounded frame, so the global 2px-outset focus ring
        // would be cut off — this row draws it inside its own edge.
        // Hover tint only on an unselected row: a selected row keeps its tint under the
        // mouse (the hover utility is emitted later and would otherwise win).
        className={`flex items-center gap-3 px-3 py-2.5 transition-colors focus-visible:outline-offset-[-2px] @xl:gap-4 ${
          selected ? 'bg-[var(--accent-subtle)]' : 'hover:bg-[var(--bg-secondary)]'
        }`}
      >
        {selectable && (
          <span
            data-testid={`row-select-indicator-${id}`}
            aria-hidden="true"
            className={`flex h-5 w-5 shrink-0 items-center justify-center rounded-full border-2 transition-colors ${
              selected
                ? 'border-[var(--accent-primary)] bg-[var(--accent-primary)] text-[var(--text-on-accent)]'
                : 'border-[var(--border-subtle)] text-transparent'
            }`}
          >
            <Check className="h-3.5 w-3.5" />
          </span>
        )}

        <div
          className={`grid min-w-0 flex-1 items-center gap-y-0.5 ${LIBRARY_LIST_COLS} ${LIBRARY_LIST_GAP}`}
        >
          <Thumb
            failed={failed}
            img={img}
            title={title}
            className="col-start-1 row-span-2 row-start-1 @xl:row-span-1 @xl:row-start-1"
          />

          <div className="col-start-2 row-start-1 flex min-w-0 flex-col gap-0.5 self-end @xl:gap-1 @xl:self-center">
            <div className="flex min-w-0 items-center gap-2">
              <span className="truncate text-sm font-semibold text-[var(--text-primary)] @xl:text-base">
                {heading}
              </span>
              {pillEl('hidden @xl:inline-flex')}
            </div>
            {failed ? (
              <FileName name={fileName} />
            ) : (
              meta && (
                <span className="truncate text-xs text-[var(--text-secondary)] @xl:text-sm">
                  {meta}
                </span>
              )
            )}
          </div>

          {/* Third line on a phone; `contents` in the table, so the two cells below
              are placed in the 字幕 / 評分 columns. */}
          <div className="col-start-2 row-start-2 flex min-w-0 items-center gap-2 self-start @xl:contents">
            {pillEl('inline-flex @xl:hidden')}
            <span
              data-testid={`list-row-subtitle-${id}`}
              className="flex shrink-0 items-center @xl:col-start-3 @xl:row-start-1"
            >
              {zhUnmarked ? (
                <Tooltip content={ZH_UNMARKED_HINT} side="top">
                  {subtitleBadge}
                </Tooltip>
              ) : (
                subtitleBadge
              )}
            </span>
            {rating && (
              <span className="flex shrink-0 items-center gap-1 font-mono text-xs text-[var(--text-secondary)] @xl:col-start-4 @xl:row-start-1">
                <Star
                  aria-hidden="true"
                  className="h-3 w-3 fill-[var(--warning)] text-[var(--warning-text)]"
                />
                {rating}
              </span>
            )}
          </div>

          <span
            data-testid={`list-row-spec-${id}`}
            title={spec ?? undefined}
            className={`hidden min-w-0 truncate font-mono text-xs @2xl:col-start-5 @2xl:row-start-1 @2xl:block ${
              spec ? 'text-[var(--text-secondary)]' : 'text-[var(--text-muted)]'
            }`}
          >
            {spec ?? '—'}
          </span>

          <ChevronRight
            aria-hidden="true"
            className="col-start-3 row-span-2 row-start-1 h-4 w-4 justify-self-center text-[var(--text-muted)] @xl:col-start-5 @xl:row-span-1 @xl:row-start-1 @2xl:col-start-6"
          />
        </div>
      </Link>
    </li>
  );
}

function Thumb({
  failed,
  img,
  title,
  className,
}: {
  failed: boolean;
  img: string | null;
  title: string;
  className: string;
}) {
  const frame = `flex h-[60px] w-10 items-center justify-center overflow-hidden rounded-[var(--radius-sm)] @xl:h-[72px] @xl:w-12 @xl:rounded-[var(--radius-md)] ${className}`;
  if (failed) {
    // PosterCard-v2/Unmatched's poster-fallback: a plain tile, a film strip.
    return (
      <div aria-hidden="true" className={`${frame} bg-[var(--bg-tertiary)]`}>
        <Film className="h-5 w-5 text-[var(--text-muted)]" />
      </div>
    );
  }
  if (img) {
    return (
      <div aria-hidden="true" className={`${frame} bg-[var(--bg-tertiary)]`}>
        <img src={img} alt="" loading="lazy" className="h-full w-full object-cover" />
      </div>
    );
  }
  // Matched but no poster: the same hash gradient + initial the poster card uses (AC #8).
  // --text-on-scrim, not --text-primary: the tile does not invert in 日巡 (see PosterCardV2).
  const [from, to] = filenameToGradient(title);
  return (
    <div
      aria-hidden="true"
      data-testid="list-row-thumb-fallback"
      className={`${frame} text-base font-bold text-[var(--text-on-scrim)]`}
      style={{ backgroundImage: `linear-gradient(135deg, ${from}, ${to})` }}
    >
      {fallbackInitial(title)}
    </div>
  );
}

/**
 * A file name cut in the MIDDLE (AC #7): the head is clipped with an ellipsis by the
 * browser at whatever width there is, the last 12 characters always show — that is
 * where the release tag and the extension are. `whitespace-pre`, not `truncate`'s
 * `nowrap`: the head usually ends in a space (「… – 」) and a block's trailing space
 * would otherwise be dropped, gluing the two halves together.
 */
function FileName({ name }: { name: string }) {
  const { head, tail } = splitFileNameTail(name);
  return (
    <span
      title={name}
      data-testid="list-row-filename"
      className="flex min-w-0 font-mono text-xs text-[var(--text-muted)]"
    >
      {head && <span className="min-w-0 overflow-hidden text-ellipsis whitespace-pre">{head}</span>}
      <span className="shrink-0 whitespace-pre">{tail}</span>
    </span>
  );
}

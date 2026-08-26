// Implements: Component/PosterCard-v2 (hD7Tw)
/**
 * v2 poster card (UX Redesign Phase 2 — UX2-2, §3.3 / §5.1 / N1).
 *
 * Tighter than the legacy card: a 2-line CJK title grid (reserves two full lines,
 * truncates with ellipsis, never clips mid-glyph — R2 fix), JetBrains Mono
 * year·meta in `text-secondary`, and ONE lifecycle/subtitle status badge on the
 * poster (§2.5). The badge is an EXCEPTION signal (ux3-0-2): a lifecycle exception
 * (整理中/失敗) wins, else a subtitle exception (缺字幕/簡中/有字幕); the happy steady
 * state (已入庫 + 繁中) and unknown states show NO badge (never errors — F3). The
 * subtitle source prefers the authoritative engine result (subtitleStatus/
 * subtitleLanguage, ux3-0-1) over embedded tracks. Links to the detail route.
 */
import { Link } from '@tanstack/react-router';
import { Check, Star } from 'lucide-react';
import { getImageUrl } from '../../lib/image';
import { filenameToGradient } from '../media/ColorPlaceholder';
import { pickPosterBadge } from '../../utils/libraryStatus';
import type { LibraryMovie, LibrarySeries } from '../../types/library';

interface PosterCardV2Props {
  id: string;
  /** Detail-route media type (series is mapped to `tv` by the caller). */
  type: 'movie' | 'tv';
  title: string;
  posterPath?: string | null;
  /** Release year (already formatted). */
  year?: string;
  /** Right-hand meta — runtime ("142 分") or season count ("3 季"). */
  meta?: string;
  voteAverage?: number;
  media: Pick<
    LibraryMovie | LibrarySeries,
    'parseStatus' | 'subtitleTracks' | 'subtitleStatus' | 'subtitleLanguage'
  >;
  /** ux3-cutover-2: selection mode — card toggles instead of navigating. */
  selectable?: boolean;
  selected?: boolean;
  onSelect?: (e: React.MouseEvent) => void;
}

/**
 * First LETTER or CJK char for the no-poster tile —「[FanSub] 未知電影」used
 * to render a giant「[」(critique R3 minor). Falls back to the raw first char
 * when the title is all symbols.
 */
function fallbackInitial(title: string): string {
  const m = title.match(/[\p{L}\p{N}]/u);
  return m ? m[0] : title.slice(0, 1) || '?';
}

export function PosterCardV2({
  id,
  type,
  title,
  posterPath,
  year,
  meta,
  voteAverage,
  media,
  selectable,
  selected,
  onSelect,
}: PosterCardV2Props) {
  const badge = pickPosterBadge(media);
  const img = getImageUrl(posterPath ?? null, 'w342');
  const [from, to] = filenameToGradient(title);
  const metaLine = [year, meta].filter(Boolean).join(' · ');

  return (
    <Link
      to="/media/$type/$id"
      params={{ type, id }}
      data-testid={`poster-v2-${id}`}
      aria-pressed={selectable ? selected : undefined}
      onClick={
        selectable
          ? (e) => {
              e.preventDefault();
              onSelect?.(e);
            }
          : undefined
      }
      className="group/card flex flex-col gap-2"
    >
      {/* ① 回應手勢, on --motion-touch and --motion-lift so reduced motion holds
          the scale at 1 while the title still turns gold — the hover is never
          silent, it just stops moving.
          NO shadow escalation on hover, deliberately: DESIGN.md's Tone-First
          Rule rations shadow to things that genuinely float (11 uses app-wide
          against 525 tone steps), and lifting every poster in a grid would spend
          that budget on the most repeated element in the app. The press state is
          the one place a card may go BELOW rest — settling back to 1 answers the
          tap on touch, where there is no hover to answer it. */}
      <div
        className={`relative aspect-[2/3] overflow-hidden rounded-[var(--radius-lg)] bg-[var(--bg-secondary)] shadow-[var(--shadow-md)] transition-transform duration-[var(--motion-touch)] ease-[var(--ease-settle)] group-hover/card:scale-[var(--motion-lift)] group-focus-visible/card:scale-[var(--motion-lift)] group-active/card:scale-100 ${
          selected ? 'ring-2 ring-[var(--accent-primary)]' : ''
        }`}
      >
        {img ? (
          <img src={img} alt={title} loading="lazy" className="h-full w-full object-cover" />
        ) : (
          <div
            // 宣紙白, not --text-on-accent: the gradient hue is title-hash-derived,
            // and dark ink over a light hash measured 2.4:1 (critique R1 P0). The
            // hash palette is lightness-clamped dark (ColorPlaceholder), so light
            // text is guaranteed ≥3:1 on every tile.
            className="flex h-full w-full items-center justify-center text-3xl font-bold text-[var(--text-primary)]"
            style={{ backgroundImage: `linear-gradient(135deg, ${from}, ${to})` }}
            aria-hidden="true"
          >
            {fallbackInitial(title)}
          </div>
        )}

        {selectable && (
          <span
            data-testid={`poster-select-indicator-${id}`}
            aria-hidden="true"
            className={`absolute left-1.5 top-1.5 flex h-6 w-6 items-center justify-center rounded-full border-2 transition-colors duration-[var(--motion-state)] ${
              selected
                ? 'border-[var(--accent-primary)] bg-[var(--accent-primary)] text-[var(--text-on-accent)]'
                : 'border-[var(--text-on-accent)] bg-[var(--overlay-scrim)] text-transparent'
            }`}
          >
            <Check className="h-4 w-4" />
          </span>
        )}

        {badge && (
          // Opaque --bg-secondary underlay: the 12% tint alone composites over
          // ARBITRARY poster art (1.58:1 on a light hash tile — critique R1 P0).
          // Over the solid underlay the composite is deterministic and the
          // *-text tokens are gate-verified on exactly that kind of surface.
          <span
            data-testid="poster-status-badge"
            className="absolute right-1.5 top-1.5 rounded-full bg-[var(--bg-secondary)]"
          >
            <span
              className={`block rounded-full px-2 py-0.5 text-[11px] font-medium ${badge.className}`}
            >
              {badge.label}
            </span>
          </span>
        )}

        {typeof voteAverage === 'number' && voteAverage > 0 && (
          // 宣紙白 on the scrim (11.9:1) — NOT --text-on-accent: that ink is cut
          // for GOLD fills and measures 1.1:1 on the 70% black scrim (critique
          // R1 P0, confirmed by two independent probes). Star inherits it too:
          // --warning is a STATUS colour (asked-but-not-happening), a decorative
          // rating star must not dilute it.
          //
          // KNOWN DEFECT, 日巡: --text-primary is ink in light, so this reads
          // 1.19–2.34:1 on the scrim. It is NOT fixable by swapping tokens —
          // --text-inverse and --text-on-accent are both ink in DARK (1.13 /
          // 1.16), so either one just moves the P0 to the default theme. The
          // scrim does not invert, so its label must not invert either, and no
          // such token exists yet. Blocked on an always-paper --text-on-scrim.
          <span className="absolute bottom-1.5 left-1.5 flex items-center gap-0.5 rounded-full bg-[var(--overlay-scrim)] px-1.5 py-0.5 font-mono text-[11px] text-[var(--text-on-scrim)]">
            <Star className="h-3 w-3 fill-current" />
            {voteAverage.toFixed(1)}
          </span>
        )}
      </div>

      {/* 2-line CJK title grid (§3.3) — reserves two lines, ellipsis on overflow */}
      <div>
        <h3
          className="line-clamp-2 min-h-[2.75em] text-sm font-medium leading-snug text-[var(--text-primary)] transition-colors duration-[var(--motion-touch)] group-hover/card:text-[var(--accent-text)]"
          title={title}
        >
          {title}
        </h3>
        {metaLine && (
          <p className="mt-0.5 truncate font-mono text-[11px] text-[var(--text-secondary)]">
            {metaLine}
          </p>
        )}
      </div>
    </Link>
  );
}

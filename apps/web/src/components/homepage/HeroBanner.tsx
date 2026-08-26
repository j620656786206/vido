// Design ref: ux-design.pen Screen H1-D-v3 (k2Otv)
// Companion frames: H7-D-v3 (EoCQ4) degraded · H2-M-v3 (uGCAU) mobile
/**
 * Own-library hero (ux3-1-8 / epic ux3-home-v3, home-v3-identity-brief §3).
 *
 * The identity flip: the homepage's largest surface now sells YOUR shelf, not
 * TMDb's. Content = the newest library items that HAVE a backdrop (max 5),
 * derived from the SAME useRecentlyAdded(RECENT_LIMIT) query the 最近新增 row
 * reads —
 * TanStack Query dedupes, so the hero costs zero extra requests.
 *
 * STATIC by ruling (⚖️ R3 autoplay + shape session 靜止＋手動): there is no
 * rotation interval, no hover/focus pause, no pause button — nothing moves
 * unless the user moves it (移除操作者的產品不該有自己會動的元件). Manual
 * switching = dots + prev/next chevrons on the scrim pill; the asymmetric
 * cross-fade (700/300) survives because it answers a user gesture.
 *
 * 例外訊號原則: no backdrops → the hero renders nothing (absent, not an empty
 * frame); exactly one item → single static slide, dots hidden. The status
 * badge speaks the poster grid's vocabulary on the grid's own precedence
 * ladder (see heroStatusBadge), with the 繁中 steady state spelled out
 * (已就緒) because on the hero the happy state IS the message.
 */
import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { ChevronLeft, ChevronRight, Star } from 'lucide-react';
import { useRecentlyAdded, RECENT_LIMIT } from '../../hooks/useLibrary';
import { getImageUrl, getBackdropSrcSet, getBackdropSizes } from '../../lib/image';
import { deriveLifecycleStatus, deriveSubtitleStatus } from '../../utils/libraryStatus';
import { cn } from '../../lib/utils';
import type { LibraryItem, LibraryMovie, LibrarySeries } from '../../types/library';

const HERO_MAX_ITEMS = 5;

export interface HeroItem {
  id: string;
  type: 'movie' | 'tv';
  title: string;
  backdropPath: string;
  year: string | null;
  voteAverage?: number;
  media: LibraryMovie | LibrarySeries;
}

/** Stable identity for one hero slide — the React key AND the selection key. */
function heroKey(item: HeroItem): string {
  return `${item.type}-${item.id}`;
}

/** Newest own-library items that can actually dress a hero (backdrop present). */
export function toHeroItems(items: LibraryItem[]): HeroItem[] {
  const result: HeroItem[] = [];
  for (const item of items) {
    const media = item.type === 'movie' ? item.movie : item.series;
    if (!media?.backdropPath) continue;
    const date = item.type === 'movie' ? item.movie?.releaseDate : item.series?.firstAirDate;
    const year = date && /^\d{4}/.test(date) ? date.slice(0, 4) : null;
    result.push({
      id: media.id,
      type: item.type === 'movie' ? 'movie' : 'tv',
      title: media.title,
      backdropPath: media.backdropPath,
      year,
      voteAverage: media.voteAverage,
      media,
    });
    if (result.length >= HERO_MAX_ITEMS) break;
  }
  return result;
}

/**
 * The hero's status badge. Same ladder as the grid's `pickPosterBadge` — a
 * LIFECYCLE exception (整理中 / 失敗) outranks anything the subtitle layer has
 * to say — because the same item is on screen twice: here, and in the 最近新增
 * row six inches below. 固定詞彙 forbids one screen dressing one truth in two
 * colours, and reading only the subtitle verdict would print a green
 * 「繁中字幕 ✓ 已就緒」 on an item whose poster is simultaneously wearing amber
 * 整理中.
 *
 * Where the hero DIVERGES from the grid, deliberately: the grid is an
 * exception-only signal and suppresses the happy steady state, while the hero
 * spells it out (繁中字幕 ✓ 已就緒) — on the page's largest surface the happy
 * state IS the message this product sells.
 */
function heroStatusBadge(media: LibraryMovie | LibrarySeries) {
  const lifecycle = deriveLifecycleStatus(media);
  if (lifecycle && !lifecycle.steadyState) {
    // A lifecycle exception is the app's own in-flight work — the Activity hub
    // owns it, not the item's page, so this badge stays a plain statement.
    return { label: lifecycle.label, className: lifecycle.className, actionable: false };
  }
  const subtitle = deriveSubtitleStatus(media);
  if (!subtitle) return null;
  const label = subtitle.label === '繁中' ? '繁中字幕 ✓ 已就緒' : subtitle.label;
  // brief §3: 「繁中✓已就緒／缺字幕→門」. A named problem with no way to act on
  // it is the dead-end pattern this redesign keeps deleting, so the states the
  // user CAN resolve (on the item's own page) become links; the happy state and
  // the "nothing to work with" states stay statements.
  const actionable = ACTIONABLE_SUBTITLE_LABELS.has(subtitle.label);
  return { label, className: subtitle.className, actionable };
}

/** Subtitle verdicts the user can act on from the item's detail page. */
const ACTIONABLE_SUBTITLE_LABELS = new Set(['缺字幕', '未翻譯', '簡中']);

function HeroSlide({ item, active }: { item: HeroItem; active: boolean }) {
  // w780 src is the safe baseline; srcset upgrades to w1280/original on wide
  // viewports so handsets never pay a 3–5MB image.
  const fallbackBackdrop = getImageUrl(item.backdropPath, 'w780');
  const backdropSrcSet = getBackdropSrcSet(item.backdropPath);
  const backdropSizes = getBackdropSizes();
  const badge = heroStatusBadge(item.media);
  const [imageBroken, setImageBroken] = useState(false);

  return (
    // The container carries NO interaction (critique R1 P1 lineage): the title
    // <Link> stretches over the whole slide (after:inset-0) so full-surface
    // click is a NATIVE anchor; the CTA sits above it on z-10. `inert` removes
    // inactive slides from focus order + the a11y tree.
    <div
      data-testid="hero-banner-slide"
      data-active={active ? 'true' : 'false'}
      inert={!active}
      // Outgoing slide fades FASTER than the incoming one (300 vs 700ms) so
      // two title layers never sit half-mixed (critique R3 P3). The fade now
      // only ever answers a user gesture — the hero itself never moves.
      className={cn(
        'absolute inset-0 ease-in-out',
        active
          ? 'opacity-100 transition-opacity duration-700'
          : 'pointer-events-none opacity-0 transition-opacity duration-300'
      )}
    >
      {fallbackBackdrop && !imageBroken && (
        <img
          src={fallbackBackdrop}
          srcSet={backdropSrcSet ?? undefined}
          sizes={backdropSizes}
          alt={item.title}
          className="h-full w-full object-cover"
          loading={active ? 'eager' : 'lazy'}
          decoding="async"
          onError={() => setImageBroken(true)}
          data-testid="hero-banner-backdrop"
        />
      )}
      {/* Bottom-up gradient for text legibility — 夜行 ground, not raw black */}
      <div className="absolute inset-0 bg-gradient-to-t from-[var(--bg-primary)] via-[var(--bg-primary)]/70 to-transparent" />

      {/* Same gutter recipe as every sibling section (critique R2 P2). */}
      <div className="absolute inset-x-0 bottom-0 pb-12 sm:pb-16 lg:pb-20">
        <div className="mx-auto max-w-7xl px-4 sm:px-6">
          {/* The eyebrow sits at the TOP of the content block, where the
              bottom-up gradient is thinnest — bare --text-secondary measured
              1.10:1 over a bright still. The scrim pill gives it a
              deterministic floor instead of trusting the photograph (the same
              Shapes-amendment treatment the dot pill and the poster overlay
              badges already use). */}
          <p
            className="inline-block rounded-[var(--radius-sm)] bg-[var(--overlay-scrim)] px-2 py-0.5 text-xs font-medium uppercase tracking-wider text-[var(--text-primary)]"
            data-testid="hero-banner-eyebrow"
          >
            最新入庫
          </p>

          <p
            className="mt-2 text-2xl font-bold sm:text-3xl lg:text-4xl"
            data-testid="hero-banner-title"
          >
            <Link
              to="/media/$type/$id"
              params={{ type: item.type, id: item.id }}
              data-testid="hero-banner-title-link"
              className="text-[var(--text-primary)] after:absolute after:inset-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--bg-primary)]"
            >
              {item.title}
            </Link>
          </p>

          {/* The meta row rides its own scrim. The status tints are 12%-alpha
              tokens whose AA guarantee is measured against the --bg-* surfaces,
              NOT against arbitrary key art — composited straight onto a bright
              backdrop the success tint measured 2.95:1. Compositing the whole
              row over a known floor keeps every token's measured contrast true
              here, instead of re-deriving a hero-only palette. */}
          <div className="mt-3 inline-flex max-w-full flex-wrap items-center gap-3 rounded-[var(--radius-md)] bg-[var(--overlay-scrim)] px-2 py-1 text-sm text-[var(--text-primary)]">
            {badge &&
              (badge.actionable ? (
                // relative z-10 keeps it above the title's stretched anchor so
                // the badge is its own target; after:-inset-y grows the touch
                // area to the 44px floor without fattening the pill.
                <Link
                  to="/media/$type/$id"
                  params={{ type: item.type, id: item.id }}
                  data-testid="hero-banner-subtitle-badge"
                  aria-label={`${badge.label}，前往 ${item.title} 的詳情頁處理`}
                  className={cn(
                    'relative z-10 rounded-[var(--radius-sm)] px-2 py-0.5 text-xs font-medium underline underline-offset-2 after:absolute after:-inset-x-1 after:-inset-y-3 after:content-[""] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]',
                    badge.className
                  )}
                >
                  {badge.label}
                </Link>
              ) : (
                <span
                  data-testid="hero-banner-subtitle-badge"
                  className={cn(
                    'rounded-[var(--radius-sm)] px-2 py-0.5 text-xs font-medium',
                    badge.className
                  )}
                >
                  {badge.label}
                </span>
              ))}
            {item.year && <span data-testid="hero-banner-year">{item.year}</span>}
            <span data-testid="hero-banner-type">{item.type === 'movie' ? '電影' : '影集'}</span>
            {item.voteAverage != null && item.voteAverage > 0 && (
              <span className="flex items-center gap-1" data-testid="hero-banner-rating">
                <Star className="h-4 w-4 fill-current" />
                {item.voteAverage.toFixed(1)}
              </span>
            )}
          </div>

          <div className="mt-5 flex flex-wrap items-center gap-3">
            <Link
              to="/media/$type/$id"
              params={{ type: item.type, id: item.id }}
              data-testid="hero-banner-detail-link"
              className="relative z-10 flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-5 py-2 text-sm font-semibold text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-hover)]"
            >
              查看詳情
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}

export function HeroBanner() {
  const { data, isLoading } = useRecentlyAdded(RECENT_LIMIT);
  // Selection is keyed by IDENTITY, not by position. useRecentlyAdded polls
  // every 30s, and a scan finishing mid-read prepends a new item — with an
  // index the slide the user deliberately switched to would silently become a
  // different movie. That is precisely the self-moving hero this story exists
  // to remove; a static hero that quietly re-points is the same defect wearing
  // a different mechanism.
  const [selectedKey, setSelectedKey] = useState<string | null>(null);

  const items = toHeroItems(data ?? []);
  const hasItems = items.length > 0;
  // Derived DURING render, never corrected afterwards in an effect: an
  // effect-based reset would paint one frame in which NO slide is active (the
  // hero would blink to bare gradient). A selection whose item has left the
  // list falls back to the newest — index 0 — which is also the initial state.
  const selectedPosition = selectedKey ? items.findIndex((i) => heroKey(i) === selectedKey) : -1;
  const activeIndex = selectedPosition >= 0 ? selectedPosition : 0;

  if (isLoading) {
    return (
      <section
        data-testid="hero-banner-skeleton"
        aria-busy="true"
        className="relative h-[250px] w-full overflow-hidden bg-[var(--bg-secondary)] md:h-[400px]"
      >
        <div className="h-full w-full animate-pulse bg-[var(--bg-tertiary)] motion-reduce:animate-none" />
      </section>
    );
  }

  // 例外訊號原則: no own item can dress the hero → the section is absent, and
  // the page below stays complete (the 最近新增 row still shows fresh items
  // without backdrops). isError shares this path via data === undefined.
  if (!hasItems) {
    return null;
  }

  const goTo = (index: number) => {
    const target = items[(index + items.length) % items.length];
    if (target) setSelectedKey(heroKey(target));
  };

  return (
    <section
      data-testid="hero-banner"
      aria-label="最新入庫精選"
      // Fixed heights (mobile 250 / desktop 400) per the design tokens —
      // predictable layout across device classes.
      className="relative h-[250px] w-full overflow-hidden bg-[var(--bg-primary)] md:h-[400px]"
    >
      {items.map((item, idx) => (
        <HeroSlide key={heroKey(item)} item={item} active={idx === activeIndex} />
      ))}

      {/* Switching slides replaces the title, badge, meta AND both link
          destinations while focus stays on the button that did it — with the
          inactive slides `inert`, a screen reader would otherwise hear nothing
          at all. One polite live region names what the hero now shows; it is
          only ever driven by a user gesture, so it can never chatter. */}
      {items.length > 1 && (
        <span aria-live="polite" className="sr-only" data-testid="hero-banner-live">
          {items[activeIndex]?.title}
        </span>
      )}

      {items.length > 1 && (
        // Manual switching only (⚖️ 靜止＋手動): chevrons + dots on the scrim
        // pill (Shapes amendment: lawful for overlay micro-elements; bare dots
        // over arbitrary stills measured 1.87:1 — critique R3 P2). Every
        // control keeps the ≥44px touch floor; the visible dot is a decorative
        // span INSIDE its button.
        <div
          className="absolute bottom-1 left-1/2 z-10 flex -translate-x-1/2 items-center rounded-full bg-[var(--overlay-scrim)] px-1"
          data-testid="hero-banner-dots"
        >
          <button
            type="button"
            aria-label="上一部"
            data-testid="hero-banner-prev"
            onClick={() => goTo(activeIndex - 1)}
            className="flex h-11 w-11 items-center justify-center text-[var(--text-primary)]/80 transition-colors hover:text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]"
          >
            <ChevronLeft className="h-4 w-4" />
          </button>
          {items.map((item, idx) => (
            <button
              key={heroKey(item) + '-dot'}
              type="button"
              aria-label={`切換到第 ${idx + 1} 部`}
              aria-current={idx === activeIndex}
              data-testid={`hero-banner-dot-${idx}`}
              onClick={() => goTo(idx)}
              className="flex h-11 min-w-[24px] items-center justify-center px-1 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]"
            >
              <span
                aria-hidden="true"
                className={cn(
                  'h-2 rounded-full transition-all',
                  idx === activeIndex
                    ? 'w-8 bg-[var(--text-primary)]'
                    : 'w-2 bg-[var(--text-primary)]/70'
                )}
              />
            </button>
          ))}
          <button
            type="button"
            aria-label="下一部"
            data-testid="hero-banner-next"
            onClick={() => goTo(activeIndex + 1)}
            className="flex h-11 w-11 items-center justify-center text-[var(--text-primary)]/80 transition-colors hover:text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]"
          >
            <ChevronRight className="h-4 w-4" />
          </button>
        </div>
      )}
    </section>
  );
}

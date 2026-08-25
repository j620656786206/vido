// Design ref: ux-design.pen Screen HP-1 Homepage Desktop (sAaCR)
import { useEffect, useRef, useState } from 'react';
import { Link } from '@tanstack/react-router';
import { Pause, Play, Star } from 'lucide-react';
import { useTrendingHero } from '../../hooks/useTrending';
import { getImageUrl, getBackdropSrcSet, getBackdropSizes } from '../../lib/image';
import { cn } from '../../lib/utils';
import { TrailerModal } from './TrailerModal';
import type { HeroBannerItem } from '../../types/tmdb';

const ROTATION_INTERVAL_MS = 8000;

function getYear(releaseDate: string | undefined): string | null {
  if (!releaseDate) return null;
  const year = releaseDate.slice(0, 4);
  return /^\d{4}$/.test(year) ? year : null;
}

interface HeroBannerSlideProps {
  item: HeroBannerItem;
  active: boolean;
  onPlayClick: (item: HeroBannerItem) => void;
}

function HeroBannerSlide({ item, active, onPlayClick }: HeroBannerSlideProps) {
  // w1280 src is the safe baseline; srcset upgrades to original for desktop and
  // downgrades to w780 for mobile so we don't push 3–5MB images at handsets.
  const fallbackBackdrop = getImageUrl(item.backdropPath, 'w1280');
  const backdropSrcSet = getBackdropSrcSet(item.backdropPath);
  const backdropSizes = getBackdropSizes();
  const year = getYear(item.releaseDate);
  const [imageBroken, setImageBroken] = useState(false);

  return (
    // M3 fix: whole slide is now a navigable surface. M1 fix: inert removes
    // inactive slides from a11y tree + tab order without ad-hoc tabIndex hacks.
    // Critique R1 P1: the slide used to be role="link" WRAPPING a button and a
    // Link — ARIA forbids interactive descendants inside a link, so SR users
    // heard a link with two buttons trapped inside. Now the container carries
    // NO interaction at all: the title <Link> stretches over the whole slide
    // (after:inset-0), so full-surface click is a NATIVE anchor, and the two
    // CTAs sit above it on z-10.
    <div
      data-testid="hero-banner-slide"
      data-active={active ? 'true' : 'false'}
      // React 19 forwards the `inert` boolean attribute natively. When true,
      // the browser removes the subtree from focus order, hit-testing, and
      // the accessibility tree (M1 fix).
      inert={!active}
      className={cn(
        'absolute inset-0 transition-opacity duration-700 ease-in-out',
        active ? 'opacity-100' : 'pointer-events-none opacity-0'
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
      {/* Bottom-up gradient for text legibility — 夜行 ground, not raw black,
          so the hero melts into the page instead of sitting on a foreign slab */}
      <div className="absolute inset-0 bg-gradient-to-t from-[var(--bg-primary)] via-[var(--bg-primary)]/70 to-transparent" />

      <div className="absolute inset-x-0 bottom-0 px-4 pb-12 sm:px-8 sm:pb-16 lg:px-12 lg:pb-20">
        <div className="mx-auto max-w-7xl">
          <div className="flex items-center gap-3 text-sm text-[var(--text-secondary)]">
            <span className="rounded bg-[var(--overlay-scrim)] px-2 py-0.5 text-xs font-semibold uppercase tracking-wider text-[var(--text-primary)]">
              {item.mediaType === 'movie' ? '電影' : '影集'}
            </span>
            {year && <span data-testid="hero-banner-year">{year}</span>}
            {item.voteAverage > 0 && (
              <span className="flex items-center gap-1" data-testid="hero-banner-rating">
                <Star className="h-4 w-4 fill-current" />
                {item.voteAverage.toFixed(1)}
              </span>
            )}
          </div>

          {/* lg was text-5xl=48px — past the Display ceiling (36px). The title
              carries the slide's accessible link now that the container is
              non-interactive. */}
          <h2
            className="mt-3 text-2xl font-bold sm:text-3xl lg:text-4xl"
            data-testid="hero-banner-title"
          >
            <Link
              to="/media/$type/$id"
              params={{ type: item.mediaType, id: String(item.id) }}
              data-testid="hero-banner-title-link"
              className="text-[var(--text-primary)] after:absolute after:inset-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--bg-primary)]"
            >
              {item.title}
            </Link>
          </h2>

          {item.overview && (
            <p
              className="mt-3 line-clamp-2 max-w-2xl text-sm text-[var(--text-primary)] sm:line-clamp-3 sm:text-base"
              data-testid="hero-banner-overview"
            >
              {item.overview}
            </p>
          )}

          <div className="mt-5 flex flex-wrap items-center gap-3">
            <button
              type="button"
              onClick={() => onPlayClick(item)}
              data-testid="hero-banner-play-trailer"
              className="relative z-10 flex min-h-[44px] items-center gap-2 rounded-full bg-[var(--accent-primary)] px-5 py-2 text-sm font-semibold text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-hover)]"
            >
              <Play className="h-4 w-4 fill-current" />
              觀看預告片
            </button>
            <Link
              to="/media/$type/$id"
              params={{ type: item.mediaType, id: String(item.id) }}
              data-testid="hero-banner-detail-link"
              className="relative z-10 flex min-h-[44px] items-center rounded-full bg-[var(--overlay-scrim)] px-5 py-2 text-sm font-semibold text-[var(--text-primary)] backdrop-blur transition-colors hover:bg-[var(--bg-tertiary)]"
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
  const { data, isLoading, isError } = useTrendingHero('week');
  const [activeIndex, setActiveIndex] = useState(0);
  // Hover-pause and the explicit pause BUTTON are separate states: hover is a
  // convenience that must not undo the user's explicit choice (WCAG 2.2.2 —
  // before critique R1 there was NO way to stop rotation on touch devices).
  const [isPaused, setIsPaused] = useState(false);
  const [userPaused, setUserPaused] = useState(false);
  const [trailerItem, setTrailerItem] = useState<HeroBannerItem | null>(null);
  const intervalRef = useRef<number | null>(null);

  const items = data ?? [];
  const hasItems = items.length > 0;

  // Auto-rotate every ROTATION_INTERVAL_MS unless paused or only one item.
  useEffect(() => {
    if (!hasItems || isPaused || userPaused || items.length < 2 || trailerItem) return;
    intervalRef.current = window.setInterval(() => {
      setActiveIndex((prev) => (prev + 1) % items.length);
    }, ROTATION_INTERVAL_MS);
    return () => {
      if (intervalRef.current !== null) {
        window.clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [hasItems, items.length, isPaused, userPaused, trailerItem]);

  // Reset active index if data shrinks beneath it.
  useEffect(() => {
    if (activeIndex >= items.length && items.length > 0) {
      setActiveIndex(0);
    }
  }, [items.length, activeIndex]);

  // AC #5: hide section gracefully on empty/error/loading.
  if (isLoading) {
    return (
      <section
        data-testid="hero-banner-skeleton"
        aria-busy="true"
        className="relative h-[250px] w-full overflow-hidden bg-[var(--bg-secondary)] md:h-[400px]"
      >
        <div className="h-full w-full animate-pulse bg-[var(--bg-tertiary)]" />
      </section>
    );
  }

  if (isError || !hasItems) {
    return null;
  }

  return (
    <>
      <section
        data-testid="hero-banner"
        aria-label="熱門推薦輪播"
        // Story 10-5 Task 4.1 — hero uses fixed heights rather than vh so the
        // layout is predictable across device classes (mobile compact 250px,
        // desktop 400px at md+). Matches design tokens in hp1/hp2 Pencil mocks.
        className="relative h-[250px] w-full overflow-hidden bg-[var(--bg-primary)] md:h-[400px]"
        onMouseEnter={() => setIsPaused(true)}
        onMouseLeave={() => setIsPaused(false)}
      >
        {items.map((item, idx) => (
          <HeroBannerSlide
            key={item.mediaType + '-' + item.id}
            item={item}
            active={idx === activeIndex}
            onPlayClick={setTrailerItem}
          />
        ))}

        {items.length > 1 && (
          // Every control here is a ≥44px touch target (the visible dot is a
          // decorative span INSIDE the button — the 8px dot itself was the
          // whole target before critique R1). The pause button is the WCAG
          // 2.2.2 stop mechanism: hover-pause never reached touch devices.
          <div
            className="absolute bottom-1 left-1/2 z-10 flex -translate-x-1/2 items-center"
            data-testid="hero-banner-dots"
          >
            <button
              type="button"
              aria-label={userPaused ? '繼續輪播' : '暫停輪播'}
              aria-pressed={userPaused}
              data-testid="hero-banner-pause"
              onClick={() => setUserPaused((p) => !p)}
              className="flex h-11 w-11 items-center justify-center text-[var(--text-primary)]/70 transition-colors hover:text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]"
            >
              {userPaused ? (
                <Play className="h-4 w-4 fill-current" />
              ) : (
                <Pause className="h-4 w-4 fill-current" />
              )}
            </button>
            {items.map((item, idx) => (
              <button
                key={item.mediaType + '-' + item.id + '-dot'}
                type="button"
                aria-label={`切換到第 ${idx + 1} 個推薦`}
                aria-current={idx === activeIndex}
                data-testid={`hero-banner-dot-${idx}`}
                onClick={() => setActiveIndex(idx)}
                className="flex h-11 min-w-[24px] items-center justify-center px-1 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]"
              >
                <span
                  aria-hidden="true"
                  className={cn(
                    'h-2 rounded-full transition-all',
                    idx === activeIndex
                      ? 'w-8 bg-[var(--text-primary)]'
                      : 'w-2 bg-[var(--text-primary)]/50'
                  )}
                />
              </button>
            ))}
          </div>
        )}
      </section>

      {trailerItem && (
        <TrailerModal
          open={true}
          onClose={() => setTrailerItem(null)}
          mediaType={trailerItem.mediaType}
          tmdbId={trailerItem.id}
          title={trailerItem.title}
        />
      )}
    </>
  );
}

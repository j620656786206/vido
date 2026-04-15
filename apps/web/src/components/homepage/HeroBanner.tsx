import { useEffect, useRef, useState } from 'react';
import { Link } from '@tanstack/react-router';
import { Play, Star } from 'lucide-react';
import { useTrendingHero } from '../../hooks/useTrending';
import { getImageUrl } from '../../lib/image';
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
  const backdropUrl = getImageUrl(item.backdropPath, 'original');
  const year = getYear(item.releaseDate);

  return (
    <div
      data-testid="hero-banner-slide"
      data-active={active ? 'true' : 'false'}
      aria-hidden={!active}
      className={cn(
        'absolute inset-0 transition-opacity duration-700 ease-in-out',
        active ? 'opacity-100' : 'pointer-events-none opacity-0'
      )}
    >
      {backdropUrl && (
        <img
          src={backdropUrl}
          alt={item.title}
          className="h-full w-full object-cover"
          loading={active ? 'eager' : 'lazy'}
          data-testid="hero-banner-backdrop"
        />
      )}
      {/* Bottom-up gradient for text legibility */}
      <div className="absolute inset-0 bg-gradient-to-t from-black via-black/70 to-transparent" />

      <div className="absolute inset-x-0 bottom-0 px-4 pb-12 sm:px-8 sm:pb-16 lg:px-12 lg:pb-20">
        <div className="mx-auto max-w-7xl">
          <div className="flex items-center gap-3 text-sm text-white/80">
            <span className="rounded bg-white/20 px-2 py-0.5 text-xs font-semibold uppercase tracking-wider">
              {item.mediaType === 'movie' ? '電影' : '影集'}
            </span>
            {year && <span data-testid="hero-banner-year">{year}</span>}
            {item.voteAverage > 0 && (
              <span className="flex items-center gap-1" data-testid="hero-banner-rating">
                <Star className="h-4 w-4 fill-[var(--warning)] text-[var(--warning)]" />
                {item.voteAverage.toFixed(1)}
              </span>
            )}
          </div>

          <h2
            className="mt-3 text-2xl font-bold text-white sm:text-4xl lg:text-5xl"
            data-testid="hero-banner-title"
          >
            {item.title}
          </h2>

          {item.overview && (
            <p
              className="mt-3 line-clamp-2 max-w-2xl text-sm text-white/90 sm:line-clamp-3 sm:text-base"
              data-testid="hero-banner-overview"
            >
              {item.overview}
            </p>
          )}

          <div className="mt-5 flex flex-wrap items-center gap-3">
            <button
              type="button"
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                onPlayClick(item);
              }}
              data-testid="hero-banner-play-trailer"
              className="flex items-center gap-2 rounded-full bg-white px-5 py-2 text-sm font-semibold text-black transition-colors hover:bg-white/90"
            >
              <Play className="h-4 w-4 fill-current" />
              觀看預告片
            </button>
            <Link
              to="/media/$type/$id"
              params={{ type: item.mediaType, id: String(item.id) }}
              data-testid="hero-banner-detail-link"
              className="rounded-full bg-white/20 px-5 py-2 text-sm font-semibold text-white backdrop-blur transition-colors hover:bg-white/30"
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
  const [isPaused, setIsPaused] = useState(false);
  const [trailerItem, setTrailerItem] = useState<HeroBannerItem | null>(null);
  const intervalRef = useRef<number | null>(null);

  const items = data ?? [];
  const hasItems = items.length > 0;

  // Auto-rotate every ROTATION_INTERVAL_MS unless paused or only one item.
  useEffect(() => {
    if (!hasItems || isPaused || items.length < 2 || trailerItem) return;
    intervalRef.current = window.setInterval(() => {
      setActiveIndex((prev) => (prev + 1) % items.length);
    }, ROTATION_INTERVAL_MS);
    return () => {
      if (intervalRef.current !== null) {
        window.clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [hasItems, items.length, isPaused, trailerItem]);

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
        className="relative h-[40vh] w-full overflow-hidden bg-[var(--bg-secondary)] sm:h-[50vh] lg:h-[70vh]"
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
        className="relative h-[40vh] w-full overflow-hidden bg-black sm:h-[50vh] lg:h-[70vh]"
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
          <div
            className="absolute bottom-4 left-1/2 z-10 flex -translate-x-1/2 gap-2"
            data-testid="hero-banner-dots"
          >
            {items.map((item, idx) => (
              <button
                key={item.mediaType + '-' + item.id + '-dot'}
                type="button"
                aria-label={`切換到第 ${idx + 1} 個推薦`}
                aria-current={idx === activeIndex}
                data-testid={`hero-banner-dot-${idx}`}
                onClick={() => setActiveIndex(idx)}
                className={cn(
                  'h-2 rounded-full transition-all',
                  idx === activeIndex ? 'w-8 bg-white' : 'w-2 bg-white/50 hover:bg-white/80'
                )}
              />
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

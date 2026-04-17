import { useState } from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { DownloadPanel, RecentMediaPanel } from '../components/dashboard';
import { NewMediaNotifications } from '../components/notifications/NewMediaNotifications';
import { useNewMediaNotifications } from '../hooks/useNewMediaNotifications';
import { QBStatusIndicator, ConnectionHistoryPanel } from '../components/health';
import { HeroBanner } from '../components/homepage/HeroBanner';
import { ExploreBlocksList } from '../components/homepage/ExploreBlocksList';
import { queryClient } from '../queryClient';
import { trendingKeys } from '../hooks/useTrending';
import tmdbService from '../services/tmdb';
import type { HeroBannerItem, Movie, TVShow } from '../types/tmdb';

// Story 10-5 Task 2.4 — router preload ('intent') fires this loader when the
// user hovers a Link to '/'. Seeding the trending cache means the HeroBanner
// has data in hand by the time it mounts, chopping a network roundtrip off LCP.
// Kept in sync with useTrendingHero: same query key, same shape, same filter.
// If that hook diverges, this loader will produce stale cache entries — when
// you change useTrendingHero, update this prefetch too.
const HERO_BANNER_LIMIT = 5;

export const Route = createFileRoute('/')({
  component: DashboardPage,
  loader: () => {
    void queryClient.prefetchQuery({
      queryKey: trendingKeys.hero('week'),
      queryFn: async () => {
        const [moviesRes, tvRes] = await Promise.all([
          tmdbService.getTrendingMovies('week', 1),
          tmdbService.getTrendingTVShows('week', 1),
        ]);
        const movies = moviesRes.results.map(movieToHeroItem);
        const tvShows = tvRes.results.map(tvShowToHeroItem);
        const merged: HeroBannerItem[] = [];
        const max = Math.max(movies.length, tvShows.length);
        for (let i = 0; i < max && merged.length < HERO_BANNER_LIMIT; i++) {
          if (movies[i] && movies[i].backdropPath) merged.push(movies[i]);
          if (merged.length >= HERO_BANNER_LIMIT) break;
          if (tvShows[i] && tvShows[i].backdropPath) merged.push(tvShows[i]);
        }
        return merged.slice(0, HERO_BANNER_LIMIT);
      },
      staleTime: 60 * 60 * 1000,
    });
    return null;
  },
});

function movieToHeroItem(movie: Movie): HeroBannerItem {
  return {
    id: movie.id,
    mediaType: 'movie',
    title: movie.title,
    overview: movie.overview,
    backdropPath: movie.backdropPath,
    releaseDate: movie.releaseDate,
    voteAverage: movie.voteAverage,
  };
}

function tvShowToHeroItem(tv: TVShow): HeroBannerItem {
  return {
    id: tv.id,
    mediaType: 'tv',
    title: tv.name,
    overview: tv.overview,
    backdropPath: tv.backdropPath,
    releaseDate: tv.firstAirDate,
    voteAverage: tv.voteAverage,
  };
}

/**
 * Homepage composition (Story 10-5 AC #1 / Task 1).
 *
 * Renders the four homepage sections as an independent vertical stack. Each
 * section owns its own data fetching (TanStack Query), its own loading
 * skeleton, and is responsible for hiding itself when empty (AC #5). The
 * outer flex wrapper gives the page consistent breathing room on mobile
 * (gap-6 = 24px) and desktop (gap-8 = 32px) per Task 1.3.
 */
function DashboardPage() {
  const { notifications, dismissNotification } = useNewMediaNotifications();
  const [historyOpen, setHistoryOpen] = useState(false);

  return (
    <div data-testid="homepage-root" className="flex flex-col gap-6 py-6 md:gap-8">
      {/* AC #1 section order: Hero → Explore → Recently Added → Downloads */}
      <HeroBanner />

      <ExploreBlocksList />

      <div className="mx-auto w-full max-w-7xl px-4 sm:px-6">
        <RecentMediaPanel hideWhenEmpty />
      </div>

      <div className="mx-auto w-full max-w-7xl px-4 sm:px-6">
        <DownloadPanel hideWhenEmpty />
      </div>

      {/* qBittorrent connection indicator — a small utility overlay that sits
          beneath the main sections; kept from the Epic 4 QB story and is not
          part of the AC #1 section order. */}
      <div className="mx-auto w-full max-w-7xl px-4 sm:px-6">
        <div className="flex justify-end">
          <QBStatusIndicator onClick={() => setHistoryOpen(true)} />
        </div>
      </div>

      <ConnectionHistoryPanel isOpen={historyOpen} onClose={() => setHistoryOpen(false)} />

      <NewMediaNotifications notifications={notifications} onDismiss={dismissNotification} />
    </div>
  );
}

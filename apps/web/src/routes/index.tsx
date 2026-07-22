import { createFileRoute } from '@tanstack/react-router';
import { HomeBrowseV2 } from '../components/homepage/HomeBrowseV2';
import { queryClient } from '../queryClient';
import { fetchTrendingHero, trendingKeys, HERO_BANNER_STALE_TIME_MS } from '../hooks/useTrending';

// Story 10-5 Task 2.4 — router preload ('intent') fires this loader when the
// user hovers a Link to '/'. Seeding the trending cache means the HeroBanner
// has data in hand by the time it mounts, chopping a network roundtrip off LCP.
// Uses the same fetchTrendingHero + trendingKeys + staleTime as useTrendingHero
// so the prefetch cannot drift from the hook. The Hero is kept in Home v2
// (below own-content, D3), so the prefetch still pays off.
export const Route = createFileRoute('/')({
  // ux3-cutover-3: legacy branch removed — HomeBrowseV2 is the only render.
  component: HomeBrowseV2,
  loader: () => {
    void queryClient.prefetchQuery({
      queryKey: trendingKeys.hero('week'),
      queryFn: () => fetchTrendingHero('week'),
      staleTime: HERO_BANNER_STALE_TIME_MS,
    });
    return null;
  },
});

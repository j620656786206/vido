import { useState } from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { DownloadPanel, RecentMediaPanel } from '../components/dashboard';
import { NewMediaNotifications } from '../components/notifications/NewMediaNotifications';
import { useNewMediaNotifications } from '../hooks/useNewMediaNotifications';
import { QBStatusIndicator, ConnectionHistoryPanel } from '../components/health';
import { HeroBanner } from '../components/homepage/HeroBanner';
import { ExploreBlocksList } from '../components/homepage/ExploreBlocksList';
import { HomeBrowseV2 } from '../components/homepage/HomeBrowseV2';
import { useShellVersion } from '../components/shell/shellVersion';
import { queryClient } from '../queryClient';
import { fetchTrendingHero, trendingKeys, HERO_BANNER_STALE_TIME_MS } from '../hooks/useTrending';

// Story 10-5 Task 2.4 — router preload ('intent') fires this loader when the
// user hovers a Link to '/'. Seeding the trending cache means the HeroBanner
// has data in hand by the time it mounts, chopping a network roundtrip off LCP.
// Uses the same fetchTrendingHero + trendingKeys + staleTime as useTrendingHero
// so the prefetch cannot drift from the hook. The Hero is kept in Home v2 (below
// own-content, D3), so the prefetch still pays off under both shells.
export const Route = createFileRoute('/')({
  component: HomeRoute,
  // ux3-1-2: migrated route — full-bleed under the v2 shell (LegacyContentContainer
  // opt-out). Content is gated by the shell version (NOT a second flag read, F4).
  staticData: { shell: 'v2' },
  loader: () => {
    void queryClient.prefetchQuery({
      queryKey: trendingKeys.hero('week'),
      queryFn: () => fetchTrendingHero('week'),
      staleTime: HERO_BANNER_STALE_TIME_MS,
    });
    return null;
  },
});

/**
 * Shell-version switch (ux3-1-2, mirrors library.tsx): under the v2 shell render
 * Home v2 (own-content above Hero/Explore — D3 ordering law); under the legacy shell
 * render the existing dashboard home byte-unchanged (P3 strangler discipline — flag
 * OFF leaves the current home exactly as-is, including the Epic 4 dashboard remnants).
 */
function HomeRoute() {
  const shell = useShellVersion();
  return shell === 'v2' ? <HomeBrowseV2 /> : <LegacyHome />;
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
function LegacyHome() {
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

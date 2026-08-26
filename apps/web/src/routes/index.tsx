import { createFileRoute } from '@tanstack/react-router';
import { HomeBrowseV2 } from '../components/homepage/HomeBrowseV2';
import { queryClient } from '../queryClient';
import { libraryKeys, RECENT_LIMIT, RECENT_STALE_TIME_MS } from '../hooks/useLibrary';
import { libraryService } from '../services/libraryService';

// Story 10-5 Task 2.4 lineage — router preload ('intent') fires this loader
// when the user hovers a Link to '/'. ux3-1-8: the hero reads the OWN library
// now (recently-added, shared with the 最近新增 row), so the prefetch seeds
// THAT query instead of the retired TMDb trending one. The limit and staleTime
// are IMPORTED from the hook rather than re-declared — a drifted value would
// seed a cache entry nothing reads and the prefetch would buy nothing while
// still looking correct.
export const Route = createFileRoute('/')({
  // ux3-cutover-3: legacy branch removed — HomeBrowseV2 is the only render.
  component: HomeBrowseV2,
  loader: () => {
    void queryClient.prefetchQuery({
      queryKey: libraryKeys.recent(RECENT_LIMIT),
      queryFn: () => libraryService.getRecentlyAdded(RECENT_LIMIT),
      staleTime: RECENT_STALE_TIME_MS,
    });
    return null;
  },
});

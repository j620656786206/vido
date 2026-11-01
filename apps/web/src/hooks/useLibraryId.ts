import { useQuery } from '@tanstack/react-query';
import { availabilityService } from '../services/availabilityService';

export const libraryIdKeys = {
  all: ['library-id'] as const,
  lookup: (type: 'movie' | 'tv', tmdbId: number) => [...libraryIdKeys.all, type, tmdbId] as const,
};

/**
 * The library id of a TMDb title the library owns, or null
 * (disc-2026-11-tmdb-detail-owned-no-link-to-library). The TMDb detail page
 * uses it to hand an owned title over to its library page, which has the
 * seasons, episodes and 生成字幕 the TMDb page cannot show.
 */
export function useLibraryId(type: 'movie' | 'tv', tmdbId: number) {
  return useQuery({
    queryKey: libraryIdKeys.lookup(type, tmdbId),
    queryFn: () => availabilityService.libraryId(type, tmdbId),
    enabled: tmdbId > 0,
    staleTime: 60 * 1000,
    retry: 1,
  });
}

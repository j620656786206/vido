/**
 * Infinite (continuous-scroll) library list hook (UX Redesign Phase 2 — UX2-2,
 * AC #11). Wraps the existing paginated `libraryService.listLibrary` in a
 * TanStack `useInfiniteQuery` so the v2 Browse grid loads continuously (no
 * pagination — competitive table-stakes) while reusing the unchanged list
 * endpoint (Rule 5; no backend change).
 */
import { useInfiniteQuery } from '@tanstack/react-query';
import { libraryService } from '../services/libraryService';
import type { LibraryItem, LibraryListParams } from '../types/library';

const PAGE_SIZE = 36;

export type LibraryInfiniteParams = Omit<LibraryListParams, 'page' | 'pageSize'>;

export function useLibraryInfinite(params: LibraryInfiniteParams) {
  const query = useInfiniteQuery({
    queryKey: ['library', 'infinite', params],
    queryFn: ({ pageParam }) =>
      libraryService.listLibrary({ ...params, page: pageParam, pageSize: PAGE_SIZE }),
    initialPageParam: 1,
    getNextPageParam: (lastPage) =>
      lastPage.page < lastPage.totalPages ? lastPage.page + 1 : undefined,
    staleTime: 30_000,
  });

  const items: LibraryItem[] = query.data?.pages.flatMap((p) => p.items) ?? [];
  const totalItems = query.data?.pages[0]?.totalItems ?? 0;

  return { ...query, items, totalItems };
}

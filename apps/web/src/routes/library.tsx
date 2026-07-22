import { createFileRoute, useMatchRoute, Outlet, redirect } from '@tanstack/react-router';
import type { LibraryMediaType } from '../types/library';
import { LibraryBrowseV2 } from '../components/library/LibraryBrowseV2';

interface LibrarySearchParams {
  page?: number;
  pageSize?: number;
  type?: LibraryMediaType;
  sortBy?: string;
  sortOrder?: string;
  view?: string;
  q?: string;
  genres?: string;
  yearMin?: number;
  yearMax?: number;
  unmatched?: boolean;
  /**
   * Forward-compatible deep-link target for the batch-subtitle "查看未找到項目"
   * link (Story 8-11 AC #6). Preserved here so the URL is valid; backend list
   * filtering by subtitle_status is a tracked follow-up (not yet wired).
   */
  subtitleStatus?: string;
}

export const Route = createFileRoute('/library')({
  validateSearch: (search: Record<string, unknown>): LibrarySearchParams => ({
    page: typeof search.page === 'number' ? search.page : 1,
    pageSize: typeof search.pageSize === 'number' ? search.pageSize : 20,
    type: ['all', 'movie', 'tv'].includes(search.type as string)
      ? (search.type as LibraryMediaType)
      : 'all',
    sortBy: typeof search.sortBy === 'string' ? search.sortBy : undefined,
    sortOrder: ['asc', 'desc'].includes(search.sortOrder as string)
      ? (search.sortOrder as string)
      : undefined,
    view: ['grid', 'list'].includes(search.view as string) ? (search.view as string) : undefined,
    q: typeof search.q === 'string' ? search.q : undefined,
    genres: typeof search.genres === 'string' ? search.genres : undefined,
    yearMin: typeof search.yearMin === 'number' ? search.yearMin : undefined,
    yearMax: typeof search.yearMax === 'number' ? search.yearMax : undefined,
    unmatched: search.unmatched === true ? true : undefined,
    subtitleStatus: typeof search.subtitleStatus === 'string' ? search.subtitleStatus : undefined,
  }),
  staticData: { shell: 'v2' },
  // ux3-0-5: old ?type= deep links → clean type routes (D2). Route-level redirect
  // (never a component redirect, F1); 'all'/absent stays at /library (merged view).
  beforeLoad: ({ search }) => {
    if (search.type === 'movie' || search.type === 'tv') {
      throw redirect({
        to: search.type === 'movie' ? '/library/movies' : '/library/tv',
        search: { ...search, type: undefined },
      });
    }
  },
  component: LibraryRoute,
});

/**
 * ux3-cutover-3: the legacy LibraryPage is gone — LibraryBrowseV2 is the only
 * render. ux3-0-5 / F5 layout discipline unchanged: the Browse UI is mounted
 * ONCE here in the layout so movies↔tv preserves filter + scroll state; the
 * active type derives from the matched clean child (path markers render null
 * via the Outlet).
 */
function LibraryRoute() {
  const matchRoute = useMatchRoute();
  const type: LibraryMediaType = matchRoute({ to: '/library/movies' })
    ? 'movie'
    : matchRoute({ to: '/library/tv' })
      ? 'tv'
      : 'all';
  return (
    <>
      <LibraryBrowseV2 type={type} />
      <Outlet />
    </>
  );
}
